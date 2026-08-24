-- Migration: Create Support Tickets and Attachments tables
CREATE TABLE IF NOT EXISTS support_tickets (
    ticket_id VARCHAR(50) PRIMARY KEY,
    account_id VARCHAR(50) NOT NULL REFERENCES accounts(account_id) ON DELETE CASCADE,
    user_role VARCHAR(20) NOT NULL,
    business_id VARCHAR(50) REFERENCES businesses(business_id) ON DELETE SET NULL,
    category VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    support_ticket_status VARCHAR(20) NOT NULL DEFAULT 'pending',
    customer_service_suggestions TEXT,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolved_by VARCHAR(50) REFERENCES accounts(account_id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_support_tickets_account_id ON support_tickets(account_id);
CREATE INDEX IF NOT EXISTS idx_support_tickets_status ON support_tickets(support_ticket_status);
CREATE INDEX IF NOT EXISTS idx_support_tickets_category ON support_tickets(category);
CREATE INDEX IF NOT EXISTS idx_support_tickets_created_at ON support_tickets(created_at DESC);

CREATE TABLE IF NOT EXISTS support_ticket_attachments (
    attachment_id VARCHAR(50) PRIMARY KEY,
    ticket_id VARCHAR(50) NOT NULL REFERENCES support_tickets(ticket_id) ON DELETE CASCADE,
    file_url TEXT NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_support_ticket_attachments_ticket_id ON support_ticket_attachments(ticket_id);
