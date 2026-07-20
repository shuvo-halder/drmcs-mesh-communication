package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/drmcs/backend/internal/alerts"
	"github.com/drmcs/backend/internal/discovery"
	"github.com/drmcs/backend/internal/fileshare"
	"github.com/drmcs/backend/internal/messaging"
	"github.com/drmcs/backend/internal/models"
	"github.com/drmcs/backend/internal/routing"
	"github.com/drmcs/backend/internal/storage"
)

// Server provides REST/gRPC API endpoints for the mesh node
type Server struct {
	store        *storage.SQLiteStore
	nodeID       string
	discoverySvc *discovery.Service
	msgHndlr     *messaging.Handler
	alertSys     *alerts.System
	fileTransfer *fileshare.Transfer
	router       *routing.Router
	httpServer   *http.Server
	wsClients    map[chan []byte]bool
	wsMu         sync.RWMutex
}

// NewServer creates a new API server
func NewServer(store *storage.SQLiteStore, nodeID string, discoverySvc *discovery.Service,
	msgHndlr *messaging.Handler, alertSys *alerts.System, fileTransfer *fileshare.Transfer,
	router *routing.Router) *Server {

	return &Server{
		store:        store,
		nodeID:       nodeID,
		discoverySvc: discoverySvc,
		msgHndlr:     msgHndlr,
		alertSys:     alertSys,
		fileTransfer: fileTransfer,
		router:       router,
		wsClients:    make(map[chan []byte]bool),
	}
}

// Start begins the HTTP API server
func (s *Server) Start(port int) {
	mux := http.NewServeMux()

	// Node endpoints
	mux.HandleFunc("/api/v1/node/info", s.handleNodeInfo)
	mux.HandleFunc("/api/v1/node/peers", s.handleGetPeers)

	// Message endpoints
	mux.HandleFunc("/api/v1/messages", s.handleMessages)       // GET, POST
	mux.HandleFunc("/api/v1/messages/send", s.handleSendMessage)

	// Alert endpoints
	mux.HandleFunc("/api/v1/alerts", s.handleAlerts)           // GET, POST
	mux.HandleFunc("/api/v1/alerts/send", s.handleSendAlert)

	// File transfer endpoints
	mux.HandleFunc("/api/v1/files", s.handleFileTransfers)     // GET, POST
	mux.HandleFunc("/api/v1/files/upload", s.handleFileUpload)
	mux.HandleFunc("/api/v1/files/download", s.handleFileDownload)

	// Routing endpoints
	mux.HandleFunc("/api/v1/routes", s.handleGetRoutes)

	// Analytics endpoints
	mux.HandleFunc("/api/v1/analytics", s.handleGetAnalytics)

	// WebSocket endpoint
	mux.HandleFunc("/api/v1/ws", s.handleWebSocket)

	// Health check
	mux.HandleFunc("/api/v1/health", s.handleHealth)

	addr := fmt.Sprintf(":%d", port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: corsMiddleware(mux),
	}

	log.Printf("API server listening on %s", addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("API server error: %v", err)
	}
}

// Stop gracefully shuts down the API server
func (s *Server) Stop() {
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.httpServer.Shutdown(ctx)
	}
}

// --- Handlers ---

func (s *Server) handleNodeInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	peers := s.discoverySvc.GetPeers()
	routes := s.router.GetRoutingTable()

	info := map[string]interface{}{
		"node_id":         s.nodeID,
		"active_peers":    len(peers),
		"active_routes":   len(routes),
		"uptime":          time.Now().Unix(),
	}

	json.NewEncoder(w).Encode(info)
}

func (s *Server) handleGetPeers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	peers := s.discoverySvc.GetPeers()
	json.NewEncoder(w).Encode(peers)
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		msgs, err := s.msgHndlr.GetMessages()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(msgs)

	case http.MethodPost:
		var req struct {
			ReceiverID string `json:"receiver_id"`
			Content    string `json:"content"`
			MsgType    string `json:"msg_type"`
			Priority   int    `json:"priority"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		msg, err := s.msgHndlr.SendMessage(req.ReceiverID, req.Content, req.MsgType, req.Priority)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(msg)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	var req struct {
		ReceiverID string `json:"receiver_id"`
		Content    string `json:"content"`
		MsgType    string `json:"msg_type"`
		Priority   int    `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	msg, err := s.msgHndlr.SendMessage(req.ReceiverID, req.Content, req.MsgType, req.Priority)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(msg)
}

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		alerts, err := s.alertSys.GetActiveAlerts()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(alerts)

	case http.MethodPost:
		var req struct {
			AlertType string `json:"alert_type"`
			Message   string `json:"message"`
			Location  string `json:"location"`
			Priority  int    `json:"priority"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		alert, err := s.alertSys.SendAlert(req.AlertType, req.Message, req.Location, req.Priority)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(alert)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSendAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	var req struct {
		AlertType string `json:"alert_type"`
		Message   string `json:"message"`
		Location  string `json:"location"`
		Priority  int    `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	alert, err := s.alertSys.SendAlert(req.AlertType, req.Message, req.Location, req.Priority)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(alert)
}

func (s *Server) handleFileTransfers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// For now return empty list
	json.NewEncoder(w).Encode([]models.FileTransfer{})
}

func (s *Server) handleFileUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Parse multipart form
	err := r.ParseMultipartForm(50 << 20) // 50MB max
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Save to temp location and upload
	tmpPath := fmt.Sprintf("./tmp_%s", header.Filename)
	// In production, use proper temp file management

	ft, err := s.fileTransfer.UploadFile(tmpPath, 2*time.Hour)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(ft)
}

func (s *Server) handleFileDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fileID := r.URL.Query().Get("file_id")
	if fileID == "" {
		http.Error(w, "file_id required", http.StatusBadRequest)
		return
	}

	ft, err := s.fileTransfer.GetTransferStatus(fileID)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", ft.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", ft.Filename))
	json.NewEncoder(w).Encode(ft)
}

func (s *Server) handleGetRoutes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	routes := s.router.GetRoutingTable()
	json.NewEncoder(w).Encode(routes)
}

func (s *Server) handleGetAnalytics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	data, err := s.store.GetRecentAnalytics(50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(data)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Simple SSE (Server-Sent Events) for real-time updates, no external deps needed
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	ch := make(chan []byte, 64)
	s.wsMu.Lock()
	s.wsClients[ch] = true
	s.wsMu.Unlock()

	// Send initial data
	initialData, _ := json.Marshal(map[string]interface{}{
		"type":    "init",
		"node_id": s.nodeID,
	})
	fmt.Fprintf(w, "data: %s\n\n", initialData)
	flusher.Flush()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			s.wsMu.Lock()
			delete(s.wsClients, ch)
			s.wsMu.Unlock()
			close(ch)
			return
		case data := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// BroadcastEvent sends an event to all connected WebSocket clients
func (s *Server) BroadcastEvent(eventType string, data interface{}) {
	payload, err := json.Marshal(map[string]interface{}{
		"type": eventType,
		"data": data,
		"time": time.Now().Unix(),
	})
	if err != nil {
		return
	}

	s.wsMu.RLock()
	defer s.wsMu.RUnlock()

	for ch := range s.wsClients {
		select {
		case ch <- payload:
		default:
			// Skip slow clients
		}
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"node_id": s.nodeID,
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}