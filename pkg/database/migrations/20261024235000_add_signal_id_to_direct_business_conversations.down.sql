-- Migration Down: Remove signal_id from direct_business_conversations
DROP INDEX IF EXISTS idx_dbc_signal_id;
ALTER TABLE direct_business_conversations DROP COLUMN IF EXISTS signal_id;
