package domain

import (
	"time"

	"github.com/google/uuid"
)

// PeriodType represents the type of usage period
type PeriodType string

const (
	PeriodTypeDaily   PeriodType = "daily"
	PeriodTypeMonthly PeriodType = "monthly"
)

// UsageRecord represents API usage for a specific period
type UsageRecord struct {
	ID                 uuid.UUID  `json:"id"`
	FunderID           uuid.UUID  `json:"funder_id"`
	PeriodType         PeriodType `json:"period_type"`
	PeriodStart        time.Time  `json:"period_start"`
	PeriodEnd          time.Time  `json:"period_end"`
	CheckCount         int        `json:"check_count"`
	RegisterCount      int        `json:"register_count"`
	PartyCheckCount    int        `json:"party_check_count"`
	PartyRegisterCount int        `json:"party_register_count"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// UsageHistory represents aggregated monthly usage
type UsageHistory struct {
	ID                  uuid.UUID `json:"id"`
	FunderID            uuid.UUID `json:"funder_id"`
	Year                int       `json:"year"`
	Month               int       `json:"month"`
	TotalChecks         int       `json:"total_checks"`
	TotalRegisters      int       `json:"total_registers"`
	TotalPartyChecks    int       `json:"total_party_checks"`
	TotalPartyRegisters int       `json:"total_party_registers"`
	PeakDailyChecks     int       `json:"peak_daily_checks"`
	PeakDailyRegisters  int       `json:"peak_daily_registers"`
	QuotaExceededCount  int       `json:"quota_exceeded_count"`
	CreatedAt           time.Time `json:"created_at"`
}

// UsageType represents the type of API usage
type UsageType string

const (
	UsageTypeCheck         UsageType = "check"
	UsageTypeRegister      UsageType = "register"
	UsageTypePartyCheck    UsageType = "party_check"
	UsageTypePartyRegister UsageType = "party_register"
)

// UsageStats represents current usage statistics
type UsageStats struct {
	// Daily usage
	DailyChecks         int `json:"daily_checks"`
	DailyRegisters      int `json:"daily_registers"`
	DailyPartyChecks    int `json:"daily_party_checks"`
	DailyPartyRegisters int `json:"daily_party_registers"`

	// Monthly usage
	MonthlyChecks         int `json:"monthly_checks"`
	MonthlyRegisters      int `json:"monthly_registers"`
	MonthlyPartyChecks    int `json:"monthly_party_checks"`
	MonthlyPartyRegisters int `json:"monthly_party_registers"`

	// Limits (nil = unlimited)
	DailyCheckLimit      *int `json:"daily_check_limit,omitempty"`
	DailyRegisterLimit   *int `json:"daily_register_limit,omitempty"`
	MonthlyCheckLimit    *int `json:"monthly_check_limit,omitempty"`
	MonthlyRegisterLimit *int `json:"monthly_register_limit,omitempty"`
	PartyQueryLimitDaily *int `json:"party_query_limit_daily,omitempty"`

	// Percentages (for progress bars)
	DailyCheckPercent      float64 `json:"daily_check_percent"`
	DailyRegisterPercent   float64 `json:"daily_register_percent"`
	MonthlyCheckPercent    float64 `json:"monthly_check_percent"`
	MonthlyRegisterPercent float64 `json:"monthly_register_percent"`

	// Period info
	DailyResetsAt   time.Time `json:"daily_resets_at"`
	MonthlyResetsAt time.Time `json:"monthly_resets_at"`
}

// UsageResponse is the API response for usage stats
type UsageResponse struct {
	Usage          UsageStats         `json:"usage"`
	Subscription   FunderSubscription `json:"subscription"`
	WarningLevel   string             `json:"warning_level,omitempty"` // "", "warning", "critical"
	WarningMessage string             `json:"warning_message,omitempty"`
}

// QuotaExceededError represents a quota exceeded error
type QuotaExceededError struct {
	Error        string    `json:"error"`
	Message      string    `json:"message"`
	QuotaType    string    `json:"quota_type"`
	PeriodType   string    `json:"period_type"`
	CurrentUsage int       `json:"current_usage"`
	Limit        int       `json:"limit"`
	ResetsAt     time.Time `json:"resets_at"`
	UpgradeURL   string    `json:"upgrade_url"`
}

// CalculatePercentage calculates usage percentage, returns 0 for unlimited
func CalculatePercentage(usage int, limit *int) float64 {
	if limit == nil || *limit == 0 {
		return 0
	}
	return float64(usage) / float64(*limit) * 100
}

// GetWarningLevel returns the warning level based on percentage
func GetWarningLevel(percent float64) string {
	if percent >= 90 {
		return "critical"
	}
	if percent >= 80 {
		return "warning"
	}
	return ""
}

// UsageHistoryResponse is the API response for usage history
type UsageHistoryResponse struct {
	History []*UsageHistory `json:"history"`
}

// EmailNotificationType represents types of email notifications
type EmailNotificationType string

const (
	EmailNotificationUsageWarning80 EmailNotificationType = "usage_warning_80"
	EmailNotificationUsageWarning90 EmailNotificationType = "usage_warning_90"
	EmailNotificationQuotaExceeded  EmailNotificationType = "quota_exceeded"
)

// EmailNotification represents a sent email notification
type EmailNotification struct {
	ID               uuid.UUID             `json:"id"`
	FunderID         uuid.UUID             `json:"funder_id"`
	NotificationType EmailNotificationType `json:"notification_type"`
	PeriodStart      time.Time             `json:"period_start"`
	SentAt           time.Time             `json:"sent_at"`
}
