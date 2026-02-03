-- Drop triggers
DROP TRIGGER IF EXISTS orders_updated_at_trigger ON orders;
DROP TRIGGER IF EXISTS user_penalty_points_updated_at_trigger ON user_penalty_points;

-- Drop tables in correct order
DROP TABLE IF EXISTS user_penalty_points;
DROP TABLE IF EXISTS cancellation_events;
DROP TABLE IF EXISTS captain_assignment_attempts;
DROP TABLE IF EXISTS order_events;
DROP TABLE IF EXISTS order_state_history;
DROP TABLE IF EXISTS orders;

-- Drop function
DROP FUNCTION IF EXISTS update_orders_updated_at();