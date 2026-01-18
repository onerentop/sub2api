-- 040_add_user_token_version.sql
-- Add token_version column to users table for JWT invalidation on password change

ALTER TABLE users ADD COLUMN IF NOT EXISTS token_version BIGINT NOT NULL DEFAULT 0;

COMMENT ON COLUMN users.token_version IS 'Token version, incremented on password change to invalidate existing JWTs';
