"""Network topology visualization using NetworkX and Matplotlib"""

import networkx as nx
import matplotlib.pyplot as plt
import matplotlib
matplotlib.use('Agg')  # Non-interactive backend
import io
import base64
import logging
from datetime import datetime
from typing import List, Dict, Optional

from ..models.models import TopologyNode, TopologyEdge, TopologyGraph

logger = logging.getLogger(__name__)


class TopologyVisualizer:
    """Generates network topology visualizations"""

    def __init__(self):
        self.graph = nx.Graph()

    def build_graph(self, nodes: List[TopologyNode], edges: List[TopologyEdge]) -> nx.Graph:
        """Build a NetworkX graph from topology data"""
        G = nx.Graph()

        # Add nodes
        for node in nodes:
            G.add_node(
                node.id,
                label=node.label,
                group=node.group,
                value=node.value,
                status=node.status
            )

        # Add edges
        for edge in edges:
            G.add_edge(
                edge.from_node,
                edge.to_node,
                weight=edge.weight,
                label=edge.label,
                status=edge.status
            )

        self.graph = G
        return G

    def render_topology_image(self, nodes: List[TopologyNode], edges: List[TopologyEdge]) -> str:
        """Render topology as base64-encoded PNG image"""
        G = self.build_graph(nodes, edges)

        plt.figure(figsize=(12, 8))
        pos = nx.spring_layout(G, k=2, iterations=50)

        # Draw nodes
        active_nodes = [n for n in G.nodes() if G.nodes[n].get('status') == 'active']
        inactive_nodes = [n for n in G.nodes() if G.nodes[n].get('status') != 'active']

        nx.draw_networkx_nodes(G, pos, nodelist=active_nodes,
                               node_color='#4CAF50', node_size=500, alpha=0.9)
        nx.draw_networkx_nodes(G, pos, nodelist=inactive_nodes,
                               node_color='#FF5252', node_size=300, alpha=0.5)

        # Draw edges
        active_edges = [(u, v) for u, v, d in G.edges(data=True)
                        if d.get('status') == 'active']
        nx.draw_networkx_edges(G, pos, edgelist=active_edges,
                               edge_color='#2196F3', width=2, alpha=0.7)

        # Draw labels
        labels = {n: G.nodes[n].get('label', n) for n in G.nodes()}
        nx.draw_networkx_labels(G, pos, labels, font_size=8)

        plt.title(f'DRMCS Network Topology - {datetime.now().strftime("%Y-%m-%d %H:%M:%S")}',
                  fontsize=14, fontweight='bold')
        plt.axis('off')

        # Convert to base64
        buf = io.BytesIO()
        plt.savefig(buf, format='png', dpi=150, bbox_inches='tight')
        plt.close()
        buf.seek(0)
        img_base64 = base64.b64encode(buf.read()).decode('utf-8')

        return img_base64

    def render_performance_chart(self, timestamps: List[datetime], latencies: List[float],
                                   deliveries: List[float]) -> str:
        """Render performance metrics as base64-encoded PNG"""
        fig, (ax1, ax2) = plt.subplots(2, 1, figsize=(10, 8))

        # Latency chart
        ax1.plot(timestamps, latencies, 'b-', linewidth=2, label='Latency (ms)')
        ax1.set_xlabel('Time')
        ax1.set_ylabel('Latency (ms)')
        ax1.set_title('Network Latency Over Time')
        ax1.grid(True, alpha=0.3)
        ax1.legend()

        # Delivery rate chart
        ax2.plot(timestamps, deliveries, 'g-', linewidth=2, label='Delivery Rate (%)')
        ax2.set_xlabel('Time')
        ax2.set_ylabel('Delivery Rate (%)')
        ax2.set_title('Message Delivery Rate Over Time')
        ax2.grid(True, alpha=0.3)
        ax2.legend()

        plt.tight_layout()

        buf = io.BytesIO()
        plt.savefig(buf, format='png', dpi=150)
        plt.close()
        buf.seek(0)
        img_base64 = base64.b64encode(buf.read()).decode('utf-8')

        return img_base64

    def analyze_network_metrics(self) -> Dict:
        """Analyze network metrics from the current graph"""
        if not self.graph:
            return {}

        num_nodes = self.graph.number_of_nodes()
        num_edges = self.graph.number_of_edges()

        metrics = {
            "num_nodes": num_nodes,
            "num_edges": num_edges,
            "density": nx.density(self.graph),
            "is_connected": nx.is_connected(self.graph) if num_nodes > 0 else False,
        }

        if num_nodes > 0:
            # Degree metrics
            degrees = [d for n, d in self.graph.degree()]
            metrics["avg_degree"] = sum(degrees) / len(degrees) if degrees else 0
            metrics["max_degree"] = max(degrees) if degrees else 0

            # Path metrics
            if nx.is_connected(self.graph):
                metrics["diameter"] = nx.diameter(self.graph)
                metrics["avg_shortest_path"] = nx.average_shortest_path_length(self.graph)

        return metrics