package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/regstrava/regstrava/internal/domain"
	"github.com/regstrava/regstrava/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication logic
type AuthService struct {
	funderRepo *repository.FunderRepository
	jwtSecret  []byte
}

// NewAuthService creates a new auth service
func NewAuthService(funderRepo *repository.FunderRepository, jwtSecret string) *AuthService {
	return &AuthService{
		funderRepo: funderRepo,
		jwtSecret:  []byte(jwtSecret),
	}
}

// ValidateAPIKey validates an API key and returns the associated funder
func (s *AuthService) ValidateAPIKey(ctx context.Context, apiKey string) (*domain.Funder, error) {
	// Get all active funders and check API key
	// In production, you might want to use a more efficient lookup
	funders, err := s.funderRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	apiKeyBytes := []byte(apiKey)

	for _, funder := range funders {
		if !funder.IsActive {
			continue
		}

		// Compare API key hash
		if err := bcrypt.CompareHashAndPassword([]byte(funder.APIKeyHash), apiKeyBytes); err == nil {
			return funder, nil
		}
	}

	return nil, ErrInvalidCredentials
}

// GenerateToken generates a JWT token for the given funder
func (s *AuthService) GenerateToken(funder *domain.Funder) (*domain.OAuthTokenResponse, error) {
	expiresIn := 3600 // 1 hour
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)

	claims := jwt.MapClaims{
		"funder_id":   funder.ID.String(),
		"funder_name": funder.Name,
		"exp":         expiresAt.Unix(),
		"iat":         time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, err
	}

	return &domain.OAuthTokenResponse{
		AccessToken: signedToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	}, nil
}

// ValidateToken validates a JWT token and returns the funder ID
func (s *AuthService) ValidateToken(tokenString string) (*uuid.UUID, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidCredentials
		}
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidCredentials
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidCredentials
	}

	funderIDStr, ok := claims["funder_id"].(string)
	if !ok {
		return nil, ErrInvalidCredentials
	}

	funderID, err := uuid.Parse(funderIDStr)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	return &funderID, nil
}

// ValidateOAuthCredentials validates OAuth client credentials
func (s *AuthService) ValidateOAuthCredentials(ctx context.Context, clientID, clientSecret string) (*domain.Funder, error) {
	funder, err := s.funderRepo.FindByOAuthClientID(ctx, clientID)
	if err != nil {
		return nil, err
	}

	if funder == nil || !funder.IsActive {
		return nil, ErrInvalidCredentials
	}

	if funder.OAuthSecretHash == nil {
		return nil, ErrInvalidCredentials
	}

	// Compare OAuth secret hash
	if err := bcrypt.CompareHashAndPassword([]byte(*funder.OAuthSecretHash), []byte(clientSecret)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return funder, nil
}

// HashAPIKey creates a bcrypt hash of an API key
func HashAPIKey(apiKey string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// GenerateAPIKey generates a random API key
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// SecureCompare performs a constant-time comparison of two strings
func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
