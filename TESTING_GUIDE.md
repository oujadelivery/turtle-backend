# 🧪 Turtle Testing Guide

Comprehensive guide for testing the Turtle GraphQL API.

---

## 📋 Table of Contents

- [Quick Start](#quick-start)
- [GraphQL Playground](#graphql-playground)
- [cURL Examples](#curl-examples)
- [Postman Collection](#postman-collection)
- [Testing Workflows](#testing-workflows)
- [Unit Tests](#unit-tests)
- [Integration Tests](#integration-tests)
- [Performance Testing](#performance-testing)

---

## 🚀 Quick Start

### **1. Start the Server**

```bash
# Start dependencies
docker-compose up -d

# Run server
go run main.go
```

**Server runs on:** http://localhost:8080

---

### **2. Access GraphQL Playground**

Open in browser:
```
http://localhost:8080
```

The playground provides:
- ✅ Interactive query editor
- ✅ Auto-completion
- ✅ Schema documentation
- ✅ Query history
- ✅ Variable editor

---

## 🎮 GraphQL Playground

### **Setting Headers**

Click "HTTP HEADERS" at the bottom:

```json
{
  "Authorization": "Bearer YOUR_ACCESS_TOKEN"
}
```

### **Using Variables**

**Query:**
```graphql
mutation RequestOTP($phone: String!) {
  requestOTP(input: { phone: $phone, purpose: LOGIN }) {
    success
    message
  }
}
```

**Variables:**
```json
{
  "phone": "+1234567890"
}
```

---

## 📝 Testing Workflows

### **Workflow 1: OTP Authentication**

#### **Step 1: Request OTP**

```graphql
mutation {
  requestOTP(input: {
    phone: "+1234567890"
    purpose: LOGIN
  }) {
    success
    message
    expiresAt
  }
}
```

**Expected Response:**
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

**Rate Limit:** 3 requests/hour

---

#### **Step 2: Verify OTP**

```graphql
mutation {
  verifyOTP(input: {
    phone: "+1234567890"
    code: "123456"
    purpose: LOGIN
    deviceType: WEB
    deviceInfo: "Chrome on MacOS"
  }) {
    tokens {
      accessToken
      refreshToken
      expiresAt
    }
    user {
      id
      firstName
      phone
      role
    }
    isNewUser
  }
}
```

**Expected Response:**
```json
{
  "data": {
    "verifyOTP": {
      "tokens": {
        "accessToken": "eyJhbGciOiJSUzI1NiIs...",
        "refreshToken": "eyJhbGciOiJSUzI1NiIs...",
        "expiresAt": "2024-02-01T15:45:00Z"
      },
      "user": {
        "id": "user_abc123",
        "firstName": "",
        "phone": "+1234567890",
        "role": "CUSTOMER"
      },
      "isNewUser": true
    }
  }
}
```

**Save the access token for next steps!**

---

#### **Step 3: Get Current User**

**Set Header:**
```json
{
  "Authorization": "Bearer eyJhbGciOiJSUzI1NiIs..."
}
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
    createdAt
  }
}
```

---

#### **Step 4: Update Profile**

```graphql
mutation {
  updateProfile(input: {
    firstName: "John"
    lastName: "Doe"
  }) {
    id
    firstName
    lastName
    updatedAt
  }
}
```

---

### **Workflow 2: Address Management**

#### **Step 1: Create Address**

```graphql
mutation {
  createAddress(input: {
    label: HOME
    addressLine1: "123 Main St"
    addressLine2: "Apt 4B"
    city: "San Francisco"
    state: "CA"
    postalCode: "94102"
    location: {
      latitude: 37.7749
      longitude: -122.4194
    }
    setAsDefault: true
  }) {
    id
    label
    formattedAddress
    isDefault
  }
}
```

---

#### **Step 2: List My Addresses**

```graphql
query {
  myAddresses {
    id
    label
    formattedAddress
    city
    isDefault
    usageCount
  }
}
```

---

#### **Step 3: Get Suggested Addresses**

```graphql
query {
  suggestedAddresses(limit: 3) {
    address {
      id
      label
      formattedAddress
    }
    confidence
    reason
  }
}
```

---

#### **Step 4: Set Default Address**

```graphql
mutation {
  setDefaultAddress(id: "addr_123") {
    id
    isDefault
  }
}
```

---

### **Workflow 3: Captain Journey**

#### **Step 1: Become Captain**

```graphql
mutation {
  becomeCaptain(input: {
    vehicleType: CAR
    vehicleNumber: "ABC-1234"
    vehicleModel: "Toyota Camry 2020"
  }) {
    id
    role
    captainProfile {
      vehicleType
      vehicleNumber
      kycStatus
    }
  }
}
```

---

#### **Step 2: Submit KYC**

```graphql
mutation {
  submitKYC(input: {
    documents: {
      driverLicense: "https://example.com/license.jpg"
      vehicleRegistration: "https://example.com/registration.jpg"
      insurance: "https://example.com/insurance.jpg"
    }
  }) {
    kycStatus
    kycDocuments
  }
}
```

---

#### **Step 3: Go Online** (After KYC approval)

```graphql
mutation {
  goOnline(input: {
    location: {
      latitude: 37.7749
      longitude: -122.4194
    }
  })
}
```

---

#### **Step 4: Update Location**

```graphql
mutation {
  updateLocation(input: {
    location: {
      latitude: 37.7750
      longitude: -122.4195
    }
  })
}
```

---

### **Workflow 4: Admin Operations**

Login as admin first, then:

#### **Search Users**

```graphql
query {
  searchUsers(input: {
    query: "john"
    role: CAPTAIN
    limit: 20
  }) {
    edges {
      node {
        id
        firstName
        phone
        role
      }
    }
    totalCount
  }
}
```

---

#### **Approve Captain KYC**

```graphql
mutation {
  approveKYC(
    captainID: "captain_123"
    reason: "All documents verified"
  ) {
    id
    captainProfile {
      kycStatus
      kycApprovedAt
    }
  }
}
```

---

#### **Block User**

```graphql
mutation {
  blockUser(
    userID: "user_123"
    reason: "Violating terms"
  ) {
    id
    status
    blockedReason
  }
}
```

---

## 💻 cURL Examples

### **Request OTP**

```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation { requestOTP(input: {phone: \"+1234567890\", purpose: LOGIN}) { success message } }"
  }'
```

---

### **Verify OTP**

```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation { verifyOTP(input: {phone: \"+1234567890\", code: \"123456\", purpose: LOGIN, deviceType: WEB, deviceInfo: \"curl\"}) { tokens { accessToken } user { id } } }"
  }'
```

---

### **Authenticated Request**

```bash
# Save token from previous response
TOKEN="eyJhbGciOiJSUzI1NiIs..."

curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "query": "query { me { id firstName phone } }"
  }'
```

---

### **Create Address**

```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "query": "mutation { createAddress(input: {label: HOME, addressLine1: \"123 Main St\", city: \"SF\", state: \"CA\", postalCode: \"94102\", location: {latitude: 37.7749, longitude: -122.4194}}) { id label } }"
  }'
```

---

## 📮 Postman Collection

### **Import Collection**

1. Download [Turtle.postman_collection.json](./Turtle.postman_collection.json)
2. Open Postman
3. Click "Import" → Select file
4. Collection appears in sidebar

### **Environment Variables**

Create environment with:
```json
{
  "baseUrl": "http://localhost:8080/graphql",
  "accessToken": "",
  "userId": "",
  "addressId": ""
}
```

### **Pre-Request Scripts**

```javascript
// Auto-set token from login response
pm.test("Save token", function() {
    var jsonData = pm.response.json();
    pm.environment.set("accessToken", jsonData.data.verifyOTP.tokens.accessToken);
});
```

---

## 🧪 Unit Tests

### **Run Tests**

```bash
# All tests
go test ./...

# Specific package
go test ./internal/domain/aggregates

# With coverage
go test -cover ./...

# Verbose
go test -v ./...
```

---

### **Example: Testing Domain Logic**

**File:** `internal/domain/aggregates/user_test.go`

```go
func TestUser_UpdateProfile(t *testing.T) {
    // Arrange
    user := NewUser("user_1", "John", "Doe", "CUSTOMER")
    
    // Act
    err := user.UpdateProfile("Jane", "Smith", "pic.jpg")
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, "Jane", user.FirstName())
    assert.Equal(t, "Smith", user.LastName())
}

func TestUser_GoOnline_NotCaptain(t *testing.T) {
    // Arrange
    user := NewUser("user_1", "John", "Doe", "CUSTOMER")
    location := NewLocation(37.7749, -122.4194)
    
    // Act
    err := user.GoOnline(location)
    
    // Assert
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "only captains")
}
```

---

### **Example: Testing Repository**

**File:** `internal/infrastructure/persistence/postgres/user_repository_test.go`

```go
func TestUserRepository_FindByID(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer cleanupTestDB(db)
    
    repo := NewUserRepository(db)
    
    // Create test user
    user := createTestUser()
    repo.Create(context.Background(), user)
    
    // Test
    found, err := repo.FindByID(context.Background(), user.ID())
    
    assert.NoError(t, err)
    assert.Equal(t, user.ID(), found.ID())
}
```

---

## 🔗 Integration Tests

### **Run Integration Tests**

```bash
go test -tags=integration ./tests/integration/...
```

---

### **Example: End-to-End Flow**

**File:** `tests/integration/auth_flow_test.go`

```go
func TestAuthFlow(t *testing.T) {
    // Start test server
    server := startTestServer(t)
    defer server.Close()
    
    client := graphql.NewClient(server.URL)
    
    // Step 1: Request OTP
    phone := "+1234567890"
    var reqOTP struct {
        RequestOTP struct {
            Success bool
        }
    }
    
    err := client.Mutate(context.Background(), &reqOTP, map[string]interface{}{
        "input": map[string]interface{}{
            "phone": phone,
            "purpose": "LOGIN",
        },
    })
    assert.NoError(t, err)
    assert.True(t, reqOTP.RequestOTP.Success)
    
    // Step 2: Verify OTP
    var verifyOTP struct {
        VerifyOTP struct {
            Tokens struct {
                AccessToken string
            }
            User struct {
                ID string
            }
        }
    }
    
    err = client.Mutate(context.Background(), &verifyOTP, map[string]interface{}{
        "input": map[string]interface{}{
            "phone": phone,
            "code": "123456", // Test code
            "purpose": "LOGIN",
            "deviceType": "WEB",
        },
    })
    assert.NoError(t, err)
    assert.NotEmpty(t, verifyOTP.VerifyOTP.Tokens.AccessToken)
}
```

---

## ⚡ Performance Testing

### **Load Testing with hey**

```bash
# Install hey
go install github.com/rakyll/hey@latest

# Test health endpoint
hey -n 1000 -c 10 http://localhost:8080/health

# Test GraphQL query
hey -n 1000 -c 10 \
  -H "Content-Type: application/json" \
  -d '{"query":"query { health }"}' \
  http://localhost:8080/graphql

# With authentication
hey -n 1000 -c 10 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"query":"query { me { id } }"}' \
  http://localhost:8080/graphql
```

**Expected Results:**
```
Summary:
  Total:        2.5432 secs
  Slowest:      0.0892 secs
  Fastest:      0.0012 secs
  Average:      0.0234 secs
  Requests/sec: 393.21

Status code distribution:
  [200] 1000 responses
```

---

### **Benchmark Tests**

**File:** `internal/domain/aggregates/user_bench_test.go`

```go
func BenchmarkUser_UpdateProfile(b *testing.B) {
    user := NewUser("user_1", "John", "Doe", "CUSTOMER")
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        user.UpdateProfile("Jane", "Smith", "pic.jpg")
    }
}
```

**Run:**
```bash
go test -bench=. ./internal/domain/aggregates
```

---

## 🐛 Debugging

### **Enable Debug Logging**

```bash
export LOG_LEVEL=debug
go run main.go
```

---

### **GraphQL Query Logging**

The server logs all GraphQL operations:

```
2024/02/01 14:30:00 GraphQL Request:
  Operation: requestOTP
  Duration: 45ms
  Status: 200
```

---

### **DataLoader Statistics**

Enable DataLoader stats logging:

```
═══════════════════════════════════════════════════════════
📊 DataLoader Statistics
═══════════════════════════════════════════════════════════
👤 User Loader:
   Loads: 100, Cache Hits: 20 (20.0%), Batches: 1
📍 Address Loader:
   Loads: 100, Cache Hits: 0 (0.0%), Batches: 1
───────────────────────────────────────────────────────────
💡 Overall Efficiency: 99.0% reduction in queries
   (200 loads → 2 batches)
═══════════════════════════════════════════════════════════
```

---

## ✅ Testing Checklist

### **Before Release**

- [ ] All unit tests pass
- [ ] Integration tests pass
- [ ] Authentication flow works
- [ ] Rate limiting enforced
- [ ] Error handling correct
- [ ] Performance benchmarks met
- [ ] Load testing completed
- [ ] Security tested (JWT, CORS, etc.)
- [ ] Documentation updated

---

## 📚 Additional Resources

- [GraphQL Playground Docs](https://github.com/graphql/graphql-playground)
- [Postman GraphQL](https://learning.postman.com/docs/sending-requests/supported-api-frameworks/graphql/)
- [Go Testing](https://golang.org/pkg/testing/)
- [hey Load Testing](https://github.com/rakyll/hey)

---

**Happy Testing! 🎉**
