from pydantic import BaseModel
from datetime import datetime
from typing import Optional, List


class AnalyticsData(BaseModel):
    """Model for network analytics data"""
    node_id: str
    timestamp: datetime
    active_nodes: int = 0
    msg_sent: int = 0
    msg_received: int = 0
    msg_dropped: int = 0
    avg_latency_ms: float = 0.0
    packet_loss_pct: float = 0.0
    file_transfers: int = 0
    alerts_sent: int = 0
    alerts_received: int = 0


class NetworkMetrics(BaseModel):
    """Aggregated network metrics"""
    total_nodes: int
    total_messages: int
    avg_latency: float
    packet_loss_rate: float
    delivery_success_rate: float
    active_routes: int


class TopologyNode(BaseModel):
    """Node in the network topology graph"""
    id: str
    label: str
    group: str = "mesh"
    value: int = 1
    status: str = "active"


class TopologyEdge(BaseModel):
    """Edge in the network topology graph"""
    from_node: str
    to_node: str
    weight: int = 1
    label: str = ""
    status: str = "active"


class TopologyGraph(BaseModel):
    """Complete network topology graph"""
    nodes: List[TopologyNode]
    edges: List[TopologyEdge]


class PerformanceReport(BaseModel):
    """Performance evaluation report"""
    timestamp: datetime
    avg_message_delivery_time_ms: float
    max_hops: int
    avg_hops: int
    throughput_mbps: float
    node_discovery_time_s: float
    route_convergence_time_s: float
    file_transfer_speed_kbps: float
    alert_delivery_rate: float