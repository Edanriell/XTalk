CREATE TABLE IF NOT EXISTS auth_outbox (
    id BIGSERIAL PRIMARY KEY,
    aggregate_id VARCHAR(36) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMP NULL
);

CREATE INDEX IF NOT EXISTS idx_auth_outbox_unpublished
    ON auth_outbox (id) WHERE published_at IS NULL;
