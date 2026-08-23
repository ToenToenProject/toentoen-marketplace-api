-- Fix user_notifications id column to accept VARCHAR(50) like NOTIF-XXX format
ALTER TABLE user_notifications ALTER COLUMN id TYPE VARCHAR(50);
