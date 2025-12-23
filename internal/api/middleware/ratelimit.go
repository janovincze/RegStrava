package middleware

import (
	"fmt"
	"net/http"

	"github.com/regstrava/regstrava/internal/repository"
	"github.com/regstrava/regstrava/internal/service"
)

// RateLimitMiddleware provides rate limiting middleware
type RateLimitMiddleware struct {
	rateLimitService *service.RateLimitService
	funderRepo       *repository.FunderRepository
}

// NewRateLimitMiddleware creates a new rate limit middleware
func NewRateLimitMiddleware(rateLimitService *service.RateLimitService, funderRepo *repository.FunderRepository) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		rateLimitService: rateLimitService,
		funderRepo:       funderRepo,
	}
}

// RateLimit checks and enforces rate limits
func (m *RateLimitMiddleware) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		funderID := GetFunderID(r.Context())
		if funderID == nil {
			// No funder ID means unauthenticated - let auth middleware handle it
			next.ServeHTTP(w, r)
			return
		}

		// Get funder's rate limits
		funder, err := m.funderRepo.FindByID(r.Context(), *funderID)
		if err != nil || funder == nil {
			http.Error(w, `{"error": "Funder not found"}`, http.StatusUnauthorized)
			return
		}

		// Check rate limit
		result, err := m.rateLimitService.CheckAndIncrement(
			r.Context(),
			*funderID,
			funder.RateLimitDaily,
			funder.RateLimitMonthly,
		)

		if err != nil {
			// Log error but don't block request
			// In production, you might want to be more strict
			next.ServeHTTP(w, r)
			return
		}

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Daily-Limit", fmt.Sprintf("%d", result.DailyLimit))
		w.Header().Set("X-RateLimit-Daily-Used", fmt.Sprintf("%d", result.DailyUsed))
		w.Header().Set("X-RateLimit-Monthly-Limit", fmt.Sprintf("%d", result.MonthlyLimit))
		w.Header().Set("X-RateLimit-Monthly-Used", fmt.Sprintf("%d", result.MonthlyUsed))

		if !result.Allowed {
			w.Header().Set("Retry-After", fmt.Sprintf("%d", result.RetryAfterSecs))
			http.Error(w, `{"error": "Rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
