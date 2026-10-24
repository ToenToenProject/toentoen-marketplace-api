-- Migration: Add signal_id to direct_business_conversations
ALTER TABLE direct_business_conversations 
ADD COLUMN IF NOT EXISTS signal_id VARCHAR(50) REFERENCES signals(signal_id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_dbc_signal_id ON direct_business_conversations(signal_id);
