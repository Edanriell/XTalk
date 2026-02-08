-- Matching queue table (tracks users waiting for a match)
CREATE TABLE IF NOT EXISTS matching_queue (
    user_id VARCHAR(36) PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    age INTEGER,
    min_age INTEGER,
    max_age INTEGER,
    interests JSONB DEFAULT '[]',
    gender VARCHAR(20),
    location VARCHAR(100),
    joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    priority INTEGER DEFAULT 0
);

CREATE INDEX idx_matching_queue_joined ON matching_queue(joined_at);
CREATE INDEX idx_matching_queue_priority ON matching_queue(priority DESC, joined_at ASC);

-- Match history table (records completed matches)
CREATE TABLE IF NOT EXISTS match_history (
    id VARCHAR(36) PRIMARY KEY,
    user1_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user2_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    match_score FLOAT DEFAULT 0,
    chat_id VARCHAR(36) DEFAULT '' REFERENCES chats(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'completed', 'cancelled')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX idx_match_history_user1 ON match_history(user1_id);
CREATE INDEX idx_match_history_user2 ON match_history(user2_id);
CREATE INDEX idx_match_history_users ON match_history(user1_id, user2_id);
CREATE INDEX idx_match_history_status ON match_history(status);
CREATE INDEX idx_match_history_created_at ON match_history(created_at DESC);
CREATE INDEX idx_match_history_recent ON match_history(user1_id, user2_id, created_at) WHERE created_at > NOW() - INTERVAL '24 hours';
