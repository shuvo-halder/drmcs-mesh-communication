package alerts

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/drmcs/backend/internal/crypto"
	"github.com/drmcs/backend/internal/messaging"
	"github.com/drmcs/backend/internal/models"
	"github.com/drmcs/backend/internal/storage"
)

const (
	alertDefaultExpiry = 1 * time.Hour
	alertMaxExpiry     = 24 * time.Hour
	alertCleanupInt    = 10 * time.Minute
)

// System manages emergency alerts
type System struct {
	nodeID   string
	msgHndlr *messaging.Handler
	store    *storage.SQLiteStore
	mu       sync.RWMutex
	stopCh   chan struct{}
	// Track seen alert IDs for duplicate prevention
	seenAlerts map[string]bool
}

// NewSystem creates a new alert system
func NewSystem(nodeID string, msgHndlr *messaging.Handler, store *storage.SQLiteStore) *System {
	return &System{
		nodeID:     nodeID,
		msgHndlr:   msgHndlr,
		store:      store,
		stopCh:     make(chan struct{}),
		seenAlerts: make(map[string]bool),
	}
}

// Start begins the alert system
func (s *System) Start() {
	// Subscribe to incoming messages for alert detection
	s.msgHndlr.RegisterHandler(func(msg *models.Message) {
		if msg.MsgType == "alert" {
			s.processIncomingAlert(msg)
		}
	})

	// Periodic cleanup
	go s.cleanupLoop()

	log.Println("Alert system started")
}

// Stop halts the alert system
func (s *System) Stop() {
	close(s.stopCh)
}

// SendAlert creates and broadcasts an emergency alert
func (s *System) SendAlert(alertType, message, location string, priority int) (*models.Alert, error) {
	alertID := crypto.HashContent([]byte(fmt.Sprintf("%s:%s:%d", s.nodeID, alertType, time.Now().UnixNano())))

	expiry := alertDefaultExpiry
	if priority >= models.PriorityCritical {
		expiry = alertMaxExpiry
	}

	alert := &models.Alert{
		AlertID:   alertID,
		SenderID:  s.nodeID,
		Timestamp: time.Now(),
		AlertType: alertType,
		Priority:  priority,
		Message:   message,
		Location:  location,
		ExpiresAt: time.Now().Add(expiry),
	}

	// Store alert
	if err := s.store.SaveAlert(alert); err != nil {
		return nil, fmt.Errorf("failed to save alert: %w", err)
	}

	// Send as a broadcast alert message
	alertContent := fmt.Sprintf("ALERT|%s|%s|%s|%d|%s",
		alert.AlertID, alert.AlertType, s.nodeID, priority, message)

	_, err := s.msgHndlr.SendMessage("broadcast", alertContent, "alert", priority)
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast alert: %w", err)
	}

	log.Printf("EMERGENCY ALERT sent: [%s] %s (priority: %d)", alertType, message, priority)
	return alert, nil
}

// SendMedicalAlert sends a medical emergency alert
func (s *System) SendMedicalAlert(message, location string) (*models.Alert, error) {
	return s.SendAlert(models.AlertTypeMedical, message, location, models.PriorityCritical)
}

// SendFireAlert sends a fire emergency alert
func (s *System) SendFireAlert(message, location string) (*models.Alert, error) {
	return s.SendAlert(models.AlertTypeFire, message, location, models.PriorityCritical)
}

// SendFloodWarning sends a flood warning alert
func (s *System) SendFloodWarning(message, location string) (*models.Alert, error) {
	return s.SendAlert(models.AlertTypeFlood, message, location, models.PriorityHigh)
}

// SendRescueRequest sends a rescue request
func (s *System) SendRescueRequest(message, location string) (*models.Alert, error) {
	return s.SendAlert(models.AlertTypeRescue, message, location, models.PriorityCritical)
}

// SendMissingPersonAlert sends a missing person notification
func (s *System) SendMissingPersonAlert(description, location string) (*models.Alert, error) {
	return s.SendAlert(models.AlertTypeMissing, description, location, models.PriorityHigh)
}

// GetActiveAlerts returns all non-expired alerts
func (s *System) GetActiveAlerts() ([]*models.Alert, error) {
	return s.store.GetActiveAlerts()
}

// AlertHandlers returns a list of available alert types and example usages
func (s *System) AlertHandlers() map[string]string {
	return map[string]string{
		"medical":   "Medical emergency - requires immediate response",
		"fire":      "Fire alert - immediate evacuation may be needed",
		"flood":     "Flood warning - rising water levels detected",
		"rescue":    "Rescue request - victims need assistance",
		"missing":   "Missing person - search and rescue needed",
		"emergency": "General emergency alert",
	}
}

func (s *System) processIncomingAlert(msg *models.Message) {
	// Parse alert from message content
	// Format: ALERT|alert_id|alert_type|sender_id|priority|message
	var alertID, alertType, senderID string
	var priority int
	var content string

	n, _ := fmt.Sscanf(msg.Content, "ALERT|%s|%s|%s|%d|%s",
		&alertID, &alertType, &senderID, &priority, &content)

	if n < 4 {
		return
	}

	// Duplicate prevention
	s.mu.Lock()
	if s.seenAlerts[alertID] {
		s.mu.Unlock()
		return
	}
	s.seenAlerts[alertID] = true
	s.mu.Unlock()

	alert := &models.Alert{
		AlertID:   alertID,
		SenderID:  senderID,
		Timestamp: msg.Timestamp,
		AlertType: alertType,
		Priority:  priority,
		Message:   content,
		ExpiresAt: time.Now().Add(alertDefaultExpiry),
	}

	s.store.SaveAlert(alert)
	log.Printf("ALERT RECEIVED: [%s] from %s: %s", alertType, senderID, alert.AlertID)
}

func (s *System) cleanupLoop() {
	ticker := time.NewTicker(alertCleanupInt)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.cleanupExpiredAlerts()
		}
	}
}

func (s *System) cleanupExpiredAlerts() {
	// Clean up seen alerts map periodically
	s.mu.Lock()
	for id := range s.seenAlerts {
		if len(s.seenAlerts) > 10000 {
			delete(s.seenAlerts, id)
		}
	}
	s.mu.Unlock()
}