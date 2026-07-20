import React, { useState, useEffect, useCallback } from 'react';
import axios from 'axios';
import TopologyView from './components/TopologyView';
import useEventStream from './hooks/useEventStream';

// API base URLs
const BACKEND_API = 'http://localhost:8080';
const ANALYTICS_API = 'http://localhost:8090';
const ANALYTICS_SSE = 'http://localhost:8090/api/v2/ws';

// Styles
const styles = {
  container: {
    padding: '20px',
    maxWidth: '1400px',
    margin: '0 auto',
  },
  header: {
    textAlign: 'center',
    padding: '20px 0',
    borderBottom: '2px solid #e94560',
    marginBottom: '30px',
  },
  title: {
    fontSize: '28px',
    fontWeight: 'bold',
    color: '#e94560',
  },
  subtitle: {
    fontSize: '14px',
    color: '#8899aa',
    marginTop: '5px',
  },
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))',
    gap: '20px',
    marginBottom: '30px',
  },
  card: {
    background: '#16213e',
    borderRadius: '10px',
    padding: '20px',
    border: '1px solid #0f3460',
  },
  cardTitle: {
    fontSize: '16px',
    fontWeight: '600',
    color: '#8899aa',
    marginBottom: '15px',
    textTransform: 'uppercase',
    letterSpacing: '1px',
  },
  metricValue: {
    fontSize: '32px',
    fontWeight: 'bold',
    color: '#e94560',
  },
  metricLabel: {
    fontSize: '12px',
    color: '#8899aa',
    marginTop: '5px',
  },
  statusBar: {
    display: 'flex',
    justifyContent: 'space-between',
    padding: '10px 0',
    borderBottom: '1px solid #0f3460',
  },
  statusLabel: { color: '#8899aa', fontSize: '14px' },
  statusValue: { color: '#e0e0e0', fontWeight: '500' },
  sectionTitle: {
    fontSize: '20px',
    fontWeight: 'bold',
    color: '#e0e0e0',
    margin: '20px 0',
  },
  table: {
    width: '100%',
    borderCollapse: 'collapse',
  },
  th: {
    textAlign: 'left',
    padding: '12px 15px',
    borderBottom: '2px solid #0f3460',
    color: '#8899aa',
    fontSize: '13px',
    textTransform: 'uppercase',
  },
  td: {
    padding: '12px 15px',
    borderBottom: '1px solid #0f3460',
    fontSize: '14px',
  },
  input: {
    width: '100%',
    padding: '10px',
    borderRadius: '5px',
    border: '1px solid #0f3460',
    background: '#1a1a2e',
    color: '#e0e0e0',
    marginBottom: '10px',
    fontSize: '14px',
  },
  button: {
    padding: '10px 20px',
    borderRadius: '5px',
    border: 'none',
    fontWeight: '600',
    cursor: 'pointer',
    fontSize: '14px',
    marginRight: '10px',
    marginBottom: '10px',
  },
  primaryBtn: {
    background: '#e94560',
    color: '#fff',
  },
  secondaryBtn: {
    background: '#0f3460',
    color: '#e0e0e0',
  },
  alertBadge: {
    display: 'inline-block',
    padding: '3px 8px',
    borderRadius: '3px',
    fontSize: '11px',
    fontWeight: '600',
    textTransform: 'uppercase',
  },
  criticalAlert: { background: '#e94560', color: '#fff' },
  highAlert: { background: '#ff6b35', color: '#fff' },
  normalAlert: { background: '#4ecca3', color: '#fff' },
  textarea: {
    width: '100%',
    padding: '10px',
    borderRadius: '5px',
    border: '1px solid #0f3460',
    background: '#1a1a2e',
    color: '#e0e0e0',
    marginBottom: '10px',
    fontSize: '14px',
    minHeight: '80px',
    resize: 'vertical',
  },
  select: {
    width: '100%',
    padding: '10px',
    borderRadius: '5px',
    border: '1px solid #0f3460',
    background: '#1a1a2e',
    color: '#e0e0e0',
    marginBottom: '10px',
    fontSize: '14px',
  },
  topologyContainer: {
    background: '#16213e',
    borderRadius: '10px',
    padding: '20px',
    border: '1px solid #0f3460',
    textAlign: 'center' },
  topologyImg: {
    maxWidth: '100%',
    height: 'auto',
    borderRadius: '5px',
  },
  tabContainer: {
    display: 'flex',
    marginBottom: '20px',
    borderBottom: '2px solid #0f3460',
  },
  tab: {
    padding: '10px 20px',
    cursor: 'pointer',
    borderBottom: '3px solid transparent',
    color: '#8899aa',
    fontWeight: '500',
  },
  activeTab: {
    borderBottomColor: '#e94560',
    color: '#e94560',
  },
  panel: { display: 'none' },
  activePanel: { display: 'block' },
};

function App() {
  const [activeTab, setActiveTab] = useState('dashboard');
  const [nodeInfo, setNodeInfo] = useState(null);
  const [peers, setPeers] = useState([]);
  const [messages, setMessages] = useState([]);
  const [alerts, setAlerts] = useState([]);
  const [routes, setRoutes] = useState([]);
  const [metrics, setMetrics] = useState(null);
  const [topologyImage, setTopologyImage] = useState('');
  const [loading, setLoading] = useState(true);

  // Real-time SSE updates
  const handleSSEEvent = useCallback((data) => {
    if (data.type === 'metrics_update' && data.data) {
      setMetrics(prev => ({ ...prev, ...data.data }));
    }
  }, []);

  useEventStream(ANALYTICS_SSE, handleSSEEvent);

  // Form states
  const [msgReceiver, setMsgReceiver] = useState('');
  const [msgContent, setMsgContent] = useState('');
  const [alertType, setAlertType] = useState('emergency');
  const [alertMsg, setAlertMsg] = useState('');
  const [alertLocation, setAlertLocation] = useState('');

  const fetchData = useCallback(async () => {
    try {
      const [nodeRes, peersRes, msgsRes, alertsRes, routesRes, metricsRes] = await Promise.allSettled([
        axios.get(`${BACKEND_API}/api/v1/node/info`),
        axios.get(`${BACKEND_API}/api/v1/node/peers`),
        axios.get(`${BACKEND_API}/api/v1/messages`),
        axios.get(`${BACKEND_API}/api/v1/alerts`),
        axios.get(`${BACKEND_API}/api/v1/routes`),
        axios.get(`${ANALYTICS_API}/api/v2/metrics`),
      ]);

      if (nodeRes.status === 'fulfilled') setNodeInfo(nodeRes.value.data);
      if (peersRes.status === 'fulfilled') setPeers(peersRes.value.data);
      if (msgsRes.status === 'fulfilled') setMessages(msgsRes.value.data);
      if (alertsRes.status === 'fulfilled') setAlerts(alertsRes.value.data);
      if (routesRes.status === 'fulfilled') setRoutes(routesRes.value.data);
      if (metricsRes.status === 'fulfilled') setMetrics(metricsRes.value.data);
    } catch (err) {
      console.error('Failed to fetch data:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchTopology = useCallback(async () => {
    try {
      const res = await axios.get(`${ANALYTICS_API}/api/v2/topology/image`);
      if (res.data.image) setTopologyImage(`data:image/png;base64,${res.data.image}`);
    } catch (err) {
      console.error('Failed to fetch topology:', err);
    }
  }, []);

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 5000);
    return () => clearInterval(interval);
  }, [fetchData]);

  useEffect(() => {
    if (activeTab === 'topology') fetchTopology();
  }, [activeTab, fetchTopology]);

  const handleSendMessage = async () => {
    if (!msgReceiver || !msgContent) return;
    try {
      await axios.post(`${BACKEND_API}/api/v1/messages/send`, {
        receiver_id: msgReceiver,
        content: msgContent,
        msg_type: 'text',
        priority: 0,
      });
      setMsgContent('');
      fetchData();
    } catch (err) {
      console.error('Failed to send message:', err);
    }
  };

  const handleSendAlert = async () => {
    if (!alertMsg) return;
    try {
      await axios.post(`${BACKEND_API}/api/v1/alerts/send`, {
        alert_type: alertType,
        message: alertMsg,
        location: alertLocation,
        priority: alertType === 'medical_emergency' || alertType === 'fire_alert' ? 2 : 1,
      });
      setAlertMsg('');
      fetchData();
    } catch (err) {
      console.error('Failed to send alert:', err);
    }
  };

  const getAlertStyle = (priority) => {
    if (priority >= 2) return styles.criticalAlert;
    if (priority >= 1) return styles.highAlert;
    return styles.normalAlert;
  };

  const getAlertTypeLabel = (type) => {
    const labels = {
      medical_emergency: 'Medical',
      fire_alert: 'Fire',
      flood_warning: 'Flood',
      rescue_request: 'Rescue',
      missing_person: 'Missing',
      emergency: 'Emergency',
    };
    return labels[type] || type;
  };

  const tabs = [
    { id: 'dashboard', label: 'Dashboard' },
    { id: 'messages', label: 'Messages' },
    { id: 'alerts', label: 'Alerts' },
    { id: 'files', label: 'File Sharing' },
    { id: 'topology', label: 'Topology' },
    { id: 'analytics', label: 'Analytics' },
  ];

  return (
    <div style={styles.container}>
      {/* Header */}
      <header style={styles.header}>
        <h1 style={styles.title}>DRMCS - Mesh Network Dashboard</h1>
        <p style={styles.subtitle}>
          Disaster Response Mesh Communication System | 
          {nodeInfo ? ` Node: ${nodeInfo.node_id?.substring(0, 8)}...` : ' Connecting...'}
        </p>
      </header>

      {/* Tabs */}
      <div style={styles.tabContainer}>
        {tabs.map(tab => (
          <div
            key={tab.id}
            style={{ ...styles.tab, ...(activeTab === tab.id ? styles.activeTab : {}) }}
            onClick={() => setActiveTab(tab.id)}
          >
            {tab.label}
          </div>
        ))}
      </div>

      {/* Dashboard Tab */}
      <div style={activeTab === 'dashboard' ? styles.activePanel : styles.panel}>
        {/* Metrics Cards */}
        <div style={styles.grid}>
          <div style={styles.card}>
            <div style={styles.cardTitle}>Active Peers</div>
            <div style={styles.metricValue}>{metrics?.active_nodes || 0}</div>
            <div style={styles.metricLabel}>Discovered nodes in mesh</div>
          </div>
          <div style={styles.card}>
            <div style={styles.cardTitle}>Messages</div>
            <div style={styles.metricValue}>{(metrics?.msg_sent || 0) + (metrics?.msg_received || 0)}</div>
            <div style={styles.metricLabel}>Total messages exchanged</div>
          </div>
          <div style={styles.card}>
            <div style={styles.cardTitle}>Latency</div>
            <div style={styles.metricValue}>{metrics?.avg_latency_ms?.toFixed(1) || '0.0'}</div>
            <div style={styles.metricLabel}>Average milliseconds</div>
          </div>
          <div style={styles.card}>
            <div style={styles.cardTitle}>Active Routes</div>
            <div style={styles.metricValue}>{routes.length}</div>
            <div style={styles.metricLabel}>Routing table entries</div>
          </div>
        </div>

        {/* Node Info */}
        <div style={styles.card}>
          <div style={styles.cardTitle}>Node Information</div>
          {nodeInfo ? (
            <>
              <div style={styles.statusBar}>
                <span style={styles.statusLabel}>Node ID</span>
                <span style={styles.statusValue}>{nodeInfo.node_id}</span>
              </div>
              <div style={styles.statusBar}>
                <span style={styles.statusLabel}>Active Peers</span>
                <span style={styles.statusValue}>{nodeInfo.active_peers}</span>
              </div>
              <div style={styles.statusBar}>
                <span style={styles.statusLabel}>Active Routes</span>
                <span style={styles.statusValue}>{nodeInfo.active_routes}</span>
              </div>
            </>
          ) : (
            <p style={{ color: '#8899aa' }}>Waiting for node connection...</p>
          )}
        </div>

        {/* Peers List */}
        <div style={{...styles.card, marginTop: '20px'}}>
          <div style={styles.cardTitle}>Connected Peers</div>
          {peers.length > 0 ? (
            <table style={styles.table}>
              <thead>
                <tr>
                  <th style={styles.th}>Peer ID</th>
                  <th style={styles.th}>IP Address</th>
                  <th style={styles.th}>Status</th>
                  <th style={styles.th}>Last Seen</th>
                </tr>
              </thead>
              <tbody>
                {peers.map((peer, idx) => (
                  <tr key={idx}>
                    <td style={styles.td}>{peer.node_id?.substring(0, 12)}...</td>
                    <td style={styles.td}>{peer.ip_address}</td>
                    <td style={styles.td}>
                      <span style={{...styles.alertBadge, background: peer.status === 'active' ? '#4ecca3' : '#8899aa', color: '#1a1a2e'}}>
                        {peer.status}
                      </span>
                    </td>
                    <td style={styles.td}>{new Date(peer.last_seen).toLocaleTimeString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <p style={{ color: '#8899aa' }}>No peers discovered yet. Ensure other nodes are running.</p>
          )}
        </div>
      </div>

      {/* Messages Tab */}
      <div style={activeTab === 'messages' ? styles.activePanel : styles.panel}>
        <div style={styles.grid}>
          {/* Send Message */}
          <div style={styles.card}>
            <div style={styles.cardTitle}>Send Message</div>
            <input
              style={styles.input}
              placeholder="Receiver ID (or 'broadcast')"
              value={msgReceiver}
              onChange={e => setMsgReceiver(e.target.value)}
            />
            <textarea
              style={styles.textarea}
              placeholder="Type your message..."
              value={msgContent}
              onChange={e => setMsgContent(e.target.value)}
            />
            <button style={{...styles.button, ...styles.primaryBtn}} onClick={handleSendMessage}>
              Send Message
            </button>
          </div>

          {/* Messages List */}
          <div style={styles.card}>
            <div style={styles.cardTitle}>Message History</div>
            {messages.length > 0 ? (
              messages.slice(0, 20).map((msg, idx) => (
                <div key={idx} style={styles.statusBar}>
                  <div>
                    <span style={{ color: '#4ecca3', fontSize: '12px' }}>{msg.sender_id?.substring(0, 8)}</span>
                    <p style={{ color: '#e0e0e0', marginTop: '3px', fontSize: '13px' }}>{msg.content}</p>
                  </div>
                  <span style={{ color: '#8899aa', fontSize: '11px' }}>
                    {new Date(msg.timestamp).toLocaleTimeString()}
                  </span>
                </div>
              ))
            ) : (
              <p style={{ color: '#8899aa' }}>No messages yet.</p>
            )}
          </div>
        </div>
      </div>

      {/* Alerts Tab */}
      <div style={activeTab === 'alerts' ? styles.activePanel : styles.panel}>
        <div style={styles.grid}>
          {/* Send Alert */}
          <div style={styles.card}>
            <div style={styles.cardTitle}>Send Emergency Alert</div>
            <select style={styles.select} value={alertType} onChange={e => setAlertType(e.target.value)}>
              <option value="emergency">General Emergency</option>
              <option value="medical_emergency">Medical Emergency</option>
              <option value="fire_alert">Fire Alert</option>
              <option value="flood_warning">Flood Warning</option>
              <option value="rescue_request">Rescue Request</option>
              <option value="missing_person">Missing Person</option>
            </select>
            <input
              style={styles.input}
              placeholder="Location (optional)"
              value={alertLocation}
              onChange={e => setAlertLocation(e.target.value)}
            />
            <textarea
              style={styles.textarea}
              placeholder="Alert message..."
              value={alertMsg}
              onChange={e => setAlertMsg(e.target.value)}
            />
            <button style={{...styles.button, ...styles.primaryBtn}} onClick={handleSendAlert}>
              Send Emergency Alert
            </button>
          </div>

          {/* Active Alerts */}
          <div style={styles.card}>
            <div style={styles.cardTitle}>Active Alerts</div>
            {alerts.length > 0 ? (
              alerts.map((alert, idx) => (
                <div key={idx} style={{
                  ...styles.statusBar,
                  borderLeft: `4px solid ${alert.priority >= 2 ? '#e94560' : alert.priority >= 1 ? '#ff6b35' : '#4ecca3'}`,
                  paddingLeft: '15px',
                  marginBottom: '5px',
                }}>
                  <div>
                    <span style={{...styles.alertBadge, ...getAlertStyle(alert.priority)}}>
                      {getAlertTypeLabel(alert.alert_type)}
                    </span>
                    <p style={{ color: '#e0e0e0', marginTop: '5px', fontSize: '13px' }}>{alert.message}</p>
                    {alert.location && <p style={{ color: '#8899aa', fontSize: '11px' }}>📍 {alert.location}</p>}
                  </div>
                </div>
              ))
            ) : (
              <p style={{ color: '#8899aa' }}>No active alerts.</p>
            )}
          </div>
        </div>
      </div>

      {/* File Sharing Tab */}
      <div style={activeTab === 'files' ? styles.activePanel : styles.panel}>
        <div style={styles.grid}>
          <div style={styles.card}>
            <div style={styles.cardTitle}>Upload File</div>
            <input type="file" style={{...styles.input, padding: '15px'}} />
            <button style={{...styles.button, ...styles.primaryBtn}}>
              Share File
            </button>
          </div>
          <div style={styles.card}>
            <div style={styles.cardTitle}>Shared Files</div>
            <p style={{ color: '#8899aa' }}>No files currently shared. Upload a file to share with the mesh network.</p>
            <p style={{ color: '#8899aa', fontSize: '12px', marginTop: '10px' }}>
              Supported: Images, PDFs, Documents, Text files (max 50MB)
            </p>
          </div>
        </div>
      </div>

      {/* Topology Tab */}
      <div style={activeTab === 'topology' ? styles.activePanel : styles.panel}>
        <TopologyView />
      </div>

      {/* Analytics Tab */}
      <div style={activeTab === 'analytics' ? styles.activePanel : styles.panel}>
        <div style={styles.grid}>
          <div style={styles.card}>
            <div style={styles.cardTitle}>Network Performance</div>
            {metrics ? (
              <>
                <div style={styles.statusBar}>
                  <span style={styles.statusLabel}>Active Nodes</span>
                  <span style={styles.statusValue}>{metrics.active_nodes}</span>
                </div>
                <div style={styles.statusBar}>
                  <span style={styles.statusLabel}>Messages Sent</span>
                  <span style={styles.statusValue}>{metrics.msg_sent}</span>
                </div>
                <div style={styles.statusBar}>
                  <span style={styles.statusLabel}>Messages Received</span>
                  <span style={styles.statusValue}>{metrics.msg_received}</span>
                </div>
                <div style={styles.statusBar}>
                  <span style={styles.statusLabel}>Average Latency</span>
                  <span style={styles.statusValue}>{metrics.avg_latency_ms?.toFixed(1)} ms</span>
                </div>
                <div style={styles.statusBar}>
                  <span style={styles.statusLabel}>Packet Loss</span>
                  <span style={styles.statusValue}>{metrics.packet_loss_pct?.toFixed(1)}%</span>
                </div>
                <div style={styles.statusBar}>
                  <span style={styles.statusLabel}>File Transfers</span>
                  <span style={styles.statusValue}>{metrics.file_transfers}</span>
                </div>
                <div style={styles.statusBar}>
                  <span style={styles.statusLabel}>Alerts Sent</span>
                  <span style={styles.statusValue}>{metrics.alerts_sent}</span>
                </div>
                <div style={styles.statusBar}>
                  <span style={styles.statusLabel}>Alerts Received</span>
                  <span style={styles.statusValue}>{metrics.alerts_received}</span>
                </div>
              </>
            ) : (
              <p style={{ color: '#8899aa' }}>No analytics data available.</p>
            )}
          </div>
          <div style={styles.card}>
            <div style={styles.cardTitle}>Routing Table</div>
            {routes.length > 0 ? (
              <table style={styles.table}>
                <thead>
                  <tr>
                    <th style={styles.th}>Destination</th>
                    <th style={styles.th}>Hops</th>
                    <th style={styles.th}>Status</th>
                  </tr>
                </thead>
                <tbody>
                  {routes.map((route, idx) => (
                    <tr key={idx}>
                      <td style={styles.td}>{route.destination_id?.substring(0, 12)}...</td>
                      <td style={styles.td}>{route.hop_count}</td>
                      <td style={styles.td}>
                        <span style={{...styles.alertBadge, background: route.status === 'active' ? '#4ecca3' : '#8899aa', color: '#1a1a2e'}}>
                          {route.status}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            ) : (
              <p style={{ color: '#8899aa' }}>No routes established yet.</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

export default App;