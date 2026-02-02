-- migrations/001_initial_schema.down.sql
-- Rollback script for initial schema

-- Drop triggers
DROP TRIGGER IF EXISTS update_addresses_updated_at ON addresses;
DROP TRIGGER IF EXISTS update_admin_profiles_updated_at ON admin_profiles;
DROP TRIGGER IF EXISTS update_captain_profiles_updated_at ON captain_profiles;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in correct order (respecting foreign keys)
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS otp_sessions;
DROP TABLE IF EXISTS addresses;
DROP TABLE IF EXISTS admin_profiles;
DROP TABLE IF EXISTS captain_profiles;
DROP TABLE IF EXISTS users;

-- Drop extensions (optional - may want to keep for other apps)
-- DROP EXTENSION IF EXISTS "pg_trgm";
-- DROP EXTENSION IF EXISTS "postgis";
-- DROP EXTENSION IF EXISTS "uuid-ossp";