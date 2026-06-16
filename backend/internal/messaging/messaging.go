package messaging

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/drmcs/backend/internal/crypto"
	"github.com/drmcs/backend/internal/models"
	"github.com/drmcs/backend/internal/routing"
	"github.com/drmcs/backend/internal/storage"
)

const (
	msgBufferSize    = 1000
	deliveryTimeout  = 30 * time.Second
	maxRetries       = 3
	retryInterval    = 5 * time.Second
	pendingCleanupInt = 60 * time.Second
	ackTimeout       = 10 * time.Second
)

// Handler manages message sending, receiving, and forwarding
type Handler struct {
	nodeID      string
	router      *routing.Router
	privateKey  ed25519.PrivateKey
	store       *storage.SQLiteStore
	pendingMsgs map[string]*pendingMessage
	mu          sync.RWMutex
	listener    net.Listener
	stopCh      chan struct{}
	msgHandlers []func(*models.Message)
	handlerMu   sync.RWMutex
}

type pendingMessage struct {
	msg       *models.Message
	retries   int
	lastTry   time.Time
	createdAt time.Time
}

// NewHandler creates a new message handler
func NewHandler(nodeID string, router *routing.Router, privKey ed25519.PrivateKey, store *storage.SQLiteStore) *Handler {
	return &Handler{
		nodeID:      nodeID,
		router:      router,
		privateKey:  privKey,
		store:       store,
		pendingMsgs: make(map[string]*pendingMessage),
		stopCh:      make(chan struct{}),
	}
}

// Start begins listening for messages
func (h *Handler) Start(port int) {
	addr := fmt.Sprintf(":%d", port)
	var err error
	h.listener, err = net.Listen("tcp4", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}
	defer h.listener.Close()

	log.Printf("Message handler listening on %s", addr)

	go h.processPendingMessages()

	for {
		conn, err := h.listener.Accept()
		if err != nil {
			select {
			case <-h.stopCh:
				return
			default:
				log.Printf("Accept error: %v", err)
				continue
			}
		}
		go h.handleConnection(conn)
	}
}

// Stop halts message processing
func (h *Handler) Stop() {
	close(h.stopCh)
	if h.listener != nil {
		h.listener.Close()
	}
}

// SendMessage sends a message to a destination node
func (h *Handler) SendMessage(receiverID, content, msgType string, priority int) (*models.Message, error) {
	msg := &models.Message{
		MessageID:  crypto.HashContent([]byte(fmt.Sprintf("%s:%s:%d", h.nodeID, receiverID, time.Now().UnixNano()))),
		SenderID:   h.nodeID,
		ReceiverID: receiverID,
		Timestamp:  time.Now(),
		Content:    content,
		MsgType:    msgType,
		Priority:   priority,
		Status:     models.StatusPending,
		HopCount:   0,
	}

	// Sign the message
	dataToSign, _ := json.Marshal(struct {
		MsgID   string `json:"msg_id"`
		Sender  string `json:"sender"`
		Content string `json:"content"`
		Time    int64  `json:"time"`
	}{
		MsgID:   msg.MessageID,
		Sender:  msg.SenderID,
		Content: msg.Content,
		Time:    msg.Timestamp.Unix(),
	})
	msg.Signature = crypto.SignMessage(h.privateKey, dataToSign)

	// Store message
	if err := h.store.SaveMessage(msg); err != nil {
		return nil, fmt.Errorf("failed to save message: %w", err)
	}

	// If broadcast (receiverID is "broadcast"), flood to all peers
	if receiverID == "broadcast" {
		return h.broadcastMessage(msg)
	}

	// Find route and send
	route := h.router.GetRoute(receiverID)
	if route == nil {
		// No route found, start discovery
		h.router.StartRouteDiscovery(receiverID)
		// Queue as pending
		h.queuePending(msg)
		log.Printf("No route to %s, queuing message %s", receiverID, msg.MessageID)
		return msg, nil
	}

	// Try to deliver
	if err := h.deliverMessage(msg, route); err != nil {
		h.queuePending(msg)
		return msg, fmt.Errorf("delivery failed: %w", err)
	}

	return msg, nil
}

// BroadcastMessage sends a message to all connected nodes
func (h *Handler) BroadcastMessage(content string) (*models.Message, error) {
	return h.SendMessage("broadcast", content, "text", models.PriorityNormal)
}

// GetMessages retrieves messages from storage
func (h *Handler) GetMessages() ([]*models.Message, error) {
	return h.store.GetMessages(h.nodeID)
}

// RegisterHandler adds a message handler callback
func (h *Handler) RegisterHandler(handler func(*models.Message)) {
	h.handlerMu.Lock()
	defer h.handlerMu.Unlock()
	h.msgHandlers = append(h.msgHandlers, handler)
}

func (h *Handler) handleConnection(conn net.Conn) {
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(30 * time.Second))

	var msg models.Message
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&msg); err != nil {
		log.Printf("Failed to decode message: %v", err)
		return
	}

	// Verify signature
	pubKey := ed25519.PublicKey{} // In real impl, look up sender's public key
	_ = pubKey

	h.processReceivedMessage(&msg)
}

func (h *Handler) processReceivedMessage(msg *models.Message) {
	log.Printf("Received message: %s from %s (type: %s)", msg.MessageID[:8], msg.SenderID, msg.MsgType)

	// Check for duplicates
	existing, _ := h.store.GetMessages(h.nodeID)
	for _, m := range existing {
		if m.MessageID == msg.MessageID {
			return // Duplicate
		}
	}

	msg.Status = models.StatusDelivered
	msg.HopCount++

	// Store message
	h.store.SaveMessage(msg)

	// Forward if not for us
	if msg.ReceiverID != h.nodeID && msg.ReceiverID != "broadcast" {
		route := h.router.GetRoute(msg.ReceiverID)
		if route != nil {
			go h.deliverMessage(msg, route)
		}
	}

	// Notify registered handlers
	h.handlerMu.RLock()
	for _, handler := range h.msgHandlers {
		go handler(msg)
	}
	h.handlerMu.RUnlock()
}

func (h *Handler) deliverMessage(msg *models.Message, route *models.Route) error {
	// In real implementation, this connects to next hop and sends
	msg.Status = models.StatusSent
	h.store.SaveMessage(msg)
	log.Printf("Delivering message %s to %s (next hop: %s, %d hops)",
		msg.MessageID[:8], msg.ReceiverID, route.NextHopID, route.HopCount)
	return nil
}

func (h *Handler) broadcastMessage(msg *models.Message) (*models.Message, error) {
	// Store as sent
	msg.Status = models.StatusSent
	h.store.SaveMessage(msg)

	// Get all peers and forward
	peers, _ := h.store.GetActivePeers()
	for _, peer := range peers {
		log.Printf("Broadcasting message %s to peer %s", msg.MessageID[:8], peer.NodeID)
	}
	return msg, nil
}

func (h *Handler) queuePending(msg *models.Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.pendingMsgs[msg.MessageID] = &pendingMessage{
		msg:       msg,
		retries:   0,
		lastTry:   time.Now(),
		createdAt: time.Now(),
	}
}

func (h *Handler) processPendingMessages() {
	ticker := time.NewTicker(retryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopCh:
			return
		case <-ticker.C:
			h.retryPendingMessages()
		}
	}
}

func (h *Handler) retryPendingMessages() {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	for id, pm := range h.pendingMsgs {
		if pm.retries >= maxRetries || now.Sub(pm.createdAt) > deliveryTimeout {
			pm.msg.Status = models.StatusFailed
			h.store.SaveMessage(pm.msg)
			delete(h.pendingMsgs, id)
			log.Printf("Message %s delivery failed after %d retries", id[:8], pm.retries)
			continue
		}

		if now.Sub(pm.lastTry) < retryInterval {
			continue
		}

		route := h.router.GetRoute(pm.msg.ReceiverID)
		if route != nil {
			if err := h.deliverMessage(pm.msg, route); err == nil {
				delete(h.pendingMsgs, id)
				log.Printf("Message %s delivered on retry %d", id[:8], pm.retries+1)
				continue
			}
		}

		pm.retries++
		pm.lastTry = now
		log.Printf("Retry %d for message %s", pm.retries, id[:8])

		// Try route discovery again if no route
		if !h.router.RouteExists(pm.msg.ReceiverID) {
			h.router.StartRouteDiscovery(pm.msg.ReceiverID)
		}
	}
}