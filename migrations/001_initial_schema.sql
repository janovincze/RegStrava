-- RegStrava Initial Schema
-- Invoice Funding Registry

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Funders table (API consumers)
CREATE TABLE IF NOT EXISTS funders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    api_key_hash VARCHAR(255) NOT NULL,
    oauth_client_id VARCHAR(255) UNIQUE,
    oauth_secret_hash VARCHAR(255),
    track_fundings BOOLEAN DEFAULT false,
    rate_limit_daily INT DEFAULT 1000,
    rate_limit_monthly INT DEFAULT 20000,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true
);

-- Invoice hashes table (core registry)
CREATE TABLE IF NOT EXISTS invoice_hashes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    hash_value VARCHAR(64) NOT NULL UNIQUE,
    hash_level SMALLINT NOT NULL CHECK (hash_level BETWEEN 1 AND 4),
    funded_at TIMESTAMP WITH TIME ZONE NOT NULL,
    funder_id UUID REFERENCES funders(id) ON DELETE SET NULL,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- API usage table (for rate limiting audit)
CREATE TABLE IF NOT EXISTS api_usage (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    funder_id UUID REFERENCES funders(id) ON DELETE CASCADE,
    endpoint VARCHAR(100) NOT NULL,
    request_count INT DEFAULT 1,
    date DATE NOT NULL,
    UNIQUE(funder_id, endpoint, date)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_invoice_hashes_hash_value ON invoice_hashes(hash_value);
CREATE INDEX IF NOT EXISTS idx_invoice_hashes_funder_id ON invoice_hashes(funder_id);
CREATE INDEX IF NOT EXISTS idx_invoice_hashes_expires_at ON invoice_hashes(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_invoice_hashes_created_at ON invoice_hashes(created_at);
CREATE INDEX IF NOT EXISTS idx_funders_oauth_client_id ON funders(oauth_client_id) WHERE oauth_client_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_funders_is_active ON funders(is_active);
CREATE INDEX IF NOT EXISTS idx_api_usage_funder_date ON api_usage(funder_id, date);

-- Comments for documentation
COMMENT ON TABLE funders IS 'Funding companies that use the invoice registry';
COMMENT ON TABLE invoice_hashes IS 'Anonymized invoice hashes - core registry';
COMMENT ON TABLE api_usage IS 'API usage tracking for rate limiting and auditing';

COMMENT ON COLUMN invoice_hashes.hash_level IS 'Hash level: 1=Basic, 2=Standard, 3=Dated, 4=Full';
COMMENT ON COLUMN invoice_hashes.funder_id IS 'NULL if funder did not consent to tracking';
COMMENT ON COLUMN funders.track_fundings IS 'Whether funder consents to having their fundings tracked';
