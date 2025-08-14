-- 0001_init_down.sql
-- This migration reverses 0001_init.sql

-- Step 1: Drop triggers (if you had added them in up – not in original, but included in improved version)
-- If you used the enhanced up with triggers, uncomment these:
-- DROP TRIGGER IF EXISTS update_users_updated_at ON users;
-- DROP TRIGGER IF EXISTS update_api_keys_updated_at ON api_keys;
-- DROP FUNCTION IF EXISTS update_updated_at_column();

-- Step 2: Drop indexes
DROP INDEX IF EXISTS api_keys_user_id_idx;
DROP INDEX IF EXISTS api_keys_key_hash_idx;
DROP INDEX IF EXISTS users_email_idx;
-- Note: Index on users(id) was redundant and auto-created by PK, so no need to drop explicitly

-- Step 3: Drop tables
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS users;

-- Step 4: Optionally drop citext extension
-- ⚠️ WARNING: Only drop citext if no other tables/schemas depend on it!
-- It's safer to leave it installed, as it's a harmless extension.
-- Uncomment only if you're certain it was created solely for this migration.
-- DROP EXTENSION IF EXISTS citext;