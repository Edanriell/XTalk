-- Chat service: owns chat rooms (no cross-DB foreign keys)
CREATE TABLE IF NOT EXISTS chats (
    id VARCHAR(36) PRIMARY KEY,
    participant1_id VARCHAR(36) NOT NULL,
    participant2_id VARCHAR(36) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'ended')),
    match_score FLOAT DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_chats_participant1 ON chats(participant1_id);
CREATE INDEX IF NOT EXISTS idx_chats_participant2 ON chats(participant2_id);
CREATE INDEX IF NOT EXISTS idx_chats_status ON chats(status);
CREATE INDEX IF NOT EXISTS idx_chats_updated_at ON chats(updated_at);
CREATE INDEX IF NOT EXISTS idx_chats_participant1_active ON chats(participant1_id) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_chats_participant2_active ON chats(participant2_id) WHERE status = 'active';

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_chats_updated_at BEFORE UPDATE ON chats
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
