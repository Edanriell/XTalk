-- Users table (shared by auth-service and user-service)
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    age INTEGER,
    gender VARCHAR(20) CHECK (gender IN ('male', 'female', 'other', 'prefer_not_to_say')),
    country VARCHAR(100),
    language VARCHAR(10),
    interests TEXT[],
    status VARCHAR(20) DEFAULT 'offline' CHECK (status IN ('online', 'offline', 'away')),
    bio TEXT DEFAULT '',
    avatar_url TEXT DEFAULT '',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_is_active ON users(is_active) WHERE is_active = TRUE;

-- Chats table (1:1 chats created by matching-service)
CREATE TABLE IF NOT EXISTS chats (
    id VARCHAR(36) PRIMARY KEY,
    participant1_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    participant2_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'ended')),
    match_score FLOAT DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP
);

CREATE INDEX idx_chats_participant1 ON chats(participant1_id);
CREATE INDEX idx_chats_participant2 ON chats(participant2_id);
CREATE INDEX idx_chats_status ON chats(status);
CREATE INDEX idx_chats_updated_at ON chats(updated_at);
CREATE INDEX idx_chats_participant1_active ON chats(participant1_id) WHERE status = 'active';
CREATE INDEX idx_chats_participant2_active ON chats(participant2_id) WHERE status = 'active';

-- Messages table
CREATE TABLE IF NOT EXISTS messages (
    id VARCHAR(36) PRIMARY KEY,
    chat_id VARCHAR(36) NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    sender_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message_type VARCHAR(20) NOT NULL DEFAULT 'text' CHECK (message_type IN ('text', 'image', 'file')),
    content TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    read_at TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_messages_chat ON messages(chat_id);
CREATE INDEX idx_messages_sender ON messages(sender_id);
CREATE INDEX idx_messages_created_at ON messages(created_at DESC);
CREATE INDEX idx_messages_chat_created ON messages(chat_id, created_at DESC);
CREATE INDEX idx_messages_unread ON messages(chat_id, sender_id) WHERE is_read = FALSE AND deleted_at IS NULL;

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers to automatically update updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_chats_updated_at BEFORE UPDATE ON chats
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
