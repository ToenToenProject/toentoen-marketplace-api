DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_product_comments_account' AND table_name = 'product_comments'
    ) THEN
        ALTER TABLE product_comments DROP CONSTRAINT fk_product_comments_account;
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'product_comments') THEN
        ALTER TABLE product_comments 
            ADD CONSTRAINT fk_product_comments_account 
            FOREIGN KEY (account_id) REFERENCES accounts(account_id) ON DELETE CASCADE;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_product_comments_replied_account' AND table_name = 'product_comments'
    ) THEN
        ALTER TABLE product_comments DROP CONSTRAINT fk_product_comments_replied_account;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'product_comments' AND column_name = 'replied_to_account_id'
    ) THEN
        ALTER TABLE product_comments 
            ADD CONSTRAINT fk_product_comments_replied_account 
            FOREIGN KEY (replied_to_account_id) REFERENCES accounts(account_id) ON DELETE SET NULL;
    END IF;
END $$;
