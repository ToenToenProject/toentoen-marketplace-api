ALTER TABLE business_admin_assignments
ADD COLUMN IF NOT EXISTS permissions JSONB NOT NULL DEFAULT '{}';

COMMENT ON COLUMN business_admin_assignments.permissions IS
'Granular permission toggle per assignment. Keys: change_product_image,
change_business_hours, edit_product_details, create_promotion,
add_premium_feature, create_post, add_delete_products, create_support_ticket.
Value: boolean. Key tidak ada = false (default deny).';
