# 📚 Turtle API Documentation

Complete GraphQL API reference for the Turtle delivery and ride-sharing platform.

---

## 📋 Table of Contents

- [Authentication](#authentication)
- [User Management](#user-management)
- [Address Management](#address-management)
- [Session Management](#session-management)
- [Error Handling](#error-handling)
- [Rate Limits](#rate-limits)

---

## 🔐 Authentication

### **Base URL**
```
POST http://localhost:8080/graphql
```

### **Headers**
```
Content-Type: application/json
Authorization: Bearer <access_token>  # For authenticated requests
```

---

## 🎫 Auth Mutations

### **1. Request OTP**

Send OTP code to phone number for verification.

**Mutation:**
```graphql
mutation RequestOTP {
  requestOTP(input: {
    phone: "+1234567890"
    purpose: LOGIN  # LOGIN | VERIFICATION | FORGOT_PASSWORD
  }) {
    success
    message
    target
    expiresAt
  }
}
```

**Response:**
```json
{
  "data": {
    "requestOTP": {
      "success": true,
      "message": "OTP sent successfully",
      "target": "+1234567890",
      "expiresAt": "2024-02-01T15:30:00Z"
    }
  }
}
```

**Rate Limit:** 3 requests per hour

---

### **2. Verify OTP & Login**

Verify OTP code and login/create account.

**Mutation:**
```graphql
mutation VerifyOTP {
  verifyOTP(input: {
    phone: "+1234567890"
    code: "123456"
    purpose: LOGIN
    deviceType: WEB  # WEB | IOS | ANDROID
    deviceInfo: "Chrome on MacOS"
  }) {
    tokens {
      accessToken
      refreshToken
      expiresAt
      tokenType
    }
    user {
      id
      firstName
      lastName
      email
      phone
      role
      status
      profilePic
      createdAt
    }
    isNewUser
  }
}
```

**Response:**
```json
{
  "data": {
    "verifyOTP": {
      "tokens": {
        "accessToken": "eyJhbGciOiJSUzI1NiIs...",
        "refreshToken": "eyJhbGciOiJSUzI1NiIs...",
        "expiresAt": "2024-02-01T15:45:00Z",
        "tokenType": "Bearer"
      },
      "user": {
        "id": "user_123",
        "firstName": "John",
        "lastName": "Doe",
        "email": null,
        "phone": "+1234567890",
        "role": "CUSTOMER",
        "status": "ACTIVE",
        "profilePic": null,
        "createdAt": "2024-02-01T14:30:00Z"
      },
      "isNewUser": true
    }
  }
}
```

**Rate Limit:** 5 attempts per 15 minutes

---

### **3. Social Login**

Login with Google or Apple.

**Mutation:**
```graphql
mutation SocialLogin {
  socialLogin(input: {
    provider: GOOGLE  # GOOGLE | APPLE
    providerID: "google_user_123"
    email: "john@example.com"
    firstName: "John"
    lastName: "Doe"
    profilePic: "https://example.com/photo.jpg"
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
      email
      role
    }
    isNewUser
    needsPhone
  }
}
```

**Rate Limit:** 10 requests per hour

---

### **4. Refresh Token**

Get new access token using refresh token.

**Mutation:**
```graphql
mutation RefreshToken {
  refreshToken(input: {
    refreshToken: "eyJhbGciOiJSUzI1NiIs..."
  }) {
    accessToken
    refreshToken
    expiresAt
    tokenType
  }
}
```

---

### **5. Link Phone to Account**

Link phone number to social login account.

**Mutation:**
```graphql
mutation LinkPhone {
  linkPhone(input: {
    phone: "+1234567890"
    code: "123456"
  }) {
    id
    phone
    phoneVerified
  }
}
```

**Requires:** Authentication

---

### **6. Logout**

Revoke current device session.

**Mutation:**
```graphql
mutation Logout {
  logout
}
```

**Requires:** Authentication  
**Returns:** Boolean (true if successful)

---

### **7. Logout All Devices**

Revoke all sessions across all devices.

**Mutation:**
```graphql
mutation LogoutAll {
  logoutAll
}
```

**Requires:** Authentication

---

## 👤 User Queries

### **1. Get Current User (Me)**

Get authenticated user's profile.

**Query:**
```graphql
query Me {
  me {
    id
    firstName
    lastName
    email
    phone
    phoneVerified
    role
    status
    profilePic
    createdAt
    updatedAt
    
    # Captain-specific fields
    captainProfile {
      vehicleType
      vehicleNumber
      vehicleModel
      kycStatus
      kycDocuments
      isAvailable
      currentLocation {
        latitude
        longitude
      }
      rating
      totalRides
    }
  }
}
```

**Requires:** Authentication

---

### **2. Get User by ID**

Get any user's public profile.

**Query:**
```graphql
query GetUser {
  user(id: "user_123") {
    id
    firstName
    lastName
    profilePic
    role
    createdAt
  }
}
```

**Requires:** Authentication

---

### **3. Search Users (Admin)**

Search users with filters and pagination.

**Query:**
```graphql
query SearchUsers {
  searchUsers(input: {
    query: "john"
    role: CUSTOMER  # Optional: CUSTOMER | CAPTAIN | ADMIN
    limit: 20
    offset: 0
  }) {
    edges {
      cursor
      node {
        id
        firstName
        lastName
        email
        phone
        role
        status
      }
    }
    pageInfo {
      hasNextPage
      hasPreviousPage
      total
    }
    totalCount
  }
}
```

**Requires:** Admin role

---

### **4. Find Nearby Captains**

Find available captains near a location.

**Query:**
```graphql
query NearbyCaptains {
  nearbyCaptains(
    latitude: 37.7749
    longitude: -122.4194
    radiusKm: 5.0
  ) {
    id
    firstName
    profilePic
    captainProfile {
      vehicleType
      vehicleNumber
      rating
      currentLocation {
        latitude
        longitude
      }
      isAvailable
    }
  }
}
```

---

### **5. User Statistics (Admin)**

Get aggregated user stats.

**Query:**
```graphql
query UserStats {
  userStats {
    totalUsers
    totalCustomers
    totalCaptains
    totalAdmins
    activeUsers
    verifiedCaptains
    onlineCaptains
  }
}
```

**Requires:** Admin role

---

## 👤 User Mutations

### **1. Update Profile**

Update user profile information.

**Mutation:**
```graphql
mutation UpdateProfile {
  updateProfile(input: {
    firstName: "John"
    lastName: "Doe"
    profilePic: "https://example.com/photo.jpg"
  }) {
    id
    firstName
    lastName
    profilePic
    updatedAt
  }
}
```

**Requires:** Authentication

---

### **2. Become Captain**

Upgrade customer account to captain.

**Mutation:**
```graphql
mutation BecomeCaptain {
  becomeCaptain(input: {
    vehicleType: CAR  # BIKE | SCOOTER | CAR | VAN
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

**Requires:** Authentication

---

### **3. Submit KYC (Captain)**

Submit KYC documents for verification.

**Mutation:**
```graphql
mutation SubmitKYC {
  submitKYC(input: {
    documents: {
      driverLicense: "https://example.com/license.jpg"
      vehicleRegistration: "https://example.com/reg.jpg"
      insurance: "https://example.com/insurance.jpg"
    }
  }) {
    kycStatus
    kycDocuments
  }
}
```

**Requires:** Captain role

---

### **4. Go Online (Captain)**

Set captain as available for rides.

**Mutation:**
```graphql
mutation GoOnline {
  goOnline(input: {
    location: {
      latitude: 37.7749
      longitude: -122.4194
    }
  })
}
```

**Requires:** Captain role  
**Returns:** Boolean

---

### **5. Go Offline (Captain)**

Set captain as unavailable.

**Mutation:**
```graphql
mutation GoOffline {
  goOffline
}
```

**Requires:** Captain role

---

### **6. Update Location (Captain)**

Update current location while online.

**Mutation:**
```graphql
mutation UpdateLocation {
  updateLocation(input: {
    location: {
      latitude: 37.7750
      longitude: -122.4195
    }
  })
}
```

**Requires:** Captain role  
**Returns:** Boolean

---

### **7. Approve KYC (Admin)**

Approve captain's KYC verification.

**Mutation:**
```graphql
mutation ApproveKYC {
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

**Requires:** Admin role

---

### **8. Reject KYC (Admin)**

Reject captain's KYC submission.

**Mutation:**
```graphql
mutation RejectKYC {
  rejectKYC(
    captainID: "captain_123"
    reason: "Invalid driver license"
  ) {
    id
    captainProfile {
      kycStatus
      kycRejectionReason
    }
  }
}
```

**Requires:** Admin role

---

### **9. Block User (Admin)**

Block a user account.

**Mutation:**
```graphql
mutation BlockUser {
  blockUser(
    userID: "user_123"
    reason: "Violating terms of service"
  ) {
    id
    status
    blockedReason
    blockedAt
  }
}
```

**Requires:** Admin role

---

### **10. Unblock User (Admin)**

Unblock a user account.

**Mutation:**
```graphql
mutation UnblockUser {
  unblockUser(userID: "user_123") {
    id
    status
    updatedAt
  }
}
```

**Requires:** Admin role

---

## 📍 Address Queries

### **1. Get Address by ID**

Fetch specific address.

**Query:**
```graphql
query GetAddress {
  address(id: "addr_123") {
    id
    label
    addressLine1
    addressLine2
    landmark
    city
    state
    country
    postalCode
    location {
      latitude
      longitude
    }
    isDefault
    isVerified
    usageCount
    createdAt
  }
}
```

**Requires:** Authentication

---

### **2. My Addresses**

Get all addresses for current user.

**Query:**
```graphql
query MyAddresses {
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

**Requires:** Authentication

---

### **3. Default Address**

Get user's default address.

**Query:**
```graphql
query DefaultAddress {
  myDefaultAddress {
    id
    label
    formattedAddress
    location {
      latitude
      longitude
    }
  }
}
```

**Requires:** Authentication

---

### **4. Suggested Addresses**

Get AI-powered address suggestions.

**Query:**
```graphql
query SuggestedAddresses {
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

**Requires:** Authentication

---

### **5. Nearest Addresses**

Find nearest saved addresses to a location.

**Query:**
```graphql
query NearestAddresses {
  nearestAddresses(
    latitude: 37.7749
    longitude: -122.4194
    limit: 5
  ) {
    id
    label
    formattedAddress
    location {
      latitude
      longitude
    }
    # Distance calculated from query point
  }
}
```

**Requires:** Authentication

---

### **6. Search Addresses**

Full-text search across user's addresses.

**Query:**
```graphql
query SearchAddresses {
  searchAddresses(input: {
    query: "office"
    limit: 20
    offset: 0
  }) {
    edges {
      cursor
      node {
        id
        label
        formattedAddress
        city
      }
    }
    pageInfo {
      hasNextPage
      total
    }
    totalCount
  }
}
```

**Requires:** Authentication

---

### **7. Address Statistics**

Get address usage analytics.

**Query:**
```graphql
query AddressStats {
  addressStats {
    totalAddresses
    verifiedAddresses
    defaultAddress {
      id
      label
    }
    mostUsedAddress {
      id
      label
      usageCount
    }
    recentAddresses {
      id
      label
      lastUsedAt
    }
  }
}
```

**Requires:** Authentication

---

## 📍 Address Mutations

### **1. Create Address**

Add new address.

**Mutation:**
```graphql
mutation CreateAddress {
  createAddress(input: {
    label: HOME  # HOME | WORK | OTHER
    addressLine1: "123 Main St"
    addressLine2: "Apt 4B"
    landmark: "Near Central Park"
    city: "San Francisco"
    state: "CA"
    country: "USA"
    postalCode: "94102"
    location: {
      latitude: 37.7749
      longitude: -122.4194
    }
    contactName: "John Doe"
    contactPhone: "+1234567890"
    setAsDefault: true
  }) {
    id
    label
    formattedAddress
    isDefault
  }
}
```

**Requires:** Authentication  
**Rate Limit:** 20 per hour

---

### **2. Update Address**

Modify existing address.

**Mutation:**
```graphql
mutation UpdateAddress {
  updateAddress(input: {
    id: "addr_123"
    label: WORK
    addressLine1: "456 Market St"
    addressLine2: "Floor 10"
    city: "San Francisco"
  }) {
    id
    label
    formattedAddress
    updatedAt
  }
}
```

**Requires:** Authentication

---

### **3. Delete Address**

Remove address (soft delete).

**Mutation:**
```graphql
mutation DeleteAddress {
  deleteAddress(id: "addr_123")
}
```

**Requires:** Authentication  
**Returns:** Boolean

---

### **4. Set Default Address**

Mark address as default.

**Mutation:**
```graphql
mutation SetDefaultAddress {
  setDefaultAddress(id: "addr_123") {
    id
    label
    isDefault
  }
}
```

**Requires:** Authentication

---

### **5. Verify Address**

Mark address location as verified.

**Mutation:**
```graphql
mutation VerifyAddress {
  verifyAddress(id: "addr_123") {
    id
    isVerified
    verifiedAt
  }
}
```

**Requires:** Authentication

---

## 🔐 Session Management

### **Get Active Sessions**

List all active login sessions.

**Query:**
```graphql
query MySessions {
  mySessions {
    id
    device
    deviceInfo
    lastUsedAt
    createdAt
    expiresAt
    isActive
  }
}
```

**Requires:** Authentication

**Response:**
```json
{
  "data": {
    "mySessions": [
      {
        "id": "session_123",
        "device": "WEB",
        "deviceInfo": "Chrome on MacOS",
        "lastUsedAt": "2024-02-01T14:30:00Z",
        "createdAt": "2024-02-01T10:00:00Z",
        "expiresAt": "2024-03-01T10:00:00Z",
        "isActive": true
      }
    ]
  }
}
```

---

## 🔥 Subscriptions (Real-time)

### **1. User Profile Updates**

Subscribe to user profile changes.

**Subscription:**
```graphql
subscription UserUpdated {
  userUpdated(userID: "user_123") {
    id
    firstName
    lastName
    profilePic
    updatedAt
  }
}
```

**Requires:** Authentication

---

### **2. Captain Location Updates**

Track captain's real-time location.

**Subscription:**
```graphql
subscription CaptainLocation {
  captainLocationUpdated(captainID: "captain_123") {
    latitude
    longitude
    address
  }
}
```

---

### **3. Captain Availability**

Subscribe to captain online/offline status.

**Subscription:**
```graphql
subscription CaptainAvailability {
  captainAvailabilityChanged(captainID: "captain_123") {
    captainID
    isAvailable
    location {
      latitude
      longitude
    }
    timestamp
  }
}
```

---

## ⚠️ Error Handling

### **Error Response Format**

```json
{
  "errors": [
    {
      "message": "Authentication required",
      "extensions": {
        "code": "UNAUTHORIZED",
        "statusCode": 401
      }
    }
  ],
  "data": null
}
```

### **Common Error Codes**

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `UNAUTHORIZED` | 401 | Not authenticated |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `VALIDATION_ERROR` | 400 | Invalid input |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Server error |
| `INVALID_TOKEN` | 401 | Invalid/expired token |
| `DUPLICATE_RESOURCE` | 409 | Resource already exists |

---

## ⚡ Rate Limits

| Operation | Limit | Window | Identifier |
|-----------|-------|--------|-----------|
| Request OTP | 3 | 1 hour | Phone number |
| Verify OTP | 5 | 15 min | Phone number |
| Social Login | 10 | 1 hour | Email |
| Create Address | 20 | 1 hour | User ID |
| Global API | 60 | 1 minute | IP/User ID |

### **Rate Limit Headers**

```
X-RateLimit-Limit: 3
X-RateLimit-Remaining: 2
X-RateLimit-Reset: 1706789234
Retry-After: 3600
```

---

## 🔗 Useful Links

- [GraphQL Playground](http://localhost:8080)
- [Health Check](http://localhost:8080/health)
- [Architecture Guide](./ARCHITECTURE.md)
- [Testing Guide](./TESTING_GUIDE.md)

---

## 📝 Notes

- All timestamps are in ISO 8601 format (UTC)
- All IDs are UUIDs
- Phone numbers must be in E.164 format (+1234567890)
- Coordinates use WGS84 (latitude: -90 to 90, longitude: -180 to 180)
- Pagination uses cursor-based approach for efficiency
- File uploads use base64 encoding (max 5MB)

---

**For detailed schema documentation, use the GraphQL Playground's Documentation Explorer.**
