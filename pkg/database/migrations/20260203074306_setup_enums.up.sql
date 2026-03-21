CREATE TYPE enum_account_roles as ENUM ('user', 'owner', 'business_admin', 
'platform_admin');
CREATE TYPE enum_account_status as ENUM ('active', 'inactive', 'suspended', 'banned');
CREATE TYPE enum_gender AS ENUM ('male', 'female', 'other');
CREATE TYPE enum_platform_admin_role AS ENUM ('super_admin', 'moderator', 'finance', 'support');
CREATE TYPE enum_business_admin_role AS ENUM ('super_admin', 'admin');
CREATE TYPE enum_business_status AS ENUM ('draft', 'active', 'suspended', 'banned');
CREATE TYPE enum_subscription AS ENUM ('free', 'basic', 'premium');
CREATE TYPE enum_stock AS ENUM ('in_stock', 'pre_order', 'limited', 'out_of_stock');
CREATE TYPE enum_signal_status AS ENUM ('open', 'fulfilled', 'expired', 'cancelled');
CREATE TYPE enum_bookmark_target AS ENUM ('business', 'product');
CREATE TYPE enum_like_target AS ENUM ('review', 'comment', 'business_post');
CREATE TYPE enum_comment_target AS ENUM ('post', 'product_review', 'business_review', 'product');
CREATE TYPE enum_ad_target AS ENUM ('business_profile', 'product');
CREATE TYPE enum_ad_status AS ENUM ('active', 'paused', 'completed');
CREATE TYPE enum_payment_status AS ENUM ('pending', 'paid', 'failed');
CREATE TYPE enum_notif_type AS ENUM ('review', 'signal_match', 'system', 'promo');