package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/regstrava/regstrava/internal/domain"
)

// UsageRepository handles usage record persistence
type UsageRepository struct {
	db *sql.DB
}

// NewUsageRepository creates a new usage repository
func NewUsageRepository(db *sql.DB) *UsageRepository {
	return &UsageRepository{db: db}
}

// GetOrCreateDailyRecord gets or creates a daily usage record for the current day
func (r *UsageRepository) GetOrCreateDailyRecord(ctx context.Context, funderID uuid.UUID) (*domain.UsageRecord, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	return r.getOrCreateRecord(ctx, funderID, domain.PeriodTypeDaily, today, tomorrow)
}

// GetOrCreateMonthlyRecord gets or creates a monthly usage record for the current month
func (r *UsageRepository) GetOrCreateMonthlyRecord(ctx context.Context, funderID uuid.UUID) (*domain.UsageRecord, error) {
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)

	return r.getOrCreateRecord(ctx, funderID, domain.PeriodTypeMonthly, monthStart, monthEnd)
}

func (r *UsageRepository) getOrCreateRecord(ctx context.Context, funderID uuid.UUID, periodType domain.PeriodType, periodStart, periodEnd time.Time) (*domain.UsageRecord, error) {
	// Try to find existing record
	query := `
		SELECT id, funder_id, period_type, period_start, period_end,
		       check_count, register_count, party_check_count, party_register_count,
		       created_at, updated_at
		FROM usage_records
		WHERE funder_id = $1 AND period_type = $2 AND period_start = $3
	`

	var record domain.UsageRecord
	err := r.db.QueryRowContext(ctx, query, funderID, periodType, periodStart.Format("2006-01-02")).Scan(
		&record.ID,
		&record.FunderID,
		&record.PeriodType,
		&record.PeriodStart,
		&record.PeriodEnd,
		&record.CheckCount,
		&record.RegisterCount,
		&record.PartyCheckCount,
		&record.PartyRegisterCount,
		&record.CreatedAt,
		&record.UpdatedAt,
	)

	if err == nil {
		return &record, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to find usage record: %w", err)
	}

	// Create new record
	record = domain.UsageRecord{
		ID:          uuid.New(),
		FunderID:    funderID,
		PeriodType:  periodType,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	insertQuery := `
		INSERT INTO usage_records (id, funder_id, period_type, period_start, period_end,
		                           check_count, register_count, party_check_count, party_register_count,
		                           created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (funder_id, period_type, period_start) DO NOTHING
		RETURNING id
	`

	err = r.db.QueryRowContext(ctx, insertQuery,
		record.ID,
		record.FunderID,
		record.PeriodType,
		record.PeriodStart.Format("2006-01-02"),
		record.PeriodEnd.Format("2006-01-02"),
		record.CheckCount,
		record.RegisterCount,
		record.PartyCheckCount,
		record.PartyRegisterCount,
		record.CreatedAt,
		record.UpdatedAt,
	).Scan(&record.ID)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to create usage record: %w", err)
	}

	// If we hit a conflict, re-fetch the existing record
	if err == sql.ErrNoRows {
		return r.getOrCreateRecord(ctx, funderID, periodType, periodStart, periodEnd)
	}

	return &record, nil
}

// IncrementUsage increments the usage counter for a specific type
func (r *UsageRepository) IncrementUsage(ctx context.Context, funderID uuid.UUID, usageType domain.UsageType) error {
	// Increment both daily and monthly records
	dailyRecord, err := r.GetOrCreateDailyRecord(ctx, funderID)
	if err != nil {
		return fmt.Errorf("failed to get daily record: %w", err)
	}

	monthlyRecord, err := r.GetOrCreateMonthlyRecord(ctx, funderID)
	if err != nil {
		return fmt.Errorf("failed to get monthly record: %w", err)
	}

	column := getUsageColumn(usageType)

	// Update daily record
	dailyQuery := fmt.Sprintf(`
		UPDATE usage_records
		SET %s = %s + 1, updated_at = NOW()
		WHERE id = $1
	`, column, column)

	_, err = r.db.ExecContext(ctx, dailyQuery, dailyRecord.ID)
	if err != nil {
		return fmt.Errorf("failed to increment daily usage: %w", err)
	}

	// Update monthly record
	monthlyQuery := fmt.Sprintf(`
		UPDATE usage_records
		SET %s = %s + 1, updated_at = NOW()
		WHERE id = $1
	`, column, column)

	_, err = r.db.ExecContext(ctx, monthlyQuery, monthlyRecord.ID)
	if err != nil {
		return fmt.Errorf("failed to increment monthly usage: %w", err)
	}

	return nil
}

func getUsageColumn(usageType domain.UsageType) string {
	switch usageType {
	case domain.UsageTypeCheck:
		return "check_count"
	case domain.UsageTypeRegister:
		return "register_count"
	case domain.UsageTypePartyCheck:
		return "party_check_count"
	case domain.UsageTypePartyRegister:
		return "party_register_count"
	default:
		return "check_count"
	}
}

// GetCurrentUsage gets the current usage for a funder
func (r *UsageRepository) GetCurrentUsage(ctx context.Context, funderID uuid.UUID) (*domain.UsageStats, error) {
	dailyRecord, err := r.GetOrCreateDailyRecord(ctx, funderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily usage: %w", err)
	}

	monthlyRecord, err := r.GetOrCreateMonthlyRecord(ctx, funderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly usage: %w", err)
	}

	stats := &domain.UsageStats{
		DailyChecks:           dailyRecord.CheckCount,
		DailyRegisters:        dailyRecord.RegisterCount,
		DailyPartyChecks:      dailyRecord.PartyCheckCount,
		DailyPartyRegisters:   dailyRecord.PartyRegisterCount,
		MonthlyChecks:         monthlyRecord.CheckCount,
		MonthlyRegisters:      monthlyRecord.RegisterCount,
		MonthlyPartyChecks:    monthlyRecord.PartyCheckCount,
		MonthlyPartyRegisters: monthlyRecord.PartyRegisterCount,
		DailyResetsAt:         dailyRecord.PeriodEnd,
		MonthlyResetsAt:       monthlyRecord.PeriodEnd,
	}

	return stats, nil
}

// GetUsageHistory gets historical usage for a funder
func (r *UsageRepository) GetUsageHistory(ctx context.Context, funderID uuid.UUID, months int) ([]*domain.UsageHistory, error) {
	query := `
		SELECT id, funder_id, year, month,
		       total_checks, total_registers, total_party_checks, total_party_registers,
		       peak_daily_checks, peak_daily_registers, quota_exceeded_count, created_at
		FROM usage_history
		WHERE funder_id = $1
		ORDER BY year DESC, month DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, funderID, months)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage history: %w", err)
	}
	defer rows.Close()

	var history []*domain.UsageHistory
	for rows.Next() {
		var h domain.UsageHistory
		err := rows.Scan(
			&h.ID,
			&h.FunderID,
			&h.Year,
			&h.Month,
			&h.TotalChecks,
			&h.TotalRegisters,
			&h.TotalPartyChecks,
			&h.TotalPartyRegisters,
			&h.PeakDailyChecks,
			&h.PeakDailyRegisters,
			&h.QuotaExceededCount,
			&h.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan usage history: %w", err)
		}
		history = append(history, &h)
	}

	return history, nil
}

// RecordQuotaExceeded records a quota exceeded event
func (r *UsageRepository) RecordQuotaExceeded(ctx context.Context, funderID uuid.UUID) error {
	now := time.Now().UTC()
	year := now.Year()
	month := int(now.Month())

	query := `
		INSERT INTO usage_history (id, funder_id, year, month, quota_exceeded_count, created_at)
		VALUES ($1, $2, $3, $4, 1, NOW())
		ON CONFLICT (funder_id, year, month)
		DO UPDATE SET quota_exceeded_count = usage_history.quota_exceeded_count + 1
	`

	_, err := r.db.ExecContext(ctx, query, uuid.New(), funderID, year, month)
	if err != nil {
		return fmt.Errorf("failed to record quota exceeded: %w", err)
	}

	return nil
}

// HasSentNotification checks if a notification has already been sent
func (r *UsageRepository) HasSentNotification(ctx context.Context, funderID uuid.UUID, notificationType domain.EmailNotificationType, periodStart time.Time) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM email_notifications
			WHERE funder_id = $1 AND notification_type = $2 AND period_start = $3
		)
	`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, funderID, notificationType, periodStart.Format("2006-01-02")).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check notification: %w", err)
	}

	return exists, nil
}

// RecordNotification records that a notification was sent
func (r *UsageRepository) RecordNotification(ctx context.Context, funderID uuid.UUID, notificationType domain.EmailNotificationType, periodStart time.Time) error {
	query := `
		INSERT INTO email_notifications (id, funder_id, notification_type, period_start, sent_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (funder_id, notification_type, period_start) DO NOTHING
	`

	_, err := r.db.ExecContext(ctx, query, uuid.New(), funderID, notificationType, periodStart.Format("2006-01-02"))
	if err != nil {
		return fmt.Errorf("failed to record notification: %w", err)
	}

	return nil
}
