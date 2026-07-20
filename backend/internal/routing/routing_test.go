package routing

import (
	"testing"
	"time"

	"github.com/drmcs/backend/internal/models"
)

func TestNewAODVRouter(t *testing.T) {
	r := NewAODVRouter("test-node")
	if r == nil {
		t.Fatal("expected non-nil router")
	}
	if r.nodeID != "test-node" {
		t.Errorf("expected nodeID 'test-node', got '%s'", r.nodeID)
	}
	if len(r.routes) != 0 {
		t.Errorf("expected empty routes, got %d", len(r.routes))
	}
}

func TestAddAndGetRoute(t *testing.T) {
	r := NewAODVRouter("test-node")

	// Add a route directly
	r.AddRouteDirect("dest-1", "next-hop-1", 1)

	route := r.GetRoute("dest-1")
	if route == nil {
		t.Fatal("expected non-nil route")
	}
	if route.DestinationID != "dest-1" {
		t.Errorf("expected destination 'dest-1', got '%s'", route.DestinationID)
	}
	if route.NextHopID != "next-hop-1" {
		t.Errorf("expected next hop 'next-hop-1', got '%s'", route.NextHopID)
	}
	if route.HopCount != 1 {
		t.Errorf("expected hop count 1, got %d", route.HopCount)
	}
	if route.Status != "active" {
		t.Errorf("expected status 'active', got '%s'", route.Status)
	}
}

func TestGetNextHop(t *testing.T) {
	r := NewAODVRouter("test-node")
	r.AddRouteDirect("dest-1", "next-hop-1", 1)

	nextHop, ok := r.GetNextHop("dest-1")
	if !ok {
		t.Fatal("expected to find next hop")
	}
	if nextHop != "next-hop-1" {
		t.Errorf("expected next hop 'next-hop-1', got '%s'", nextHop)
	}

	_, ok = r.GetNextHop("nonexistent")
	if ok {
		t.Fatal("expected no route for nonexistent destination")
	}
}

func TestRouteExists(t *testing.T) {
	r := NewAODVRouter("test-node")
	r.AddRouteDirect("dest-1", "next-hop-1", 1)

	if !r.RouteExists("dest-1") {
		t.Fatal("expected route to exist")
	}
	if r.RouteExists("nonexistent") {
		t.Fatal("expected route to not exist")
	}
}

func TestRemoveRoute(t *testing.T) {
	r := NewAODVRouter("test-node")
	r.AddRouteDirect("dest-1", "next-hop-1", 1)

	r.RemoveRoute("dest-1")
	route := r.GetRoute("dest-1")
	if route != nil {
		t.Fatal("expected route to be removed")
	}
}

func TestBetterRouteReplaces(t *testing.T) {
	r := NewAODVRouter("test-node")

	// Add a route with 3 hops
	r.AddRouteDirect("dest-1", "next-hop-1", 3)
	route := r.GetRoute("dest-1")
	if route.HopCount != 3 {
		t.Errorf("expected hop count 3, got %d", route.HopCount)
	}

	// Add a better route with 1 hop
	r.AddRouteDirect("dest-1", "better-hop", 1)
	route = r.GetRoute("dest-1")
	if route.HopCount != 1 {
		t.Errorf("expected hop count 1 after better route, got %d", route.HopCount)
	}
	if route.NextHopID != "better-hop" {
		t.Errorf("expected next hop 'better-hop', got '%s'", route.NextHopID)
	}
}

func TestRouteExpiration(t *testing.T) {
	r := NewAODVRouter("test-node")
	r.AddRouteDirect("dest-1", "next-hop-1", 1)

	// Route should exist
	if !r.RouteExists("dest-1") {
		t.Fatal("expected route to exist")
	}

	// Manually expire the route
	r.mu.Lock()
	if route, ok := r.routes["dest-1"]; ok {
		route.ExpiresAt = time.Now().Add(-1 * time.Second)
	}
	r.mu.Unlock()

	// Route should be expired
	if r.RouteExists("dest-1") {
		t.Fatal("expected route to be expired")
	}
}

func TestGetRoutingTable(t *testing.T) {
	r := NewAODVRouter("test-node")
	r.AddRouteDirect("dest-1", "next-hop-1", 1)
	r.AddRouteDirect("dest-2", "next-hop-2", 2)

	routes := r.GetRoutingTable()
	if len(routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(routes))
	}
}

func TestRouteUpdateSubscription(t *testing.T) {
	r := NewAODVRouter("test-node")
	ch := r.SubscribeRouteUpdates()

	r.AddRouteDirect("dest-1", "next-hop-1", 1)

	select {
	case update := <-ch:
		if update.DestinationID != "dest-1" {
			t.Errorf("expected destination 'dest-1', got '%s'", update.DestinationID)
		}
		if update.Status != "active" {
			t.Errorf("expected status 'active', got '%s'", update.Status)
		}
	case <-time.After(time.Second):
		t.Fatal("expected route update notification")
	}
}

func TestRouteDiscovery(t *testing.T) {
	r := NewAODVRouter("test-node")
	// Should not panic
	r.StartRouteDiscovery("some-node")
}

func TestRouteRequestWithDestAsSelf(t *testing.T) {
	r := NewAODVRouter("dest-node")

	rreq := &models.RouteRequest{
		RequestID:     1,
		SourceID:      "source-node",
		DestinationID: "dest-node",
		SequenceNum:   1,
		HopCount:      2,
		TTL:           5,
		BroadcastID:   1,
	}

	reply, shouldReply := r.HandleRouteRequest(rreq)
	if !shouldReply {
		t.Fatal("expected to reply since we are the destination")
	}
	if reply == nil {
		t.Fatal("expected non-nil reply")
	}
	if reply.DestinationID != "source-node" {
		t.Errorf("expected reply destination 'source-node', got '%s'", reply.DestinationID)
	}
}

func TestDuplicateRouteRequest(t *testing.T) {
	r := NewAODVRouter("dest-node")

	rreq := &models.RouteRequest{
		RequestID:     1,
		SourceID:      "source-node",
		DestinationID: "dest-node",
		SequenceNum:   1,
		HopCount:      2,
		TTL:           5,
		BroadcastID:   1,
	}

	// First request should be processed
	_, shouldReply := r.HandleRouteRequest(rreq)
	if !shouldReply {
		t.Fatal("expected to process first request")
	}

	// Second identical request should be dropped
	_, shouldReply = r.HandleRouteRequest(rreq)
	if shouldReply {
		t.Fatal("expected duplicate request to be dropped")
	}
}

func TestRouteRequestForward(t *testing.T) {
	r := NewAODVRouter("intermediate-node")

	rreq := &models.RouteRequest{
		RequestID:     1,
		SourceID:      "source-node",
		DestinationID: "far-away-node",
		SequenceNum:   1,
		HopCount:      1,
		TTL:           3,
		BroadcastID:   1,
	}

	// We're not the destination, should forward (not reply directly)
	reply, shouldReply := r.HandleRouteRequest(rreq)
	if shouldReply {
		t.Fatal("should not reply since we are not the destination")
	}
	if reply != nil {
		t.Fatal("expected nil reply")
	}

	// Should have created a reverse route to source
	if !r.RouteExists("source-node") {
		t.Fatal("expected reverse route to source")
	}
}

func TestHandleRouteReply(t *testing.T) {
	r := NewAODVRouter("source-node")

	rrep := &models.RouteReply{
		RequestID:     1,
		SourceID:      "dest-node",
		DestinationID: "source-node",
		SequenceNum:   1,
		HopCount:      2,
		Lifetime:      60,
	}

	r.HandleRouteReply(rrep)
	if !r.RouteExists("source-node") {
		t.Fatal("expected route to source after handling RREP")
	}
}