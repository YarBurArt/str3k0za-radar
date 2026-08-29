CREATE TABLE IF NOT EXISTS preferences (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    apt_groups TEXT[] NOT NULL DEFAULT '{}',
    digest_enabled BOOLEAN NOT NULL DEFAULT true,
    delivery_time TIME
);
