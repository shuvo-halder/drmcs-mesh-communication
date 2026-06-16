package models

import "time"

// Node represents a device in the mesh network
type Node struct {
	NodeID    string    `json:"node_id"`
	Name      string    `json:"name"`
	IPAddress string    `json:"ip_address"`
	Port      int       `json:"port"`
	PublicKey []byte    `json:"public_key"`
	Status    string    `json:"status"` // "active", "inactive", "unknown"
	LastSeen  time.Time `json:"last_seen"`
	HopCount  int       `json:"hop_count"`
}

// Peer represents a discovered neighboring node
type Peer struct {
	PeerID    string    `json:"peer_id"`
	NodeID    string    `json:"node_id"`
	IPAddress string    `json:"ip_address"`
	Port      int       `json:"port"`
	PublicKey []byte    `json:"public_key"`
	Status    string    `json:"status"`
	LastSeen  time.Time `json:"last_seen"`
	RTT       int64     `json:"rtt_ms"` // Round trip time in milliseconds
}

// Message represents a communication message between nodes
type Message struct {
	MessageID  string    `json:"message_id"`
	SenderID   string    `json:"sender_id"`
	ReceiverID string    `json:"receiver_id"`
	Timestamp  time.Time `json:"timestamp"`
	Content    string    `json:"content"`
	MsgType    string    `json:"msg_type"` // "text", "alert", "file", "system"
	Priority   int       `json:"priority"` // 0=normal, 1=important, 2=critical
	Status     string    `json:"status"`   // "pending", "sent", "delivered", "failed"
	HopCount   int       `json:"hop_count"`
	Signature  []byte    `json:"signature"`
}

// Alert represents an emergency alert
type Alert struct {
	AlertID    string    `json:"alert_id"`
	SenderID   string    `json:"sender_id"`
	Timestamp  time.Time `json:"timestamp"`
	AlertType  string    `json:"alert_type"`
	Priority   int       `json:"priority"`
	Message    string    `json:"message"`
	Location   string    `json:"location"`
	ExpiresAt  time.Time `json:"expires_at"`
	ReceivedAt time.Time `json:"received_at"`
}

// FileTransfer represents a file being shared on the network
type FileTransfer struct {
	FileID       string    `json:"file_id"`
	SenderID     string    `json:"sender_id"`
	Filename     string    `json:"filename"`
	FileSize     int64     `json:"file_size"`
	ContentType  string    `json:"content_type"`
	ChunkCount   int       `json:"chunk_count"`
	Checksum     string    `json:"checksum"`
	ExpiresAt    time.Time `json:"expires_at"`
	Status       string    `json:"status"` // "active", "completed", "expired", "failed"
	Progress     float64   `json:"progress"`
}

// FileChunk represents a chunk of a file transfer
type FileChunk struct {
	ChunkID   int    `json:"chunk_id"`
	FileID    string `json:"file_id"`
	Data      []byte `json:"data"`
	Checksum  string `json:"checksum"`
	Size      int    `json:"size"`
}

// Route represents a network route to a destination node
type Route struct {
	DestinationID string    `json:"destination_id"`
	NextHopID     string    `json:"next_hop_id"`
	HopCount      int       `json:"hop_count"`
	SequenceNum   uint32    `json:"sequence_num"`
	LastUpdated   time.Time `json:"last_updated"`
	ExpiresAt     time.Time `json:"expires_at"`
	Status        string    `json:"status"` // "active", "stale", "invalid"
}

// RouteRequest (RREQ) for AODV route discovery
type RouteRequest struct {
	RequestID      uint32    `json:"request_id"`
	SourceID       string    `json:"source_id"`
	DestinationID  string    `json:"destination_id"`
	SequenceNum    uint32    `json:"sequence_num"`
	HopCount       int       `json:"hop_count"`
	TTL            int       `json:"ttl"`
	BroadcastID    uint32    `json:"broadcast_id"`
}

// RouteReply (RREP) for AODV route reply
type RouteReply struct {
	RequestID      uint32    `json:"request_id"`
	SourceID       string    `json:"source_id"`
	DestinationID  string    `json:"destination_id"`
	SequenceNum    uint32    `json:"sequence_num"`
	HopCount       int       `json:"hop_count"`
	Lifetime       int64     `json:"lifetime"`
}

// AnalyticsData represents collected network statistics
type AnalyticsData struct {
	NodeID        string    `json:"node_id"`
	Timestamp     time.Time `json:"timestamp"`
	ActiveNodes   int       `json:"active_nodes"`
	MsgSent       int64     `json:"msg_sent"`
	MsgReceived   int64     `json:"msg_received"`
	MsgDropped    int64     `json:"msg_dropped"`
	AvgLatency    float64   `json:"avg_latency_ms"`
	PacketLoss    float64   `json:"packet_loss_pct"`
	FileTransfers int       `json:"file_transfers"`
	AlertsSent    int       `json:"alerts_sent"`
	AlertsReceived int      `json:"alerts_received"`
}

// DiscoveryPacket is the UDP broadcast packet for peer discovery
type DiscoveryPacket struct {
	NodeID    string `json:"node_id"`
	NodeName  string `json:"node_name"`
	Port      int    `json:"port"`
	PublicKey []byte `json:"public_key"`
	Timestamp int64  `json:"timestamp"`
	Signature []byte `json:"signature"`
}

// Alert types constants
const (
	AlertTypeMedical   = "medical_emergency"
	AlertTypeFire      = "fire_alert"
	AlertTypeFlood     = "flood_warning"
	AlertTypeRescue    = "rescue_request"
	AlertTypeMissing   = "missing_person"
	AlertTypeGeneric   = "emergency"
)

// Priority constants
const (
	PriorityNormal   = 0
	PriorityHigh     = 1
	PriorityCritical = 2
)

// Status constants
const (
	StatusActive   = "active"
	StatusInactive = "inactive"
	StatusPending  = "pending"
	StatusSent     = "sent"
	StatusDelivered = "delivered"
	StatusFailed   = "failed"
	StatusExpired  = "expired"
)