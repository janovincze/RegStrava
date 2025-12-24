-- RegStrava Extended Hash Levels Migration
-- Adds document types, party-level hashing, and updated hash level structure

-- ============================================================================
-- DOCUMENT TYPES TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS document_types (
    code VARCHAR(10) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert default document types for factoring
INSERT INTO document_types (code, name, description) VALUES
    ('INV', 'Invoice', 'Standard commercial invoice for goods/services delivered'),
    ('CN', 'Credit Note', 'Adjustment reducing amount owed'),
    ('DN', 'Debit Note', 'Additional charges or adjustments'),
    ('BOE', 'Bill of Exchange', 'Negotiable instrument / trade bill'),
    ('PN', 'Promissory Note', 'Written promise to pay'),
    ('PO', 'Purchase Order', 'Confirmed purchase order (for PO financing)'),
    ('CR', 'Contract Receivable', 'Long-term contract-based receivable'),
    ('PB', 'Progress Billing', 'Milestone/progress payment for projects'),
    ('LR', 'Lease Receivable', 'Receivables from lease agreements'),
    ('TA', 'Trade Acceptance', 'Accepted trade document'),
    ('GRN', 'Goods Receipt Note', 'Proof of delivery document')
ON CONFLICT (code) DO NOTHING;

COMMENT ON TABLE document_types IS 'Configurable document types for factoring/financing';

-- ============================================================================
-- PARTY HASHES TABLE (L0 - Party Level)
-- ============================================================================

-- Party type enum
CREATE TYPE party_type AS ENUM ('buyer', 'supplier');

CREATE TABLE IF NOT EXISTS party_hashes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    hash_value VARCHAR(64) NOT NULL,
    party_type party_type NOT NULL,
    first_checked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    first_registered_at TIMESTAMP WITH TIME ZONE,
    last_checked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_registered_at TIMESTAMP WITH TIME ZONE,
    check_count INT DEFAULT 1,
    register_count INT DEFAULT 0,
    first_checker_id UUID REFERENCES funders(id) ON DELETE SET NULL,
    first_registerer_id UUID REFERENCES funders(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(hash_value, party_type)
);

CREATE INDEX IF NOT EXISTS idx_party_hashes_hash_value ON party_hashes(hash_value);
CREATE INDEX IF NOT EXISTS idx_party_hashes_party_type ON party_hashes(party_type);
CREATE INDEX IF NOT EXISTS idx_party_hashes_last_checked ON party_hashes(last_checked_at);
CREATE INDEX IF NOT EXISTS idx_party_hashes_last_registered ON party_hashes(last_registered_at);

COMMENT ON TABLE party_hashes IS 'Party-level (L0) hashes for buyer/supplier duplicate detection';
COMMENT ON COLUMN party_hashes.hash_value IS 'HMAC-SHA256 hash of normalized tax_id|country_code';
COMMENT ON COLUMN party_hashes.first_checker_id IS 'NULL if first checker did not consent to tracking';
COMMENT ON COLUMN party_hashes.first_registerer_id IS 'NULL if first registerer did not consent to tracking';

-- ============================================================================
-- UPDATE INVOICE HASHES TABLE
-- ============================================================================

-- Update hash_level constraint for new levels (0-3 instead of 1-4)
-- First drop the old constraint
ALTER TABLE invoice_hashes DROP CONSTRAINT IF EXISTS invoice_hashes_hash_level_check;

-- Add new constraint allowing 0-3
ALTER TABLE invoice_hashes ADD CONSTRAINT invoice_hashes_hash_level_check
    CHECK (hash_level BETWEEN 0 AND 3);

-- Add document_type column
ALTER TABLE invoice_hashes ADD COLUMN IF NOT EXISTS document_type VARCHAR(10)
    REFERENCES document_types(code) DEFAULT 'INV';

-- Update comment for new hash levels
COMMENT ON COLUMN invoice_hashes.hash_level IS 'Hash level: 0=Party, 1=DocType+Parties, 2=+DocID, 3=+Amount';

-- Index for document type
CREATE INDEX IF NOT EXISTS idx_invoice_hashes_document_type ON invoice_hashes(document_type);

-- ============================================================================
-- FUNDER SETTINGS FOR PARTY QUERIES
-- ============================================================================

-- Add party query configuration to funders
ALTER TABLE funders ADD COLUMN IF NOT EXISTS party_query_limit_daily INT DEFAULT 100;
ALTER TABLE funders ADD COLUMN IF NOT EXISTS party_lookback_days INT DEFAULT 30;
ALTER TABLE funders ADD COLUMN IF NOT EXISTS subscription_tier VARCHAR(20) DEFAULT 'basic';
ALTER TABLE funders ADD COLUMN IF NOT EXISTS notification_consent BOOLEAN DEFAULT false;

COMMENT ON COLUMN funders.party_query_limit_daily IS 'Max party history queries per day';
COMMENT ON COLUMN funders.party_lookback_days IS 'How many days back party history queries can look';
COMMENT ON COLUMN funders.subscription_tier IS 'Subscription tier: basic, premium, enterprise';
COMMENT ON COLUMN funders.notification_consent IS 'Whether funder consents to receive notifications';

-- ============================================================================
-- NOTIFICATIONS TABLE (Premium Feature)
-- ============================================================================

CREATE TYPE notification_type AS ENUM ('hash_checked', 'hash_registered', 'party_checked', 'party_registered');

CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    funder_id UUID NOT NULL REFERENCES funders(id) ON DELETE CASCADE,
    notification_type notification_type NOT NULL,
    hash_value VARCHAR(64),
    hash_level SMALLINT,
    party_type party_type,
    triggered_by_funder_id UUID REFERENCES funders(id) ON DELETE SET NULL,
    message TEXT,
    is_read BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_funder_id ON notifications(funder_id);
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at);
CREATE INDEX IF NOT EXISTS idx_notifications_is_read ON notifications(funder_id, is_read);

COMMENT ON TABLE notifications IS 'Notifications for premium subscribers when their hashes are checked/registered';
COMMENT ON COLUMN notifications.triggered_by_funder_id IS 'NULL if triggering funder did not consent to tracking';

-- ============================================================================
-- SUBSCRIPTION TIERS TABLE
-- ============================================================================

CREATE TABLE IF NOT EXISTS subscription_tiers (
    name VARCHAR(20) PRIMARY KEY,
    display_name VARCHAR(100) NOT NULL,
    max_daily_requests INT NOT NULL,
    max_monthly_requests INT NOT NULL,
    party_query_limit_daily INT NOT NULL,
    party_lookback_days INT NOT NULL,
    notifications_enabled BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

INSERT INTO subscription_tiers (name, display_name, max_daily_requests, max_monthly_requests, party_query_limit_daily, party_lookback_days, notifications_enabled) VALUES
    ('basic', 'Basic', 1000, 20000, 100, 30, false),
    ('premium', 'Premium', 5000, 100000, 500, 90, true),
    ('enterprise', 'Enterprise', 50000, 1000000, 5000, 365, true)
ON CONFLICT (name) DO NOTHING;

COMMENT ON TABLE subscription_tiers IS 'Subscription tier definitions with limits and features';
