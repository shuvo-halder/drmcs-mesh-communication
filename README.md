# Disaster Response Mesh Communication System (DRMCS)

A decentralized peer-to-peer mesh communication system designed for disaster scenarios where traditional communication infrastructure fails. DRMCS enables automatic peer discovery, message routing, emergency alerts, and temporary file sharing without requiring internet access.

## Architecture

```
┌─────────────────────────────────────────────┐
│         User Device Layer                    │
│  (Go CLI / React Dashboard / Mobile Web)    │
├─────────────────────────────────────────────┤
│         Node Discovery Layer                 │
│  UDP Broadcast → Signature Verification     │
├─────────────────────────────────────────────┤
│         Mesh Routing Layer                   │
│  AODV-based Dynamic Routing (RREQ/RREP)     │
├─────────────────────────────────────────────┤
│         Communication Layer                  │
│  REST API / SSE Streams → Encrypted Channels │
├─────────────────────────────────────────────┤
│         Analytics Layer                      │
│  Python FastAPI → Pandas → NetworkX          │
├─────────────────────────────────────────────┤
│         Data Storage Layer                   │
│  SQLite (local) for resilience              │
└─────────────────────────────────────────────┘
```

## Project Structure

```
drmcs/
├── backend/                          # Go mesh networking core
│   ├── cmd/drmcsd/                   # Main daemon entry point
│   ├── internal/
│   │   ├── discovery/                # Peer discovery (UDP broadcast + Ed25519 auth)
│   │   ├── routing/                  # AODV routing protocol (RREQ/RREP)
│   │   ├── messaging/                # Message handling & delivery
│   │   ├── alerts/                   # Emergency alert system
│   │   ├── fileshare/                # Chunk-based file transfer (256KB)
│   │   ├── crypto/                   # AES-256-GCM encryption, Ed25519 signing
│   │   ├── storage/                  # SQLite persistence
│   │   └── models/                   # Shared data models & types
│   └── api/                          # REST API server with SSE streaming
├── analytics/                        # Python analytics engine
│   ├── api/                          # FastAPI server with WebSocket
│   ├── collectors/                   # Metrics collection from mesh nodes
│   ├── models/                       # Data models & schemas
│   └── visualization/                # NetworkX/Matplotlib topology graphs
├── frontend/                         # React dashboard (dark theme)
│   ├── public/
│   ├── src/
│   │   ├── components/              # TopologyView, MetricsDisplay, etc.
│   │   └── hooks/                   # useEventStream (SSE client)
├── docker/                           # Docker deployment
│   ├── Dockerfile.backend
│   ├── Dockerfile.analytics
│   └── docker-compose.yml
├── docs/                             # Documentation
└── tests/                            # Integration tests
```

## Quick Start

### Prerequisites

- Go 1.21+
- Python 3.11+
- Node.js 18+
- Docker & Docker Compose (optional)

### Running Locally

#### 1. Start a Mesh Node (Terminal 1)

```bash
cd drmcs/backend
go mod tidy
go run ./cmd/drmcsd --port 8080 --name node1 --data ./data
```

#### 2. Start Another Node (Terminal 2)

```bash
cd drmcs/backend
go run ./cmd/drmcsd --port 9080 --name node2 --data ./data2
```

The nodes will automatically discover each other via UDP broadcast on ports 8081/9081.

#### 3. Start Analytics Server (Terminal 3)

```bash
cd drmcs/analytics
pip install -r requirements.txt
python -m api.server
```

#### 4. Start Frontend Dashboard (Terminal 4)

```bash
cd drmcs/frontend
npm install
npm start
```

Open http://localhost:3000 in your browser. The dashboard provides:
- Real-time network metrics (peers, latency, throughput)
- Messaging interface (direct & broadcast)
- Emergency alert management
- File sharing capabilities
- Network topology visualization
- Routing table overview

### Running with Docker Compose

```bash
cd drmcs
docker compose -f docker/docker-compose.yml up --build
```

This starts:
- 3 mesh nodes (ports 8080, 9080, 10080 — each with discovery + file ports)
- 1 analytics server (port 8090)
- 1 React frontend (port 3000)

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `8080` | Primary listening port (API + messaging) |
| `--name` | hostname | Human-readable node name |
| `--data` | `./data` | Data directory for SQLite storage |

The daemon automatically allocates `port+1` for UDP discovery and `port+2` for file transfers.

## API Endpoints

### Backend API (Port 8080)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/node/info` | Node information (peers, routes, uptime) |
| GET | `/api/v1/node/peers` | Active peers list |
| GET | `/api/v1/messages` | Message history |
| POST | `/api/v1/messages` | Send a message (body: receiver_id, content, msg_type, priority) |
| POST | `/api/v1/messages/send` | Send a message (alternative) |
| GET | `/api/v1/alerts` | Active alerts |
| POST | `/api/v1/alerts` | Send emergency alert (body: alert_type, message, location, priority) |
| POST | `/api/v1/alerts/send` | Send emergency alert (alternative) |
| GET | `/api/v1/routes` | Routing table |
| GET | `/api/v1/analytics` | Analytics data (last 50 records) |
| GET | `/api/v1/files` | File transfers |
| POST | `/api/v1/files/upload` | Upload a file (multipart form, max 50MB) |
| GET | `/api/v1/files/download?file_id=` | Download/query file transfer status |
| GET | `/api/v1/ws` | Server-Sent Events (SSE) for real-time updates |

### Analytics API (Port 8090)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v2/health` | Health check |
| GET | `/api/v2/metrics` | Current network metrics |
| GET | `/api/v2/metrics/history?limit=100` | Historical metrics |
| GET | `/api/v2/topology` | Network topology graph (nodes + edges) |
| GET | `/api/v2/topology/image` | Topology visualization as base64 PNG |
| GET | `/api/v2/performance` | Performance report |
| GET | `/api/v2/performance/chart` | Performance chart as base64 PNG |
| GET | `/api/v2/network-metrics` | Network metrics summary (delivery rate, packet loss) |
| GET | `/api/v2/node/list` | List of all discovered nodes |
| GET | `/api/v2/alerts/types` | Available alert types & priorities |
| GET | `/api/v2/alerts/summary` | Summary of active alerts (proxied from backend) |
| WS | `/api/v2/ws` | WebSocket for real-time metrics streaming |

## Features

### 1. Peer Discovery
- Automatic device discovery via UDP broadcast
- Ed25519 signature verification on all discovery packets
- Replay attack prevention using timestamps
- Heartbeat-based liveness tracking (5s interval)
- Automatic removal of inactive peers (30s timeout)
- Self-discovery prevention (ignores own broadcasts)

### 2. Message Routing (AODV)
- On-demand route discovery (RREQ/RREP)
- Dynamic route maintenance with sequence numbers
- Multi-hop message forwarding
- Route recovery and timeout-based cleanup
- TTL-based hop limiting (initial TTL=3, increment=2)
- Duplicate RREQ detection
- Route listener notifications for external subscribers

### 3. Emergency Alerts
- Broadcast to all connected nodes
- Priority levels: 0 (normal), 1 (high), 2 (critical)
- Duplicate prevention
- Automatic expiration
- Alert types: Medical Emergency, Fire Alert, Flood Warning, Rescue Request, Missing Person, General Emergency

### 4. File Sharing
- Chunk-based transfer (256KB chunks)
- SHA-256 integrity verification using crypto.HashContent
- Content type detection
- Auto-expiration (configurable TTL, default 2 hours)
- Download tracking via file transfer status API

### 5. Security
- Node identity: Ed25519 key pair generation at startup
- Message signing: All discovery packets signed with sender's private key
- Signature verification: Every discovery packet verified on receipt
- Encryption: AES-256-GCM for message payloads
- Session key derivation: Deterministic shared AES key from sorted public keys
- Integrity: SHA-256 hashing for content identification

### 6. Analytics
- Real-time network metrics via WebSocket streaming
- Topology visualization with NetworkX/Matplotlib
- Performance charts (latency, delivery rates)
- Historical data tracking
- Network metrics summary (delivery rate, packet loss)
- Proxied alert aggregation from backend

### 7. Frontend Dashboard
- Dark-themed React UI (6 tabs: Dashboard, Messages, Alerts, File Sharing, Topology, Analytics)
- Real-time updates via Server-Sent Events (from backend) + WebSocket (from analytics)
- Metrics cards: Active peers, total messages, latency, active routes
- Message composition and history view
- Emergency alert management with priority badges
- File upload interface
- Topology visualization component
- Routing table with hop counts and status
- 5-second auto-refresh polling

## Security Implementation

- **Identity**: Ed25519 key pair generated per node on startup
- **Encryption**: AES-256-GCM for message payloads
- **Authentication**: Ed25519 signatures on all discovery broadcasts
- **Verification**: Signature verification on receipt of discovery packets
- **Session Keys**: Deterministic key derivation from sorted public keys (SHA-256)
- **Integrity**: SHA-256 checksums for content identification and file chunks
- **Replay Protection**: Timestamp verification in discovery packets

## Testing

```bash
# Backend tests
cd drmcs/backend
go test ./...

# Analytics tests
cd drmcs/analytics
pytest

# Frontend tests
cd drmcs/frontend
npm test
```

## Performance Characteristics

| Metric | Target | Current |
|--------|--------|---------|
| Message delivery | < 2 seconds | — |
| Peer discovery | < 3 seconds | ~2.5s (simulated) |
| Route convergence | < 5 seconds | ~5s (simulated) |
| Scalability | 500+ nodes | — |
| File transfer | Chunked with resume support | 256KB chunks, SHA-256 verified |
| Packet loss recovery | < 5% | — |

## Docker Configuration

The `docker-compose.yml` creates an isolated `drmcs-mesh` bridge network with:

- **3 mesh nodes**: Each with API, discovery, and file ports mapped to unique host ports
- **Analytics server**: Connects to node1's API for metrics collection
- **React frontend**: Installs dependencies and starts dev server
- **Persistent volumes**: Separate data and transfer volumes per node
- **Network capabilities**: `NET_ADMIN` and `NET_RAW` for UDP broadcast support

## Future Enhancements

- GPS integration for location-aware routing
- Mobile app (Android/iOS) support
- AI-assisted route optimization
- Drone relay support
- Satellite gateway integration
- Offline map support
- Voice communication
- End-to-end encryption improvements (per-message keys)
- gRPC integration for efficient multi-hop forwarding

## License

This project is developed as part of disaster response research.

## Contributors

- DRMCS Team - Disaster Response Mesh Communication System