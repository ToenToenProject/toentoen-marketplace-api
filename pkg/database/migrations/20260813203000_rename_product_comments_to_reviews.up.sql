DO $$
BEGIN
    -- Rename tabel jika product_comments ada dan product_reviews belum ada
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'product_comments') AND
       NOT EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'product_reviews') THEN
        ALTER TABLE product_comments RENAME TO product_reviews;
    END IF;

    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'product_comment_likes') AND
       NOT EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'product_review_likes') THEN
        ALTER TABLE product_comment_likes RENAME TO product_review_likes;
    END IF;

    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'product_comment_reports') AND
       NOT EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'product_review_reports') THEN
        ALTER TABLE product_comment_reports RENAME TO product_review_reports;
    END IF;
END $$;

-- Ubah nama kolom jika kolom lama masih ada
DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'product_reviews' AND column_name = 'comment_id') THEN
        ALTER TABLE product_reviews RENAME COLUMN comment_id TO review_id;
    END IF;

    IF EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'product_reviews' AND column_name = 'reply_to_comment_id') THEN
        ALTER TABLE product_reviews RENAME COLUMN reply_to_comment_id TO reply_to_review_id;
    END IF;

    IF EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'product_review_likes' AND column_name = 'comment_id') THEN
        ALTER TABLE product_review_likes RENAME COLUMN comment_id TO review_id;
    END IF;

    IF EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'product_review_reports' AND column_name = 'comment_id') THEN
        ALTER TABLE product_review_reports RENAME COLUMN comment_id TO review_id;
    END IF;
END $$;

-- Hapus kolom comment_count di products jika masih ada
ALTER TABLE IF EXISTS products DROP COLUMN IF EXISTS comment_count;
