package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/regstrava/regstrava/internal/domain"
	"github.com/regstrava/regstrava/internal/service"
)

// QuotaMiddleware provides quota enforcement middleware
type QuotaMiddleware struct {
	usageService *service.UsageService
}

// NewQuotaMiddleware creates a new quota middleware
func NewQuotaMiddleware(usageService *service.UsageService) *QuotaMiddleware {
	return &QuotaMiddleware{
		usageService: usageService,
	}
}

// EnforceQuota checks and enforces quotas before processing requests
func (m *QuotaMiddleware) EnforceQuota(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get funder ID from context
		funderID := GetFunderID(r.Context())
		if funderID == nil {
			// No funder ID means auth middleware should have rejected this
			next.ServeHTTP(w, r)
			return
		}

		// Determine usage type based on the request path and method
		usageType := getUsageType(r)
		if usageType == "" {
			// Not a metered endpoint, proceed
			next.ServeHTTP(w, r)
			return
		}

		// Check quota
		quotaError, err := m.usageService.CheckQuota(r.Context(), *funderID, usageType)
		if err != nil {
			http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
			return
		}

		if quotaError != nil {
			// Record the quota exceeded event
			_ = m.usageService.RecordUsage(r.Context(), *funderID, usageType)

			// Return quota exceeded error
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(quotaError)
			return
		}

		// Record usage after successful quota check
		// We record before processing to count the request
		_ = m.usageService.RecordUsage(r.Context(), *funderID, usageType)

		// Check if we need to send warning emails (async, don't block request)
		go func() {
			_, _, _ = m.usageService.CheckAndSendWarnings(r.Context(), *funderID)
		}()

		next.ServeHTTP(w, r)
	})
}

// getUsageType determines the usage type based on the request
func getUsageType(r *http.Request) domain.UsageType {
	path := r.URL.Path
	method := r.Method

	// Only POST requests for check/register endpoints are metered
	if method != http.MethodPost {
		return ""
	}

	// Invoice endpoints
	if strings.Contains(path, "/invoices/check") {
		return domain.UsageTypeCheck
	}
	if strings.Contains(path, "/invoices/register") {
		return domain.UsageTypeRegister
	}

	// Party endpoints
	if strings.Contains(path, "/party/check") {
		return domain.UsageTypePartyCheck
	}
	if strings.Contains(path, "/party/register") {
		return domain.UsageTypePartyRegister
	}

	return ""
}
