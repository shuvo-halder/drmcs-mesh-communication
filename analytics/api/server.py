"""FastAPI server for DRMCS Analytics Dashboard"""

import asyncio
import httpx
import logging
from contextlib import asynccontextmanager
from datetime import datetime
from typing import List, Dict, Optional

from fastapi import FastAPI, HTTPException, WebSocket, WebSocketDisconnect
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse

from ..models.models import (
    AnalyticsData, NetworkMetrics, TopologyNode,
    TopologyEdge, TopologyGraph, PerformanceReport
)
from ..collectors.metrics_collector import MetricsCollector
from ..visualization.graph import TopologyVisualizer

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Global state
collector = MetricsCollector()
visualizer = TopologyVisualizer()
active_websockets: List[WebSocket] = []
collection_task: Optional[asyncio.Task] = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Start background tasks on startup"""
    global collection_task
    collection_task = asyncio.create_task(collector.continuous_collection(10))
    logger.info("Analytics server started")
    yield
    if collection_task:
        collection_task.cancel()
    logger.info("Analytics server stopped")


app = FastAPI(
    title="DRMCS Analytics Dashboard",
    description="Disaster Response Mesh Communication System - Analytics Engine",
    version="1.0.0",
    lifespan=lifespan
)

# CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.get("/api/v2/health")
async def health_check():
    """Health check endpoint"""
    return {"status": "ok", "service": "DRMCS Analytics", "timestamp": datetime.utcnow().isoformat()}


@app.get("/api/v2/metrics", response_model=AnalyticsData)
async def get_current_metrics():
    """Get current network metrics"""
    metrics = collector.get_latest_metrics()
    if not metrics:
        return AnalyticsData(
            node_id="unknown",
            timestamp=datetime.utcnow(),
            active_nodes=0
        )
    return metrics


@app.get("/api/v2/metrics/history", response_model=List[AnalyticsData])
async def get_metrics_history(limit: int = 100):
    """Get historical metrics data"""
    data = collector.get_collected_data()
    return data[-limit:] if limit < len(data) else data


@app.get("/api/v2/topology", response_model=TopologyGraph)
async def get_network_topology():
    """Get current network topology graph"""
    # Build topology from collected data
    latest = collector.get_latest_metrics()
    if not latest:
        return TopologyGraph(nodes=[], edges=[])

    # In production, this would query all peer nodes for their connections
    # For now, return a simplified graph
    nodes = [
        TopologyNode(id=latest.node_id, label=f"Node: {latest.node_id[:8]}", status="active")
    ]
    edges = []

    # Add peer nodes
    peers_data = await collector.collect_peers()
    for peer in peers_data:
        peer_id = peer.get("node_id", peer.get("peer_id", ""))
        if peer_id:
            nodes.append(
                TopologyNode(id=peer_id, label=f"Peer: {peer_id[:8]}", status=peer.get("status", "active"))
            )
            edges.append(
                TopologyEdge(from_node=latest.node_id, to_node=peer_id, weight=1, status="active")
            )

    return TopologyGraph(nodes=nodes, edges=edges)


@app.get("/api/v2/topology/image")
async def get_topology_image():
    """Get network topology as base64 image"""
    topology = await get_network_topology()
    img_base64 = visualizer.render_topology_image(
        list(topology.nodes) if topology.nodes else [],
        list(topology.edges) if topology.edges else []
    )
    return JSONResponse(content={"image": img_base64, "format": "png"})


@app.get("/api/v2/performance", response_model=PerformanceReport)
async def get_performance_report():
    """Get performance evaluation report"""
    data = collector.get_collected_data()
    if not data:
        return PerformanceReport(
            timestamp=datetime.utcnow(),
            avg_message_delivery_time_ms=0.0,
            max_hops=0,
            avg_hops=0,
            throughput_mbps=0.0,
            node_discovery_time_s=0.0,
            route_convergence_time_s=0.0,
            file_transfer_speed_kbps=0.0,
            alert_delivery_rate=0.0
        )

    # Compute metrics from collected data
    latencies = [d.avg_latency_ms for d in data if d.avg_latency_ms > 0]
    deliveries = [d.msg_received for d in data if d.msg_received > 0]

    report = PerformanceReport(
        timestamp=datetime.utcnow(),
        avg_message_delivery_time_ms=sum(latencies) / len(latencies) if latencies else 0.0,
        max_hops=10,  # Simulated
        avg_hops=3,   # Simulated
        throughput_mbps=0.0,
        node_discovery_time_s=2.5,  # Simulated
        route_convergence_time_s=5.0,  # Simulated
        file_transfer_speed_kbps=500.0,  # Simulated
        alert_delivery_rate=sum(deliveries) / (len(deliveries) or 1) if deliveries else 0.0
    )
    return report


@app.get("/api/v2/network-metrics", response_model=NetworkMetrics)
async def get_network_metrics_summary():
    """Get summary of network metrics"""
    latest = collector.get_latest_metrics()
    data = collector.get_collected_data()

    total_msgs = sum(d.msg_sent + d.msg_received for d in data)
    total_dropped = sum(d.msg_dropped for d in data)

    delivery_rate = ((total_msgs - total_dropped) / total_msgs * 100) if total_msgs > 0 else 0.0

    return NetworkMetrics(
        total_nodes=latest.active_nodes if latest else 0,
        total_messages=total_msgs,
        avg_latency=latest.avg_latency_ms if latest else 0.0,
        packet_loss_rate=latest.packet_loss_pct if latest else 0.0,
        delivery_success_rate=delivery_rate,
        active_routes=10  # Simulated
    )


@app.get("/api/v2/performance/chart")
async def get_performance_chart():
    """Get performance chart as base64 image"""
    data = collector.get_collected_data()
    if not data:
        return JSONResponse(content={"image": "", "format": "png"})

    timestamps = [d.timestamp for d in data]
    latencies = [d.avg_latency_ms for d in data]
    deliveries = [
        ((d.msg_received / (d.msg_sent or 1)) * 100) if d.msg_sent > 0 else 0
        for d in data
    ]

    img_base64 = visualizer.render_performance_chart(timestamps, latencies, deliveries)
    return JSONResponse(content={"image": img_base64, "format": "png"})


@app.get("/api/v2/alerts/summary")
async def get_alerts_summary():
    """Get summary of alerts"""
    try:
        async with httpx.AsyncClient(timeout=5.0) as client:
            resp = await client.get("http://localhost:8080/api/v1/alerts")
            if resp.status_code == 200:
                alerts = resp.json()
                return {
                    "total_alerts": len(alerts),
                    "active_alerts": alerts if isinstance(alerts, list) else []
                }
    except Exception as e:
        logger.error(f"Failed to fetch alerts: {e}")
        return {"total_alerts": 0, "active_alerts": []}


@app.websocket("/api/v2/ws")
async def websocket_endpoint(websocket: WebSocket):
    """WebSocket endpoint for real-time updates"""
    await websocket.accept()
    active_websockets.append(websocket)
    logger.info("WebSocket client connected")

    try:
        while True:
            # Send heartbeat every 15s to detect dead connections
            try:
                metrics = collector.get_latest_metrics()
                payload = {
                    "type": "metrics_update",
                    "data": metrics.model_dump() if metrics else None,
                    "timestamp": datetime.utcnow().isoformat()
                }
                await websocket.send_json(payload)
                await asyncio.sleep(5)
            except asyncio.CancelledError:
                break
    except WebSocketDisconnect:
        logger.info("WebSocket client disconnected")
    except Exception as e:
        logger.error(f"WebSocket error: {e}")
    finally:
        if websocket in active_websockets:
            active_websockets.remove(websocket)
        logger.info(f"WebSocket cleaned up. Active connections: {len(active_websockets)}")


@app.get("/api/v2/node/list")
async def get_all_nodes():
    """Get list of all discovered nodes"""
    try:
        peers = await collector.collect_peers()
    except Exception:
        peers = []
    try:
        node_info = await collector.collect_node_info()
    except Exception:
        node_info = None

    nodes = []
    if node_info:
        nodes.append({
            "id": node_info.get("node_id", "unknown"),
            "type": "self",
            "active_peers": node_info.get("active_peers", 0),
            "active_routes": node_info.get("active_routes", 0)
        })

    for peer in peers:
        nodes.append({
            "id": peer.get("node_id", peer.get("peer_id", "unknown")),
            "type": "peer",
            "ip": peer.get("ip_address", ""),
            "status": peer.get("status", "unknown")
        })

    return {"nodes": nodes}


@app.get("/api/v2/alerts/types")
async def get_alert_types():
    """Get available alert types"""
    return {
        "alert_types": [
            {"type": "medical_emergency", "label": "Medical Emergency", "priority": 2},
            {"type": "fire_alert", "label": "Fire Alert", "priority": 2},
            {"type": "flood_warning", "label": "Flood Warning", "priority": 1},
            {"type": "rescue_request", "label": "Rescue Request", "priority": 2},
            {"type": "missing_person", "label": "Missing Person", "priority": 1},
            {"type": "emergency", "label": "General Emergency", "priority": 0},
        ]
    }


if __name__ == "__main__":
    import uvicorn
    uvicorn.run("analytics.api.server:app", host="0.0.0.0", port=8090, reload=True)
