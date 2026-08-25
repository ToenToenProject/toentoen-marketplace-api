-- Migration: Create Direct Business Chat Tables (Customer <-> Merchant)
CREATE TABLE IF NOT EXISTS direct_business_conversations (
    conversation_id VARCHAR(50) PRIMARY KEY,
    business_id VARCHAR(50) NOT NULL REFERENCES businesses(business_id) ON DELETE CASCADE,
    customer_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    last_message_preview TEXT,
    last_message_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_business_customer UNIQUE (business_id, customer_id)
);

CREATE INDEX IF NOT EXISTS idx_dbc_business_id ON direct_business_conversations(business_id);
CREATE INDEX IF NOT EXISTS idx_dbc_customer_id ON direct_business_conversations(customer_id);
CREATE INDEX IF NOT EXISTS idx_dbc_last_message_at ON direct_business_conversations(last_message_at DESC);

CREATE TABLE IF NOT EXISTS direct_business_messages (
    message_id VARCHAR(50) PRIMARY KEY,
    conversation_id VARCHAR(50) NOT NULL REFERENCES direct_business_conversations(conversation_id) ON DELETE CASCADE,
    sender_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    sender_type VARCHAR(20) NOT NULL,
    sender_name VARCHAR(100) NOT NULL,
    message_type VARCHAR(20) NOT NULL DEFAULT 'text',
    content TEXT NOT NULL,
    media_url TEXT,
    product_id VARCHAR(50) REFERENCES products(product_id) ON DELETE SET NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    read_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_dbm_conversation_id ON direct_business_messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_dbm_created_at ON direct_business_messages(created_at ASC);
