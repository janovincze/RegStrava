package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/regstrava/regstrava/internal/domain"
	"github.com/regstrava/regstrava/internal/repository"
)

// UsageService handles usage tracking and quota checking
type UsageService struct {
	usageRepo        *repository.UsageRepository
	subscriptionRepo *repository.SubscriptionRepository
}

// NewUsageService creates a new usage service
func NewUsageService(usageRepo *repository.UsageRepository, subscriptionRepo *repository.SubscriptionRepository) *UsageService {
	return &UsageService{
		usageRepo:        usageRepo,
		subscriptionRepo: subscriptionRepo,
	}
}

// CheckQuota checks if the funder has quota available for the given usage type
func (s *UsageService) CheckQuota(ctx context.Context, funderID uuid.UUID, usageType domain.UsageType) (*domain.QuotaExceededError, error) {
	// Get the funder's subscription tier
	tier, err := s.subscriptionRepo.GetFunderTier(ctx, funderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription tier: %w", err)
	}

	if tier == nil {
		// No tier, use default (free)
		tier, err = s.subscriptionRepo.GetDefaultTier(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get default tier: %w", err)
		}
	}

	// Get current usage
	usage, err := s.usageRepo.GetCurrentUsage(ctx, funderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current usage: %w", err)
	}

	// Check daily limits
	dailyLimit, dailyUsage := s.getDailyLimitAndUsage(tier, usage, usageType)
	if dailyLimit != nil && dailyUsage >= *dailyLimit {
		return &domain.QuotaExceededError{
			Error:        "quota_exceeded",
			Message:      fmt.Sprintf("Daily %s quota exceeded (%d/%d). Upgrade your plan for higher limits.", usageType, dailyUsage, *dailyLimit),
			QuotaType:    string(usageType),
			PeriodType:   "daily",
			CurrentUsage: dailyUsage,
			Limit:        *dailyLimit,
			ResetsAt:     usage.DailyResetsAt,
			UpgradeURL:   "/dashboard.html#subscription",
		}, nil
	}

	// Check monthly limits
	monthlyLimit, monthlyUsage := s.getMonthlyLimitAndUsage(tier, usage, usageType)
	if monthlyLimit != nil && monthlyUsage >= *monthlyLimit {
		return &domain.QuotaExceededError{
			Error:        "quota_exceeded",
			Message:      fmt.Sprintf("Monthly %s quota exceeded (%d/%d). Upgrade your plan for higher limits.", usageType, monthlyUsage, *monthlyLimit),
			QuotaType:    string(usageType),
			PeriodType:   "monthly",
			CurrentUsage: monthlyUsage,
			Limit:        *monthlyLimit,
			ResetsAt:     usage.MonthlyResetsAt,
			UpgradeURL:   "/dashboard.html#subscription",
		}, nil
	}

	return nil, nil
}

func (s *UsageService) getDailyLimitAndUsage(tier *domain.SubscriptionTier, usage *domain.UsageStats, usageType domain.UsageType) (*int, int) {
	switch usageType {
	case domain.UsageTypeCheck:
		return tier.CheckLimitDaily, usage.DailyChecks
	case domain.UsageTypeRegister:
		return tier.RegisterLimitDaily, usage.DailyRegisters
	case domain.UsageTypePartyCheck:
		return tier.PartyQueryLimitDaily, usage.DailyPartyChecks
	case domain.UsageTypePartyRegister:
		return tier.PartyQueryLimitDaily, usage.DailyPartyRegisters
	default:
		return nil, 0
	}
}

func (s *UsageService) getMonthlyLimitAndUsage(tier *domain.SubscriptionTier, usage *domain.UsageStats, usageType domain.UsageType) (*int, int) {
	switch usageType {
	case domain.UsageTypeCheck:
		return tier.CheckLimitMonthly, usage.MonthlyChecks
	case domain.UsageTypeRegister:
		return tier.RegisterLimitMonthly, usage.MonthlyRegisters
	case domain.UsageTypePartyCheck:
		return nil, usage.MonthlyPartyChecks // Party checks don't have monthly limit
	case domain.UsageTypePartyRegister:
		return nil, usage.MonthlyPartyRegisters
	default:
		return nil, 0
	}
}

// RecordUsage records usage for a funder
func (s *UsageService) RecordUsage(ctx context.Context, funderID uuid.UUID, usageType domain.UsageType) error {
	return s.usageRepo.IncrementUsage(ctx, funderID, usageType)
}

// GetUsageStats gets the current usage stats for a funder
func (s *UsageService) GetUsageStats(ctx context.Context, funderID uuid.UUID) (*domain.UsageResponse, error) {
	// Get the funder's subscription tier
	tier, err := s.subscriptionRepo.GetFunderTier(ctx, funderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription tier: %w", err)
	}

	if tier == nil {
		tier, err = s.subscriptionRepo.GetDefaultTier(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get default tier: %w", err)
		}
	}

	// Get current usage
	usage, err := s.usageRepo.GetCurrentUsage(ctx, funderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current usage: %w", err)
	}

	// Set limits from tier
	usage.DailyCheckLimit = tier.CheckLimitDaily
	usage.DailyRegisterLimit = tier.RegisterLimitDaily
	usage.MonthlyCheckLimit = tier.CheckLimitMonthly
	usage.MonthlyRegisterLimit = tier.RegisterLimitMonthly
	usage.PartyQueryLimitDaily = tier.PartyQueryLimitDaily

	// Calculate percentages
	usage.DailyCheckPercent = domain.CalculatePercentage(usage.DailyChecks, tier.CheckLimitDaily)
	usage.DailyRegisterPercent = domain.CalculatePercentage(usage.DailyRegisters, tier.RegisterLimitDaily)
	usage.MonthlyCheckPercent = domain.CalculatePercentage(usage.MonthlyChecks, tier.CheckLimitMonthly)
	usage.MonthlyRegisterPercent = domain.CalculatePercentage(usage.MonthlyRegisters, tier.RegisterLimitMonthly)

	// Determine warning level (use the highest percentage)
	maxPercent := max(usage.DailyCheckPercent, usage.DailyRegisterPercent, usage.MonthlyCheckPercent, usage.MonthlyRegisterPercent)
	warningLevel := domain.GetWarningLevel(maxPercent)

	var warningMessage string
	if warningLevel == "critical" {
		warningMessage = "You are at 90% of your quota. Consider upgrading to avoid service interruption."
	} else if warningLevel == "warning" {
		warningMessage = "You are at 80% of your quota. Consider upgrading for higher limits."
	}

	response := &domain.UsageResponse{
		Usage: *usage,
		Subscription: domain.FunderSubscription{
			TierID:   tier.ID,
			TierName: tier.Name,
			Tier:     tier,
			Status:   domain.SubscriptionStatusActive,
		},
		WarningLevel:   warningLevel,
		WarningMessage: warningMessage,
	}

	return response, nil
}

// GetUsageHistory gets historical usage for a funder
func (s *UsageService) GetUsageHistory(ctx context.Context, funderID uuid.UUID, months int) (*domain.UsageHistoryResponse, error) {
	history, err := s.usageRepo.GetUsageHistory(ctx, funderID, months)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage history: %w", err)
	}

	return &domain.UsageHistoryResponse{
		History: history,
	}, nil
}

// CheckAndSendWarnings checks if warnings need to be sent and marks them
func (s *UsageService) CheckAndSendWarnings(ctx context.Context, funderID uuid.UUID) (shouldSend80, shouldSend90 bool, err error) {
	// Get current usage stats
	stats, err := s.GetUsageStats(ctx, funderID)
	if err != nil {
		return false, false, err
	}

	maxPercent := max(
		stats.Usage.DailyCheckPercent,
		stats.Usage.DailyRegisterPercent,
		stats.Usage.MonthlyCheckPercent,
		stats.Usage.MonthlyRegisterPercent,
	)

	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Check 90% warning first
	if maxPercent >= 90 {
		sent, err := s.usageRepo.HasSentNotification(ctx, funderID, domain.EmailNotificationUsageWarning90, monthStart)
		if err != nil {
			return false, false, err
		}
		if !sent {
			err = s.usageRepo.RecordNotification(ctx, funderID, domain.EmailNotificationUsageWarning90, monthStart)
			if err != nil {
				return false, false, err
			}
			return false, true, nil
		}
	}

	// Check 80% warning
	if maxPercent >= 80 {
		sent, err := s.usageRepo.HasSentNotification(ctx, funderID, domain.EmailNotificationUsageWarning80, monthStart)
		if err != nil {
			return false, false, err
		}
		if !sent {
			err = s.usageRepo.RecordNotification(ctx, funderID, domain.EmailNotificationUsageWarning80, monthStart)
			if err != nil {
				return false, false, err
			}
			return true, false, nil
		}
	}

	return false, false, nil
}

func max(values ...float64) float64 {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values[1:] {
		if v > m {
			m = v
		}
	}
	return m
}
