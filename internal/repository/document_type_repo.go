package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/regstrava/regstrava/internal/domain"
)

// DocumentTypeRepository handles document type persistence
type DocumentTypeRepository struct {
	db *sql.DB
}

// NewDocumentTypeRepository creates a new document type repository with a shared database connection
func NewDocumentTypeRepository(db *sql.DB) *DocumentTypeRepository {
	return &DocumentTypeRepository{db: db}
}

// FindByCode finds a document type by its code
func (r *DocumentTypeRepository) FindByCode(ctx context.Context, code string) (*domain.DocumentType, error) {
	query := `
		SELECT code, name, description, is_active, created_at
		FROM document_types
		WHERE code = $1
	`

	var docType domain.DocumentType
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&docType.Code,
		&docType.Name,
		&docType.Description,
		&docType.IsActive,
		&docType.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find document type: %w", err)
	}

	return &docType, nil
}

// ListActive lists all active document types
func (r *DocumentTypeRepository) ListActive(ctx context.Context) ([]domain.DocumentType, error) {
	query := `
		SELECT code, name, description, is_active, created_at
		FROM document_types
		WHERE is_active = true
		ORDER BY code
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list document types: %w", err)
	}
	defer rows.Close()

	var docTypes []domain.DocumentType
	for rows.Next() {
		var docType domain.DocumentType
		if err := rows.Scan(
			&docType.Code,
			&docType.Name,
			&docType.Description,
			&docType.IsActive,
			&docType.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan document type: %w", err)
		}
		docTypes = append(docTypes, docType)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating document types: %w", err)
	}

	return docTypes, nil
}

// IsValidCode checks if a document type code is valid and active
func (r *DocumentTypeRepository) IsValidCode(ctx context.Context, code string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM document_types WHERE code = $1 AND is_active = true)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, code).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check document type: %w", err)
	}

	return exists, nil
}
