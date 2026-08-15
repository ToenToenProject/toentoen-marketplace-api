ALTER TABLE IF EXISTS signal_conversations DROP CONSTRAINT IF EXISTS fk_signal_conversations_business;
ALTER TABLE IF EXISTS signal_messages DROP CONSTRAINT IF EXISTS fk_signal_messages_conversation;
ALTER TABLE IF EXISTS signal_conversations DROP CONSTRAINT IF EXISTS fk_signals_conversations;
ALTER TABLE IF EXISTS signal_images DROP CONSTRAINT IF EXISTS fk_signals_images;
ALTER TABLE IF EXISTS business_reports DROP CONSTRAINT IF EXISTS fk_business_reports_business;
