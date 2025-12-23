package api

import (
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
	funderHandler := handlers.NewFunderHandler(funderRepo)

	// Create middleware
	authMiddleware := middleware.NewAuthMiddleware(authService)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(rateLimitService, funderRepo)

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public endpoints (no auth required)
		r.Post("/oauth/token", authHandler.Token)
		r.Post("/funders/register", funderHandler.Register)

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

			// Funder profile endpoints
			r.Route("/funders", func(r chi.Router) {
				r.Get("/me", funderHandler.GetProfile)
				r.Patch("/me", funderHandler.UpdateProfile)
				r.Post("/me/regenerate-api-key", funderHandler.RegenerateAPIKey)
				r.Get("/me/usage", funderHandler.GetUsageStats)
			})
		})
	})

	// Serve static files for the landing page
	staticDir := getStaticDir()
	fileServer(r, "/", http.Dir(staticDir))

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

// fileServer conveniently sets up a http.FileServer handler to serve static files
func fileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))

		// Try to serve the file
		filePath := strings.TrimPrefix(r.URL.Path, pathPrefix)
		if filePath == "" || filePath == "/" {
			filePath = "/index.html"
		}

		// Check if file exists
		f, err := root.Open(filepath.Clean(filePath))
		if err != nil {
			// If file doesn't exist, serve index.html (for SPA routing)
			r.URL.Path = pathPrefix + "/index.html"
		} else {
			f.Close()
		}

		fs.ServeHTTP(w, r)
	})
}
