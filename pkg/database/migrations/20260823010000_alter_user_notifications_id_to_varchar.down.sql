-- Revert user_notifications id column back to UUID
ALTER TABLE user_notifications ALTER COLUMN id TYPE UUID USING (id::uuid);
