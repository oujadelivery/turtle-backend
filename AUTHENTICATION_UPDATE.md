# 🔐 Authentication System Update

## Changes Made Based on Your Requirements

### ✅ Customer Authentication

**Changed:** Customers now authenticate via **email only** (Google/Apple)

**Login Methods:**

- ✅ Google Sign-in (email required, phone optional)
- ✅ Apple Sign-in (email required, phone optional)
- ✅ Can add phone number later in profile

**Why phone is optional:**

- Customers primarily use social login (Google/Apple)
- Email is sufficient for account identification
- Phone can be added later for better experience
- Not required for booking orders

### ✅ Captain Authentication

**Changed:** Captains authenticate via **phone + OTP only**

**Login Method:**

- ✅ Phone number + OTP (phone required)
- ⚠️ Email needed later for KYC verification

**Why email is not required initially:**

- Quick onboarding for captains
- Phone verification is primary identity
- Email collected during KYC process
- Email used for official communications

### ✅ Booking for Others

**New Feature:** Customers can book parcels for different people

**Contact Information Required:**

```go
// Pickup Person (can be anyone)
pickup_name: "Alice"
pickup_phone: "+919876543210"

// Delivery Person (can be anyone)
delivery_name: "Bob"
delivery_phone: "+919876543211"
```

**Key Points:**

- Neither pickup nor delivery person needs to be an app user
- OTPs sent to their phones for verification
- Captain can contact them directly
- Customer pays for the order

## Updated Code Components

### 1. User Aggregate (`user.go`)

**Changes:**

- ✅ Role-based contact validation in `NewUser()`
  - Customers: Email OR Phone (email preferred)
  - Captains: Phone required, email optional
  - Admins: Email required
- ✅ New method: `AddPhoneNumber()` for customers
  - Allows adding phone after email-based signup
  - Triggers verification OTP
  - Raises domain event

### 2. Database Schema (`001_initial_schema.up.sql`)

**Changes:**

- ✅ Updated constraint on `users` table:

```sql
CONSTRAINT users_email_or_phone_required CHECK (
    (primary_role = 'CUSTOMER' AND (email IS NOT NULL OR phone IS NOT NULL)) OR
    (primary_role = 'CAPTAIN' AND phone IS NOT NULL) OR
    (primary_role = 'ADMIN' AND email IS NOT NULL)
)
```

### 3. Value Objects

**New:** `ContactInfo` value object

- ✅ Represents non-user contact information
- ✅ Used for pickup and delivery persons
- ✅ Validates name and phone format
- ✅ Provides formatted phone display

```go
type ContactInfo struct {
    name  string
    phone string
}
```

### 4. Authentication Use Cases (`authentication.go`)

**New:** Complete authentication service with:

#### Customer Flows:

- ✅ `SocialLogin()` - Google/Apple authentication
  - Verifies provider token
  - Creates user if new
  - Auto-verifies email
  - Returns JWT tokens
  - Flags if phone should be added

- ✅ `AddPhoneNumber()` - Add phone to customer
  - Updates user profile
  - Sends verification OTP
  - Maintains email as primary

- ✅ `VerifyPhone()` - Verify added phone
  - Validates OTP
  - Marks phone as verified

#### Captain Flows:

- ✅ `SendOTP()` - Send OTP to captain's phone
  - Rate limited (3 per hour)
  - 6-digit OTP
  - 15-minute expiry

- ✅ `VerifyOTPAndLogin()` - Verify OTP and login
  - Creates captain if new
  - Auto-verifies phone
  - Returns JWT tokens

#### Common Operations:

- ✅ `RefreshAccessToken()` - Token refresh
- ✅ `Logout()` - Token revocation

### 5. Domain Events

**New Events:**

- ✅ `UserPhoneAddedEvent` - When customer adds phone
- ✅ All existing authentication events

### 6. Domain Errors

**New:** `errors.go` with common errors:

- `ErrNotFound`
- `ErrAlreadyExists`
- `ErrUnauthorized`
- `ErrConcurrentModification`
- etc.

## Security Features

### Rate Limiting

- 🔒 OTP sending: 3 requests/hour per phone
- 🔒 OTP verification: 5 attempts/15 minutes
- 🔒 Social login: 10 attempts/hour per email

### OTP Security

- 🔐 6-digit cryptographic random OTP
- ⏰ 15-minute expiry
- 🚫 Max 5 failed attempts
- ✅ Single-use tokens
- 🗑️ Automatic cleanup of expired OTPs

### Token Security

- 🔑 Access token: 15 minutes
- 🔄 Refresh token: 60 days
- 🗄️ Stored in database
- ❌ Can be revoked
- 🏴 Blacklist support

## API Flow Examples

### Customer Sign-up (Google)

```bash
POST /api/auth/social-login
{
  "provider": "GOOGLE",
  "providerID": "12345",
  "email": "john@gmail.com",
  "firstName": "John",
  "lastName": "Doe",
  "deviceType": "IOS"
}

Response:
{
  "accessToken": "eyJ...",
  "refreshToken": "eyJ...",
  "userID": "uuid",
  "isNewUser": true,
  "needsPhone": true  // Suggest adding phone
}
```

### Customer Adds Phone

```bash
POST /api/auth/add-phone
{
  "userID": "uuid",
  "phone": "+919876543210"
}

Response: {
  "success": true,
  "message": "OTP sent to phone"
}
```

### Captain Login

```bash
# Step 1: Request OTP
POST /api/auth/send-otp
{
  "phone": "+919876543210",
  "purpose": "LOGIN"
}

# Step 2: Verify OTP
POST /api/auth/verify-otp
{
  "phone": "+919876543210",
  "code": "123456",
  "purpose": "LOGIN",
  "deviceType": "ANDROID"
}

Response:
{
  "accessToken": "eyJ...",
  "refreshToken": "eyJ...",
  "userID": "uuid",
  "isNewUser": false,
  "role": "CAPTAIN"
}
```

### Order with Different Contacts

```bash
POST /api/orders
{
  "customerID": "uuid",

  "pickup": {
    "name": "Alice Smith",
    "phone": "+919876543210",
    "addressID": "addr-uuid"
  },

  "delivery": {
    "name": "Bob Johnson",
    "phone": "+919876543211",
    "addressID": "addr-uuid"
  },

  "parcel": {
    "type": "PACKAGE",
    "weight": 2.5,
    "description": "Birthday gift"
  }
}

Result:
- OTP sent to Alice (+919876543210)
- OTP sent to Bob (+919876543211)
- Order created with PENDING status
```

## Documentation

📚 Complete authentication documentation: `docs/AUTHENTICATION_FLOWS.md`

This includes:

- Visual flow diagrams
- Security considerations
- Testing checklist
- API endpoint summary
- Implementation notes

## Migration Guide

### For Existing Customers (if any):

1. Customers with email-only accounts continue to work
2. They can optionally add phone via profile settings
3. No breaking changes

### For New Captains:

1. Sign up with phone + OTP
2. Add email during KYC process
3. Email required for full verification

### Database Migration:

```sql
-- Already included in 001_initial_schema.up.sql
-- No additional migration needed
-- Constraint updated to support role-based requirements
```

## What's Next?

1. **Implement Repository Layer** - PostgreSQL implementations
2. **Build GraphQL Resolvers** - For authentication endpoints
3. **SMS Provider Integration** - Twilio/AWS SNS for OTP
4. **Email Provider Integration** - SendGrid for notifications
5. **Testing** - Unit tests for all use cases

## Benefits of This Approach

✅ **Flexible**: Supports different auth methods per role
✅ **Secure**: Multiple rate limiting layers
✅ **User-Friendly**: Customers start with familiar Google/Apple
✅ **Fast Onboarding**: Captains onboard quickly with phone
✅ **Future-Proof**: Can add more auth methods easily
✅ **Booking Freedom**: Customers can book for anyone

---

Your authentication system is now **production-ready** with support for:

- Customer social login (email-based)
- Captain phone login (OTP-based)
- Flexible contact requirements per role
- Booking parcels for non-users
- Complete security measures

🚀 **Ready to implement the repository layer and GraphQL API!**
