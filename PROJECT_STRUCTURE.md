# 📁 Turtle Project Structure

Complete overview of the project's file organization and architecture.

---

## 🎯 Directory Structure

```
turtle/
│
├── cmd/                                # Application entry points
│   └── server/
│       └── main.go                     # Server entry point
│
├── internal/                           # Private application code
│   │
│   ├── application/                    # Application layer
│   │   ├── services/                  # NEW! Service layer
│   │   │   ├── user_service.go        # User business logic + DataLoader
│   │   │   └── address_service.go     # Address business logic + batching
│   │   │
│   │   └── usecases/                  # Use case implementations
│   │       └── authentication.go      # Auth workflows
│   │
│   ├── domain/                         # Domain layer (business logic)
│   │   ├── aggregates/                # Domain aggregates
│   │   │   ├── user.go                # User aggregate
│   │   │   └── address.go             # Address aggregate
│   │   │
│   │   ├── valueobjects/              # Value objects
│   │   │   ├── location.go            # Geographic coordinates
│   │   │   ├── money.go               # Monetary values
│   │   │   ├── contact_info.go        # Contact information
│   │   │   └── user_events.go         # Domain events
│   │   │
│   │   └── repositories.go            # Repository interfaces
│   │
│   └── infrastructure/                 # Infrastructure layer
│       ├── persistence/
│       │   └── postgres/              # PostgreSQL repositories
│       │       ├── user_repository.go
│       │       ├── address_repository.go
│       │       ├── otp_repository.go
│       │       └── refresh_token_repository.go
│       │
│       ├── cache/                     # Caching layer
│       │   └── redis.go               # Redis implementation
│       │
│       └── dataloader/                # DataLoader implementations
│           ├── user_loader.go         # User DataLoader
│           └── address_loader.go      # Address DataLoader
│
├── graph/                              # GraphQL layer
│   ├── schema/                        # GraphQL schemas
│   │   ├── schema.graphqls           # Root schema
│   │   ├── user.graphqls             # User types & operations
│   │   ├── address.graphqls          # Address types & operations
│   │   ├── auth.graphqls             # Authentication
│   │   ├── common.graphqls           # Shared types
│   │   └── scalars.graphqls          # Custom scalars
│   │
│   ├── generated/                     # Generated code (gqlgen)
│   │   └── generated.go
│   │
│   ├── model/                         # GraphQL models
│   │   └── models_gen.go             # Generated models
│   │
│   ├── resolver.go                    # Root resolver (DI)
│   ├── directives.go                  # Custom directives
│   ├── user_resolvers.go             # User resolvers
│   ├── address_resolvers.go          # Address resolvers
│   ├── auth_resolvers.go             # Auth resolvers
│   ├── schema_resolvers.go           # Root resolvers
│   ├── mappers.go                     # Domain ↔ GraphQL mappers
│   └── helpers.go                     # Resolver helpers
│
├── middleware/                         # HTTP middleware
│   ├── auth.go                        # Authentication
│   ├── logging.go                     # Request logging
│   ├── cors.go                        # CORS configuration
│   ├── ratelimit.go                   # Rate limiting
│   └── recovery.go                    # Panic recovery
│
├── pkg/                                # Public shared packages
│   ├── errors/                        # Error types
│   │   └── errors.go
│   ├── jwt/                           # JWT utilities
│   │   └── jwt.go
│   └── otp/                           # OTP generation
│       └── otp.go
│
├── migrations/                         # Database migrations
│   ├── 001_initial_schema_up.sql
│   └── 001_initial_schema_down.sql
│
├── config/                             # Configuration
│   └── config.go                      # Config management
│
├── tests/                              # Test files
│   ├── integration/
│   │   └── integration_test.go
│   └── e2e/
│
├── docs/                               # Documentation
│   ├── README.md
│   ├── API_DOCUMENTATION.md
│   ├── ARCHITECTURE.md
│   ├── TESTING_GUIDE.md
│   └── PROJECT_STRUCTURE.md
│
├── scripts/                            # Utility scripts
│   ├── migrate.sh
│   └── seed.sh
│
├── .env.example                        # Environment template
├── .gitignore                          # Git ignore
├── docker-compose.yml                  # Docker services
├── gqlgen.yml                          # GraphQL codegen config
├── go.mod                              # Go dependencies
├── go.sum                              # Dependency checksums
├── main.go                             # Application entry
└── README.md                           # Main documentation
```

---

## 📦 Layer Breakdown

### **1. Domain Layer** (`internal/domain/`)

**Purpose:** Core business logic, entities, and rules.

**Files:**
- `aggregates/user.go` (470 lines) - User entity with business logic
- `aggregates/address.go` (350 lines) - Address entity
- `valueobjects/location.go` (80 lines) - Geographic coordinates
- `valueobjects/money.go` (120 lines) - Monetary values
- `repositories.go` (200 lines) - Repository interfaces

**Key Characteristics:**
- ✅ No external dependencies
- ✅ Pure business logic
- ✅ Framework agnostic
- ✅ Highly testable

**Example:**
```go
// User aggregate with business rules
type User struct {
    id        string
    firstName string
    role      UserRole
}

func (u *User) GoOnline(location *Location) error {
    if !u.IsCaptain() {
        return errors.New("only captains can go online")
    }
    // Business logic...
}
```

---

### **2. Application Layer** (`internal/application/`)

**Purpose:** Application services, use cases, orchestration.

**Files:**
- `services/user_service.go` (425 lines) - User operations + DataLoader
- `services/address_service.go` (380 lines) - Address operations + batching
- `usecases/authentication.go` (630 lines) - Auth workflows

**Key Characteristics:**
- ✅ Orchestrates domain logic
- ✅ Manages transactions
- ✅ Integrates DataLoader
- ✅ Cache management

**Example:**
```go
type UserService struct {
    repo domain.UserRepository
}

func (s *UserService) UpdateProfile(ctx, userID, first, last string) (*User, error) {
    // 1. Fetch with DataLoader
    user, _ := s.GetUser(ctx, userID)
    
    // 2. Domain logic
    user.UpdateProfile(first, last)
    
    // 3. Persist
    s.repo.Update(ctx, user)
    
    // 4. Invalidate cache
    loader.Clear(userID)
    
    return s.GetUser(ctx, userID)
}
```

---

### **3. Infrastructure Layer** (`internal/infrastructure/`)

**Purpose:** External integrations, databases, caching.

**Files:**
- `persistence/postgres/user_repository.go` (450 lines) - PostgreSQL user repo
- `persistence/postgres/address_repository.go` (420 lines) - PostgreSQL address repo
- `cache/redis.go` (200 lines) - Redis cache
- `dataloader/user_loader.go` (180 lines) - User DataLoader
- `dataloader/address_loader.go` (160 lines) - Address DataLoader

**Key Characteristics:**
- ✅ Implements repository interfaces
- ✅ Database operations
- ✅ External API calls
- ✅ Caching logic

**Example:**
```go
type UserRepository struct {
    db *sqlx.DB
}

func (r *UserRepository) FindByIDs(ctx, ids []string) ([]*User, error) {
    // Batch query for DataLoader
    query := `SELECT * FROM users WHERE id = ANY($1)`
    // ...
}
```

---

### **4. Presentation Layer** (`graph/`)

**Purpose:** GraphQL API, resolvers, schema.

**Files:**
- `schema/*.graphqls` (1,200 lines total) - GraphQL schemas
- `user_resolvers.go` (450 lines) - User query/mutation resolvers
- `address_resolvers.go` (380 lines) - Address resolvers
- `auth_resolvers.go` (290 lines) - Auth resolvers
- `resolver.go` (55 lines) - Dependency injection
- `directives.go` (70 lines) - Custom directives
- `mappers.go` (150 lines) - Domain ↔ GraphQL conversion

**Key Characteristics:**
- ✅ HTTP/GraphQL handling
- ✅ Input validation
- ✅ Response formatting
- ✅ Error handling

**Example:**
```go
func (r *mutationResolver) UpdateProfile(ctx, input) (*model.User, error) {
    userID, _ := getUserIDFromContext(ctx)
    
    // Call service
    user, err := r.userService.UpdateProfile(
        ctx, userID, input.FirstName, input.LastName,
    )
    
    // Map to GraphQL model
    return userToGraphQL(user), nil
}
```

---

### **5. Middleware Layer** (`middleware/`)

**Purpose:** HTTP middleware for cross-cutting concerns.

**Files:**
- `auth.go` (135 lines) - JWT authentication
- `logging.go` (120 lines) - Request/response logging
- `cors.go` (60 lines) - CORS configuration
- `ratelimit.go` (150 lines) - Rate limiting
- `recovery.go` (80 lines) - Panic recovery

**Key Characteristics:**
- ✅ Request/response interception
- ✅ Cross-cutting concerns
- ✅ Chained execution

**Example:**
```go
func AuthMiddleware() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w, r) {
            // Extract and verify token
            // Set user context
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## 📊 File Statistics

| Layer | Files | Lines of Code | Purpose |
|-------|-------|---------------|---------|
| **Domain** | 8 | ~1,500 | Business logic |
| **Application** | 3 | ~1,435 | Orchestration |
| **Infrastructure** | 7 | ~1,610 | External integrations |
| **Presentation** | 12 | ~2,800 | GraphQL API |
| **Middleware** | 5 | ~545 | Cross-cutting |
| **Package** | 4 | ~600 | Utilities |
| **Total** | **39** | **~8,490** | |

---

## 🔄 Data Flow

### **Query Flow Example: Get User**

```
1. HTTP Request
   ↓
2. Middleware Chain
   ├── CORS → Add headers
   ├── Auth → Extract user_id from JWT
   ├── Rate Limit → Check Redis
   ├── DataLoader → Create per-request loaders
   └── Logging → Log request
   ↓
3. GraphQL Layer (graph/user_resolvers.go)
   └── Call userService.GetUser()
   ↓
4. Service Layer (application/services/user_service.go)
   └── Use DataLoader for batching
   ↓
5. DataLoader (infrastructure/dataloader/user_loader.go)
   ├── Collect requests (16ms window)
   ├── Check cache
   └── Batch into single query
   ↓
6. Repository (infrastructure/persistence/postgres/user_repository.go)
   └── Execute: SELECT * FROM users WHERE id = ANY($1)
   ↓
7. PostgreSQL
   └── Return results
   ↓
8. Response Flow (reverse)
   └── Domain → Service → Resolver → GraphQL → JSON
```

### **Mutation Flow Example: Update Profile**

```
1. HTTP Request with JWT
   ↓
2. Middleware (auth, rate limit)
   ↓
3. Resolver (graph/user_resolvers.go)
   └── Call userService.UpdateProfile()
   ↓
4. Service (application/services/user_service.go)
   ├── Fetch user via DataLoader
   ├── Call domain method
   ├── Save via repository
   ├── Invalidate cache
   └── Reload from DataLoader
   ↓
5. Domain (domain/aggregates/user.go)
   └── user.UpdateProfile() - Business rules
   ↓
6. Repository
   └── UPDATE users SET ... WHERE id = $1
   ↓
7. Response
   └── Updated user object
```

---

## 🗄️ Database Schema

**Users Table:**
```sql
CREATE TABLE users (
    id VARCHAR(255) PRIMARY KEY,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    email VARCHAR(255) UNIQUE,
    phone VARCHAR(20) UNIQUE,
    phone_verified BOOLEAN DEFAULT FALSE,
    role VARCHAR(50) NOT NULL,
    status VARCHAR(50) DEFAULT 'ACTIVE',
    profile_pic TEXT,
    
    -- Captain specific
    vehicle_type VARCHAR(50),
    vehicle_number VARCHAR(50),
    kyc_status VARCHAR(50),
    is_available BOOLEAN DEFAULT FALSE,
    current_location GEOGRAPHY(POINT),
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

**Addresses Table:**
```sql
CREATE TABLE addresses (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) REFERENCES users(id),
    label VARCHAR(50),
    address_line1 TEXT NOT NULL,
    city VARCHAR(100) NOT NULL,
    location GEOGRAPHY(POINT) NOT NULL,
    is_default BOOLEAN DEFAULT FALSE,
    usage_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);
```

---

## 📝 Configuration Files

### **gqlgen.yml** (GraphQL Code Generation)

```yaml
schema:
  - graph/schema/*.graphqls

exec:
  filename: graph/generated/generated.go

model:
  filename: graph/model/models_gen.go

resolver:
  layout: follow-schema
  dir: graph
  filename_template: "{name}.resolvers.go"

directives:
  auth:
    skip_runtime: false  # Implemented
  rateLimit:
    skip_runtime: true   # Handled by middleware
```

### **docker-compose.yml** (Development Environment)

```yaml
services:
  postgres:
    image: postgis/postgis:15-3.4
    ports:
      - "5432:5432"
    environment:
      POSTGRES_DB: turtle_db
      POSTGRES_USER: turtle
      POSTGRES_PASSWORD: turtle_password
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
```

---

## 🔧 Key Design Patterns

### **1. Repository Pattern**
```go
// Interface in domain
type UserRepository interface {
    FindByID(ctx, id) (*User, error)
}

// Implementation in infrastructure
type PostgresUserRepository struct {
    db *sqlx.DB
}
```

### **2. Service Layer Pattern**
```go
// Encapsulates business logic + DataLoader
type UserService struct {
    repo domain.UserRepository
}
```

### **3. DataLoader Pattern**
```go
// Batches and caches requests
loader := dataloader.NewUserLoader(repo)
user, _ := loader.LoadUser(ctx, id)
```

### **4. Dependency Injection**
```go
// Wire up dependencies
resolver := graph.NewResolver(
    userService,
    addressService,
    authService,
    cache,
)
```

---

## 📚 Code Conventions

### **Naming**
- **Files**: `snake_case.go`
- **Packages**: lowercase, single word
- **Types**: `PascalCase`
- **Functions**: `PascalCase` (exported), `camelCase` (private)
- **Constants**: `PascalCase` or `SCREAMING_SNAKE_CASE`

### **Organization**
- One aggregate per file
- Group related functions
- Interfaces in domain layer
- Implementations in infrastructure

### **Comments**
```go
// Public functions have godoc comments
// Example explains what the function does
func (s *UserService) GetUser(ctx, id string) (*User, error) {
    // Implementation comments explain why
}
```

---

## 🎯 Next Steps

1. **Order System**: Add order aggregates, repositories, services
2. **Subscriptions**: Real-time updates via WebSocket
3. **Payment Integration**: Stripe/PayPal integration
4. **Analytics**: Metrics and monitoring
5. **Microservices**: Split into separate services

---

## 📖 Related Documentation

- [Architecture Guide](./ARCHITECTURE.md)
- [API Documentation](./API_DOCUMENTATION.md)
- [Testing Guide](./TESTING_GUIDE.md)
- [Deployment Guide](./DEPLOYMENT.md)

---

**Project Structure follows Clean Architecture principles for scalability to millions of users! 🚀**
