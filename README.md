# Disaster Response Mesh Communication System (DRMCS)

A decentralized peer-to-peer mesh communication system designed for disaster scenarios where traditional communication infrastructure fails. DRMCS enables automatic peer discovery, message routing, emergency alerts, and temporary file sharing without requiring internet access.

## Architecture

```
┌─────────────────────────────────────────────┐
│         User Device Layer                    │
│  (Go CLI / React Dashboard / Mobile Web)    │
├─────────────────────────────────────────────┤
│         Node Discovery Layer                 │
│  UDP Broadcast → mDNS → TCP Verify          │
├─────────────────────────────────────────────┤
│         Mesh Routing Layer                   │
│  AODV-based Dynamic Routing                 │
├─────────────────────────────────────────────┤
│         Communication Layer                  │
│  gRPC/HTTP Streams → Encrypted Channels      │
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
│   │   ├── discovery/                # Peer discovery (UDP broadcast)
│   │   ├── routing/                  # AODV routing protocol
│   │   ├── messaging/                # Message handling & delivery
│   │   ├── alerts/                   # Emergency alert system
│   │   ├── fileshare/                # Chunk-based file transfer
│   │   ├── crypto/                   # AES-256 encryption, Ed25519 signing
│   │   └── storage/                  # SQLite persistence
│   ├── api/                          # REST API endpoints
│   └── proto/                        # Protocol Buffers (future gRPC)
├── analytics/                        # Python analytics engine
│   ├── api/                          # FastAPI server
│   ├── collectors/                   # Metrics collection
│   ├── models/                       # Data models
│   └── visualization/                # NetworkX/Matplotlib graphs
├── frontend/                         # React dashboard
│   ├── public/
│   └── src/
│       └── components/
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
go mod init github.com/drmcs/backend  # Run once
go mod tidy
go run ./cmd/drmcsd --port 8080 --name node1
```

#### 2. Start Another Node (Terminal 2)

```bash
cd drmcs/backend
go run ./cmd/drmcsd --port 9080 --name node2
```

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

Open http://localhost:3000 in your browser.

### Running with Docker Compose

```bash
cd drmcs
docker-compose -f docker/docker-compose.yml up --build
```

This starts:
- 3 mesh nodes (ports 8080, 9080, 10080)
- 1 analytics server (port 8090)
- 1 React frontend (port 3000)

## API Endpoints

### Backend API (Port 8080)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/node/info` | Node information |
| GET | `/api/v1/node/peers` | Active peers list |
| GET | `/api/v1/messages` | Message history |
| POST | `/api/v1/messages/send` | Send a message |
| GET | `/api/v1/alerts` | Active alerts |
| POST | `/api/v1/alerts/send` | Send emergency alert |
| GET | `/api/v1/routes` | Routing table |
| GET | `/api/v1/analytics` | Analytics data |
| GET | `/api/v1/files` | File transfers |
| POST | `/api/v1/files/upload` | Upload a file |

### Analytics API (Port 8090)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v2/health` | Health check |
| GET | `/api/v2/metrics` | Current network metrics |
| GET | `/api/v2/metrics/history` | Historical metrics |
| GET | `/api/v2/topology` | Network topology graph |
| GET | `/api/v2/topology/image` | Topology visualization image |
| GET | `/api/v2/performance` | Performance report |
| GET | `/api/v2/performance/chart` | Performance chart image |
| GET | `/api/v2/network-metrics` | Network metrics summary |
| WS | `/api/v2/ws` | Real-time WebSocket updates |

## Features

### 1. Peer Discovery
- Automatic device discovery via UDP broadcast
- mDNS-style service announcements
- TCP handshake verification
- Heartbeat-based liveness tracking
- Automatic removal of inactive peers

### 2. Message Routing (AODV)
- On-demand route discovery (RREQ/RREP)
- Dynamic route maintenance
- Multi-hop message forwarding
- Route recovery on node failure
- TTL-based hop limiting

### 3. Emergency Alerts
- Broadcast to all connected nodes
- Priority levels (normal, high, critical)
- Duplicate prevention
- Automatic expiration
- Alert types: Medical, Fire, Flood, Rescue, Missing Person

### 4. File Sharing
- Chunk-based transfer (256KB chunks)
- SHA-256 integrity verification
- Auto-expiration of shared files
- Content type detection
- Resume capability for interrupted transfers

### 5. Security
- AES-256-GCM encryption
- Ed25519 digital signatures
- Peer identity verification
- Replay attack prevention
- Secure key exchange

### 6. Analytics
- Real-time network metrics
- Topology visualization
- Performance charts
- Historical data tracking
- WebSocket streaming

## Security Implementation

- **Encryption**: AES-256-GCM for message payloads
- **Authentication**: Ed25519 public/private key pairs
- **Signing**: All messages signed with sender's private key
- **Verification**: Signature verification on receipt
- **Integrity**: SHA-256 checksums for file chunks

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

- Message delivery: < 2 seconds (target)
- Peer discovery: < 3 seconds
- Route convergence: < 5 seconds
- Scalability: 500+ nodes
- File transfer: Chunked with resume support

## Future Enhancements

- GPS integration for location-aware routing
- Mobile app (Android/iOS) support
- AI-assisted route optimization
- Drone relay support
- Satellite gateway integration
- Offline map support
- Voice communication
- End-to-end encryption improvements

## License

This project is developed as part of disaster response research.

## Contributors

- DRMCS Team - Disaster Response Mesh Communication System