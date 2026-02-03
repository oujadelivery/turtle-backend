-- ============================================================================
-- ORDERS TABLE (Parcel Delivery Orders)
-- ============================================================================

CREATE TABLE orders (
    -- Identity
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    version INT NOT NULL DEFAULT 1, -- Optimistic locking
    
    -- Order Type (future-proof)
    order_type VARCHAR(20) NOT NULL DEFAULT 'PARCEL', -- PARCEL, RIDE, MULTI_STOP, SCHEDULED
    
    -- Parties (Dual Role + Third-Party Support)
    booked_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    assigned_captain_id UUID REFERENCES users(id) ON DELETE RESTRICT,
    
    -- Sender Details (Value Object - NOT a user reference)
    sender_name VARCHAR(100) NOT NULL,
    sender_phone VARCHAR(20) NOT NULL,
    sender_alternate_phone VARCHAR(20),
    sender_notes VARCHAR(500),
    
    -- Receiver Details (Value Object - NOT a user reference)
    receiver_name VARCHAR(100) NOT NULL,
    receiver_phone VARCHAR(20) NOT NULL,
    receiver_alternate_phone VARCHAR(20),
    receiver_notes VARCHAR(500),
    
    -- Pickup Address
    pickup_address_line1 VARCHAR(255) NOT NULL,
    pickup_address_line2 VARCHAR(255),
    pickup_landmark VARCHAR(255),
    pickup_city VARCHAR(100) NOT NULL,
    pickup_state VARCHAR(100) NOT NULL,
    pickup_postal_code VARCHAR(20),
    pickup_location GEOGRAPHY(POINT, 4326) NOT NULL,
    
    -- Delivery Address
    delivery_address_line1 VARCHAR(255) NOT NULL,
    delivery_address_line2 VARCHAR(255),
    delivery_landmark VARCHAR(255),
    delivery_city VARCHAR(100) NOT NULL,
    delivery_state VARCHAR(100) NOT NULL,
    delivery_postal_code VARCHAR(20),
    delivery_location GEOGRAPHY(POINT, 4326) NOT NULL,
    
    -- Parcel Details (for PARCEL orders)
    parcel_type VARCHAR(20), -- DOCUMENT, PACKAGE, FOOD, FRAGILE, etc.
    parcel_weight DECIMAL(5,2), -- in kg
    parcel_description VARCHAR(500),
    parcel_value BIGINT, -- declared value in paise
    parcel_images TEXT[], -- Array of image URLs
    
    -- Pricing (stored as snapshot)
    base_price BIGINT NOT NULL, -- in paise
    distance_price BIGINT NOT NULL, -- in paise
    weight_price BIGINT, -- in paise
    surge_multiplier DECIMAL(3,2) DEFAULT 1.00,
    surge_amount BIGINT DEFAULT 0,
    waiting_charges BIGINT DEFAULT 0,
    insurance_fee BIGINT DEFAULT 0,
    platform_commission BIGINT DEFAULT 0,
    discount_amount BIGINT DEFAULT 0,
    subtotal BIGINT NOT NULL,
    total BIGINT NOT NULL, -- Final amount
    captain_earning BIGINT, -- Captain's share
    
    -- Pricing Metadata
    distance_km DECIMAL(6,2) NOT NULL,
    estimated_minutes INT,
    currency VARCHAR(3) NOT NULL DEFAULT 'INR',
    
    -- State Machine
    state VARCHAR(30) NOT NULL DEFAULT 'CREATED',
    state_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- OTPs for Security
    pickup_otp VARCHAR(10),
    delivery_otp VARCHAR(10),
    otp_expires_at TIMESTAMPTZ,
    
    -- Timing
    scheduled_pickup_time TIMESTAMPTZ,
    actual_pickup_time TIMESTAMPTZ,
    estimated_delivery TIMESTAMPTZ,
    actual_delivery_time TIMESTAMPTZ,
    
    -- Waiting Time
    waiting_start_time TIMESTAMPTZ,
    waiting_minutes INT DEFAULT 0,
    
    -- Cancellation
    cancellation_cancelled_by UUID REFERENCES users(id),
    cancellation_cancelled_by_role VARCHAR(20), -- CUSTOMER, CAPTAIN, ADMIN, SYSTEM
    cancellation_reason VARCHAR(500),
    cancellation_refund_amount BIGINT,
    cancellation_penalty_amount BIGINT,
    cancellation_cancelled_at TIMESTAMPTZ,
    
    -- Payment
    payment_method VARCHAR(20) NOT NULL DEFAULT 'CASH', -- CASH, WALLET, ONLINE
    payment_status VARCHAR(20) NOT NULL DEFAULT 'PENDING', -- PENDING, CAPTURED, FAILED, REFUNDED, CANCELLED
    payment_intent_id VARCHAR(255), -- For online payments (Stripe/Razorpay)
    
    -- Notes
    customer_notes VARCHAR(1000),
    captain_notes VARCHAR(1000),
    internal_notes VARCHAR(2000), -- Admin/system notes
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    -- Constraints
    CONSTRAINT orders_valid_state CHECK (
        state IN (
            'CREATED', 'PRICE_ESTIMATED', 'SEARCHING_CAPTAIN',
            'CAPTAIN_ASSIGNED', 'CAPTAIN_ACCEPTED', 'CAPTAIN_EN_ROUTE',
            'PICKED_UP', 'IN_TRANSIT', 'AT_DELIVERY', 'DELIVERED',
            'COMPLETED', 'CANCELLED', 'FAILED'
        )
    ),
    CONSTRAINT orders_valid_order_type CHECK (
        order_type IN ('PARCEL', 'RIDE', 'MULTI_STOP', 'SCHEDULED')
    ),
    CONSTRAINT orders_valid_payment_method CHECK (
        payment_method IN ('CASH', 'WALLET', 'ONLINE')
    ),
    CONSTRAINT orders_valid_payment_status CHECK (
        payment_status IN ('PENDING', 'CAPTURED', 'FAILED', 'REFUNDED', 'CANCELLED')
    ),
    CONSTRAINT orders_prevent_self_assignment CHECK (
        assigned_captain_id IS NULL OR assigned_captain_id != booked_by_user_id
    ),
    CONSTRAINT orders_positive_amounts CHECK (
        base_price >= 0 AND
        distance_price >= 0 AND
        (weight_price IS NULL OR weight_price >= 0) AND
        subtotal >= 0 AND
        total >= 0
    ),
    CONSTRAINT orders_captain_assigned_state CHECK (
        (assigned_captain_id IS NULL AND state IN ('CREATED', 'PRICE_ESTIMATED', 'SEARCHING_CAPTAIN', 'CANCELLED', 'FAILED')) OR
        (assigned_captain_id IS NOT NULL)
    )
);

-- ============================================================================
-- INDEXES FOR ORDERS
-- ============================================================================

-- Primary lookups
CREATE INDEX idx_orders_booked_by_user_id ON orders(booked_by_user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_orders_assigned_captain_id ON orders(assigned_captain_id) WHERE deleted_at IS NULL AND assigned_captain_id IS NOT NULL;

-- State queries
CREATE INDEX idx_orders_state ON orders(state) WHERE deleted_at IS NULL;
CREATE INDEX idx_orders_active ON orders(state) WHERE deleted_at IS NULL AND state NOT IN ('COMPLETED', 'CANCELLED', 'FAILED');

-- Spatial indexes for location-based queries
CREATE INDEX idx_orders_pickup_location ON orders USING GIST(pickup_location) WHERE deleted_at IS NULL;
CREATE INDEX idx_orders_delivery_location ON orders USING GIST(delivery_location) WHERE deleted_at IS NULL;

-- Captain matching (find orders near captain's location)
CREATE INDEX idx_orders_searching_pickup ON orders USING GIST(pickup_location) 
WHERE state = 'SEARCHING_CAPTAIN' AND deleted_at IS NULL;

-- Time-based queries
CREATE INDEX idx_orders_created_at ON orders(created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_orders_scheduled ON orders(scheduled_pickup_time) WHERE scheduled_pickup_time IS NOT NULL AND deleted_at IS NULL;

-- Payment queries
CREATE INDEX idx_orders_payment_status ON orders(payment_status) WHERE deleted_at IS NULL;
CREATE INDEX idx_orders_payment_intent ON orders(payment_intent_id) WHERE payment_intent_id IS NOT NULL;

-- Analytics
CREATE INDEX idx_orders_completed_at ON orders(actual_delivery_time DESC) WHERE state = 'COMPLETED' AND deleted_at IS NULL;
CREATE INDEX idx_orders_captain_earnings ON orders(assigned_captain_id, captain_earning) WHERE state = 'COMPLETED' AND deleted_at IS NULL;

-- Composite indexes for common queries
CREATE INDEX idx_orders_user_state ON orders(booked_by_user_id, state) WHERE deleted_at IS NULL;
CREATE INDEX idx_orders_captain_state ON orders(assigned_captain_id, state) WHERE deleted_at IS NULL AND assigned_captain_id IS NOT NULL;

-- Optimistic locking support
CREATE INDEX idx_orders_id_version ON orders(id, version);

-- ============================================================================
-- ORDER STATE HISTORY TABLE (Audit Trail)
-- ============================================================================

CREATE TABLE order_state_history (
    id SERIAL PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    
    from_state VARCHAR(30) NOT NULL,
    to_state VARCHAR(30) NOT NULL,
    triggered_by UUID NOT NULL REFERENCES users(id),
    reason VARCHAR(500),
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT osh_valid_states CHECK (
        from_state IN (
            'CREATED', 'PRICE_ESTIMATED', 'SEARCHING_CAPTAIN',
            'CAPTAIN_ASSIGNED', 'CAPTAIN_ACCEPTED', 'CAPTAIN_EN_ROUTE',
            'PICKED_UP', 'IN_TRANSIT', 'AT_DELIVERY', 'DELIVERED',
            'COMPLETED', 'CANCELLED', 'FAILED'
        ) AND
        to_state IN (
            'CREATED', 'PRICE_ESTIMATED', 'SEARCHING_CAPTAIN',
            'CAPTAIN_ASSIGNED', 'CAPTAIN_ACCEPTED', 'CAPTAIN_EN_ROUTE',
            'PICKED_UP', 'IN_TRANSIT', 'AT_DELIVERY', 'DELIVERED',
            'COMPLETED', 'CANCELLED', 'FAILED'
        )
    )
);

-- Indexes for state history
CREATE INDEX idx_osh_order_id ON order_state_history(order_id);
CREATE INDEX idx_osh_created_at ON order_state_history(created_at DESC);
CREATE INDEX idx_osh_to_state ON order_state_history(to_state);

-- ============================================================================
-- ORDER EVENTS TABLE (Future: Event Sourcing)
-- ============================================================================

CREATE TABLE order_events (
    id SERIAL PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB NOT NULL,
    aggregate_version INT NOT NULL,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT oe_positive_version CHECK (aggregate_version > 0)
);

-- Indexes for order events
CREATE INDEX idx_oe_order_id ON order_events(order_id);
CREATE INDEX idx_oe_created_at ON order_events(created_at DESC);
CREATE INDEX idx_oe_event_type ON order_events(event_type);
CREATE INDEX idx_oe_aggregate_version ON order_events(order_id, aggregate_version);

-- ============================================================================
-- CAPTAIN ASSIGNMENT ATTEMPTS TABLE (For Retry Logic)
-- ============================================================================

CREATE TABLE captain_assignment_attempts (
    id SERIAL PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    captain_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    attempt_number INT NOT NULL,
    status VARCHAR(20) NOT NULL, -- SENT, ACCEPTED, DECLINED, TIMEOUT
    decline_reason VARCHAR(500),
    
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    responded_at TIMESTAMPTZ,
    timeout_at TIMESTAMPTZ NOT NULL,
    
    CONSTRAINT caa_valid_status CHECK (
        status IN ('SENT', 'ACCEPTED', 'DECLINED', 'TIMEOUT')
    ),
    CONSTRAINT caa_unique_attempt UNIQUE(order_id, captain_id, attempt_number)
);

-- Indexes for captain assignment attempts
CREATE INDEX idx_caa_order_id ON captain_assignment_attempts(order_id);
CREATE INDEX idx_caa_captain_id ON captain_assignment_attempts(captain_id);
CREATE INDEX idx_caa_status ON captain_assignment_attempts(status);
CREATE INDEX idx_caa_timeout ON captain_assignment_attempts(timeout_at) WHERE status = 'SENT';

-- ============================================================================
-- CANCELLATION EVENTS TABLE (Track Cancellation Patterns)
-- ============================================================================

CREATE TABLE cancellation_events (
    id SERIAL PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    cancelled_by_role VARCHAR(20) NOT NULL, -- CUSTOMER, CAPTAIN, ADMIN, SYSTEM
    order_state_at_cancellation VARCHAR(30) NOT NULL,
    reason VARCHAR(500),
    
    refund_amount BIGINT,
    penalty_amount BIGINT,
    penalty_points INT DEFAULT 0,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT ce_valid_role CHECK (
        cancelled_by_role IN ('CUSTOMER', 'CAPTAIN', 'ADMIN', 'SYSTEM')
    )
);

-- Indexes for cancellation events
CREATE INDEX idx_ce_order_id ON cancellation_events(order_id);
CREATE INDEX idx_ce_user_id ON cancellation_events(user_id);
CREATE INDEX idx_ce_role ON cancellation_events(cancelled_by_role);
CREATE INDEX idx_ce_created_at ON cancellation_events(created_at DESC);

-- ============================================================================
-- USER PENALTY POINTS TABLE (Cancellation Penalty System)
-- ============================================================================

CREATE TABLE user_penalty_points (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    total_penalty_points INT NOT NULL DEFAULT 0,
    total_reward_points INT NOT NULL DEFAULT 0,
    net_score INT NOT NULL DEFAULT 0, -- reward - penalty
    
    last_penalty_at TIMESTAMPTZ,
    last_reward_at TIMESTAMPTZ,
    
    is_suspended BOOLEAN NOT NULL DEFAULT FALSE,
    suspension_until TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT upp_unique_user UNIQUE(user_id),
    CONSTRAINT upp_non_negative_points CHECK (
        total_penalty_points >= 0 AND
        total_reward_points >= 0
    )
);

-- Indexes for user penalty points
CREATE INDEX idx_upp_user_id ON user_penalty_points(user_id);
CREATE INDEX idx_upp_suspended ON user_penalty_points(is_suspended) WHERE is_suspended = TRUE;

-- ============================================================================
-- TRIGGER: Auto-update updated_at timestamp
-- ============================================================================

CREATE OR REPLACE FUNCTION update_orders_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER orders_updated_at_trigger
BEFORE UPDATE ON orders
FOR EACH ROW
EXECUTE FUNCTION update_orders_updated_at();

-- Similar trigger for user_penalty_points
CREATE TRIGGER user_penalty_points_updated_at_trigger
BEFORE UPDATE ON user_penalty_points
FOR EACH ROW
EXECUTE FUNCTION update_orders_updated_at();