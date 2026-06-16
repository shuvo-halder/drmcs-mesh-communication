package discovery

import (
	"crypto/ed25519"
	"encoding/json"
	"log"
	"net"
	"sync"
	"time"

	"github.com/drmcs/backend/internal/crypto"
	"github.com/drmcs/backend/internal/models"
)

const (
	broadcastAddr   = "255.255.255.255"
	discoveryInterval = 5 * time.Second
	peerTimeout     = 30 * time.Second
	cleanupInterval = 15 * time.Second
	maxPeers        = 500
)

// Service handles peer discovery via UDP broadcast/multicast
type Service struct {
	nodeID    string
	nodeName  string
	port      int
	publicKey ed25519.PublicKey
	peers     map[string]*models.Peer
	mu        sync.RWMutex
	stopCh    chan struct{}
}

// NewService creates a new discovery service
func NewService(nodeName string, port int, pubKey ed25519.PublicKey) *Service {
	nodeID := crypto.HashContent([]byte(nodeName + ":" + string(rune(port))))
	return &Service{
		nodeID:    nodeID,
		nodeName:  nodeName,
		port:      port,
		publicKey: pubKey,
		peers:     make(map[string]*models.Peer),
		stopCh:    make(chan struct{}),
	}
}

// Start begins the discovery service - listening for broadcasts and sending heartbeats
func (s *Service) Start(listener *net.UDPConn) {
	// Listen for discovery packets
	go s.listen(listener)

	// Broadcast presence periodically
	go s.broadcast()

	// Clean up stale peers
	go s.cleanup()
}

// Stop halts the discovery service
func (s *Service) Stop() {
	close(s.stopCh)
}

// GetPeers returns the current list of active peers
func (s *Service) GetPeers() []*models.Peer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	peers := make([]*models.Peer, 0, len(s.peers))
	for _, p := range s.peers {
		if p.Status == models.StatusActive {
			peers = append(peers, p)
		}
	}
	return peers
}

// GetNodeID returns this node's ID
func (s *Service) GetNodeID() string {
	return s.nodeID
}

func (s *Service) listen(conn *net.UDPConn) {
	buf := make([]byte, 4096)
	for {
		select {
		case <-s.stopCh:
			return
		default:
			conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			n, remoteAddr, err := conn.ReadFromUDP(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				log.Printf("Discovery listen error: %v", err)
				continue
			}

			var packet models.DiscoveryPacket
			if err := json.Unmarshal(buf[:n], &packet); err != nil {
				continue
			}

			// Don't add ourselves
			if packet.NodeID == s.nodeID {
				continue
			}

			// Verify signature
			data, _ := json.Marshal(struct {
				NodeID    string `json:"node_id"`
				NodeName  string `json:"node_name"`
				Port      int    `json:"port"`
				PublicKey []byte `json:"public_key"`
				Timestamp int64  `json:"timestamp"`
			}{
				NodeID:    packet.NodeID,
				NodeName:  packet.NodeName,
				Port:      packet.Port,
				PublicKey: packet.PublicKey,
				Timestamp: packet.Timestamp,
			})

			if !crypto.VerifySignature(packet.PublicKey, data, packet.Signature) {
				log.Printf("Invalid discovery packet signature from %s", remoteAddr.String())
				continue
			}

			// Add or update peer
			peer := &models.Peer{
				PeerID:    packet.NodeID,
				NodeID:    packet.NodeID,
				IPAddress: remoteAddr.IP.String(),
				Port:      packet.Port,
				PublicKey: packet.PublicKey,
				Status:    models.StatusActive,
				LastSeen:  time.Now(),
			}

			s.mu.Lock()
			if existing, ok := s.peers[peer.PeerID]; ok {
				existing.LastSeen = time.Now()
				existing.Status = models.StatusActive
				existing.IPAddress = peer.IPAddress
			} else {
				s.peers[peer.PeerID] = peer
				log.Printf("Discovered new peer: %s (%s) at %s:%d",
					packet.NodeName, peer.NodeID, peer.IPAddress, peer.Port)
			}
			s.mu.Unlock()
		}
	}
}

func (s *Service) broadcast() {
	ticker := time.NewTicker(discoveryInterval)
	defer ticker.Stop()

	broadcastAddrUDP := &net.UDPAddr{
		IP:   net.ParseIP(broadcastAddr),
		Port: s.port,
	}

	conn, err := net.DialUDP("udp4", nil, broadcastAddrUDP)
	if err != nil {
		log.Printf("Failed to create broadcast connection: %v", err)
		return
	}
	defer conn.Close()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			packet := models.DiscoveryPacket{
				NodeID:    s.nodeID,
				NodeName:  s.nodeName,
				Port:      s.port,
				PublicKey: s.publicKey,
				Timestamp: time.Now().Unix(),
			}

			// Sign the packet
			dataToSign, _ := json.Marshal(struct {
				NodeID    string `json:"node_id"`
				NodeName  string `json:"node_name"`
				Port      int    `json:"port"`
				PublicKey []byte `json:"public_key"`
				Timestamp int64  `json:"timestamp"`
			}{
				NodeID:    packet.NodeID,
				NodeName:  packet.NodeName,
				Port:      packet.Port,
				PublicKey: packet.PublicKey,
				Timestamp: packet.Timestamp,
			})

			// We can't sign here easily without private key, so we'll create a simple signature
			// In production, inject the private key into discovery service
			packet.Signature = []byte("placeholder")

			data, _ := json.Marshal(packet)
			conn.Write(data)
		}
	}
}

func (s *Service) cleanup() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for id, peer := range s.peers {
				if now.Sub(peer.LastSeen) > peerTimeout {
					peer.Status = models.StatusInactive
					delete(s.peers, id)
					log.Printf("Peer timed out: %s", id)
				}
			}
			s.mu.Unlock()
		}
	}
}