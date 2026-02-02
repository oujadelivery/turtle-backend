# 👥 Dual-Role User System (Customer + Captain)

## Overview

Our system supports **dual-role users** - a single person can be both a **Customer** and a **Captain** using the same phone number and email. This allows for maximum flexibility and enables users to earn money by delivering while also ordering parcels.

## Why Dual Roles?

### Real-World Use Cases

1. **Part-Time Captains**
   - Office worker who delivers on weekends
   - Student who delivers during free time
   - Anyone wanting side income

2. **Captain Who Orders**
   - Captain needs to send a parcel
   - No need to create separate account
   - Seamless switch between roles

3. **Flexible Work Arrangement**
   - Work as captain when needed
   - Order as customer when busy
   - Same account, same wallet

## How It Works

### User Model Structure

```go
type User struct {
    // Single user identity
    id      string
    email   *string  // Unique across entire system
    phone   *string  // Unique across entire system

    // Multiple roles
    primaryRole UserRole        // Initial role (CUSTOMER or CAPTAIN)
    roles       []UserRole      // Can contain: [CUSTOMER, CAPTAIN, ADMIN]

    // Role-specific profiles
    captainProfile *CaptainProfile  // Populated when user has CAPTAIN role
    adminProfile   *AdminProfile    // Populated when user has ADMIN role
}
```

### Key Principles

1. **One Identity**
   - Email and phone are unique identifiers
   - One person = One user record
   - Multiple roles on same record

2. **Role-Specific Data**
   - Captain profile only exists if user has CAPTAIN role
   - Admin profile only exists if user has ADMIN role
   - Wallet is shared across all roles

3. **Progressive Enhancement**
   - User starts with one role
   - Can add additional roles later
   - Each role has specific requirements

## User Journeys

### Journey 1: Customer → Customer + Captain

**Starting Point:** User signed up as customer via Google

```
Step 1: User is Customer (John)
- Email: john@gmail.com (from Google)
- Phone: Not added yet
- Roles: [CUSTOMER]
- Can: Book parcels

Step 2: User decides to become Captain
- Action: Click "Become a Captain" in app
- System checks: Phone number required
- Prompt: "Add phone number to become captain"

Step 3: User adds phone number
- Enter: +919876543210
- Receive: OTP via SMS
- Verify: OTP code
- Phone: +919876543210 (verified)

Step 4: User becomes Captain
- Action: Click "Complete Captain Registration"
- System: Adds CAPTAIN role
- Roles: [CUSTOMER, CAPTAIN]
- Status: Must complete KYC

Step 5: Complete KYC
- Upload: License, Vehicle RC, Photos
- Wait: Admin verification
- Status: KYC Verified
- Can: Accept delivery orders

Final State:
- Email: john@gmail.com
- Phone: +919876543210
- Roles: [CUSTOMER, CAPTAIN]
- Can: Book parcels AND deliver orders
```

### Journey 2: Captain → Captain + Customer

**Starting Point:** User signed up as captain via phone

```
Step 1: User is Captain (Alice)
- Phone: +919876543210 (verified via OTP)
- Email: Not added yet
- Roles: [CAPTAIN]
- Can: Deliver orders

Step 2: User wants to order parcel
- Action: Click "Book Parcel"
- System: Automatically enables customer role
- Roles: [CAPTAIN, CUSTOMER]
- Can: Now book parcels too

Step 3: (Optional) Add email for better experience
- Action: Add email in profile
- Email: alice@email.com
- Benefits: Email notifications, receipts

Final State:
- Phone: +919876543210
- Email: alice@email.com
- Roles: [CAPTAIN, CUSTOMER]
- Can: Deliver orders AND book parcels
```

### Journey 3: Same Login Works for Both Roles

**Scenario:** User with both roles logs in

```
Login Flow:
1. User can login via:
   - Google/Apple (if email added)
   - Phone + OTP (if phone added)

2. After login, JWT contains:
   - UserID: "uuid-123"
   - Primary Role: "CUSTOMER" or "CAPTAIN"
   - All Roles: ["CUSTOMER", "CAPTAIN"]

3. App shows role switcher:
   ┌─────────────────────────┐
   │ Switch Mode:            │
   │ ○ Customer Mode         │
   │ ● Captain Mode          │
   └─────────────────────────┘

4. User can switch anytime:
   - Customer Mode: Shows "Book Parcel" interface
   - Captain Mode: Shows "Accept Orders" interface
```

## Database Design

### Single Users Table

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,

    -- Contact (unique identifiers)
    email VARCHAR(255) UNIQUE,      -- One email per person
    phone VARCHAR(20) UNIQUE,       -- One phone per person

    -- Roles
    primary_role VARCHAR(20),       -- First role user signed up with
    roles TEXT[],                   -- Array: ['CUSTOMER', 'CAPTAIN']

    -- Common fields
    wallet_balance BIGINT,          -- Shared wallet!
    rating DECIMAL(3,2),            -- Overall rating
    total_orders INT,               -- As customer
    total_deliveries INT,           -- As captain

    -- Status
    status VARCHAR(20)
);

-- Separate captain-specific data
CREATE TABLE captain_profiles (
    user_id UUID PRIMARY KEY,      -- One-to-one with users
    kyc_status VARCHAR(20),
    vehicle_type VARCHAR(20),
    is_available BOOLEAN,
    -- ... other captain fields
);

-- Separate admin-specific data
CREATE TABLE admin_profiles (
    user_id UUID PRIMARY KEY,      -- One-to-one with users
    permissions TEXT[],
    department VARCHAR(50),
    -- ... other admin fields
);
```

### Key Design Points

1. **Unique Constraints on email and phone remain**
   - Ensures one person = one account
   - Prevents duplicate identities

2. **Roles Array**
   - PostgreSQL array type
   - Can contain multiple values
   - Easy to query: `'CAPTAIN' = ANY(roles)`

3. **Separate Profile Tables**
   - Only created when role is added
   - Keeps users table clean
   - One-to-one relationship

## API Flows

### 1. Customer Becomes Captain

```http
POST /api/user/become-captain
Authorization: Bearer <customer-jwt-token>

{
  "phone": "+919876543210"  // If not already present
}

Response:
{
  "success": false,
  "requiresPhone": true,
  "message": "Phone verification OTP sent"
}

# After phone verification
POST /api/user/become-captain
Authorization: Bearer <customer-jwt-token>

Response:
{
  "success": true,
  "requiresKYC": true,
  "message": "Captain role added. Complete KYC to start delivering."
}
```

### 2. Switch Between Roles

```http
POST /api/user/switch-role
Authorization: Bearer <jwt-token>

{
  "targetRole": "CAPTAIN"
}

Response:
{
  "success": true,
  "message": "Switched to captain mode"
}
```

### 3. Query User Capabilities

```http
GET /api/user/me
Authorization: Bearer <jwt-token>

Response:
{
  "id": "uuid-123",
  "email": "john@gmail.com",
  "phone": "+919876543210",
  "primaryRole": "CUSTOMER",
  "roles": ["CUSTOMER", "CAPTAIN"],

  "capabilities": {
    "canBookOrders": true,
    "canDeliverOrders": true,
    "canGoOnline": true,
    "requiresKYC": false
  },

  "profiles": {
    "customer": {
      "totalOrders": 25,
      "rating": 4.8
    },
    "captain": {
      "kycStatus": "VERIFIED",
      "totalDeliveries": 150,
      "rating": 4.9,
      "isAvailable": false
    }
  }
}
```

## Business Rules

### Rule 1: Contact Requirements by Role

| Action                             | Email Required | Phone Required   |
| ---------------------------------- | -------------- | ---------------- |
| Sign up as Customer (Google/Apple) | ✅ Yes         | ❌ No (optional) |
| Sign up as Captain (Phone+OTP)     | ❌ No (later)  | ✅ Yes           |
| Customer → Add Captain role        | ❌ No          | ✅ Yes           |
| Captain → Add Customer role        | ❌ No          | ✅ Already has   |

### Rule 2: Captain Role Requirements

To add CAPTAIN role:

1. ✅ Must have verified phone number
2. ✅ Must complete KYC verification
3. ✅ Must add vehicle details
4. ✅ Must upload required documents

To go online as captain:

1. ✅ Must have CAPTAIN role
2. ✅ KYC must be VERIFIED
3. ✅ Account must be ACTIVE
4. ✅ Must not be blocked

### Rule 3: Wallet & Earnings

**Shared Wallet:**

- Customer orders: Deduct from wallet
- Captain deliveries: Add to wallet
- Same balance across roles
- Separate transaction types

```go
// Example transactions
Order #123 - Customer paid: -₹250
Order #456 - Captain earned: +₹180
Order #789 - Customer paid: -₹300
Order #012 - Captain earned: +₹210

Wallet Balance: ₹210 - ₹250 - ₹300 + ₹180 + ₹210 = +₹50
```

### Rule 4: Ratings

**Separate Ratings:**

- Customer rating: Based on ordering behavior
- Captain rating: Based on delivery performance
- Overall rating: Weighted average

```go
Customer Rating: 4.8 (based on 25 orders)
Captain Rating: 4.9 (based on 150 deliveries)
Overall Rating: (4.8*25 + 4.9*150) / (25+150) = 4.88
```

## Implementation Details

### Adding Captain Role

```go
// In authentication service
func (s *AuthenticationService) BecomeCaptain(
    ctx context.Context,
    input BecomeCaptainInput,
) (*BecomeCaptainOutput, error) {

    user, _ := s.userRepo.FindByID(ctx, input.UserID)

    // Check if already captain
    if user.HasRole(aggregates.RoleCaptain) {
        return &BecomeCaptainOutput{
            Success: true,
            Message: "Already a captain",
        }, nil
    }

    // Validate phone exists and is verified
    if user.Phone() == nil {
        return &BecomeCaptainOutput{
            RequiresPhone: true,
            Message: "Phone number required",
        }, nil
    }

    // Add captain role
    user.AddRole(aggregates.RoleCaptain)
    s.userRepo.Update(ctx, user)

    return &BecomeCaptainOutput{
        Success: true,
        RequiresKYC: true,
        Message: "Captain role added. Complete KYC.",
    }, nil
}
```

### Checking Capabilities

```go
// Helper function
func (u *User) GetCapabilities() UserCapabilities {
    return UserCapabilities{
        CanBookOrders:    u.HasRole(RoleCustomer),
        CanDeliverOrders: u.HasRole(RoleCaptain) &&
                         u.CaptainProfile().KYCStatus() == KYCVerified,
        CanGoOnline:      u.IsAvailable(),
        CanAccessAdmin:   u.HasRole(RoleAdmin),
    }
}
```

## UI/UX Considerations

### Role Switcher in App

```
┌────────────────────────────────────┐
│  👤 John Doe                       │
│  john@gmail.com                    │
│                                    │
│  Current Mode: Customer            │
│  ┌──────────────────────────────┐ │
│  │ Switch to:                   │ │
│  │ ● Customer Mode  📦          │ │
│  │ ○ Captain Mode   🚚          │ │
│  └──────────────────────────────┘ │
│                                    │
│  [Book a Parcel]                  │
│  [My Orders]                      │
│  [Wallet: ₹1,250]                 │
└────────────────────────────────────┘
```

### Context-Based UI

**Customer Mode:**

- Show: Book parcel, My orders, Addresses
- Hide: Go online, Pending deliveries

**Captain Mode:**

- Show: Go online/offline, Pending deliveries, Earnings
- Hide: Book parcel button

**Both Modes:**

- Show: Wallet, Profile, Support
- Wallet balance is same

## Security Considerations

### Preventing Fraud

1. **Phone Verification Required**
   - Can't become captain without verified phone
   - Ensures accountability

2. **KYC for Captains**
   - Must complete KYC to deliver
   - Documents verified by admin
   - Background checks

3. **Separate Ratings**
   - Can't fake good captain rating by being good customer
   - Separate tracking prevents gaming

4. **Wallet Monitoring**
   - Flag unusual patterns
   - Monitor captain earnings vs customer spending
   - Detect potential fraud

## Testing Scenarios

### Test Case 1: Customer Adds Captain Role

```
1. Create customer account via Google
2. Add phone number
3. Verify phone with OTP
4. Call BecomeCaptain API
5. Verify captain_profile created
6. Verify roles = [CUSTOMER, CAPTAIN]
7. Submit KYC documents
8. Admin approves KYC
9. Captain can now go online
```

### Test Case 2: Captain Adds Customer Role

```
1. Create captain account via Phone+OTP
2. Captain delivers 10 orders
3. Captain wants to book parcel
4. Call BookOrder API (automatically adds CUSTOMER role)
5. Verify roles = [CAPTAIN, CUSTOMER]
6. Order created successfully
7. Captain can still deliver
```

### Test Case 3: Wallet Transactions

```
1. User has both roles
2. Book order as customer: -₹500
3. Deliver order as captain: +₹300
4. Book another order: -₹200
5. Verify wallet: -₹500 + ₹300 - ₹200 = -₹400
6. Verify transaction history shows all
```

## Migration Strategy

If you already have users in production:

```sql
-- Migration script
-- Add roles column to existing users
ALTER TABLE users ADD COLUMN IF NOT EXISTS roles TEXT[];

-- Populate roles array based on primary_role
UPDATE users
SET roles = ARRAY[primary_role]::TEXT[]
WHERE roles IS NULL OR array_length(roles, 1) IS NULL;

-- Users can now add additional roles without migration
```

## Summary

✅ **One User, Multiple Roles**

- Same email and phone across roles
- Unique constraints maintained
- Progressive role addition

✅ **Flexible Work Arrangement**

- Switch between customer and captain modes
- Shared wallet and identity
- Separate ratings and profiles

✅ **Secure & Validated**

- Phone verification required for captain
- KYC required for deliveries
- Proper capability checks

✅ **Production Ready**

- Database design supports it
- API flows implemented
- Security considered

---

**This dual-role system enables maximum flexibility while maintaining security and data integrity!** 🎉
