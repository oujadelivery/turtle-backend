# 🐢 Turtle Backend

Turtle Backend is a **production-grade Go + GraphQL backend** for building delivery platforms.  
It is designed with clean architecture, scalability, and real-world startup practices.

---

## Tech Stack

| Layer         | Tech                             |
| ------------- | -------------------------------- |
| Language      | Go                               |
| API           | GraphQL (gqlgen)                 |
| Server        | Gin                              |
| Database      | PostgreSQL                       |
| ORM           | GORM                             |
| Cache / Queue | Redis                            |
| Auth          | JWT                              |
| Logging       | Zap                              |
| Config        | Multi-Env (.env.dev / .env.prod) |

---

## Project Structure

```
cmd/api          → API entrypoint
config           → Environment loader
db               → PostgreSQL connector
models           → GORM models
graph            → GraphQL schema & resolvers
services         → Business logic
pkg              → Shared libraries (JWT etc)
infra            → Logging, Redis, tracing
queue            → Async background jobs
```

---

## Requirements

- Go 1.20+
- PostgreSQL 14+
- Redis

---

## Setup

### 1. Clone the repository

```bash
git clone <your-repo-url>
cd turtle-backend
```

### 2. Install dependencies

```bash
go mod tidy
```

### 3. Create databases

```bash
createdb turtle_db_dev
createdb turtle_db_prod
```

### 4. Configure environment

Create `.env.dev`

```env
DB_URL=host=localhost user=postgres password=postgres dbname=turtle_db_dev port=5432 sslmode=disable
JWT_SECRET=supersecretkey
```

### 5. Run backend

```bash
APP_ENV=dev go run cmd/api/main.go
```

### 6. Open GraphQL Playground

http://localhost:8080

---

## GraphQL API

### Health

```graphql
query {
  health
}
```

### Social Login

```graphql
mutation {
  socialLogin(provider: GOOGLE, providerToken: "demo-token") {
    token
  }
}
```

---

## Production Ready Features

- Multi-environment configuration
- Structured logging
- Crash-safe server
- Health monitoring
- Queue system ready
- Clean layered architecture

---

## Roadmap

- JWT middleware
- Role-based users (Customer / Captain / Admin)
- Order creation
- Captain matching
- Live tracking
- Payments & wallet settlement
