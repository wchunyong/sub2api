-- Keep only one-way hashes of registration IP and browser identifier.
ALTER TABLE users ADD COLUMN IF NOT EXISTS signup_ip_hash VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS signup_device_hash VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS welcome_recharge_order_id BIGINT NULL;
CREATE INDEX IF NOT EXISTS idx_users_signup_ip_created_at ON users (signup_ip_hash, created_at) WHERE signup_ip_hash <> '';
CREATE INDEX IF NOT EXISTS idx_users_signup_device_created_at ON users (signup_device_hash, created_at) WHERE signup_device_hash <> '';
