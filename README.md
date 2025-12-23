# RegStrava

A privacy-preserving invoice funding registry that prevents double-funding fraud.

## Problem

Invoice factoring fraud occurs when the same invoice is funded by multiple funders simultaneously. This was famously exploited in the [First Brands Group case](https://en.wikipedia.org/wiki/First_Brands_Group), resulting in significant financial losses.

## Solution

RegStrava provides a shared registry where funders can:
1. **Check** if an invoice has already been funded
2. **Register** an invoice as funded to prevent others from funding it

All while maintaining **complete anonymity** of invoice issuers through blind hashing.

## Features

- **Self-service signup** - Landing page with instant API key generation
- **Funder dashboard** - Manage API keys, view docs, test the API
- **Multi-level hashing (L1-L4)** - Different hash levels for varying invoice detail availability
- **Hybrid hashing** - Server-side for convenience, client SDK for maximum privacy
- **Dual authentication** - API Keys for simplicity, OAuth 2.0 for enterprise
- **Per-funder rate limiting** - Configurable daily/monthly limits via Redis
- **Optional funder tracking** - Consent-based tracking for audit trails
- **Configurable expiration** - Funders decide how long registrations last

## Hash Levels

| Level | Fields | Use Case |
|-------|--------|----------|
| L1 (Basic) | invoice_number + issuer_tax_id | Minimum viable check |
| L2 (Standard) | L1 + amount + currency | Most common |
| L3 (Dated) | L2 + invoice_date | Full standard invoice |
| L4 (Full) | L3 + buyer_tax_id | Complete invoice data |

## Quick Start

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- Make (optional)

### Run with Docker

```bash
# Clone the repository
git clone https://github.com/janovincze/RegStrava.git
cd RegStrava

# Start all services
docker-compose up -d

# Open http://localhost:8080 to access the landing page
# Sign up to get your API key instantly!
```

### Run Locally

```bash
# Start infrastructure only (uses ports 5433 and 6380 to avoid conflicts)
docker-compose up -d postgres redis

# Set environment variables
export HMAC_KEY="your-super-secret-hmac-key"
export JWT_SECRET="your-super-secret-jwt-key"
export DATABASE_URL="postgres://regstrava:regstrava@localhost:5433/regstrava?sslmode=disable"
export REDIS_URL="redis://localhost:6380"

# Build and run
make run

# Open http://localhost:8080 for the landing page
```

## API Usage

### Authentication

**API Key:**
```bash
curl -H "X-API-Key: your-api-key" ...
```

**OAuth 2.0:**
```bash
# Get token
curl -X POST http://localhost:8080/api/v1/oauth/token \
  -H "Content-Type: application/json" \
  -d '{
    "grant_type": "client_credentials",
    "client_id": "your-client-id",
    "client_secret": "your-client-secret"
  }'

# Use token
curl -H "Authorization: Bearer <token>" ...
```

### Check Invoice

**With server-side hashing (simpler):**
```bash
curl -X POST http://localhost:8080/api/v1/invoices/check-raw \
  -H "X-API-Key: test-api-key-12345" \
  -H "Content-Type: application/json" \
  -d '{
    "invoice_number": "INV-2024-001",
    "issuer_tax_id": "DE123456789",
    "amount": 10000.00,
    "currency": "EUR"
  }'
```

**With pre-computed hashes (more private):**
```bash
curl -X POST http://localhost:8080/api/v1/invoices/check \
  -H "X-API-Key: test-api-key-12345" \
  -H "Content-Type: application/json" \
  -d '{
    "hashes": ["abc123...", "def456..."]
  }'
```

**Response:**
```json
{
  "funded": false
}
```
or
```json
{
  "funded": true,
  "matched_level": 2,
  "funded_at": "2024-01-15T10:30:00Z"
}
```

### Register Invoice

```bash
curl -X POST http://localhost:8080/api/v1/invoices/register-raw \
  -H "X-API-Key: test-api-key-12345" \
  -H "Content-Type: application/json" \
  -d '{
    "invoice_number": "INV-2024-001",
    "issuer_tax_id": "DE123456789",
    "amount": 10000.00,
    "currency": "EUR",
    "funding_date": "2024-12-23",
    "track_funder": true,
    "expires_in_days": 730
  }'
```

**Response:**
```json
{
  "success": true,
  "registered_at": "2024-12-23T14:30:00Z",
  "hash_levels": [1, 2]
}
```

### Unregister Invoice

Only allowed within 24 hours, only by the original registrant:

```bash
curl -X DELETE http://localhost:8080/api/v1/invoices/{hash} \
  -H "X-API-Key: test-api-key-12345"
```

## Go SDK

For maximum privacy, use client-side hashing with our SDK:

```go
package main

import (
    "context"
    "fmt"
    regstrava "github.com/janovincze/RegStrava/sdk/go"
)

func main() {
    // Create client with client-side hashing
    client := regstrava.NewClient(
        "http://localhost:8080",
        "your-api-key",
        regstrava.WithHasher("your-hmac-key"),
    )

    // Check invoice - data never leaves your system unhashed
    amount := 10000.00
    invoice := &regstrava.InvoiceData{
        InvoiceNumber: "INV-2024-001",
        IssuerTaxID:   "DE123456789",
        Amount:        &amount,
        Currency:      "EUR",
    }

    result, err := client.CheckWithClientHashing(context.Background(), invoice)
    if err != nil {
        panic(err)
    }

    if result.Funded {
        fmt.Println("Invoice already funded!")
    } else {
        fmt.Println("Invoice is available for funding")
    }
}
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/ready` | Readiness probe |
| POST | `/api/v1/oauth/token` | Get OAuth token |
| POST | `/api/v1/invoices/check` | Check with pre-hashed data |
| POST | `/api/v1/invoices/check-raw` | Check with server-side hashing |
| POST | `/api/v1/invoices/register` | Register with pre-hashed data |
| POST | `/api/v1/invoices/register-raw` | Register with server-side hashing |
| DELETE | `/api/v1/invoices/{hash}` | Unregister invoice |

## Rate Limiting

Response headers include rate limit info:

```
X-RateLimit-Daily-Limit: 1000
X-RateLimit-Daily-Used: 42
X-RateLimit-Monthly-Limit: 20000
X-RateLimit-Monthly-Used: 1337
```

When exceeded, returns `429 Too Many Requests` with `Retry-After` header.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `DATABASE_URL` | - | PostgreSQL connection string |
| `REDIS_URL` | - | Redis connection string |
| `HMAC_KEY` | - | **Required.** Secret key for hashing |
| `JWT_SECRET` | - | **Required.** Secret for JWT tokens |

## Security

- **Blind hashing**: Only HMAC-SHA256 hashes are stored, never raw invoice data
- **HMAC with secret key**: Prevents rainbow table attacks
- **Bcrypt**: API keys and OAuth secrets are bcrypt-hashed in the database
- **Rate limiting**: Prevents enumeration attacks
- **Time-limited unregister**: Reduces fraud window

## Architecture

```
┌─────────────┐     ┌─────────────────┐     ┌──────────────┐
│   Funder    │────▶│   RegStrava     │────▶│  PostgreSQL  │
│   Client    │     │   API (Go)      │     │   Database   │
└─────────────┘     └─────────────────┘     └──────────────┘
                            │
                            ▼
                    ┌──────────────┐
                    │    Redis     │
                    │ (Rate Limit) │
                    └──────────────┘
```

## Development

```bash
# Format code
make fmt

# Run tests
make test

# Build binary
make build

# Clean
make clean
```

## License

MIT

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.
