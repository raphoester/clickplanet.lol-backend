> **This repo has moved.** ClickPlanet is now a monorepo at [github.com/raphoester/clickplanet](https://github.com/raphoester/clickplanet) (this app lives under `apps/backend`, with full history preserved). This repo is archived and read-only.

# ClickPlanet — Backend

Backend for [clickplanet.lol](https://clickplanet.lol), a real-time multiplayer game where countries compete to own tiles on a world map. Written in Go.

## How it works

Every tile on the map has an owner (a country code). Players click tiles to claim them for their country. All connected clients see ownership changes in real time.

The backend is split into two binaries:

- **`api`** — serves HTTP endpoints and WebSocket connections
- **`bookkeeper`** — background job that reads update history from the Redis stream and posts highlights to X/Twitter

## Architecture

The project follows **hexagonal architecture** (ports & adapters), keeping the domain model isolated from infrastructure concerns.

```
internal/clicks/
├── domain/          # Core interfaces and business logic
├── adapters/
│   ├── primary/     # Inbound: HTTP REST, WebSocket
│   └── secondary/   # Outbound: Redis storage, X publisher
└── app/             # Wires everything together
```

**Click flow:**

```
POST /v2/rpc/click
  → validate country + tile ID
  → Redis Lua script (atomic SET + XADD to stream)
  → WebSocket publisher fans updates out to all connected clients
```

The Lua script (`static/setAndPublishOnStream.lua`) atomically updates tile ownership and appends to a Redis Stream only when the owner actually changed — avoiding redundant fan-out.

## Stack

| Concern          | Technology                                                      |
|------------------|-----------------------------------------------------------------|
| Language         | Go 1.23                                                         |
| Real-time        | WebSockets (`coder/websocket`) + Redis Streams                  |
| Storage          | Redis                                                           |
| API contracts    | Protocol Buffers (supports JSON and binary wire formats)        |
| Metrics          | Prometheus (decorator pattern over the core service)            |
| Config           | YAML + environment variable overrides (`koanf`)                 |
| Scheduling       | `gocron` (bookkeeper interval jobs)                             |
| Testing          | `testify` + `dockertest` (integration tests against real Redis) |
| Containerization | Docker (multi-stage build, non-root runtime)                    |

## API

| Method | Path                          | Description                                |
|--------|-------------------------------|--------------------------------------------|
| `POST` | `/v2/rpc/click`               | Claim a tile for a country                 |
| `GET`  | `/v2/rpc/map-density`         | Total number of clicks across the map      |
| `POST` | `/v2/rpc/ownerships-by-batch` | Bulk fetch tile ownership for a tile range |
| `GET`  | `/ws`                         | WebSocket stream of real-time tile updates |
| `GET`  | `/metrics`                    | Prometheus metrics                         |

Request/response bodies use Protocol Buffers. The server supports both binary and JSON wire formats, configured via `httpServer.format`.

## Running locally

```bash
# Start Redis
make dbUp

# Run the API server
go run ./cmd/api -config cmd/api/example.yaml

# Run the bookkeeper
go run ./cmd/bookkeeper -config cmd/bookkeeper/example.yaml

# Run tests
make test
```

See `cmd/api/example.yaml` for the full configuration schema.
