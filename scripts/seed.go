// +build ignore

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://regstrava:regstrava@localhost:5432/regstrava?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a test funder
	funderID := uuid.New()
	apiKey := "test-api-key-12345" // In production, use a secure random key

	apiKeyHash, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash API key: %v", err)
	}

	oauthClientID := "test-client-id"
	oauthSecret := "test-client-secret"
	oauthSecretHash, err := bcrypt.GenerateFromPassword([]byte(oauthSecret), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash OAuth secret: %v", err)
	}

	query := `
		INSERT INTO funders (id, name, api_key_hash, oauth_client_id, oauth_secret_hash,
		                     track_fundings, rate_limit_daily, rate_limit_monthly, created_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (oauth_client_id) DO UPDATE SET
			name = EXCLUDED.name,
			api_key_hash = EXCLUDED.api_key_hash
	`

	_, err = db.ExecContext(ctx, query,
		funderID,
		"Test Funder",
		string(apiKeyHash),
		oauthClientID,
		string(oauthSecretHash),
		true,  // track_fundings
		1000,  // rate_limit_daily
		20000, // rate_limit_monthly
		time.Now(),
		true, // is_active
	)

	if err != nil {
		log.Fatalf("Failed to create test funder: %v", err)
	}

	fmt.Println("Test funder created successfully!")
	fmt.Println()
	fmt.Println("=== API Key Authentication ===")
	fmt.Printf("API Key: %s\n", apiKey)
	fmt.Println("Header: X-API-Key: test-api-key-12345")
	fmt.Println()
	fmt.Println("=== OAuth Authentication ===")
	fmt.Printf("Client ID: %s\n", oauthClientID)
	fmt.Printf("Client Secret: %s\n", oauthSecret)
	fmt.Println()
	fmt.Println("Example token request:")
	fmt.Println(`curl -X POST http://localhost:8080/api/v1/oauth/token \
  -H "Content-Type: application/json" \
  -d '{"grant_type": "client_credentials", "client_id": "test-client-id", "client_secret": "test-client-secret"}'`)
}
