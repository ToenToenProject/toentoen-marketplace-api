CREATE TABLE accounts (
  account_id SERIAL PRIMARY KEY,
  auth_provider VARCHAR(50) NOT NULL DEFAULT 'local',
  email VARCHAR(255) UNIQUE,
  phone_number VARCHAR(20) UNIQUE,
  fullname VARCHAR(255) NOT NULL,
  username VARCHAR(50) UNIQUE NOT NULL,
  password_hash TEXT,
  role enum_account_roles NOT NULL,
  gender enum_gender,
  account_status enum_account_status NOT NULL,
  last_login_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);