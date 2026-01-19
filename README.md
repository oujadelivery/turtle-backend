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




# Critical Improvements & Edge Cases - Turtle Backend

## 🎯 Overview
This document outlines critical improvements made to the GraphQL backend for production readiness, performance, and scalability.

---

## 🚨 Critical Edge Cases Addressed

### 1. **Race Conditions & Concurrent Modifications**

#### Problem:
Multiple captains accepting the same order simultaneously.

#### Solution:
```go
// Triple-layer protection:
1. Distributed lock with Redis (prevents concurrent processing)
2. Row-level database locking with GORM (FOR UPDATE)
3. Optimistic locking with version field (detects concurrent updates)

// In order service:
lockKey := fmt.Sprintf("order:accept:%d", orderID)
locked, err := infra.Redis.SetNX(ctx, lockKey, captainID, 10*time.Second).Result()
if !locked {
    return nil, ErrOrderLocked
}

// Database transaction with row locking
tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", orderID).First(&order)

// Optimistic locking check
result := tx.Model(&order).Where("version = ?", order.Version).Updates(...)
if result.RowsAffected == 0 {
    return ErrConcurrentUpdate
}
```

### 2. **Duplicate Order Prevention**

#### Problem:
User submits order twice due to slow network/double-click.

#### Solution:
```go
// Idempotency key enforcement
type CreateOrderInput struct {
    IdempotencyKey string // Required!
    // ... other fields
}

// Check for duplicate submissions
idempotencyLock := fmt.Sprintf("idempotency:%s", input.IdempotencyKey)
locked, err := infra.Redis.SetNX(ctx, idempotencyLock, "1", 10*time.Minute).Result()
if !locked {
    // Return existing order instead of creating duplicate
    var existingOrder models.Order
    if err := db.DB.Where("idempotency_key = ?", input.IdempotencyKey).First(&existingOrder).Error; err == nil {
        return &existingOrder, nil
    }
}
```

### 3. **N+1 Query Problem**

#### Problem:
Loading 100 orders causes 100+ database queries (1 for orders + 1 per customer + 1 per captain).

#### Solution:
```go
// DataLoader implementation batches queries
// Before: 101 queries
// After: 2-3 queries

// Usage in resolver:
func (r *orderResolver) Customer(ctx context.Context, obj *models.Order) (*models.User, error) {
    return dataloader.For(ctx).UserLoader.Load(ctx, obj.CustomerID)
}

// All customer loads are automatically batched into a single query:
// SELECT * FROM users WHERE id IN (1,2,3,...,100)
```

### 4. **OTP Security Issues**

#### Problem:
- Predictable OTPs (sequential numbers)
- No expiry time
- Unlimited attempts
- Replay attacks

#### Solution:
```go
// Secure OTP generation with crypto/rand
func generateSecureOTP() string {
    otp := make([]byte, 6)
    _, err := rand.Read(otp)
    // ... convert to numeric OTP
}

// OTP expiry tracking
type Order struct {
    PickupOTPExpiresAt   *time.Time
    DeliveryOTPExpiresAt *time.Time
    OTPAttempts          int // Max 5 attempts
}

// Validation with expiry and rate limiting
func (o *Order) ValidatePickupOTP(otp string) error {
    if o.OTPAttempts >= 5 {
        return errors.New("maximum OTP attempts exceeded")
    }
    if time.Now().After(*o.PickupOTPExpiresAt) {
        return errors.New("OTP has expired")
    }
    // ... validate OTP
}
```

### 5. **Payment Edge Cases**

#### Problem:
- Concurrent wallet deductions
- Refund processing during cancellation
- Payment gateway webhooks arriving out of order

#### Solution:
```go
// Wallet optimistic locking
type User struct {
    WalletBalance float64
    WalletVersion int // Optimistic locking for concurrent transactions
}

// Safe wallet deduction
result := tx.Model(&user).
    Where("id = ? AND wallet_version = ?", userID, currentVersion).
    Updates(map[string]interface{}{
        "wallet_balance": balance - amount,
        "wallet_version": gorm.Expr("wallet_version + 1"),
    })

if result.RowsAffected == 0 {
    return ErrConcurrentUpdate
}
```

### 6. **Captain Going Offline Mid-Delivery**

#### Problem:
Captain disconnects/app crashes during active delivery.

#### Solution:
```go
// Track last active time
type User struct {
    LastActiveAt *time.Time
    CurrentLocationUpdatedAt *time.Time
}

// Background job to detect offline captains
func detectOfflineCaptains() {
    threshold := time.Now().Add(-5 * time.Minute)
    
    var offlineCaptains []User
    db.DB.Where("role = ? AND is_available = true AND last_active_at < ?", 
        "CAPTAIN", threshold).Find(&offlineCaptains)
    
    for _, captain := range offlineCaptains {
        // Mark as unavailable
        // Reassign active orders
        // Notify admin
    }
}
```

### 7. **Geolocation Precision Issues**

#### Problem:
Inaccurate distance calculations, missing nearby orders.

#### Solution:
```go
// Use PostGIS for accurate geospatial queries
db.Exec(`CREATE EXTENSION IF NOT EXISTS postgis`)

// Add geography column
db.Exec(`ALTER TABLE users ADD COLUMN location geography(POINT, 4326)`)

// Efficient nearby search with spatial index
db.Exec(`CREATE INDEX idx_users_location_gist 
    ON users USING GIST(location) 
    WHERE role = 'CAPTAIN' AND is_available = true`)

// Query with proper distance calculation
query := `
    SELECT * FROM users
    WHERE ST_DWithin(
        location,
        ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
        $3 * 1000 -- radius in meters
    )
    ORDER BY location <-> ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography
    LIMIT 20
`
```

### 8. **Time Zone Handling**

#### Problem:
Inconsistent timestamps across different regions.

#### Solution:
```go
// Store timezone in order
type Order struct {
    Timezone string `gorm:"size:50;default:'Asia/Kolkata'"`
    PlacedAt time.Time
}

// Always work in UTC, convert for display
func (o *Order) GetLocalPlacedAt() time.Time {
    loc, _ := time.LoadLocation(o.Timezone)
    return o.PlacedAt.In(loc)
}
```

### 9. **Status Transition Validation**

#### Problem:
Invalid status changes (e.g., DELIVERED → PENDING).

#### Solution:
```go
// Define valid transitions
var validTransitions = map[string][]string{
    "PENDING":          {"ACCEPTED", "CANCELLED"},
    "ACCEPTED":         {"CAPTAIN_ARRIVING", "CANCELLED"},
    "CAPTAIN_ARRIVING": {"PICKED_UP", "CANCELLED"},
    "PICKED_UP":        {"IN_TRANSIT"},
    "IN_TRANSIT":       {"DELIVERED"},
}

func validateTransition(from, to string) error {
    allowed, ok := validTransitions[from]
    if !ok {
        return ErrInvalidTransition
    }
    
    for _, valid := range allowed {
        if valid == to {
            return nil
        }
    }
    
    return fmt.Errorf("cannot transition from %s to %s", from, to)
}
```

### 10. **Memory Leaks in Subscriptions**

#### Problem:
WebSocket connections not properly cleaned up.

#### Solution:
```go
// Proper context management
func (r *subscriptionResolver) OrderUpdated(ctx context.Context, orderID int) (<-chan *models.Order, error) {
    ch := make(chan *models.Order, 10) // Buffered channel
    
    go func() {
        defer close(ch) // Always close channel
        
        for {
            select {
            case event := <-eventCh:
                select {
                case ch <- order:
                case <-ctx.Done(): // Respect context cancellation
                    return
                }
            case <-ctx.Done(): // Client disconnected
                return
            }
        }
    }()
    
    return ch, nil
}
```

---

## 🚀 Performance Improvements

### 1. **Database Indexing**

```sql
-- Composite index for common queries
CREATE INDEX CONCURRENTLY idx_orders_customer_status 
    ON orders(customer_id, status) WHERE deleted_at IS NULL;

CREATE INDEX CONCURRENTLY idx_orders_status_created 
    ON orders(status, created_at DESC) WHERE deleted_at IS NULL;

-- Partial index for pending orders
CREATE INDEX CONCURRENTLY idx_orders_pending_pickup 
    ON orders(status, pickup_lat, pickup_lng) 
    WHERE deleted_at IS NULL AND status = 'PENDING';

-- GIST index for geospatial queries
CREATE INDEX idx_users_location_gist 
    ON users USING GIST(location) 
    WHERE role = 'CAPTAIN' AND is_available = true;
```

### 2. **Connection Pooling**

```go
// Optimized connection pool settings
sqlDB.SetMaxOpenConns(100)        // Max connections
sqlDB.SetMaxIdleConns(10)         // Idle connections
sqlDB.SetConnMaxLifetime(30 * time.Minute)  // Connection lifetime
sqlDB.SetConnMaxIdleTime(10 * time.Minute)  // Idle timeout
```

### 3. **Query Optimization**

```go
// Use prepared statements
config := &gorm.Config{
    PrepareStmt: true, // Reuse prepared statements
}

// Skip default transactions for reads
config.SkipDefaultTransaction = true

// Use Select to fetch only needed fields
db.DB.Select("id", "order_number", "status").Find(&orders)

// Use Preload wisely (avoid over-fetching)
db.DB.Preload("Customer", func(db *gorm.DB) *gorm.DB {
    return db.Select("id", "first_name", "last_name")
}).Find(&orders)
```

### 4. **Caching Strategy**

```go
// Cache hot data in Redis
func GetUserByID(userID uint) (*models.User, error) {
    cacheKey := fmt.Sprintf("user:%d", userID)
    
    // Try cache first
    cached, err := infra.Redis.Get(ctx, cacheKey).Bytes()
    if err == nil {
        var user models.User
        json.Unmarshal(cached, &user)
        return &user, nil
    }
    
    // Cache miss - fetch from DB
    var user models.User
    if err := db.DB.First(&user, userID).Error; err != nil {
        return nil, err
    }
    
    // Store in cache (5 minutes TTL)
    data, _ := json.Marshal(user)
    infra.Redis.Set(ctx, cacheKey, data, 5*time.Minute)
    
    return &user, nil
}
```

---

## 🔒 Security Improvements

### 1. **Rate Limiting**

```go
// Global rate limit
router.Use(httprate.LimitByIP(100, 1*time.Minute))

// Stricter for GraphQL
r.Use(httprate.LimitByIP(30, 1*time.Minute))

// Very strict for file uploads
r.Use(httprate.LimitByIP(10, 5*time.Minute))
```

### 2. **Request Size Limits**

```go
server := &http.Server{
    MaxHeaderBytes: 1 << 20, // 1 MB max headers
    ReadTimeout:    15 * time.Second,
    WriteTimeout:   15 * time.Second,
}

// File upload limits
transport.MultipartForm{
    MaxMemory:     32 << 20, // 32 MB in memory
    MaxUploadSize: 10 << 20, // 10 MB max file
}
```

### 3. **Security Headers**

```go
w.Header().Set("X-Content-Type-Options", "nosniff")
w.Header().Set("X-Frame-Options", "DENY")
w.Header().Set("X-XSS-Protection", "1; mode=block")
w.Header().Set("Strict-Transport-Security", "max-age=31536000")
```

### 4. **Input Validation**

```go
// Validate all inputs
func validateOrderInput(input CreateOrderInput) error {
    if input.PickupPhone == "" || len(input.PickupPhone) < 10 {
        return errors.New("invalid pickup phone")
    }
    
    if input.ParcelWeight < 0 || input.ParcelWeight > 100 {
        return errors.New("invalid parcel weight")
    }
    
    // Validate lat/lng ranges
    if input.PickupLat < -90 || input.PickupLat > 90 {
        return errors.New("invalid latitude")
    }
    
    return nil
}
```

---

## 📊 Monitoring & Observability

### 1. **Prometheus Metrics**

```go
// Request metrics
httpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration)

// Business metrics
ordersCreated.Inc()
ordersCompleted.Inc()
captainOnline.Set(float64(count))
```

### 2. **Health Checks**

```go
// Liveness: Is the service running?
GET /live → 200 OK

// Readiness: Can it serve traffic?
GET /ready → Checks DB, Redis, etc.

// Detailed health
GET /health → Service version, uptime, etc.
```

### 3. **Distributed Tracing**

```go
// Add request ID to all logs
requestID := middleware.GetReqID(ctx)
logger.Info("Processing order",
    "request_id", requestID,
    "order_id", orderID,
    "user_id", userID,
)
```

---

## 🎯 Deployment Checklist

### Before Production:

- [ ] Enable HTTPS with valid certificates
- [ ] Configure proper CORS origins
- [ ] Set up database replication (read replicas)
- [ ] Configure Redis Sentinel/Cluster
- [ ] Set up automated backups
- [ ] Configure log aggregation (ELK/DataDog)
- [ ] Set up alerts (PagerDuty/OpsGenie)
- [ ] Load testing (artillery.io / k6)
- [ ] Security audit (penetration testing)
- [ ] Configure CDN for static assets
- [ ] Set up DDoS protection (CloudFlare)
- [ ] Document API (OpenAPI/Swagger)
- [ ] Set up monitoring dashboards (Grafana)
- [ ] Configure auto-scaling rules
- [ ] Implement circuit breakers
- [ ] Set up feature flags
- [ ] Create runbooks for incidents

---

## 🔧 Performance Benchmarks

Expected performance with proper setup:

- **Orders per second**: 1000+ (with caching)
- **GraphQL query latency**: <50ms (p95)
- **Database query time**: <10ms (with indexes)
- **Nearby captain search**: <20ms (with PostGIS)
- **Concurrent users**: 10,000+

---

## 📚 Additional Resources

- [GORM Performance](https://gorm.io/docs/performance.html)
- [GraphQL Best Practices](https://graphql.org/learn/best-practices/)
- [PostgreSQL Performance](https://wiki.postgresql.org/wiki/Performance_Optimization)
- [Redis Best Practices](https://redis.io/docs/management/optimization/)
- [Go Concurrency Patterns](https://go.dev/blog/pipelines)

---

## 🤝 Contributing

When adding new features:
1. Always use transactions for multi-step operations
2. Add appropriate indexes for new query patterns
3. Use DataLoader for related entities
4. Implement rate limiting for new endpoints
5. Add metrics for monitoring
6. Write tests (unit + integration)
7. Update documentation