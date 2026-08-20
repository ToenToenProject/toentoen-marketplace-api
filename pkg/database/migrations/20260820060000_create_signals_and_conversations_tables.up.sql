-- 1. Pastikan ekstensi PostGIS aktif
CREATE EXTENSION IF NOT EXISTS postgis;

-- 2. Tabel Signals (Permintaan kebutuhan customer)
CREATE TABLE IF NOT EXISTS signals (
    signal_id VARCHAR(50) PRIMARY KEY,
    customer_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    category VARCHAR(100) NOT NULL,
    latitude DECIMAL(10, 8) NOT NULL,
    longitude DECIMAL(11, 8) NOT NULL,
    radius_km INT DEFAULT 5,
    is_urgent BOOLEAN DEFAULT FALSE,
    status VARCHAR(20) DEFAULT 'active', -- 'active', 'completed', 'expired'
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    location geometry(Point, 4326)
);

CREATE INDEX IF NOT EXISTS idx_signals_customer_id ON signals(customer_id);
CREATE INDEX IF NOT EXISTS idx_signals_status ON signals(status);
CREATE INDEX IF NOT EXISTS idx_signals_category ON signals(category);
CREATE INDEX IF NOT EXISTS idx_signals_expires_at ON signals(expires_at);
CREATE INDEX IF NOT EXISTS idx_signals_location ON signals USING GIST(location);

-- 3. Tabel Signal Images (Lampiran gambar pada sinyal)
CREATE TABLE IF NOT EXISTS signal_images (
    image_id VARCHAR(50) PRIMARY KEY,
    signal_id VARCHAR(50) NOT NULL REFERENCES signals(signal_id) ON DELETE CASCADE,
    image_url TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_signal_images_signal_id ON signal_images(signal_id);

-- 4. Tabel Signal Conversations (Ruang obrolan / roomchat antara customer dan merchant)
CREATE TABLE IF NOT EXISTS signal_conversations (
    conversation_id VARCHAR(50) PRIMARY KEY,
    signal_id VARCHAR(50) NOT NULL REFERENCES signals(signal_id) ON DELETE CASCADE,
    merchant_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    business_id VARCHAR(50) NOT NULL REFERENCES businesses(business_id) ON DELETE CASCADE,
    status VARCHAR(20) DEFAULT 'open', -- 'open', 'locked'
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_signal_conversations_signal_id ON signal_conversations(signal_id);
CREATE INDEX IF NOT EXISTS idx_signal_conversations_merchant_id ON signal_conversations(merchant_id);
CREATE INDEX IF NOT EXISTS idx_signal_conversations_business_id ON signal_conversations(business_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_signal_conv_signal_merchant ON signal_conversations(signal_id, merchant_id);

-- 5. Tabel Signal Messages (Pesan chat dalam ruang obrolan)
CREATE TABLE IF NOT EXISTS signal_messages (
    message_id VARCHAR(50) PRIMARY KEY,
    conversation_id VARCHAR(50) NOT NULL REFERENCES signal_conversations(conversation_id) ON DELETE CASCADE,
    sender_type VARCHAR(20) NOT NULL, -- 'customer' | 'merchant'
    sender_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    message_text TEXT,
    attachment_url TEXT,
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_signal_messages_conversation_id ON signal_messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_signal_messages_sender_id ON signal_messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_signal_messages_created_at ON signal_messages(created_at);
