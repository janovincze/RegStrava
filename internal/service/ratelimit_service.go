package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RateLimitService handles rate limiting using Redis
type RateLimitService struct {
	client *redis.Client
}

// NewRateLimitService creates a new rate limit service
func NewRateLimitService(redisURL string) (*RateLimitService, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RateLimitService{client: client}, nil
}

// RateLimitResult contains the result of a rate limit check
type RateLimitResult struct {
	Allowed        bool
	DailyUsed      int
	DailyLimit     int
	MonthlyUsed    int
	MonthlyLimit   int
	RetryAfterSecs int
}

// CheckAndIncrement checks if the request is within rate limits and increments counters
func (s *RateLimitService) CheckAndIncrement(ctx context.Context, funderID uuid.UUID, dailyLimit, monthlyLimit int) (*RateLimitResult, error) {
	now := time.Now()
	dailyKey := fmt.Sprintf("ratelimit:daily:%s:%s", funderID.String(), now.Format("2006-01-02"))
	monthlyKey := fmt.Sprintf("ratelimit:monthly:%s:%s", funderID.String(), now.Format("2006-01"))

	// Get current counts
	dailyCount, err := s.client.Get(ctx, dailyKey).Int()
	if err != nil && err != redis.Nil {
		return nil, err
	}

	monthlyCount, err := s.client.Get(ctx, monthlyKey).Int()
	if err != nil && err != redis.Nil {
		return nil, err
	}

	result := &RateLimitResult{
		DailyUsed:    dailyCount,
		DailyLimit:   dailyLimit,
		MonthlyUsed:  monthlyCount,
		MonthlyLimit: monthlyLimit,
	}

	// Check limits
	if dailyCount >= dailyLimit {
		// Calculate seconds until midnight
		tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		result.RetryAfterSecs = int(tomorrow.Sub(now).Seconds())
		result.Allowed = false
		return result, nil
	}

	if monthlyCount >= monthlyLimit {
		// Calculate seconds until next month
		nextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
		result.RetryAfterSecs = int(nextMonth.Sub(now).Seconds())
		result.Allowed = false
		return result, nil
	}

	// Increment counters
	pipe := s.client.Pipeline()

	// Daily counter with expiry at end of day
	pipe.Incr(ctx, dailyKey)
	tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	pipe.ExpireAt(ctx, dailyKey, tomorrow)

	// Monthly counter with expiry at end of month
	pipe.Incr(ctx, monthlyKey)
	nextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	pipe.ExpireAt(ctx, monthlyKey, nextMonth)

	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}

	result.Allowed = true
	result.DailyUsed++
	result.MonthlyUsed++

	return result, nil
}

// GetUsage gets current usage without incrementing
func (s *RateLimitService) GetUsage(ctx context.Context, funderID uuid.UUID) (daily int, monthly int, err error) {
	now := time.Now()
	dailyKey := fmt.Sprintf("ratelimit:daily:%s:%s", funderID.String(), now.Format("2006-01-02"))
	monthlyKey := fmt.Sprintf("ratelimit:monthly:%s:%s", funderID.String(), now.Format("2006-01"))

	dailyStr, err := s.client.Get(ctx, dailyKey).Result()
	if err != nil && err != redis.Nil {
		return 0, 0, err
	}
	if dailyStr != "" {
		daily, _ = strconv.Atoi(dailyStr)
	}

	monthlyStr, err := s.client.Get(ctx, monthlyKey).Result()
	if err != nil && err != redis.Nil {
		return 0, 0, err
	}
	if monthlyStr != "" {
		monthly, _ = strconv.Atoi(monthlyStr)
	}

	return daily, monthly, nil
}

// Close closes the Redis connection
func (s *RateLimitService) Close() error {
	return s.client.Close()
}
