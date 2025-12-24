-- Migration: 003_subscription_tiers.sql
-- Description: Enhance subscription tiers and add usage tracking

-- Add missing columns to existing subscription_tiers table
ALTER TABLE subscription_tiers ADD COLUMN IF NOT EXISTS id UUID DEFAULT gen_random_uuid();
ALTER TABLE subscription_tiers ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE subscription_tiers ADD COLUMN IF NOT EXISTS check_limit_daily INTEGER;
ALTER TABLE subscription_tiers ADD COLUMN IF NOT EXISTS check_limit_monthly INTEGER;
ALTER TABLE subscription_tiers ADD COLUMN IF NOT EXISTS register_limit_daily INTEGER;
ALTER TABLE subscription_tiers ADD COLUMN IF NOT EXISTS register_limit_monthly INTEGER;
ALTER TABLE subscription_tiers ADD COLUMN IF NOT EXISTS notification_email BOOLEAN DEFAULT false;
ALTER TABLE subscription_tiers ADD COLUMN IF NOT EXISTS notification_webhook BOOLEAN DEFAULT false;
ALTER TABLE subscription_tiers ADD COLUMN IF NOT EXISTS notification_priority BOOLEAN DEFAULT false;
ALTER TABLE subscription_tiers ADD COLUMN IF NOT EXISTS price_monthly_cents INTEGER DEFAULT 0;
ALTER TABLE subscription_tiers ADD COLUMN IF NOT EXISTS price_yearly_cents INTEGER DEFAULT 0;
ALTER TABLE subscription_tiers ADD COLUMN IF NOT EXISTS display_order INTEGER DEFAULT 0;
ALTER TABLE subscription_tiers ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;
ALTER TABLE subscription_tiers ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();

-- Update existing tiers with new data
UPDATE subscription_tiers SET
    description = 'Get started with basic invoice checking',
    check_limit_daily = max_daily_requests,
    check_limit_monthly = max_monthly_requests,
    register_limit_daily = CASE WHEN name = 'free' THEN 2 WHEN name = 'basic' THEN 20 WHEN name = 'premium' THEN 100 ELSE NULL END,
    register_limit_monthly = CASE WHEN name = 'free' THEN 10 WHEN name = 'basic' THEN 100 WHEN name = 'premium' THEN 1000 ELSE NULL END,
    notification_email = CASE WHEN name IN ('basic', 'premium', 'enterprise') THEN true ELSE false END,
    notification_webhook = CASE WHEN name IN ('premium', 'enterprise') THEN true ELSE false END,
    notification_priority = CASE WHEN name = 'enterprise' THEN true ELSE false END,
    price_monthly_cents = CASE WHEN name = 'free' THEN 0 WHEN name = 'basic' THEN 2900 WHEN name = 'premium' THEN 9900 ELSE 0 END,
    display_order = CASE WHEN name = 'free' THEN 1 WHEN name = 'basic' THEN 2 WHEN name = 'premium' THEN 3 WHEN name = 'enterprise' THEN 4 ELSE 0 END,
    is_active = true
WHERE name IN ('free', 'basic', 'premium', 'enterprise');

-- Update descriptions
UPDATE subscription_tiers SET description = 'Get started with basic invoice checking' WHERE name = 'free';
UPDATE subscription_tiers SET description = 'For small factoring operations' WHERE name = 'basic';
UPDATE subscription_tiers SET description = 'For growing factoring businesses' WHERE name = 'premium';
UPDATE subscription_tiers SET description = 'Unlimited access with priority support' WHERE name = 'enterprise';

-- Add subscription fields to funders table
ALTER TABLE funders ADD COLUMN IF NOT EXISTS subscription_tier_name VARCHAR(20) REFERENCES subscription_tiers(name);
ALTER TABLE funders ADD COLUMN IF NOT EXISTS subscription_status VARCHAR(20) DEFAULT 'active';
ALTER TABLE funders ADD COLUMN IF NOT EXISTS subscription_started_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE funders ADD COLUMN IF NOT EXISTS subscription_expires_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE funders ADD COLUMN IF NOT EXISTS trial_ends_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE funders ADD COLUMN IF NOT EXISTS email VARCHAR(255);
ALTER TABLE funders ADD COLUMN IF NOT EXISTS company VARCHAR(255);
ALTER TABLE funders ADD COLUMN IF NOT EXISTS usage_warning_sent_80 BOOLEAN DEFAULT false;
ALTER TABLE funders ADD COLUMN IF NOT EXISTS usage_warning_sent_90 BOOLEAN DEFAULT false;
ALTER TABLE funders ADD COLUMN IF NOT EXISTS last_usage_warning_reset TIMESTAMP WITH TIME ZONE;

-- Set default tier (free) for existing funders
UPDATE funders
SET subscription_tier_name = 'free',
    subscription_status = 'active',
    subscription_started_at = created_at
WHERE subscription_tier_name IS NULL;

-- Usage records table for tracking API usage
CREATE TABLE IF NOT EXISTS usage_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    funder_id UUID NOT NULL REFERENCES funders(id) ON DELETE CASCADE,
    -- Period tracking
    period_type VARCHAR(10) NOT NULL,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    -- Usage counts
    check_count INTEGER NOT NULL DEFAULT 0,
    register_count INTEGER NOT NULL DEFAULT 0,
    party_check_count INTEGER NOT NULL DEFAULT 0,
    party_register_count INTEGER NOT NULL DEFAULT 0,
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    -- Ensure one record per funder per period
    UNIQUE(funder_id, period_type, period_start)
);

-- Usage history for reporting
CREATE TABLE IF NOT EXISTS usage_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    funder_id UUID NOT NULL REFERENCES funders(id) ON DELETE CASCADE,
    year INTEGER NOT NULL,
    month INTEGER NOT NULL,
    total_checks INTEGER NOT NULL DEFAULT 0,
    total_registers INTEGER NOT NULL DEFAULT 0,
    total_party_checks INTEGER NOT NULL DEFAULT 0,
    total_party_registers INTEGER NOT NULL DEFAULT 0,
    peak_daily_checks INTEGER DEFAULT 0,
    peak_daily_registers INTEGER DEFAULT 0,
    quota_exceeded_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(funder_id, year, month)
);

-- Email notification log
CREATE TABLE IF NOT EXISTS email_notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    funder_id UUID NOT NULL REFERENCES funders(id) ON DELETE CASCADE,
    notification_type VARCHAR(50) NOT NULL,
    period_start DATE NOT NULL,
    sent_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(funder_id, notification_type, period_start)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_usage_records_funder_period ON usage_records(funder_id, period_type, period_start);
CREATE INDEX IF NOT EXISTS idx_usage_records_period_start ON usage_records(period_start);
CREATE INDEX IF NOT EXISTS idx_usage_history_funder ON usage_history(funder_id, year, month);
CREATE INDEX IF NOT EXISTS idx_email_notifications_funder ON email_notifications(funder_id, notification_type);
CREATE INDEX IF NOT EXISTS idx_funders_subscription ON funders(subscription_tier_name, subscription_status);

-- Add check constraints
ALTER TABLE funders DROP CONSTRAINT IF EXISTS funders_subscription_status_check;
ALTER TABLE funders ADD CONSTRAINT funders_subscription_status_check
    CHECK (subscription_status IS NULL OR subscription_status IN ('active', 'trial', 'cancelled', 'expired', 'suspended'));

ALTER TABLE usage_records DROP CONSTRAINT IF EXISTS usage_records_period_type_check;
ALTER TABLE usage_records ADD CONSTRAINT usage_records_period_type_check
    CHECK (period_type IN ('daily', 'monthly'));
