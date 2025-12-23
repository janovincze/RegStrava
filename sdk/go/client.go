// Package regstrava provides a Go client for the RegStrava API
package regstrava

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the RegStrava API client
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	hasher     *Hasher
}

// ClientOption configures the client
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithHasher sets a custom hasher for client-side hashing
func WithHasher(hmacKey string) ClientOption {
	return func(c *Client) {
		c.hasher = NewHasher(hmacKey)
	}
}

// NewClient creates a new RegStrava client
func NewClient(baseURL, apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// CheckRequest represents a check request
type CheckRequest struct {
	Hashes []string `json:"hashes"`
}

// CheckRawRequest represents a check request with raw invoice data
type CheckRawRequest struct {
	InvoiceNumber string   `json:"invoice_number"`
	IssuerTaxID   string   `json:"issuer_tax_id"`
	Amount        *float64 `json:"amount,omitempty"`
	Currency      string   `json:"currency,omitempty"`
	InvoiceDate   string   `json:"invoice_date,omitempty"`
	BuyerTaxID    string   `json:"buyer_tax_id,omitempty"`
}

// CheckResponse represents the check response
type CheckResponse struct {
	Funded       bool       `json:"funded"`
	MatchedLevel *int       `json:"matched_level,omitempty"`
	FundedAt     *time.Time `json:"funded_at,omitempty"`
}

// RegisterRequest represents a register request
type RegisterRequest struct {
	Hashes        []string `json:"hashes"`
	FundingDate   string   `json:"funding_date"`
	TrackFunder   bool     `json:"track_funder"`
	ExpiresInDays *int     `json:"expires_in_days,omitempty"`
}

// RegisterRawRequest represents a register request with raw invoice data
type RegisterRawRequest struct {
	InvoiceNumber string   `json:"invoice_number"`
	IssuerTaxID   string   `json:"issuer_tax_id"`
	Amount        *float64 `json:"amount,omitempty"`
	Currency      string   `json:"currency,omitempty"`
	InvoiceDate   string   `json:"invoice_date,omitempty"`
	BuyerTaxID    string   `json:"buyer_tax_id,omitempty"`
	FundingDate   string   `json:"funding_date"`
	TrackFunder   bool     `json:"track_funder"`
	ExpiresInDays *int     `json:"expires_in_days,omitempty"`
}

// RegisterResponse represents the register response
type RegisterResponse struct {
	Success      bool      `json:"success"`
	RegisteredAt time.Time `json:"registered_at"`
	HashLevels   []int     `json:"hash_levels"`
}

// Check checks if an invoice is funded using pre-computed hashes
func (c *Client) Check(ctx context.Context, hashes []string) (*CheckResponse, error) {
	req := CheckRequest{Hashes: hashes}
	return c.doCheck(ctx, "/api/v1/invoices/check", req)
}

// CheckRaw checks if an invoice is funded using raw invoice data (server-side hashing)
func (c *Client) CheckRaw(ctx context.Context, req *CheckRawRequest) (*CheckResponse, error) {
	return c.doCheck(ctx, "/api/v1/invoices/check-raw", req)
}

// CheckWithClientHashing checks invoice using client-side hashing (requires hasher)
func (c *Client) CheckWithClientHashing(ctx context.Context, invoice *InvoiceData) (*CheckResponse, error) {
	if c.hasher == nil {
		return nil, fmt.Errorf("client-side hashing requires a hasher; use WithHasher option")
	}

	hashes := c.hasher.GenerateHashes(invoice)
	return c.Check(ctx, hashes.ToSlice())
}

// Register registers an invoice as funded using pre-computed hashes
func (c *Client) Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	return c.doRegister(ctx, "/api/v1/invoices/register", req)
}

// RegisterRaw registers an invoice as funded using raw invoice data (server-side hashing)
func (c *Client) RegisterRaw(ctx context.Context, req *RegisterRawRequest) (*RegisterResponse, error) {
	return c.doRegister(ctx, "/api/v1/invoices/register-raw", req)
}

// RegisterWithClientHashing registers invoice using client-side hashing (requires hasher)
func (c *Client) RegisterWithClientHashing(ctx context.Context, invoice *InvoiceData, fundingDate string, trackFunder bool, expiresInDays *int) (*RegisterResponse, error) {
	if c.hasher == nil {
		return nil, fmt.Errorf("client-side hashing requires a hasher; use WithHasher option")
	}

	hashes := c.hasher.GenerateHashes(invoice)
	req := &RegisterRequest{
		Hashes:        hashes.ToSlice(),
		FundingDate:   fundingDate,
		TrackFunder:   trackFunder,
		ExpiresInDays: expiresInDays,
	}
	return c.Register(ctx, req)
}

// Unregister removes an invoice from the registry
func (c *Client) Unregister(ctx context.Context, hash string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", c.baseURL+"/api/v1/invoices/"+hash, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s (status %d)", string(body), resp.StatusCode)
	}

	return nil
}

func (c *Client) doCheck(ctx context.Context, path string, body interface{}) (*CheckResponse, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s (status %d)", string(body), resp.StatusCode)
	}

	var result CheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (c *Client) doRegister(ctx context.Context, path string, body interface{}) (*RegisterResponse, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s (status %d)", string(body), resp.StatusCode)
	}

	var result RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
