package api

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/regstrava/regstrava/internal/api/handlers"
	"github.com/regstrava/regstrava/internal/api/middleware"
	"github.com/regstrava/regstrava/internal/repository"
	"github.com/regstrava/regstrava/internal/service"
)

// NewRouter creates and configures the HTTP router
func NewRouter(
	invoiceService *service.InvoiceService,
	partyService *service.PartyService,
	authService *service.AuthService,
	hashService *service.HashService,
	rateLimitService *service.RateLimitService,
	usageService *service.UsageService,
	funderRepo *repository.FunderRepository,
	docTypeRepo *repository.DocumentTypeRepository,
	subscriptionRepo *repository.SubscriptionRepository,
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
	partyHandler := handlers.NewPartyHandler(partyService)
	authHandler := handlers.NewAuthHandler(authService)
	funderHandler := handlers.NewFunderHandler(funderRepo)
	docTypeHandler := handlers.NewDocumentTypeHandler(docTypeRepo)
	subscriptionHandler := handlers.NewSubscriptionHandler(subscriptionRepo, usageService)

	// Create middleware
	authMiddleware := middleware.NewAuthMiddleware(authService)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(rateLimitService, funderRepo)
	quotaMiddleware := middleware.NewQuotaMiddleware(usageService)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public endpoints (no auth required)
		r.Post("/oauth/token", authHandler.Token)
		r.Post("/funders/register", funderHandler.Register)
		r.Get("/subscription-tiers", subscriptionHandler.ListTiers)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Use(rateLimitMiddleware.RateLimit)
			r.Use(quotaMiddleware.EnforceQuota)

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

			// Funder profile endpoints
			r.Route("/funders", func(r chi.Router) {
				r.Get("/me", funderHandler.GetProfile)
				r.Patch("/me", funderHandler.UpdateProfile)
				r.Post("/me/regenerate-api-key", funderHandler.RegenerateAPIKey)
				r.Get("/me/usage", subscriptionHandler.GetUsage)
				r.Get("/me/usage/history", subscriptionHandler.GetUsageHistory)
				r.Get("/me/subscription", subscriptionHandler.GetSubscription)
				r.Post("/me/subscription/upgrade", subscriptionHandler.RequestUpgrade)
			})

			// Party endpoints (L0 - party level checks)
			r.Route("/party", func(r chi.Router) {
				r.Post("/check", partyHandler.Check)
				r.Post("/register", partyHandler.Register)
				r.Post("/history", partyHandler.History)
			})

			// Document types endpoint
			r.Get("/document-types", docTypeHandler.List)
		})
	})

	// Serve static files for the landing page
	staticDir := getStaticDir()
	staticHandler := newStaticHandler(staticDir)

	// Serve specific static routes
	r.Get("/", staticHandler.serveIndex)
	r.Get("/dashboard.html", staticHandler.serveFile)
	r.Get("/css/*", staticHandler.serveFile)
	r.Get("/js/*", staticHandler.serveFile)

	return r
}

// getStaticDir returns the path to the static files directory
func getStaticDir() string {
	// Check for STATIC_DIR environment variable
	if dir := os.Getenv("STATIC_DIR"); dir != "" {
		return dir
	}

	// Default to ./web/static relative to working directory
	return "./web/static"
}

// staticHandler serves static files
type staticHandler struct {
	dir string
}

func newStaticHandler(dir string) *staticHandler {
	return &staticHandler{dir: dir}
}

func (h *staticHandler) serveIndex(w http.ResponseWriter, r *http.Request) {
	h.serveStaticFile(w, r, "index.html")
}

func (h *staticHandler) serveFile(w http.ResponseWriter, r *http.Request) {
	// Get the path after the base
	filePath := strings.TrimPrefix(r.URL.Path, "/")
	h.serveStaticFile(w, r, filePath)
}

func (h *staticHandler) serveStaticFile(w http.ResponseWriter, r *http.Request, filePath string) {
	// Clean and join paths
	cleanPath := filepath.Clean(filePath)

	// Security: prevent path traversal
	if strings.Contains(cleanPath, "..") {
		http.NotFound(w, r)
		return
	}

	fullPath := filepath.Join(h.dir, cleanPath)

	file, err := os.Open(fullPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	// Get file info for content type
	stat, err := file.Stat()
	if err != nil || stat.IsDir() {
		http.NotFound(w, r)
		return
	}

	// Set content type based on extension
	ext := filepath.Ext(filePath)
	switch ext {
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case ".json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".ico":
		w.Header().Set("Content-Type", "image/x-icon")
	}

	// Serve the file
	io.Copy(w, file)
}
