-- GIN index on users.interests (TEXT[]) for matching queries
CREATE INDEX IF NOT EXISTS idx_users_interests ON users USING GIN(interests);

-- GIN index on messages.metadata (JSONB) for metadata lookups
CREATE INDEX IF NOT EXISTS idx_messages_metadata ON messages USING GIN(metadata);

-- GIN index on matching_queue.interests (JSONB) for interest-based filtering
CREATE INDEX IF NOT EXISTS idx_matching_queue_interests ON matching_queue USING GIN(interests);

-- Index on match_history.chat_id for looking up match by chat
CREATE INDEX IF NOT EXISTS idx_match_history_chat ON match_history(chat_id);

-- Index on match_history.completed_at for recent-completion queries
CREATE INDEX IF NOT EXISTS idx_match_history_completed ON match_history(completed_at DESC)
    WHERE completed_at IS NOT NULL;
