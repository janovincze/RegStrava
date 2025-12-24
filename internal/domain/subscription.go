package domain

import (
	"time"

	"github.com/google/uuid"
)

// SubscriptionStatus represents the status of a subscription
type SubscriptionStatus string

const (
	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusTrial     SubscriptionStatus = "trial"
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"
	SubscriptionStatusExpired   SubscriptionStatus = "expired"
	SubscriptionStatusSuspended SubscriptionStatus = "suspended"
)

// SubscriptionTier represents a subscription tier with its limits
type SubscriptionTier struct {
	ID                   uuid.UUID `json:"id"`
	Name                 string    `json:"name"`
	DisplayName          string    `json:"display_name"`
	Description          string    `json:"description,omitempty"`
	CheckLimitDaily      *int      `json:"check_limit_daily,omitempty"`
	CheckLimitMonthly    *int      `json:"check_limit_monthly,omitempty"`
	RegisterLimitDaily   *int      `json:"register_limit_daily,omitempty"`
	RegisterLimitMonthly *int      `json:"register_limit_monthly,omitempty"`
	NotificationEmail    bool      `json:"notification_email"`
	NotificationWebhook  bool      `json:"notification_webhook"`
	NotificationPriority bool      `json:"notification_priority"`
	PartyQueryLimitDaily *int      `json:"party_query_limit_daily,omitempty"`
	PartyLookbackDays    int       `json:"party_lookback_days"`
	PriceMonthlyCents    int       `json:"price_monthly_cents"`
	PriceYearlyCents     int       `json:"price_yearly_cents"`
	DisplayOrder         int       `json:"display_order"`
	IsActive             bool      `json:"is_active"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// IsUnlimited checks if a limit is unlimited (nil)
func (t *SubscriptionTier) IsUnlimited(limitType string) bool {
	switch limitType {
	case "check_daily":
		return t.CheckLimitDaily == nil
	case "check_monthly":
		return t.CheckLimitMonthly == nil
	case "register_daily":
		return t.RegisterLimitDaily == nil
	case "register_monthly":
		return t.RegisterLimitMonthly == nil
	case "party_query_daily":
		return t.PartyQueryLimitDaily == nil
	default:
		return false
	}
}

// GetLimit returns the limit value or -1 if unlimited
func (t *SubscriptionTier) GetLimit(limitType string) int {
	switch limitType {
	case "check_daily":
		if t.CheckLimitDaily == nil {
			return -1
		}
		return *t.CheckLimitDaily
	case "check_monthly":
		if t.CheckLimitMonthly == nil {
			return -1
		}
		return *t.CheckLimitMonthly
	case "register_daily":
		if t.RegisterLimitDaily == nil {
			return -1
		}
		return *t.RegisterLimitDaily
	case "register_monthly":
		if t.RegisterLimitMonthly == nil {
			return -1
		}
		return *t.RegisterLimitMonthly
	case "party_query_daily":
		if t.PartyQueryLimitDaily == nil {
			return -1
		}
		return *t.PartyQueryLimitDaily
	default:
		return 0
	}
}

// SubscriptionTierResponse is the API response for subscription tiers
type SubscriptionTierResponse struct {
	Tiers []SubscriptionTier `json:"tiers"`
}

// FunderSubscription represents a funder's subscription details
type FunderSubscription struct {
	TierID      uuid.UUID          `json:"tier_id"`
	TierName    string             `json:"tier_name"`
	Tier        *SubscriptionTier  `json:"tier,omitempty"`
	Status      SubscriptionStatus `json:"status"`
	StartedAt   *time.Time         `json:"started_at,omitempty"`
	ExpiresAt   *time.Time         `json:"expires_at,omitempty"`
	TrialEndsAt *time.Time         `json:"trial_ends_at,omitempty"`
}

// SubscriptionUpgradeRequest is the request to upgrade subscription
type SubscriptionUpgradeRequest struct {
	TierName string `json:"tier_name"`
}

// SubscriptionUpgradeResponse is the response for upgrade request
type SubscriptionUpgradeResponse struct {
	Success    bool              `json:"success"`
	Message    string            `json:"message"`
	NewTier    *SubscriptionTier `json:"new_tier,omitempty"`
	UpgradeURL string            `json:"upgrade_url,omitempty"`
}
