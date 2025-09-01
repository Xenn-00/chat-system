# Real-Time Chat System
<p>
  A real-time chat backend built with <b>Go</b> for scalability, concurrency safety, and clean architecture.
</p>
<p>
  Supports <b>private and group chats</b>, with <b>WebSocket-based communication, MongoDB for messages</b>, and <b>PostgreSQL</b> for room metadata.
</p>

## ✨ Features
- 🔒 Concurrency-safe message handling
- ⚡ Real-time communication via WebSocket
- 📬 Private chat flow (lazy room creation) → room dibuat saat first message dikirim
- 👥 Group chat flow → WhatsApp/Discord-like group creation & invites
- 📨 Async worker for background tasks (priority queue, message persistence)
- 💾 Hybrid storage → MongoDB for messages, SQL for metadata
- ⏱️ Queue-based architecture (priority queue + worker)

## 🏗️ Project Tree Architecture
```
.
├── cmd/                # Entry point (main.go)
├── config/             # App configuration (config.go)
├── internal/           # Core business logic
│   ├── dtos/           # Data transfer objects (request/response)
│   ├── entity/         # Domain entities (Message, Room, User, dll.)
│   ├── errors/         # Custom error handling
│   ├── handlers/       # HTTP/WebSocket handlers
│   ├── middleware/     # Middleware (auth, logging, dsb.)
│   ├── queue/          # Queue management (priority queue)
│   ├── repo/           # Repository layer (Mongo + SQL access)
│   ├── routers/        # Routing definitions
│   ├── use-case/       # Application/business logic
│   ├── utils/          # Utility functions
│   ├── websocket/      # WebSocket hub & client management
│   └── worker/         # Async workers (process tasks dari queue)
├── migrations/         # Database migrations
├── state/              # State initialization (Mongo, Postgres, Redis, Secret)
│   ├── initMongo.go
│   ├── initPostgres.go
│   ├── initRedis.go
│   ├── initSecret.go
│   └── initState.go
├── docker-compose.yaml # Docker setup (Postgres, Mongo, Redis, dsb.)
└── ...
```
## 💡 Private Chat Flow 
1. User sent first message to subject -> system would checking if there any chat room respect to participant is already exists or not
  - If yes -> reuse
  - If not -> create new room + add chat participant
2. Store message content in MongoDB, metadata in SQL
3. Broadcast to subject user via WebSocket

## 💡 Group Chat Flow
- Working on it

## 🚀 Getting Started
<b>Prerequisites</b>
- Go `>=1.21`
- PostgreSQL / MySQL (just fit with your preference SQL)
- MongoDB
- Redis (for working queue)
<b>Run Locally</b>
```
# clone repo
git clone https://github.com/username/chat-system-go.git
cd chat-system-go

# install deps
go mod tidy

# run service
go run main.go
```
## 🛠️ Tech Stack
- Go (chi, goroutines, channels)
- WebSocket
- MongoDB
- PostgreSQL
- Redis
- Docker

## License
