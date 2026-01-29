# 🐢 Turtle Backend

Turtle Backend is a **production-grade Go + GraphQL backend** for building delivery platforms.  
It is designed with clean architecture, scalability, and real-world startup practices.

---

## Tech Stack

| Layer         | Tech                                        |
| ------------- | ------------------------------------------- |
| Language      | Go                                          |
| API           | GraphQL (gqlgen)                            |
| Server        | Gin                                         |
| Database      | PostgreSQL                                  |
| ORM           | GORM                                        |
| Cache / Queue | Redis                                       |
| Auth          | JWT (Rotating Access/Refresh)               |
| Logging       | Zap                                         |
| Config        | Multi-Env (.env.dev / .env.uat / .env.prod) |

---

## Project Structure

```
cmd/api          → API entrypoint
config           → Environment loader
db               → PostgreSQL connector
models           → GORM models
graph            → GraphQL schema & resolvers
services         → Business logic (auth, otp, etc)
pkg              → Shared libraries (JWT etc)
infra            → Logging, Redis, tracing
queue            → Async background jobs
middlewares      → HTTP & Auth middlewares
```

---

## Authentication Architecture

| Feature                   | Implemented |
| ------------------------- | ----------- |
| Short-lived access tokens | ✅          |
| Rotating refresh tokens   | ✅          |
| Multi-device sessions     | ✅          |
| Replay attack protection  | ✅          |
| Forced logout             | ✅          |
| OTP onboarding            | ✅          |
| Redis backed infra        | ✅          |

---

## Auth Flows

| Role     | Login Method               |
| -------- | -------------------------- |
| Customer | Apple / Google + Email OTP |
| Captain  | Mobile number + OTP        |
| Admin    | Firebase Identity + RBAC   |

---

## Requirements

- Go 1.20+
- PostgreSQL 14+
- Redis

---

## Setup

### 1. Clone repository

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
REDIS_URL=localhost:6379
```

### 5. Run backend

```bash
APP_ENV=dev go run cmd/api/main.go
```

### 6. Open Playground

http://localhost:8080

---

## GraphQL Examples

### Health

```graphql
query {
  health
}
```

### Social Login + Send OTP + Verify OTP

```graphql
mutation socialLogin {
  socialLogin(provider: GOOGLE, providerToken: "demo-token") {
    accessToken
    refreshToken
    userId
    role
  }
}

mutation sendOTP {
  sendOtp(target: "7543875613", purpose: "CUSTOMER_LOGIN")
}

mutation verifyOTP {
  verifyOtp(target: "7543875613", code: "656014", purpose: "CUSTOMER_LOGIN") {
    accessToken
    refreshToken
    userId
    role
    userId
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

| Module            | Status |
| ----------------- | ------ |
| Authentication    | ✅     |
| Orders            | ⏳     |
| Captain Matching  | ⏳     |
| Live Tracking     | ⏳     |
| Wallet / Payments | ⏳     |
| Admin APIs        | ⏳     |
