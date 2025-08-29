-- Migration 002: Create clients table
-- This migration creates the clients table to replace file-based storage

-- Create clients table
CREATE TABLE IF NOT EXISTS clients (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    active BOOLEAN DEFAULT true,
    api_keys TEXT[], -- Array of API keys
    encryption_key_id VARCHAR(255),
    rate_limit_requests_per_minute INTEGER DEFAULT 100,
    rate_limit_burst_limit INTEGER DEFAULT 20,
    rate_limit_enabled BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    last_accessed_at TIMESTAMP,
    integration_count INTEGER DEFAULT 0
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_clients_name ON clients(name);
CREATE INDEX IF NOT EXISTS idx_clients_active ON clients(active);
CREATE INDEX IF NOT EXISTS idx_clients_created_at ON clients(created_at);
CREATE INDEX IF NOT EXISTS idx_clients_api_keys ON clients USING GIN(api_keys);
CREATE INDEX IF NOT EXISTS idx_clients_metadata ON clients USING GIN(metadata);

-- Create trigger for updated_at (reuse existing function)
CREATE TRIGGER update_clients_updated_at 
    BEFORE UPDATE ON clients 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Add foreign key constraint to existing client_integration_configs
-- This ensures referential integrity between clients and their integrations
ALTER TABLE client_integration_configs 
ADD CONSTRAINT fk_client_integration_configs_client_id 
FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE CASCADE;

-- Grant permissions to the application user
GRANT ALL PRIVILEGES ON TABLE clients TO ddfelts;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO ddfelts;
