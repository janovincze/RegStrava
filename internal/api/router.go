package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/regstrava/regstrava/internal/api/handlers"
	"github.com/regstrava/regstrava/internal/api/middleware"
	"github.com/regstrava/regstrava/internal/repository"
	"github.com/regstrava/regstrava/internal/service"
)

// NewRouter creates and configures the HTTP router
func NewRouter(
	invoiceService *service.InvoiceService,
	authService *service.AuthService,
	hashService *service.HashService,
	rateLimitService *service.RateLimitService,
	funderRepo *repository.FunderRepository,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logging)
	r.Use(middleware.RequestID)
	r.Use(middleware.CORS)

	// Health checks (no auth required)
	r.Get("/health", handlers.Health)
	r.Get("/ready", handlers.Ready)

	// Create handlers
	invoiceHandler := handlers.NewInvoiceHandler(invoiceService, hashService)
	authHandler := handlers.NewAuthHandler(authService)

	// Create middleware
	authMiddleware := middleware.NewAuthMiddleware(authService)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(rateLimitService, funderRepo)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// OAuth token endpoint (no auth required)
		r.Post("/oauth/token", authHandler.Token)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Use(rateLimitMiddleware.RateLimit)

			// Invoice endpoints
			r.Route("/invoices", func(r chi.Router) {
				// Check endpoints
				r.Post("/check", invoiceHandler.Check)
				r.Post("/check-raw", invoiceHandler.CheckRaw)

				// Register endpoints
				r.Post("/register", invoiceHandler.Register)
				r.Post("/register-raw", invoiceHandler.RegisterRaw)

				// Unregister endpoint
				r.Delete("/{hash}", invoiceHandler.Unregister)
			})
		})
	})

	return r
}
