-- migrations/001_initial_schema.up.sql
-- Production-Grade Delivery App Database Schema
-- PostgreSQL 15+ with PostGIS extension

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "postgis";
CREATE EXTENSION IF NOT EXISTS "pg_trgm"; -- For text search

-- ============================================================================
-- USERS TABLE (Central user table for all roles)
-- ============================================================================

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    version INT NOT NULL DEFAULT 1, -- Optimistic locking
    
    -- Profile
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    profile_pic VARCHAR(500),
    
    -- Contact
    email VARCHAR(255) UNIQUE, -- Still unique - one person, one email
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    phone VARCHAR(20) UNIQUE, -- Still unique - one person, one phone
    phone_verified BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Role & Status
    primary_role VARCHAR(20) NOT NULL, -- The initial role (CUSTOMER or CAPTAIN)
    roles TEXT[] NOT NULL DEFAULT '{}', -- Can have multiple roles: ['CUSTOMER', 'CAPTAIN']
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    
    -- Authentication
    provider VARCHAR(50), -- GOOGLE, APPLE, PHONE, EMAIL
    provider_id VARCHAR(255) UNIQUE,
    
    -- Wallet (in paise/cents for precision)
    wallet_balance BIGINT NOT NULL DEFAULT 0,
    wallet_currency VARCHAR(3) NOT NULL DEFAULT 'INR',
    wallet_version INT NOT NULL DEFAULT 1, -- Separate version for wallet operations
    
    -- Stats
    total_orders INT NOT NULL DEFAULT 0,
    total_deliveries INT NOT NULL DEFAULT 0,
    rating DECIMAL(3,2) DEFAULT 0.00,
    total_ratings INT NOT NULL DEFAULT 0,
    
    -- Metadata
    last_active_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    -- Constraints
    CONSTRAINT users_email_or_phone_required CHECK (
        -- Customers: Must have email OR phone (email preferred for Google/Apple login)
        -- Captains: Must have phone (email optional)
        -- Admins: Must have email
        (primary_role = 'CUSTOMER' AND (email IS NOT NULL OR phone IS NOT NULL)) OR
        (primary_role = 'CAPTAIN' AND phone IS NOT NULL) OR
        (primary_role = 'ADMIN' AND email IS NOT NULL)
    ),
    CONSTRAINT users_rating_range CHECK (rating >= 0 AND rating <= 5),
    CONSTRAINT users_wallet_balance_non_negative CHECK (wallet_balance >= 0)
);

-- Indexes for users table
CREATE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL;
CREATE INDEX idx_users_phone ON users(phone) WHERE phone IS NOT NULL;
CREATE INDEX idx_users_primary_role ON users(primary_role);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_provider ON users(provider, provider_id);
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_last_active ON users(last_active_at) WHERE last_active_at IS NOT NULL;
CREATE INDEX idx_users_rating ON users(rating) WHERE rating > 0;

-- Text search on names
CREATE INDEX idx_users_name_trgm ON users USING gin((first_name || ' ' || last_name) gin_trgm_ops);

-- Index for efficient role queries (dual-role support)
CREATE INDEX idx_users_roles_gin ON users USING GIN(roles);

-- ============================================================================
-- CAPTAIN PROFILES TABLE (Captain-specific data)
-- ============================================================================

CREATE TABLE captain_profiles (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    version INT NOT NULL DEFAULT 1,
    
    -- Vehicle
    vehicle_type VARCHAR(20), -- BIKE, CAR, VAN, TRUCK
    vehicle_number VARCHAR(50) UNIQUE,
    vehicle_model VARCHAR(100),
    
    -- License
    license_number VARCHAR(100) UNIQUE,
    license_expiry TIMESTAMPTZ,
    
    -- KYC
    kyc_status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    kyc_documents JSONB NOT NULL DEFAULT '{}',
    
    -- Operational
    is_available BOOLEAN NOT NULL DEFAULT FALSE,
    current_location GEOGRAPHY(POINT, 4326), -- PostGIS geography for accurate distance
    location_updated_at TIMESTAMPTZ,
    
    -- Performance
    completion_rate DECIMAL(5,2) DEFAULT 0.00,
    cancellation_rate DECIMAL(5,2) DEFAULT 0.00,
    on_time_rate DECIMAL(5,2) DEFAULT 0.00,
    
    -- Timestamps
    onboarded_at TIMESTAMPTZ,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    CONSTRAINT captain_kyc_status_valid CHECK (kyc_status IN ('PENDING', 'VERIFIED', 'REJECTED')),
    CONSTRAINT captain_vehicle_type_valid CHECK (vehicle_type IN ('BIKE', 'CAR', 'VAN', 'TRUCK'))
);

-- Spatial index for captain location (CRITICAL for nearby captain queries)
CREATE INDEX idx_captain_location ON captain_profiles USING GIST(current_location);

-- Other indexes
CREATE INDEX idx_captain_availability ON captain_profiles(is_available, kyc_status) WHERE is_available = TRUE;
CREATE INDEX idx_captain_kyc_status ON captain_profiles(kyc_status);
CREATE INDEX idx_captain_vehicle_number ON captain_profiles(vehicle_number) WHERE vehicle_number IS NOT NULL;
CREATE INDEX idx_captain_license_number ON captain_profiles(license_number) WHERE license_number IS NOT NULL;

-- ============================================================================
-- ADMIN PROFILES TABLE (Admin-specific data)
-- ============================================================================

CREATE TABLE admin_profiles (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    version INT NOT NULL DEFAULT 1,
    
    -- Permissions
    permissions TEXT[] NOT NULL DEFAULT '{}',
    
    -- Department
    department VARCHAR(50) NOT NULL,
    employee_id VARCHAR(50) UNIQUE,
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_admin_department ON admin_profiles(department);
CREATE INDEX idx_admin_employee_id ON admin_profiles(employee_id) WHERE employee_id IS NOT NULL;

-- ============================================================================
-- ADDRESSES TABLE (with comprehensive analytics)
-- ============================================================================

CREATE TABLE addresses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    version INT NOT NULL DEFAULT 1,
    
    -- Basic Information
    label VARCHAR(20), -- HOME, WORK, OTHER
    address_line1 VARCHAR(255) NOT NULL,
    address_line2 VARCHAR(255),
    landmark VARCHAR(255),
    city VARCHAR(100) NOT NULL,
    state VARCHAR(100) NOT NULL,
    country VARCHAR(100) NOT NULL DEFAULT 'India',
    postal_code VARCHAR(20),
    
    -- Geolocation
    location GEOGRAPHY(POINT, 4326) NOT NULL,
    
    -- Contact
    contact_name VARCHAR(100),
    contact_phone VARCHAR(20),
    
    -- Status
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Usage Analytics
    usage_count INT NOT NULL DEFAULT 0,
    last_used_at TIMESTAMPTZ,
    
    -- Time-based Analytics
    morning_usage_count INT NOT NULL DEFAULT 0,    -- 6 AM - 12 PM
    afternoon_usage_count INT NOT NULL DEFAULT 0,  -- 12 PM - 6 PM
    evening_usage_count INT NOT NULL DEFAULT 0,    -- 6 PM - 12 AM
    night_usage_count INT NOT NULL DEFAULT 0,      -- 12 AM - 6 AM
    
    -- Day-based Analytics
    weekday_usage_count INT NOT NULL DEFAULT 0,    -- Mon-Fri
    weekend_usage_count INT NOT NULL DEFAULT 0,    -- Sat-Sun
    
    -- Monthly Analytics (stored as JSONB for flexibility)
    monthly_usage_count JSONB NOT NULL DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Spatial index for location-based queries
CREATE INDEX idx_addresses_location ON addresses USING GIST(location);

-- Other indexes
CREATE INDEX idx_addresses_user_id ON addresses(user_id);
CREATE INDEX idx_addresses_city ON addresses(city);
CREATE INDEX idx_addresses_state ON addresses(state);
CREATE INDEX idx_addresses_postal_code ON addresses(postal_code);
CREATE INDEX idx_addresses_is_default ON addresses(user_id, is_default) WHERE is_default = TRUE;
CREATE INDEX idx_addresses_is_active ON addresses(user_id, is_active) WHERE is_active = TRUE;
CREATE INDEX idx_addresses_last_used ON addresses(user_id, last_used_at DESC) WHERE last_used_at IS NOT NULL;
CREATE INDEX idx_addresses_usage_score ON addresses(user_id, usage_count DESC, last_used_at DESC);

-- ============================================================================
-- OTP SESSIONS TABLE (for authentication)
-- ============================================================================

CREATE TABLE otp_sessions (
    id SERIAL PRIMARY KEY,
    target VARCHAR(255) NOT NULL, -- email or phone
    code VARCHAR(10) NOT NULL,
    purpose VARCHAR(50) NOT NULL, -- LOGIN, VERIFICATION, PASSWORD_RESET
    used BOOLEAN NOT NULL DEFAULT FALSE,
    attempts INT NOT NULL DEFAULT 0, -- Track failed attempts
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    
    CONSTRAINT otp_attempts_limit CHECK (attempts <= 5)
);

-- Indexes for OTP lookups
CREATE INDEX idx_otp_target_purpose ON otp_sessions(target, purpose, used) WHERE NOT used;
CREATE INDEX idx_otp_expires_at ON otp_sessions(expires_at);

-- Auto-delete expired OTP sessions (runs daily)
CREATE INDEX idx_otp_cleanup ON otp_sessions(expires_at) WHERE NOT used;

-- ============================================================================
-- REFRESH TOKENS TABLE (for session management)
-- ============================================================================

CREATE TABLE refresh_tokens (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(500) NOT NULL UNIQUE,
    device VARCHAR(50), -- IOS, ANDROID, WEB
    device_info JSONB, -- Device details
    
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
    revoked_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ
);

-- Indexes for token management
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token ON refresh_tokens(token) WHERE NOT is_revoked;
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- Function to update updated_at timestamp automatically
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply updated_at trigger to all tables
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_captain_profiles_updated_at BEFORE UPDATE ON captain_profiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_admin_profiles_updated_at BEFORE UPDATE ON admin_profiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_addresses_updated_at BEFORE UPDATE ON addresses
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- INITIAL DATA
-- ============================================================================

-- Insert default admin user (password should be changed immediately)
-- This is just for initial setup
-- INSERT INTO users (id, first_name, last_name, email, email_verified, primary_role, roles, status)
-- VALUES (
--     uuid_generate_v4(),
--     'System',
--     'Admin',
--     'admin@turtledelivery.com',
--     TRUE,
--     'ADMIN',
--     ARRAY['ADMIN'],
--     'ACTIVE'
-- );