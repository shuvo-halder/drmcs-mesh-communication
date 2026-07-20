package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/drmcs/backend/internal/models"
	_ "github.com/glebarez/go-sqlite"
)

// SQLiteStore implements data persistence using SQLite
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite database store
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return store, nil
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) createTables() error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS peers (
			peer_id TEXT PRIMARY KEY,
			node_id TEXT NOT NULL,
			ip_address TEXT NOT NULL,
			port INTEGER NOT NULL,
			public_key BLOB,
			status TEXT DEFAULT 'active',
			last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
			rtt_ms INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			message_id TEXT PRIMARY KEY,
			sender_id TEXT NOT NULL,
			receiver_id TEXT NOT NULL,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			content TEXT,
			msg_type TEXT DEFAULT 'text',
			priority INTEGER DEFAULT 0,
			status TEXT DEFAULT 'pending',
			hop_count INTEGER DEFAULT 0,
			signature BLOB
		)`,
		`CREATE TABLE IF NOT EXISTS alerts (
			alert_id TEXT PRIMARY KEY,
			sender_id TEXT NOT NULL,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			alert_type TEXT NOT NULL,
			priority INTEGER DEFAULT 0,
			message TEXT,
			location TEXT,
			expires_at DATETIME,
			received_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS file_transfers (
			file_id TEXT PRIMARY KEY,
			sender_id TEXT NOT NULL,
			filename TEXT NOT NULL,
			file_size INTEGER DEFAULT 0,
			content_type TEXT,
			chunk_count INTEGER DEFAULT 0,
			checksum TEXT,
			expires_at DATETIME,
			status TEXT DEFAULT 'active',
			progress REAL DEFAULT 0.0
		)`,
		`CREATE TABLE IF NOT EXISTS routes (
			destination_id TEXT NOT NULL,
			next_hop_id TEXT NOT NULL,
			hop_count INTEGER DEFAULT 0,
			sequence_num INTEGER DEFAULT 0,
			last_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME,
			status TEXT DEFAULT 'active',
			PRIMARY KEY (destination_id, next_hop_id)
		)`,
		`CREATE TABLE IF NOT EXISTS analytics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			node_id TEXT NOT NULL,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			active_nodes INTEGER DEFAULT 0,
			msg_sent INTEGER DEFAULT 0,
			msg_received INTEGER DEFAULT 0,
			msg_dropped INTEGER DEFAULT 0,
			avg_latency REAL DEFAULT 0.0,
			packet_loss REAL DEFAULT 0.0,
			file_transfers INTEGER DEFAULT 0,
			alerts_sent INTEGER DEFAULT 0,
			alerts_received INTEGER DEFAULT 0
		)`,
	}

	for _, table := range tables {
		if _, err := s.db.Exec(table); err != nil {
			return err
		}
	}
	return nil
}

// SavePeer inserts or updates a peer record
func (s *SQLiteStore) SavePeer(peer *models.Peer) error {
	query := `INSERT INTO peers (peer_id, node_id, ip_address, port, public_key, status, last_seen, rtt_ms)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			  ON CONFLICT(peer_id) DO UPDATE SET
			  	status=excluded.status, last_seen=excluded.last_seen, rtt_ms=excluded.rtt_ms`
	_, err := s.db.Exec(query, peer.PeerID, peer.NodeID, peer.IPAddress, peer.Port,
		peer.PublicKey, peer.Status, peer.LastSeen, peer.RTT)
	return err
}

// GetActivePeers returns all active peers
func (s *SQLiteStore) GetActivePeers() ([]*models.Peer, error) {
	rows, err := s.db.Query("SELECT peer_id, node_id, ip_address, port, public_key, status, last_seen, rtt_ms FROM peers WHERE status = 'active'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var peers []*models.Peer
	for rows.Next() {
		p := &models.Peer{}
		err := rows.Scan(&p.PeerID, &p.NodeID, &p.IPAddress, &p.Port, &p.PublicKey, &p.Status, &p.LastSeen, &p.RTT)
		if err != nil {
			return nil, err
		}
		peers = append(peers, p)
	}
	return peers, nil
}

// RemoveInactivePeers marks peers as inactive if not seen since threshold
func (s *SQLiteStore) RemoveInactivePeers(threshold time.Duration) error {
	cutoff := time.Now().Add(-threshold)
	_, err := s.db.Exec("UPDATE peers SET status = 'inactive' WHERE last_seen < ?", cutoff)
	return err
}

// SaveMessage stores a message
func (s *SQLiteStore) SaveMessage(msg *models.Message) error {
	query := `INSERT OR IGNORE INTO messages (message_id, sender_id, receiver_id, timestamp, content, msg_type, priority, status, hop_count, signature)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, msg.MessageID, msg.SenderID, msg.ReceiverID, msg.Timestamp,
		msg.Content, msg.MsgType, msg.Priority, msg.Status, msg.HopCount, msg.Signature)
	return err
}

// GetMessages returns messages for a given receiver
func (s *SQLiteStore) GetMessages(receiverID string) ([]*models.Message, error) {
	rows, err := s.db.Query("SELECT message_id, sender_id, receiver_id, timestamp, content, msg_type, priority, status, hop_count FROM messages WHERE receiver_id = ? ORDER BY timestamp DESC LIMIT 100", receiverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*models.Message
	for rows.Next() {
		m := &models.Message{}
		err := rows.Scan(&m.MessageID, &m.SenderID, &m.ReceiverID, &m.Timestamp,
			&m.Content, &m.MsgType, &m.Priority, &m.Status, &m.HopCount)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}

// SaveAlert stores an alert
func (s *SQLiteStore) SaveAlert(alert *models.Alert) error {
	query := `INSERT OR IGNORE INTO alerts (alert_id, sender_id, timestamp, alert_type, priority, message, location, expires_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, alert.AlertID, alert.SenderID, alert.Timestamp,
		alert.AlertType, alert.Priority, alert.Message, alert.Location, alert.ExpiresAt)
	return err
}

// GetActiveAlerts returns non-expired alerts
func (s *SQLiteStore) GetActiveAlerts() ([]*models.Alert, error) {
	rows, err := s.db.Query("SELECT alert_id, sender_id, timestamp, alert_type, priority, message, location, expires_at FROM alerts WHERE expires_at > datetime('now') ORDER BY priority DESC, timestamp DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*models.Alert
	for rows.Next() {
		a := &models.Alert{}
		err := rows.Scan(&a.AlertID, &a.SenderID, &a.Timestamp, &a.AlertType,
			&a.Priority, &a.Message, &a.Location, &a.ExpiresAt)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}
	return alerts, nil
}

// SaveFileTransfer stores a file transfer record
func (s *SQLiteStore) SaveFileTransfer(ft *models.FileTransfer) error {
	query := `INSERT OR REPLACE INTO file_transfers (file_id, sender_id, filename, file_size, content_type, chunk_count, checksum, expires_at, status, progress)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, ft.FileID, ft.SenderID, ft.Filename, ft.FileSize,
		ft.ContentType, ft.ChunkCount, ft.Checksum, ft.ExpiresAt, ft.Status, ft.Progress)
	return err
}

// GetFileTransfer retrieves a file transfer record
func (s *SQLiteStore) GetFileTransfer(fileID string) (*models.FileTransfer, error) {
	ft := &models.FileTransfer{}
	err := s.db.QueryRow("SELECT file_id, sender_id, filename, file_size, content_type, chunk_count, checksum, expires_at, status, progress FROM file_transfers WHERE file_id = ?", fileID).
		Scan(&ft.FileID, &ft.SenderID, &ft.Filename, &ft.FileSize, &ft.ContentType,
			&ft.ChunkCount, &ft.Checksum, &ft.ExpiresAt, &ft.Status, &ft.Progress)
	if err != nil {
		return nil, err
	}
	return ft, nil
}

// SaveRoute stores a route
func (s *SQLiteStore) SaveRoute(route *models.Route) error {
	query := `INSERT OR REPLACE INTO routes (destination_id, next_hop_id, hop_count, sequence_num, last_updated, expires_at, status)
			  VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, route.DestinationID, route.NextHopID, route.HopCount,
		route.SequenceNum, route.LastUpdated, route.ExpiresAt, route.Status)
	return err
}

// GetRoute retrieves a route for a destination
func (s *SQLiteStore) GetRoute(destinationID string) (*models.Route, error) {
	route := &models.Route{}
	err := s.db.QueryRow("SELECT destination_id, next_hop_id, hop_count, sequence_num, last_updated, expires_at, status FROM routes WHERE destination_id = ? AND status = 'active' AND expires_at > datetime('now')", destinationID).
		Scan(&route.DestinationID, &route.NextHopID, &route.HopCount, &route.SequenceNum,
			&route.LastUpdated, &route.ExpiresAt, &route.Status)
	if err != nil {
		return nil, err
	}
	return route, nil
}

// SaveAnalytics stores analytics data
func (s *SQLiteStore) SaveAnalytics(data *models.AnalyticsData) error {
	query := `INSERT INTO analytics (node_id, timestamp, active_nodes, msg_sent, msg_received, msg_dropped, avg_latency, packet_loss, file_transfers, alerts_sent, alerts_received)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, data.NodeID, data.Timestamp, data.ActiveNodes,
		data.MsgSent, data.MsgReceived, data.MsgDropped, data.AvgLatency,
		data.PacketLoss, data.FileTransfers, data.AlertsSent, data.AlertsReceived)
	return err
}

// GetRecentAnalytics returns the most recent N analytics records
func (s *SQLiteStore) GetRecentAnalytics(limit int) ([]*models.AnalyticsData, error) {
	rows, err := s.db.Query("SELECT node_id, timestamp, active_nodes, msg_sent, msg_received, msg_dropped, avg_latency, packet_loss, file_transfers, alerts_sent, alerts_received FROM analytics ORDER BY timestamp DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []*models.AnalyticsData
	for rows.Next() {
		d := &models.AnalyticsData{}
		err := rows.Scan(&d.NodeID, &d.Timestamp, &d.ActiveNodes, &d.MsgSent,
			&d.MsgReceived, &d.MsgDropped, &d.AvgLatency, &d.PacketLoss,
			&d.FileTransfers, &d.AlertsSent, &d.AlertsReceived)
		if err != nil {
			return nil, err
		}
		data = append(data, d)
	}
	return data, nil
}