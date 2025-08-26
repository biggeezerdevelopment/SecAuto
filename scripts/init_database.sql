-- SecAuto Database Initialization Script
-- This script creates the basic database structure for SecAuto
-- 
-- Usage: psql -U postgres -f init_database.sql
--
-- Make sure to modify the database name, user, and password as needed

-- Create database user
CREATE USER ddfelts WITH PASSWORD 'your_password_here';

-- Create database
CREATE DATABASE soar_auto OWNER ddfelts;

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE soar_auto TO ddfelts;

-- Connect to the new database
\c soar_auto

-- Enable UUID extension (required for migrations)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create client_integration_configs table (from migration 001)
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

-- Create trigger function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger
CREATE TRIGGER update_client_integration_configs_updated_at 
    BEFORE UPDATE ON client_integration_configs 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Grant permissions on all tables to user
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO ddfelts;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO ddfelts;
GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO ddfelts;

-- Set default privileges for future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO ddfelts;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO ddfelts;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON FUNCTIONS TO ddfelts;