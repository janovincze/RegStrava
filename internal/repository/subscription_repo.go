package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/regstrava/regstrava/internal/domain"
)

// SubscriptionRepository handles subscription tier persistence
type SubscriptionRepository struct {
	db *sql.DB
}

// NewSubscriptionRepository creates a new subscription repository
func NewSubscriptionRepository(db *sql.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

// FindByID finds a subscription tier by ID
func (r *SubscriptionRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.SubscriptionTier, error) {
	query := `
		SELECT id, name, display_name, description,
		       check_limit_daily, check_limit_monthly,
		       register_limit_daily, register_limit_monthly,
		       notification_email, notification_webhook, notification_priority,
		       party_query_limit_daily, party_lookback_days,
		       price_monthly_cents, price_yearly_cents,
		       display_order, is_active, created_at, updated_at
		FROM subscription_tiers
		WHERE id = $1
	`

	var tier domain.SubscriptionTier
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tier.ID,
		&tier.Name,
		&tier.DisplayName,
		&tier.Description,
		&tier.CheckLimitDaily,
		&tier.CheckLimitMonthly,
		&tier.RegisterLimitDaily,
		&tier.RegisterLimitMonthly,
		&tier.NotificationEmail,
		&tier.NotificationWebhook,
		&tier.NotificationPriority,
		&tier.PartyQueryLimitDaily,
		&tier.PartyLookbackDays,
		&tier.PriceMonthlyCents,
		&tier.PriceYearlyCents,
		&tier.DisplayOrder,
		&tier.IsActive,
		&tier.CreatedAt,
		&tier.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find subscription tier: %w", err)
	}

	return &tier, nil
}

// FindByName finds a subscription tier by name
func (r *SubscriptionRepository) FindByName(ctx context.Context, name string) (*domain.SubscriptionTier, error) {
	query := `
		SELECT id, name, display_name, description,
		       check_limit_daily, check_limit_monthly,
		       register_limit_daily, register_limit_monthly,
		       notification_email, notification_webhook, notification_priority,
		       party_query_limit_daily, party_lookback_days,
		       price_monthly_cents, price_yearly_cents,
		       display_order, is_active, created_at, updated_at
		FROM subscription_tiers
		WHERE name = $1
	`

	var tier domain.SubscriptionTier
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&tier.ID,
		&tier.Name,
		&tier.DisplayName,
		&tier.Description,
		&tier.CheckLimitDaily,
		&tier.CheckLimitMonthly,
		&tier.RegisterLimitDaily,
		&tier.RegisterLimitMonthly,
		&tier.NotificationEmail,
		&tier.NotificationWebhook,
		&tier.NotificationPriority,
		&tier.PartyQueryLimitDaily,
		&tier.PartyLookbackDays,
		&tier.PriceMonthlyCents,
		&tier.PriceYearlyCents,
		&tier.DisplayOrder,
		&tier.IsActive,
		&tier.CreatedAt,
		&tier.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find subscription tier: %w", err)
	}

	return &tier, nil
}

// ListActive returns all active subscription tiers ordered by display order
func (r *SubscriptionRepository) ListActive(ctx context.Context) ([]*domain.SubscriptionTier, error) {
	query := `
		SELECT id, name, display_name, description,
		       check_limit_daily, check_limit_monthly,
		       register_limit_daily, register_limit_monthly,
		       notification_email, notification_webhook, notification_priority,
		       party_query_limit_daily, party_lookback_days,
		       price_monthly_cents, price_yearly_cents,
		       display_order, is_active, created_at, updated_at
		FROM subscription_tiers
		WHERE is_active = true
		ORDER BY display_order ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscription tiers: %w", err)
	}
	defer rows.Close()

	var tiers []*domain.SubscriptionTier
	for rows.Next() {
		var tier domain.SubscriptionTier
		err := rows.Scan(
			&tier.ID,
			&tier.Name,
			&tier.DisplayName,
			&tier.Description,
			&tier.CheckLimitDaily,
			&tier.CheckLimitMonthly,
			&tier.RegisterLimitDaily,
			&tier.RegisterLimitMonthly,
			&tier.NotificationEmail,
			&tier.NotificationWebhook,
			&tier.NotificationPriority,
			&tier.PartyQueryLimitDaily,
			&tier.PartyLookbackDays,
			&tier.PriceMonthlyCents,
			&tier.PriceYearlyCents,
			&tier.DisplayOrder,
			&tier.IsActive,
			&tier.CreatedAt,
			&tier.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subscription tier: %w", err)
		}
		tiers = append(tiers, &tier)
	}

	return tiers, nil
}

// GetFunderTier gets the subscription tier for a funder
func (r *SubscriptionRepository) GetFunderTier(ctx context.Context, funderID uuid.UUID) (*domain.SubscriptionTier, error) {
	query := `
		SELECT t.id, t.name, t.display_name, t.description,
		       t.check_limit_daily, t.check_limit_monthly,
		       t.register_limit_daily, t.register_limit_monthly,
		       t.notification_email, t.notification_webhook, t.notification_priority,
		       t.party_query_limit_daily, t.party_lookback_days,
		       t.price_monthly_cents, t.price_yearly_cents,
		       t.display_order, t.is_active, t.created_at, t.updated_at
		FROM subscription_tiers t
		JOIN funders f ON f.subscription_tier_name = t.name
		WHERE f.id = $1
	`

	var tier domain.SubscriptionTier
	err := r.db.QueryRowContext(ctx, query, funderID).Scan(
		&tier.ID,
		&tier.Name,
		&tier.DisplayName,
		&tier.Description,
		&tier.CheckLimitDaily,
		&tier.CheckLimitMonthly,
		&tier.RegisterLimitDaily,
		&tier.RegisterLimitMonthly,
		&tier.NotificationEmail,
		&tier.NotificationWebhook,
		&tier.NotificationPriority,
		&tier.PartyQueryLimitDaily,
		&tier.PartyLookbackDays,
		&tier.PriceMonthlyCents,
		&tier.PriceYearlyCents,
		&tier.DisplayOrder,
		&tier.IsActive,
		&tier.CreatedAt,
		&tier.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Return free tier as default
		return r.FindByName(ctx, "free")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get funder tier: %w", err)
	}

	return &tier, nil
}

// GetDefaultTier returns the default (free) tier
func (r *SubscriptionRepository) GetDefaultTier(ctx context.Context) (*domain.SubscriptionTier, error) {
	return r.FindByName(ctx, "free")
}
