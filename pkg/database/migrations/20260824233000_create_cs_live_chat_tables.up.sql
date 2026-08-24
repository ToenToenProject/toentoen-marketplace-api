-- Migration: Create CS Live Chat Tables
CREATE TABLE IF NOT EXISTS cs_chat_rooms (
    room_id VARCHAR(50) PRIMARY KEY,
    account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    user_role VARCHAR(20) NOT NULL,
    business_id VARCHAR(50) REFERENCES businesses(business_id) ON DELETE SET NULL,
    cs_admin_id VARCHAR(50) REFERENCES accounts(account_id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'open',
    user_hidden BOOLEAN NOT NULL DEFAULT FALSE,
    ended_at TIMESTAMP WITH TIME ZONE,
    ended_by VARCHAR(20),
    rating INTEGER,
    feedback TEXT,
    last_message_preview TEXT,
    last_message_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cs_chat_rooms_account_id ON cs_chat_rooms(account_id);
CREATE INDEX IF NOT EXISTS idx_cs_chat_rooms_status ON cs_chat_rooms(status);
CREATE INDEX IF NOT EXISTS idx_cs_chat_rooms_user_hidden ON cs_chat_rooms(user_hidden);
CREATE INDEX IF NOT EXISTS idx_cs_chat_rooms_last_message_at ON cs_chat_rooms(last_message_at DESC);

CREATE TABLE IF NOT EXISTS cs_chat_messages (
    message_id VARCHAR(50) PRIMARY KEY,
    room_id VARCHAR(50) NOT NULL REFERENCES cs_chat_rooms(room_id) ON DELETE CASCADE,
    sender_id VARCHAR(50) NOT NULL,
    sender_type VARCHAR(20) NOT NULL,
    sender_name VARCHAR(100) NOT NULL,
    message_type VARCHAR(20) NOT NULL DEFAULT 'text',
    content TEXT NOT NULL,
    media_url TEXT,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    read_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cs_chat_messages_room_id ON cs_chat_messages(room_id);
CREATE INDEX IF NOT EXISTS idx_cs_chat_messages_created_at ON cs_chat_messages(created_at ASC);
