package domain

import (
	"time"

	"github.com/google/uuid"
)

// Funder represents a funding company that uses the registry
type Funder struct {
	ID               uuid.UUID `json:"id" db:"id"`
	Name             string    `json:"name" db:"name"`
	Email            *string   `json:"email,omitempty" db:"email"`
	Company          *string   `json:"company,omitempty" db:"company"`
	APIKeyHash       string    `json:"-" db:"api_key_hash"`
	OAuthClientID    *string   `json:"oauth_client_id,omitempty" db:"oauth_client_id"`
	OAuthSecretHash  *string   `json:"-" db:"oauth_secret_hash"`
	TrackFundings    bool      `json:"track_fundings" db:"track_fundings"`
	RateLimitDaily   int       `json:"rate_limit_daily" db:"rate_limit_daily"`
	RateLimitMonthly int       `json:"rate_limit_monthly" db:"rate_limit_monthly"`
	// Subscription fields
	SubscriptionTierName  *string            `json:"subscription_tier_name,omitempty" db:"subscription_tier_name"`
	SubscriptionStatus    SubscriptionStatus `json:"subscription_status" db:"subscription_status"`
	SubscriptionStartedAt *time.Time         `json:"subscription_started_at,omitempty" db:"subscription_started_at"`
	SubscriptionExpiresAt *time.Time         `json:"subscription_expires_at,omitempty" db:"subscription_expires_at"`
	TrialEndsAt           *time.Time         `json:"trial_ends_at,omitempty" db:"trial_ends_at"`
	// Usage warning tracking
	UsageWarningSent80    bool       `json:"-" db:"usage_warning_sent_80"`
	UsageWarningSent90    bool       `json:"-" db:"usage_warning_sent_90"`
	LastUsageWarningReset *time.Time `json:"-" db:"last_usage_warning_reset"`
	NotificationConsent   bool       `json:"notification_consent" db:"notification_consent"`
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
	IsActive              bool       `json:"is_active" db:"is_active"`
}

// APIUsage tracks API usage for rate limiting
type APIUsage struct {
	ID           uuid.UUID `json:"id" db:"id"`
	FunderID     uuid.UUID `json:"funder_id" db:"funder_id"`
	Endpoint     string    `json:"endpoint" db:"endpoint"`
	RequestCount int       `json:"request_count" db:"request_count"`
	Date         time.Time `json:"date" db:"date"`
}

// TokenClaims represents JWT claims for OAuth tokens
type TokenClaims struct {
	FunderID   uuid.UUID `json:"funder_id"`
	FunderName string    `json:"funder_name"`
	ExpiresAt  int64     `json:"exp"`
	IssuedAt   int64     `json:"iat"`
}

// OAuthTokenRequest represents an OAuth token request
type OAuthTokenRequest struct {
	GrantType    string `json:"grant_type" validate:"required,eq=client_credentials"`
	ClientID     string `json:"client_id" validate:"required"`
	ClientSecret string `json:"client_secret" validate:"required"`
}

// OAuthTokenResponse represents an OAuth token response
type OAuthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}
