ALTER TABLE business_review_comments 
DROP CONSTRAINT IF EXISTS fk_reviews_comments,
ADD CONSTRAINT fk_reviews_comments FOREIGN KEY (review_id) REFERENCES reviews(review_id) ON DELETE CASCADE;

ALTER TABLE business_post_comments 
DROP CONSTRAINT IF EXISTS fk_business_post_comments_post,
ADD CONSTRAINT fk_business_post_comments_post FOREIGN KEY (post_id) REFERENCES business_posts(post_id) ON DELETE CASCADE;
