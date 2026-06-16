package routing

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/drmcs/backend/internal/models"
)

const (
	routeTimeout      = 60 * time.Second
	routeCleanupInt   = 30 * time.Second
	maxHopCount       = 15
	ttlInitial        = 3
	ttlIncrement      = 2
	ttlThreshold      = 7
	rreqRetries       = 3
	helloInterval     = 10 * time.Second
)

// Router implements an AODV-inspired routing protocol
type Router struct {
	nodeID    string
	routes    map[string]*models.Route
	mu        sync.RWMutex
	stopCh    chan struct{}
	seqNum    uint32 // Node sequence number
	broadcastID uint32
	// Request tracking for duplicate detection
	seenRequests map[uint32]map[string]bool // broadcastID -> sourceID -> seen
	requestMu    sync.Mutex
	routeListeners []chan RouteUpdate
	listenerMu     sync.RWMutex
}

// RouteUpdate represents a change in routing table
type RouteUpdate struct {
	DestinationID string `json:"destination_id"`
	NextHopID     string `json:"next_hop_id"`
	HopCount      int    `json:"hop_count"`
	Status        string `json:"status"`
}

// NewAODVRouter creates a new AODV routing instance
func NewAODVRouter(nodeID string) *Router {
	r := &Router{
		nodeID:       nodeID,
		routes:       make(map[string]*models.Route),
		stopCh:       make(chan struct{}),
		seqNum:       1,
		broadcastID:  0,
		seenRequests: make(map[uint32]map[string]bool),
	}
	go r.cleanupLoop()
	return r
}

// StartRouteDiscovery initiates route discovery for a destination
func (r *Router) StartRouteDiscovery(destinationID string) {
	r.broadcastID++
	r.broadcastID %= 1 << 30

	rreq := &models.RouteRequest{
		RequestID:     r.broadcastID,
		SourceID:      r.nodeID,
		DestinationID: destinationID,
		SequenceNum:   r.getNextSeqNum(),
		HopCount:      0,
		TTL:           ttlInitial,
		BroadcastID:   r.broadcastID,
	}

	log.Printf("Starting route discovery for %s (broadcast %d)", destinationID, r.broadcastID)

	// Broadcast RREQ to all neighbors (simulated)
	r.processRREQ(rreq)
}

// HandleRouteRequest processes an incoming RREQ
func (r *Router) HandleRouteRequest(rreq *models.RouteRequest) (*models.RouteReply, bool) {
	// Check for duplicate
	r.requestMu.Lock()
	if _, exists := r.seenRequests[rreq.BroadcastID]; !exists {
		r.seenRequests[rreq.BroadcastID] = make(map[string]bool)
	}
	if r.seenRequests[rreq.BroadcastID][rreq.SourceID] {
		r.requestMu.Unlock()
		return nil, false // Already processed
	}
	r.seenRequests[rreq.BroadcastID][rreq.SourceID] = true
	r.requestMu.Unlock()

	// Decrement TTL
	rreq.HopCount++
	rreq.TTL--

	// Check if we can reply (we are the destination)
	if rreq.DestinationID == r.nodeID {
		reply := &models.RouteReply{
			RequestID:     rreq.RequestID,
			SourceID:      r.nodeID,
			DestinationID: rreq.SourceID,
			SequenceNum:   r.getNextSeqNum(),
			HopCount:      0,
			Lifetime:      int64(routeTimeout.Seconds()),
		}

		// Add reverse route
		r.addRoute(rreq.SourceID, "", rreq.HopCount)

		log.Printf("Route found! Replying to %s via reverse path (%d hops)", rreq.SourceID, rreq.HopCount)
		return reply, true
	}

	// Forward RREQ if TTL > 0
	if rreq.TTL > 0 {
		r.processRREQ(rreq)
	}

	return nil, false
}

// HandleRouteReply processes an incoming RREP
func (r *Router) HandleRouteReply(rrep *models.RouteReply) {
	r.addRoute(rrep.DestinationID, "", rrep.HopCount)
	log.Printf("Route established to %s via next hop (%d hops)", rrep.DestinationID, rrep.HopCount)
}

// GetRoute returns the route to a destination
func (r *Router) GetRoute(destinationID string) *models.Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	route, exists := r.routes[destinationID]
	if !exists || route.Status != models.StatusActive || time.Now().After(route.ExpiresAt) {
		return nil
	}
	return route
}

// GetNextHop returns the next hop for a destination
func (r *Router) GetNextHop(destinationID string) (string, bool) {
	route := r.GetRoute(destinationID)
	if route == nil {
		return "", false
	}
	return route.NextHopID, true
}

// AddRoute adds or updates a route
func (r *Router) AddRoute(route *models.Route) {
	r.addRoute(route.DestinationID, route.NextHopID, route.HopCount)
}

// GetRoutingTable returns all active routes
func (r *Router) GetRoutingTable() []*models.Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	routes := make([]*models.Route, 0, len(r.routes))
	for _, route := range r.routes {
		if route.Status == models.StatusActive {
			routes = append(routes, route)
		}
	}
	return routes
}

// SubscribeRouteUpdates returns a channel for route update notifications
func (r *Router) SubscribeRouteUpdates() chan RouteUpdate {
	r.listenerMu.Lock()
	defer r.listenerMu.Unlock()

	ch := make(chan RouteUpdate, 100)
	r.routeListeners = append(r.routeListeners, ch)
	return ch
}

// RemoveRoute marks a route as invalid
func (r *Router) RemoveRoute(destinationID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if route, exists := r.routes[destinationID]; exists {
		route.Status = "invalid"
		r.notifyListeners(RouteUpdate{
			DestinationID: destinationID,
			NextHopID:     route.NextHopID,
			HopCount:      route.HopCount,
			Status:        "invalid",
		})
	}
}

// RouteExists checks if a route exists to the destination
func (r *Router) RouteExists(destinationID string) bool {
	return r.GetRoute(destinationID) != nil
}

func (r *Router) addRoute(destID, nextHopID string, hopCount int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	route := &models.Route{
		DestinationID: destID,
		NextHopID:     nextHopID,
		HopCount:      hopCount,
		SequenceNum:   r.getNextSeqNum(),
		LastUpdated:   time.Now(),
		ExpiresAt:     time.Now().Add(routeTimeout),
		Status:        models.StatusActive,
	}

	// Only add if better (shorter) than existing route
	if existing, exists := r.routes[destID]; exists && existing.Status == models.StatusActive {
		if existing.HopCount <= hopCount {
			return
		}
	}

	r.routes[destID] = route

	r.notifyListeners(RouteUpdate{
		DestinationID: destID,
		NextHopID:     nextHopID,
		HopCount:      hopCount,
		Status:        models.StatusActive,
	})
}

func (r *Router) processRREQ(rreq *models.RouteRequest) {
	data, _ := json.Marshal(rreq)
	log.Printf("Processing RREQ: %s", string(data))
	// In real implementation, this broadcasts to all neighbors
}

func (r *Router) getNextSeqNum() uint32 {
	r.seqNum++
	return r.seqNum
}

func (r *Router) cleanupLoop() {
	ticker := time.NewTicker(routeCleanupInt)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.cleanupExpiredRoutes()
		}
	}
}

func (r *Router) cleanupExpiredRoutes() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for id, route := range r.routes {
		if now.After(route.ExpiresAt) {
			route.Status = "stale"
			delete(r.routes, id)
			log.Printf("Route expired: %s", id)
		}
	}
}

func (r *Router) notifyListeners(update RouteUpdate) {
	r.listenerMu.RLock()
	defer r.listenerMu.RUnlock()

	for _, ch := range r.routeListeners {
		select {
		case ch <- update:
		default:
			// Channel full, skip
		}
	}
}