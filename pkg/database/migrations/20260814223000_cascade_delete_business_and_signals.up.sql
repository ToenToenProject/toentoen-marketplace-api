-- 1. Bersihkan data orphan di signal_messages dan signal_conversations jika ada business_id yang tidak valid
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'signal_conversations') AND
       EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'businesses') THEN
        IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'signal_messages') THEN
            DELETE FROM signal_messages WHERE conversation_id IN (
                SELECT conversation_id FROM signal_conversations WHERE business_id NOT IN (SELECT business_id FROM businesses)
            );
        END IF;
        DELETE FROM signal_conversations WHERE business_id NOT IN (SELECT business_id FROM businesses);
    END IF;
END $$;

-- 2. Pastikan foreign key signal_conversations -> businesses memiliki ON DELETE CASCADE
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'signal_conversations') THEN
        ALTER TABLE signal_conversations DROP CONSTRAINT IF EXISTS fk_signal_conversations_business;
        ALTER TABLE signal_conversations 
            ADD CONSTRAINT fk_signal_conversations_business 
            FOREIGN KEY (business_id) REFERENCES businesses(business_id) ON DELETE CASCADE;
    END IF;
END $$;

-- 3. Pastikan foreign key signal_messages -> signal_conversations memiliki ON DELETE CASCADE
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'signal_messages') THEN
        ALTER TABLE signal_messages DROP CONSTRAINT IF EXISTS fk_signal_conversations_messages;
        ALTER TABLE signal_messages DROP CONSTRAINT IF EXISTS fk_signal_messages_conversation;
        ALTER TABLE signal_messages 
            ADD CONSTRAINT fk_signal_messages_conversation 
            FOREIGN KEY (conversation_id) REFERENCES signal_conversations(conversation_id) ON DELETE CASCADE;
    END IF;
END $$;

-- 4. Pastikan foreign key signal_conversations -> signals memiliki ON DELETE CASCADE
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'signal_conversations') THEN
        ALTER TABLE signal_conversations DROP CONSTRAINT IF EXISTS fk_signals_conversations;
        ALTER TABLE signal_conversations 
            ADD CONSTRAINT fk_signals_conversations 
            FOREIGN KEY (signal_id) REFERENCES signals(signal_id) ON DELETE CASCADE;
    END IF;
END $$;

-- 5. Pastikan foreign key signal_images -> signals memiliki ON DELETE CASCADE
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'signal_images') THEN
        ALTER TABLE signal_images DROP CONSTRAINT IF EXISTS fk_signals_images;
        ALTER TABLE signal_images 
            ADD CONSTRAINT fk_signals_images 
            FOREIGN KEY (signal_id) REFERENCES signals(signal_id) ON DELETE CASCADE;
    END IF;
END $$;

-- 6. Pastikan business_reports memiliki foreign key ON DELETE CASCADE ke businesses
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'business_reports') THEN
        IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'business_report_evidences') THEN
            DELETE FROM business_report_evidences WHERE report_id IN (
                SELECT report_id FROM business_reports WHERE business_id NOT IN (SELECT business_id FROM businesses)
            );
        END IF;
        DELETE FROM business_reports WHERE business_id NOT IN (SELECT business_id FROM businesses);

        ALTER TABLE business_reports DROP CONSTRAINT IF EXISTS fk_business_reports_business;
        ALTER TABLE business_reports 
            ADD CONSTRAINT fk_business_reports_business 
            FOREIGN KEY (business_id) REFERENCES businesses(business_id) ON DELETE CASCADE;
    END IF;
END $$;
