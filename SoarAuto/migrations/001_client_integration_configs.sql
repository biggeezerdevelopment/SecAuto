-- Migration: Create client_integration_configs table for multi-tenant integration storage
-- This replaces the file-based storage system with a scalable database solution

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS client_integration_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id VARCHAR(255) NOT NULL,
    integration_name VARCHAR(255) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    config_encrypted TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(client_id, integration_name)
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_client_integrations ON client_integration_configs(client_id);
CREATE INDEX IF NOT EXISTS idx_integration_name ON client_integration_configs(integration_name);
CREATE INDEX IF NOT EXISTS idx_enabled_integrations ON client_integration_configs(client_id, enabled) WHERE enabled = true;

-- Create trigger to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_client_integration_configs_updated_at 
    BEFORE UPDATE ON client_integration_configs 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();