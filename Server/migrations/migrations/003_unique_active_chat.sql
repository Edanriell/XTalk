-- Enforce at most one active chat per participant at the database level
-- to prevent TOCTOU races in concurrent create_chat calls.
DROP INDEX IF EXISTS idx_chats_participant1_active;
DROP INDEX IF EXISTS idx_chats_participant2_active;

CREATE UNIQUE INDEX idx_chats_participant1_active ON chats(participant1_id) WHERE status = 'active';
CREATE UNIQUE INDEX idx_chats_participant2_active ON chats(participant2_id) WHERE status = 'active';
