# 🚀 Turtle Quick Start Guide

Get up and running with Turtle in under 10 minutes!

---

## 📋 Prerequisites

- **Go** 1.21+ ([Download](https://golang.org/dl/))
- **Docker** & Docker Compose ([Download](https://www.docker.com/get-started))
- **Git** ([Download](https://git-scm.com/downloads))
- **Postman** (Optional - for API testing)

---

## ⚡ 5-Minute Setup

### **Step 1: Clone Repository**

```bash
git clone https://github.com/yourusername/turtle.git
cd turtle
```

### **Step 2: Start Dependencies**

```bash
# Start PostgreSQL and Redis
docker-compose up -d

# Verify services are running
docker-compose ps
```

**Expected output:**

```
NAME              STATUS              PORTS
turtle-postgres   Up 10 seconds       0.0.0.0:5432->5432/tcp
turtle-redis      Up 10 seconds       0.0.0.0:6379->6379/tcp
```

### **Step 3: Configure Environment**

```bash
# Copy example config
cp .env.example .env

# Edit if needed (defaults work for local development)
vim .env
```

**Default `.env` for development:**

```bash
SERVER_PORT=8080
SERVER_ENV=development

DB_HOST=localhost
DB_PORT=5432
DB_USER=turtle
DB_PASSWORD=turtle_password
DB_NAME=turtle_dev_db
DB_SSL_MODE=disable

REDIS_HOST=localhost
REDIS_PORT=6379

JWT_SECRET=dev-secret-change-in-production
JWT_ACCESS_TOKEN_DURATION=15m
JWT_REFRESH_TOKEN_DURATION=1440h

OTP_ENABLED=true
OTP_EXPIRY=15m
```

### **Step 4: Run Application**

```bash
# Install dependencies
go mod download

# Run server
go run main.go
```

**Expected output:**

```
🚀 Starting Turtle GraphQL Server...
📊 Running database migrations...
✅ Migrations complete!
🌐 Server running on http://localhost:8080
📚 GraphQL Playground: http://localhost:8080
```

### **Step 5: Test the API**

**Option A: GraphQL Playground (Browser)**

1. Open http://localhost:8080
2. Try this query:

```graphql
query {
  health
}
```

Expected response:

```json
{
  "data": {
    "health": true
  }
}
```

**Option B: cURL**

```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query":"query { health }"}'
```

**Option C: Postman**

1. Import `docs/Turtle_API.postman_collection.json`
2. Set environment variable: `baseUrl` = `http://localhost:8080/graphql`
3. Run "Health Check" request

---

## 🎯 Your First API Call

### **1. Request OTP**

**GraphQL Playground:**

```graphql
mutation {
  requestOTP(input: { phone: "+1234567890", purpose: LOGIN }) {
    success
    message
    expiresAt
  }
}
```

**cURL:**

```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation { requestOTP(input: {phone: \"+1234567890\", purpose: LOGIN}) { success message } }"
  }'
```

**Response:**

```json
{
  "data": {
    "requestOTP": {
      "success": true,
      "message": "OTP sent successfully",
      "expiresAt": "2024-02-01T15:30:00Z"
    }
  }
}
```

---

### **2. Verify OTP & Login**

**In development, use test code: `123456`**

```graphql
mutation {
  verifyOTP(
    input: {
      phone: "+1234567890"
      code: "123456"
      purpose: LOGIN
      deviceType: WEB
      deviceInfo: "Chrome on MacOS"
    }
  ) {
    tokens {
      accessToken
      refreshToken
    }
    user {
      id
      phone
      role
    }
    isNewUser
  }
}
```

**Save the `accessToken` for authenticated requests!**

---

### **3. Get Your Profile**

**Add Authorization header:**

```
Authorization: Bearer <your_access_token>
```

**Query:**

```graphql
query {
  me {
    id
    firstName
    lastName
    phone
    role
    status
  }
}
```

---

### **4. Create an Address**

```graphql
mutation {
  createAddress(
    input: {
      label: HOME
      addressLine1: "123 Main St"
      city: "San Francisco"
      state: "CA"
      postalCode: "94102"
      location: { latitude: 37.7749, longitude: -122.4194 }
      setAsDefault: true
    }
  ) {
    id
    label
    formattedAddress
  }
}
```

---

## 📚 Documentation Links

| Document                                    | Description                 |
| ------------------------------------------- | --------------------------- |
| [README](./README.md)                       | Project overview & features |
| [API Documentation](./API_DOCUMENTATION.md) | Complete API reference      |
| [Architecture](./ARCHITECTURE.md)           | System design & patterns    |
| [Project Structure](./PROJECT_STRUCTURE.md) | File organization           |
| [Testing Guide](./TESTING_GUIDE.md)         | How to test the API         |
| [Deployment Guide](./DEPLOYMENT.md)         | Production deployment       |

---

## 🧪 Testing with Postman

### **Import Collection**

1. Download: `docs/Turtle_API.postman_collection.json`
2. Open Postman
3. Click **Import** → Select file
4. Create environment with:
   - `baseUrl`: `http://localhost:8080/graphql`
   - `accessToken`: (auto-filled after login)

### **Run Authentication Flow**

1. **Request OTP** → Sends code to phone
2. **Verify OTP** → Returns access token (auto-saved)
3. **Get Current User** → Uses saved token
4. **Create Address** → Creates first address

---

## 🔍 Explore the Schema

### **GraphQL Playground Features:**

**1. Documentation Explorer:**

- Click "DOCS" on the right
- Browse all queries, mutations, and types
- See field descriptions and requirements

**2. Schema Tab:**

- View complete GraphQL schema
- See all available operations

**3. Auto-complete:**

- Type `Ctrl+Space` for suggestions
- GraphQL validates your queries

---

## 🛠️ Development Tools

### **Hot Reload (Optional)**

Install Air for auto-reload:

```bash
# Install air
go install github.com/cosmtrek/air@latest

# Run with hot reload
air
```

### **Database GUI**

**pgAdmin** (Recommended):

```bash
docker run -p 5050:80 \
  -e PGADMIN_DEFAULT_EMAIL=admin@turtle.com \
  -e PGADMIN_DEFAULT_PASSWORD=admin \
  dpage/pgadmin4
```

Access: http://localhost:5050

**Connection details:**

- Host: `localhost` (or `host.docker.internal` on Mac/Windows)
- Port: `5432`
- Database: `turtle_db`
- Username: `turtle`
- Password: `turtle_password`

### **Redis GUI**

**RedisInsight**:

```bash
docker run -d -p 8001:8001 redislabs/redisinsight
```

Access: http://localhost:8001

---

## 🎯 Common Commands

```bash
# Start services
docker-compose up -d

# Stop services
docker-compose down

# View logs
docker-compose logs -f

# Run tests
go test ./...

# Run with coverage
go test -cover ./...

# Regenerate GraphQL code
go run github.com/99designs/gqlgen generate

# Format code
go fmt ./...

# Build for production
go build -o turtle main.go

# Run production build
./turtle
```

---

## 📊 Project Status

```
✅ Authentication (OTP + Social Login)
✅ User Management (Customers, Captains, Admins)
✅ Address Management (CRUD + Geospatial)
✅ Service Layer + DataLoader (98% query reduction)
✅ Rate Limiting & Security
✅ GraphQL API with Playground
🔄 Order System (In Progress)
🔄 Real-time Subscriptions (Planned)
🔄 Payment Integration (Planned)
```

---

## 🐛 Troubleshooting

### **Port Already in Use**

```bash
# Find process using port 8080
lsof -i :8080

# Kill process
kill -9 <PID>

# Or change port in .env
SERVER_PORT=8081
```

### **Database Connection Error**

```bash
# Check PostgreSQL is running
docker-compose ps

# View PostgreSQL logs
docker-compose logs postgres

# Restart PostgreSQL
docker-compose restart postgres
```

### **Redis Connection Error**

```bash
# Test Redis connection
redis-cli -h localhost -p 6379 ping

# Should return: PONG
```

### **GraphQL Playground Not Loading**

```bash
# Check server is running
curl http://localhost:8080/health

# Check server logs
# Look for errors in console output
```

---

## 📈 Next Steps

### **For Development:**

1. ✅ Complete authentication flow
2. ✅ Create test data (users, addresses)
3. ✅ Test captain features (KYC, go online)
4. ✅ Explore admin operations
5. 📖 Read [Architecture Guide](./ARCHITECTURE.md)

### **For Production:**

1. 📖 Read [Deployment Guide](./DEPLOYMENT.md)
2. 🔒 Set production environment variables
3. 🐳 Build Docker image
4. ☸️ Deploy to Kubernetes/Cloud
5. 📊 Set up monitoring

---

## 🤝 Need Help?

### **Resources:**

- 📚 [Full Documentation](./docs/)
- 💬 GitHub Issues
- 📧 Email: support@turtle.com
- 🌐 Website: https://turtle.com

### **Common Questions:**

**Q: How do I test SMS in development?**
A: Use the test OTP code: `123456`. Works in development mode.

**Q: How do I create an admin user?**
A: Run the seed script: `go run scripts/seed.go` (creates admin user)

**Q: Where are database migrations?**
A: In `migrations/` folder. Auto-run on server start.

**Q: How do I add custom GraphQL types?**
A: Edit `.graphqls` files, then run: `go run github.com/99designs/gqlgen generate`

---

## 🎉 You're All Set!

Your Turtle backend is now running! 🚀

**Next:** Try the complete authentication flow in Postman or GraphQL Playground!

**Bookmark:** http://localhost:8080 (GraphQL Playground)

---

## 📊 Performance Metrics

After setup, you should see:

| Metric            | Target  | Your Setup         |
| ----------------- | ------- | ------------------ |
| Server Start Time | < 3s    | ⏱️ Check logs      |
| Health Check      | < 10ms  | 🧪 Test now        |
| GraphQL Query     | < 100ms | 🧪 Test `me` query |
| Database Query    | < 20ms  | 📊 Check pgAdmin   |

---

## 🔥 Pro Tips

1. **Use Variables in Playground:**
   - Write reusable queries
   - Avoid hardcoding values

2. **Enable DataLoader Stats:**
   - See query optimization in logs
   - Watch for N+1 problems

3. **Test Rate Limiting:**
   - Try 4 OTP requests quickly
   - Should get rate limit error

4. **Explore Schema:**
   - Click on types in Playground
   - See all available fields

---

**Happy Coding! 🚀**

**Questions?** Check [API Documentation](./API_DOCUMENTATION.md) for complete reference.
