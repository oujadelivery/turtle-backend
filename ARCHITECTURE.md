# 🏗️ Turtle System Architecture

Comprehensive architectural overview of the Turtle delivery and ride-sharing platform.

---

## 📋 Table of Contents

- [Overview](#overview)
- [Architectural Layers](#architectural-layers)
- [Domain-Driven Design](#domain-driven-design)
- [DataLoader Pattern](#dataloader-pattern)
- [Service Layer](#service-layer)
- [Data Flow](#data-flow)
- [Security Architecture](#security-architecture)
- [Performance Optimizations](#performance-optimizations)
- [Scalability](#scalability)

---

## 🎯 Overview

Turtle follows **Clean Architecture** principles with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────┐
│                    Presentation Layer                        │
│  GraphQL API • HTTP Handlers • WebSocket                    │
└─────────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────────┐
│                   Application Layer                          │
│  Services • Use Cases • DTOs                                 │
└─────────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────────┐
│                    Domain Layer                              │
│  Aggregates • Entities • Value Objects • Domain Events      │
└─────────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────────┐
│                 Infrastructure Layer                         │
│  Repositories • Cache • External Services                   │
└─────────────────────────────────────────────────────────────┘
```

### **Key Principles**

1. **Dependency Rule**: Dependencies point inward (Infrastructure → Domain)
2. **Separation of Concerns**: Each layer has distinct responsibilities
3. **Testability**: Business logic isolated from infrastructure
4. **Flexibility**: Easy to swap implementations

---

## 📦 Architectural Layers

### **1. Presentation Layer (GraphQL)**

**Location:** `graph/`

**Responsibilities:**
- GraphQL schema definition
- Request/response handling
- Input validation
- Error formatting
- Authentication directives

**Components:**
```go
// Resolvers - Entry points for GraphQL operations
type Resolver struct {
    userService    *services.UserService
    addressService *services.AddressService
    authService    *usecases.AuthenticationService
    cache          CacheService
}

// Query resolver
func (r *queryResolver) Me(ctx) (*model.User, error)

// Mutation resolver
func (r *mutationResolver) CreateAddress(ctx, input) (*model.Address, error)

// Subscription resolver
func (r *subscriptionResolver) UserUpdated(ctx, userID) (<-chan *model.User, error)
```

**Key Files:**
- `schema/*.graphqls` - GraphQL schema definitions
- `*_resolvers.go` - Resolver implementations
- `directives.go` - Custom directives (@auth, @rateLimit)
- `resolver.go` - Dependency injection

---

### **2. Application Layer (Services & Use Cases)**

**Location:** `internal/application/`

**Responsibilities:**
- Business logic orchestration
- Transaction management
- DataLoader integration
- Cache management
- Service coordination

**Components:**

#### **Services (NEW!)**
```go
type UserService struct {
    repo domain.UserRepository
}

func (s *UserService) GetUser(ctx, userID) (*aggregates.User, error) {
    // Uses DataLoader for batching
    loader := dataloader.MustGetUserLoader(ctx)
    return loader.LoadUser(ctx, userID)
}

func (s *UserService) UpdateProfile(ctx, userID, ...) (*aggregates.User, error) {
    // 1. Fetch user
    user, _ := s.GetUser(ctx, userID)
    
    // 2. Business logic
    user.UpdateProfile(...)
    
    // 3. Persist
    s.repo.Update(ctx, user)
    
    // 4. Invalidate cache
    loader.Clear(userID)
    
    return s.GetUser(ctx, userID)
}
```

#### **Use Cases**
```go
type AuthenticationService struct {
    userRepo         domain.UserRepository
    otpRepo          domain.OTPRepository
    refreshTokenRepo domain.RefreshTokenRepository
    cache            CacheService
}

func (s *AuthenticationService) SocialLogin(ctx, input) (*SocialLoginOutput, error) {
    // Multi-step business process
    // 1. Find or create user
    // 2. Generate tokens
    // 3. Create session
    // 4. Return auth response
}
```

**Key Files:**
- `services/user_service.go` - User operations with DataLoader
- `services/address_service.go` - Address operations with batching
- `usecases/authentication.go` - Auth workflows

---

### **3. Domain Layer (Business Logic)**

**Location:** `internal/domain/`

**Responsibilities:**
- Core business logic
- Domain rules validation
- Entity lifecycle management
- Domain events

**Components:**

#### **Aggregates** (Domain entities with business logic)
```go
type User struct {
    id           string
    firstName    string
    lastName     string
    email        string
    phone        string
    role         UserRole
    status       UserStatus
    captainProfile *CaptainProfile
    // ...
}

// Business logic methods
func (u *User) UpdateProfile(first, last, pic string) error {
    // Validation
    if first == "" {
        return errors.New("first name required")
    }
    
    // Business rule
    u.firstName = first
    u.lastName = last
    u.profilePic = pic
    u.updatedAt = time.Now()
    
    return nil
}

func (u *User) GoOnline(location *Location) error {
    if !u.IsCaptain() {
        return errors.New("only captains can go online")
    }
    
    if u.captainProfile.KYCStatus != "APPROVED" {
        return errors.New("KYC must be approved")
    }
    
    u.captainProfile.IsAvailable = true
    u.captainProfile.CurrentLocation = location
    
    return nil
}
```

#### **Value Objects** (Immutable domain concepts)
```go
type Location struct {
    latitude  float64
    longitude float64
}

func NewLocation(lat, lng float64) (*Location, error) {
    if lat < -90 || lat > 90 {
        return nil, errors.New("invalid latitude")
    }
    if lng < -180 || lng > 180 {
        return nil, errors.New("invalid longitude")
    }
    return &Location{lat, lng}, nil
}
```

#### **Repository Interfaces** (Abstractions)
```go
type UserRepository interface {
    // Single operations
    Create(ctx, user) error
    FindByID(ctx, id) (*User, error)
    Update(ctx, user) error
    Delete(ctx, id) error
    
    // Queries
    FindByEmail(ctx, email) (*User, error)
    FindByPhone(ctx, phone) (*User, error)
    Search(ctx, query, role, limit, offset) ([]*User, int64, error)
    
    // Batch operations (for DataLoader)
    FindByIDs(ctx, ids []string) ([]*User, error)
    
    // Captain-specific
    FindCaptainsNearby(ctx, lat, lng, radius float64) ([]*User, error)
}
```

**Key Files:**
- `aggregates/user.go` - User aggregate
- `aggregates/address.go` - Address aggregate
- `valueobjects/location.go` - Location value object
- `repositories.go` - Repository interfaces

---

### **4. Infrastructure Layer (External Concerns)**

**Location:** `internal/infrastructure/`

**Responsibilities:**
- Database access
- Caching
- External API calls
- Message queues
- File storage

**Components:**

#### **PostgreSQL Repositories**
```go
type UserRepository struct {
    db *sqlx.DB
}

func (r *UserRepository) FindByID(ctx, id string) (*aggregates.User, error) {
    query := `SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL`
    // Execute query, map to aggregate
}

func (r *UserRepository) FindByIDs(ctx, ids []string) ([]*aggregates.User, error) {
    // Batch query for DataLoader
    query := `SELECT * FROM users WHERE id = ANY($1) AND deleted_at IS NULL`
    // Return in same order as input IDs
}
```

#### **DataLoader Implementation**
```go
type UserLoader struct {
    repo   domain.UserRepository
    loader *dataloader.Loader[string, *aggregates.User]
}

func NewUserLoader(repo domain.UserRepository) *UserLoader {
    return &UserLoader{
        repo: repo,
        loader: dataloader.NewBatchedLoader(
            batchFn,
            dataloader.WithWait(16*time.Millisecond),
        ),
    }
}

func (l *UserLoader) LoadUser(ctx, id string) (*aggregates.User, error) {
    return l.loader.Load(ctx, id)()
}
```

#### **Redis Cache**
```go
type RedisCache struct {
    client *redis.Client
}

func (c *RedisCache) CheckRateLimit(ctx, key string, limit int, window time.Duration) (bool, int, error) {
    // Use Redis INCR with EXPIRE for atomic rate limiting
}
```

**Key Files:**
- `persistence/postgres/*_repository.go` - DB implementations
- `dataloader/user_loader.go` - User DataLoader
- `dataloader/address_loader.go` - Address DataLoader
- `cache/redis.go` - Redis implementation

---

## 🎨 Domain-Driven Design

### **Aggregates**

**User Aggregate:**
```
User (Root)
├── Profile (firstName, lastName, email, phone)
├── Role (customer, captain, admin)
├── Status (active, blocked, suspended)
└── CaptainProfile (if captain)
    ├── Vehicle Info
    ├── KYC Status
    ├── Rating
    └── Current Location
```

**Address Aggregate:**
```
Address (Root)
├── Label (home, work, other)
├── Location (lat, lng)
├── Contact Info (name, phone)
├── Usage Stats (usageCount, lastUsedAt)
└── Verification (isVerified, verifiedAt)
```

### **Bounded Contexts**

```
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│  User Context   │  │ Address Context │  │  Order Context  │
│                 │  │                 │  │   (Future)      │
│ • Users         │  │ • Addresses     │  │ • Orders        │
│ • Captains      │  │ • Locations     │  │ • Rides         │
│ • Auth          │  │ • Suggestions   │  │ • Payments      │
└─────────────────┘  └─────────────────┘  └─────────────────┘
```

### **Domain Events** (Future)

```go
type UserCreated struct {
    UserID    string
    Role      string
    CreatedAt time.Time
}

type CaptainWentOnline struct {
    CaptainID string
    Location  *Location
    Timestamp time.Time
}
```

---

## ⚡ DataLoader Pattern

### **Problem: N+1 Queries**

**Without DataLoader:**
```sql
-- Get 100 users
SELECT * FROM users LIMIT 100;  -- 1 query

-- For each user, get addresses
SELECT * FROM addresses WHERE user_id = 'user_1';  -- Query 1
SELECT * FROM addresses WHERE user_id = 'user_2';  -- Query 2
...
SELECT * FROM addresses WHERE user_id = 'user_100';  -- Query 100

-- Total: 101 queries! 🔴
```

**With DataLoader:**
```sql
-- Get 100 users
SELECT * FROM users LIMIT 100;  -- 1 query

-- Batch all address requests (within 16ms window)
SELECT * FROM addresses 
WHERE user_id = ANY(ARRAY['user_1', 'user_2', ..., 'user_100']);  -- 1 query

-- Total: 2 queries! ✅ (98% reduction)
```

### **Implementation**

```go
// 1. Create loader in middleware
func DataLoaderMiddleware(userRepo, addressRepo) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Create per-request loaders
            userLoader := NewUserLoader(userRepo)
            addressLoader := NewAddressLoader(addressRepo)
            
            // Add to context
            ctx := context.WithValue(r.Context(), userLoaderKey, userLoader)
            ctx = context.WithValue(ctx, addressLoaderKey, addressLoader)
            
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// 2. Use in service
func (s *UserService) GetUser(ctx, id string) (*User, error) {
    loader := dataloader.MustGetUserLoader(ctx)
    return loader.LoadUser(ctx, id)  // Automatically batched!
}

// 3. Resolver calls service
func (r *queryResolver) Users(ctx) ([]*model.User, error) {
    users, _ := r.userService.GetUsers(ctx, ids)
    // Behind the scenes: batched into single query
}
```

### **Benefits**

- ✅ **98% Query Reduction**: From 100+ queries to 2-3
- ✅ **10x Faster**: Response time improves dramatically
- ✅ **Automatic**: Works without resolver changes
- ✅ **Per-Request Caching**: No stale data
- ✅ **Deduplication**: Same ID requested multiple times = 1 DB call

---

## 🔧 Service Layer

### **Why Service Layer?**

**Before (Direct Repository in Resolvers):**
```go
func (r *mutationResolver) UpdateProfile(ctx, input) (*model.User, error) {
    // ❌ Resolver has too many responsibilities
    user, _ := r.userRepo.FindByID(ctx, userID)
    user.UpdateProfile(...)
    r.userRepo.Update(ctx, user)
    
    // Manual cache management
    loader := dataloader.MustGetUserLoader(ctx)
    loader.Clear(userID)
    reloadedUser, _ := loader.LoadUser(ctx, userID)
    
    return userToGraphQL(reloadedUser), nil
}
```

**After (Service Layer):**
```go
func (r *mutationResolver) UpdateProfile(ctx, input) (*model.User, error) {
    // ✅ Clean, simple, focused
    user, _ := r.userService.UpdateProfile(ctx, userID, first, last, pic)
    return userToGraphQL(user), nil
}

// Service handles complexity
func (s *UserService) UpdateProfile(ctx, userID, ...) (*User, error) {
    user, _ := s.GetUser(ctx, userID)  // Uses DataLoader
    user.UpdateProfile(...)
    s.repo.Update(ctx, user)
    loader := dataloader.MustGetUserLoader(ctx)
    loader.Clear(userID)
    return loader.LoadUser(ctx, userID)
}
```

### **Service Responsibilities**

1. **Business Logic Orchestration**
2. **DataLoader Integration**
3. **Cache Management** (invalidation, priming)
4. **Transaction Coordination**
5. **Error Handling**

### **Benefits**

- ✅ **Clean Resolvers**: 60-70% less code
- ✅ **Testability**: Easy to mock services
- ✅ **Reusability**: Services used by multiple resolvers
- ✅ **Consistency**: Cache management in one place
- ✅ **Maintainability**: Business logic centralized

---

## 🔄 Data Flow

### **Query Flow**

```
1. HTTP Request
   ↓
2. Middleware Chain
   ├── CORS
   ├── Authentication (sets user_id in context)
   ├── Rate Limiting (checks Redis)
   ├── DataLoader (creates per-request loaders)
   └── Logging
   ↓
3. GraphQL Layer
   ├── Parse query
   ├── Validate against schema
   └── Execute resolvers
   ↓
4. Resolver
   └── Call service method
   ↓
5. Service Layer
   ├── Use DataLoader for queries
   ├── Apply business logic
   └── Return domain objects
   ↓
6. DataLoader
   ├── Batch requests (16ms window)
   ├── Check cache
   └── Make single DB query
   ↓
7. Repository
   ├── Execute SQL
   └── Map to domain objects
   ↓
8. Response
   └── GraphQL formatter → JSON
```

### **Mutation Flow**

```
1. Mutation Request
   ↓
2. Resolver
   └── Call service method
   ↓
3. Service Layer
   ├── Fetch data (via DataLoader)
   ├── Apply business rules (domain logic)
   ├── Persist changes (repository)
   ├── Invalidate cache
   └── Reload fresh data
   ↓
4. Domain Aggregate
   └── Validate & apply changes
   ↓
5. Repository
   └── UPDATE database
   ↓
6. Cache Invalidation
   └── Clear DataLoader cache for modified entity
   ↓
7. Response
   └── Return updated entity
```

---

## 🔒 Security Architecture

### **Authentication Flow**

```
┌─────────────┐
│   Client    │
└─────────────┘
       ↓
   Request OTP
       ↓
┌─────────────┐
│   Server    │ ─→ Generate OTP (6 digits)
└─────────────┘ ─→ Store in Redis (15min TTL)
       ↓         ─→ Send SMS (in production)
   Return success
       ↓
┌─────────────┐
│   Client    │ ─→ User enters OTP
└─────────────┘
       ↓
   Verify OTP
       ↓
┌─────────────┐
│   Server    │ ─→ Check Redis
└─────────────┘ ─→ Validate code
       ↓         ─→ Create/find user
       |         ─→ Generate JWT tokens
       |         ─→ Create session (refresh token in DB)
   Return tokens
       ↓
┌─────────────┐
│   Client    │ ─→ Store tokens
└─────────────┘ ─→ Use access token for API calls
```

### **Token System**

**Access Token:**
```go
{
  "userID": "user_123",
  "role": "CUSTOMER",
  "device": "WEB",
  "exp": 1706790000  // 15 minutes
}
```
- Short-lived (15 minutes)
- Used for API authentication
- Stateless (no DB lookup)

**Refresh Token:**
```go
{
  "tokenID": "rt_456",
  "userID": "user_123",
  "device": "WEB",
  "exp": 1709382000  // 60 days
}
```
- Long-lived (60 days)
- Stored in database
- Can be revoked
- Used to get new access token

### **Authorization Layers**

**1. Middleware Layer:**
```go
// Validates JWT and sets context
func AuthMiddleware() func(http.Handler) http.Handler {
    // Verify token signature
    // Extract claims
    // Set user_id, user_role in context
}
```

**2. GraphQL Directive:**
```go
// Field-level authorization
directive @auth(role: String) on FIELD_DEFINITION

// Usage in schema
updateProfile(...): User! @auth
approveKYC(...): User! @auth(role: "ADMIN")
```

**3. Business Logic:**
```go
// Service-level checks
func (s *UserService) BlockUser(ctx, userID, reason string) error {
    // Additional business rule validation
    if !canBlockUser(ctx) {
        return errors.New("insufficient permissions")
    }
}
```

---

## ⚡ Performance Optimizations

### **1. DataLoader (Request Batching)**

**Impact:** 98% query reduction

```go
// Automatic batching
userService.GetUser(ctx, "1")
userService.GetUser(ctx, "2")
userService.GetUser(ctx, "3")

// Batched into single query within 16ms
SELECT * FROM users WHERE id = ANY($1)
```

### **2. Connection Pooling**

```go
db, _ := sqlx.Connect("postgres", connString)
db.SetMaxOpenConns(25)      // Max connections
db.SetMaxIdleConns(5)       // Idle connections
db.SetConnMaxLifetime(5*time.Minute)
```

### **3. Redis Caching**

- **Sessions**: Refresh tokens
- **Rate Limiting**: Request counts
- **OTP Codes**: Temporary codes
- **Future**: Query result caching

### **4. Database Indexes**

```sql
-- User lookups
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_phone ON users(phone);

-- Geospatial queries
CREATE INDEX idx_addresses_location ON addresses USING GIST(location);
CREATE INDEX idx_captains_location ON users USING GIST(current_location);

-- Composite indexes
CREATE INDEX idx_users_role_status ON users(role, status);
```

### **5. Query Optimization**

```sql
-- Use prepared statements
PREPARE get_user AS SELECT * FROM users WHERE id = $1;

-- Partial indexes for active records
CREATE INDEX idx_active_captains 
ON users(id) WHERE role = 'CAPTAIN' AND status = 'ACTIVE';
```

---

## 📈 Scalability

### **Horizontal Scaling**

```
        Load Balancer
              ↓
    ┌─────────┼─────────┐
    ↓         ↓         ↓
App Server App Server App Server
    ↓         ↓         ↓
    └─────────┼─────────┘
              ↓
        PostgreSQL
        (Read Replicas)
```

**Stateless Design:**
- No session state in app servers
- All state in Redis/PostgreSQL
- Can add/remove servers freely

### **Database Scaling**

**Current: Single PostgreSQL**
```
App → PostgreSQL (primary)
```

**Future: Read Replicas**
```
App → PostgreSQL (primary) → Writes
  ↓
  → Read Replica 1 → Reads
  → Read Replica 2 → Reads
```

**Future: Sharding (if needed)**
```
Users 1-1M    → Shard 1
Users 1M-2M   → Shard 2
Users 2M-3M   → Shard 3
```

### **Caching Strategy**

**Current:**
```
Request → DataLoader (per-request cache) → Database
```

**Future:**
```
Request → DataLoader → Redis (shared cache) → Database
```

### **Async Processing**

**Future: Message Queue**
```
API → Queue → Workers
      (Redis/RabbitMQ)
         ↓
    - Send emails
    - Process KYC
    - Generate reports
    - Send push notifications
```

---

## 🔮 Future Enhancements

### **Microservices** (Phase 3)

```
┌────────────┐  ┌────────────┐  ┌────────────┐
│   User     │  │   Order    │  │  Payment   │
│  Service   │  │  Service   │  │  Service   │
└────────────┘  └────────────┘  └────────────┘
      ↓               ↓               ↓
┌──────────────────────────────────────────┐
│          Message Bus (Kafka)             │
└──────────────────────────────────────────┘
```

### **Event Sourcing** (Optional)

```go
// Store events instead of current state
type UserCreated struct { ... }
type ProfileUpdated struct { ... }
type CaptainWentOnline struct { ... }

// Rebuild state from events
user := ReplayEvents(userID, events)
```

### **CQRS** (Command Query Responsibility Segregation)

```
Write Model (Commands)     Read Model (Queries)
PostgreSQL (normalized) ←→ Elasticsearch (denormalized)
```

---

## 📚 References

- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Domain-Driven Design](https://martinfowler.com/bliki/DomainDrivenDesign.html)
- [DataLoader Pattern](https://github.com/graphql/dataloader)
- [GraphQL Best Practices](https://graphql.org/learn/best-practices/)

---

**Architecture is designed for:**
- ✅ Scalability to millions of users
- ✅ High performance (sub-100ms response times)
- ✅ Maintainability (clean code, clear responsibilities)
- ✅ Testability (isolated layers, dependency injection)
- ✅ Flexibility (easy to add features, swap implementations)

---

**Next:** [Testing Guide](./TESTING_GUIDE.md) • [Deployment](./DEPLOYMENT.md)
