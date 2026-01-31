# 🚚 Turtle Delivery - Production-Grade Backend

> A scalable, production-ready parcel delivery platform built with Clean Architecture and Domain-Driven Design

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Architecture](https://img.shields.io/badge/Architecture-Clean%20Architecture-blue)](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
[![DDD](https://img.shields.io/badge/Design-Domain%20Driven-green)](https://martinfowler.com/tags/domain%20driven%20design.html)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

## 🎯 What We've Built (Phase 1 Complete!)

A production-grade foundation for a delivery platform that can scale to millions of users. We've completed the **entire foundation layer** with enterprise patterns, comprehensive error handling, and production-ready code.

### ✅ Completed Components

- ✅ **Domain Layer** - User & Address aggregates with rich business logic
- ✅ **Value Objects** - Money, Location, Parcel, ContactInfo (all immutable & validated)
- ✅ **Authentication** - Social login, OTP, dual-role support
- ✅ **Database** - PostgreSQL with PostGIS, 20+ optimized indexes
- ✅ **Infrastructure** - Redis caching, distributed locks, rate limiting
- ✅ **Security** - JWT, OTP, token blacklisting, optimistic locking
- ✅ **Error Handling** - 30+ error types with proper HTTP codes
- ✅ **Documentation** - Comprehensive guides for all systems

**Total: 29 production-grade files | 100% foundation complete**

---

## 🌟 Key Features

### 1. Dual-Role Users 🔄
**Unique competitive advantage** - One person can be both customer AND captain:

```
John's Journey:
1. Signs up via Google (Customer) → Books 5 parcels
2. Clicks "Become Captain" → Completes KYC
3. Now delivers orders on weekends (Captain)
4. Still books parcels when needed (Customer)
5. Same wallet for earning & spending!
```

### 2. Smart Address Suggestions 🧠
ML-ready analytics learn from user behavior:

```go
// Intelligent address suggestions
address.IsLikelyHomeAddress()       // True if >60% evening/night usage
address.ShouldSuggestAt(time.Now()) // Matches historical patterns
address.GetUsageScore()             // 0-100 relevance score
```

### 3. Production-Grade Money Handling 💰
Currency-aware calculations with zero precision loss:

```go
price := money.FromMajorUnit(50.99, "INR")  // ₹50.99
tax := price.Multiply(0.18)                  // 18% GST
total := price.Add(tax)                      // ₹60.17

// Split bill 3 ways
shares := total.Allocate(3)  // [₹20.06, ₹20.06, ₹20.05]
```

---

## 📁 Project Structure

```
turtle-backend/
├── internal/domain/           # ✅ Business logic (framework-independent)
│   ├── aggregates/           # User, Address with business rules
│   ├── valueobjects/         # Money, Location, Parcel, ContactInfo
│   ├── events/               # Domain events for state changes
│   └── constants/            # Business constants
│
├── internal/application/      # ✅ Use cases (orchestration)
│   └── usecases/
│       └── authentication.go # All auth flows
│
├── internal/infrastructure/   # ✅ External systems
│   ├── database/             # PostgreSQL + PostGIS
│   └── cache/                # Redis (locks, rate limits)
│
├── pkg/                      # ✅ Reusable utilities
│   ├── jwt/                  # Token management
│   ├── otp/                  # OTP generation
│   └── errors/               # Error handling
│
├── migrations/               # ✅ Database schema
└── docs/                     # ✅ Comprehensive documentation
```

---

## 🗺 Roadmap

### ✅ Phase 1: Foundation (COMPLETE - Week 1-2)
- [x] Domain models (User, Address)
- [x] Value objects (Money, Location, Parcel)
- [x] Authentication (Social, OTP, Dual-role)
- [x] Database schema (PostgreSQL + PostGIS)
- [x] Infrastructure (Redis, connection pooling)
- [x] Security (JWT, rate limiting, locking)
- [x] Documentation

**Status**: 100% Complete ✨

---

### ✅ Phase 2: Repository Layer (COMPLETE - Week 3-4)
**Priority**: HIGH | **Status**: Not Started

#### Week 3: PostgreSQL Implementations
- [x] UserRepository (CRUD, geospatial queries)
- [x] AddressRepository (smart suggestions)
- [x] OTPRepository (session management)
- [x] RefreshTokenRepository (token lifecycle)

#### Week 4: Testing & Optimization
- [x] Integration tests (80%+ coverage)
- [x] Transaction management
- [x] Query optimization
- [x] Performance benchmarks

**Deliverables**: All repositories + tests

---

### ⏳ Phase 3: GraphQL API (Week 5-6)
**Priority**: HIGH | **Status**: Week 5 completed

#### Week 5: Core API
- [x] GraphQL schema (.graphqls files)
- [x] Authentication resolvers (login, signup, OTP)
- [x] User & Address resolvers (CRUD operations)
- [x] Middleware (auth, rate limiting, logging)

#### Week 6: Advanced Features
- [ ] DataLoader (N+1 prevention)
- [ ] Subscriptions setup
- [ ] API testing & documentation
- [ ] Playground for development

**Deliverables**: Complete GraphQL API

---

### ⏳ Phase 4: Order System (Week 7-9)
**Priority**: HIGH | **Status**: Not Started

#### Week 7: Order Domain
- [ ] Order aggregate with state machine
- [ ] Price calculation engine
- [ ] OTP verification for pickup/delivery
- [ ] Cancellation & refund logic

#### Week 8: Order Persistence
- [ ] OrderRepository with complex queries
- [ ] All order use cases (create, accept, complete)
- [ ] Transaction management

#### Week 9: Captain Matching
- [ ] Geospatial matching algorithm
- [ ] Auto-assignment logic
- [ ] Order GraphQL API
- [ ] Real-time subscriptions

**Deliverables**: End-to-end order flow

---

### ⏳ Phase 5: Payment & Wallet (Week 10-11)
- [ ] Razorpay integration
- [ ] Wallet top-up & withdrawal
- [ ] Transaction history
- [ ] Refund processing
- [ ] Payment webhooks

---

### ⏳ Phase 6: Real-Time Features (Week 12-13)
- [ ] Live location tracking (Captain → Customer)
- [ ] Chat system (Customer ↔ Captain)
- [ ] Push notifications (FCM)
- [ ] SMS & Email notifications

---

### ⏳ Phase 7: Admin Dashboard (Week 14-15)
- [ ] Captain KYC approval
- [ ] User management
- [ ] Order monitoring
- [ ] Analytics dashboard

---

### ⏳ Phase 8: Production Deployment (Week 16-18)
- [ ] Monitoring (Prometheus + Grafana)
- [ ] Logging (Zap + ELK)
- [ ] Load testing (k6)
- [ ] Security audit
- [ ] CI/CD pipeline
- [ ] Production deployment

---

## 📊 Progress Tracker

```
Overall Progress:     ████░░░░░░░░░░░░░░░░  20% (Phase 1 complete)

Foundation:           ████████████████████ 100%
Repositories:         ░░░░░░░░░░░░░░░░░░░░   0%
GraphQL API:          ░░░░░░░░░░░░░░░░░░░░   0%
Order System:         ░░░░░░░░░░░░░░░░░░░░   0%
Payments:             ░░░░░░░░░░░░░░░░░░░░   0%
Real-time:            ░░░░░░░░░░░░░░░░░░░░   0%
Admin:                ░░░░░░░░░░░░░░░░░░░░   0%
Production:           ░░░░░░░░░░░░░░░░░░░░   0%

Time to MVP:          16 weeks remaining
Time to Production:   18 weeks remaining
```

---

## 🛠 Technology Stack

### Core
- **Language**: Go 1.21+
- **Architecture**: Clean Architecture + DDD
- **API**: GraphQL (gqlgen)

### Database
- **Primary**: PostgreSQL 15+ with PostGIS
- **ORM**: GORM v2
- **Migration**: golang-migrate

### Cache
- **Cache**: Redis 7+
- **Features**: Distributed locks, rate limiting, sessions

### Security
- **Auth**: JWT (HS256)
- **OTP**: Cryptographic random
- **Encryption**: bcrypt for passwords

---

## 🚀 Getting Started

### Prerequisites
- Go 1.21+
- PostgreSQL 15+ with PostGIS
- Redis 7+

### Quick Start

1. **Clone & Install**
```bash
git clone <repository>
cd turtle-backend
go mod download
```

2. **Setup Environment**
```bash
cp .env.example .env
# Edit .env with your credentials
```

3. **Run Migrations**
```bash
psql -U postgres -d turtle_delivery -f migrations/001_initial_schema.up.sql
```

4. **Start Services** (Docker recommended)
```bash
docker-compose up -d  # PostgreSQL + Redis
```

5. **Run Application** *(after Phase 3)*
```bash
go run cmd/server/main.go
```

---

## 📚 Documentation

- [**Authentication Flows**](docs/AUTHENTICATION_FLOWS.md) - Complete auth guide
- [**Dual-Role System**](docs/DUAL_ROLE_SYSTEM.md) - Customer + Captain feature
- [**Production Review**](docs/PRODUCTION_REVIEW.md) - Code quality report
- [**API Spec**](docs/API.md) - GraphQL schema (Phase 3)

---

## 🎯 What Makes This Special

### 1. Production Patterns from Day 1
- Optimistic locking for concurrency
- Distributed locks for critical sections
- Event sourcing ready
- CQRS-friendly design

### 2. Scalability Built-In
- Connection pooling
- Geospatial indexes (PostGIS)
- Efficient caching strategy
- Repository pattern for flexibility

### 3. Security First
- Rate limiting on all endpoints
- Token blacklisting
- Input validation at all layers
- Comprehensive error handling

### 4. Developer Experience
- Clear separation of concerns
- Comprehensive documentation
- Type-safe domain models
- Test-friendly architecture

---

## 📈 Performance Targets

Based on current architecture:

- **API Latency**: <300ms (p95)
- **Database Queries**: <100ms (p95)
- **Cache Operations**: <10ms (p95)
- **Throughput**: 1000+ req/sec
- **Scale**: 1M+ users, 100K+ orders/day

---

## 🤝 Contributing

We follow:
- [Uber Go Style Guide](https://github.com/uber-go/guide)
- [Conventional Commits](https://www.conventionalcommits.org/)
- 80%+ test coverage for new code

---

## 📄 License

MIT License - see [LICENSE](LICENSE)

---

## 🙏 Acknowledgments

- Clean Architecture (Robert C. Martin)
- Domain-Driven Design (Eric Evans)
- Inspired by Porter, Dunzo, Shadowfax

---

<div align="center">

**Phase 1 Complete! Ready for Phase 2: Repository Layer** 🚀

**Next Sprint**: PostgreSQL implementations (Week 3-4)

[⭐ Star this repo] | [🐛 Report Bug] | [💡 Request Feature]

Built with ❤️ using Clean Architecture and DDD

</div>