package domain

import (
	"time"

	"github.com/google/uuid"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationHashChecked     NotificationType = "hash_checked"
	NotificationHashRegistered  NotificationType = "hash_registered"
	NotificationPartyChecked    NotificationType = "party_checked"
	NotificationPartyRegistered NotificationType = "party_registered"
)

// Notification represents a notification for a funder
type Notification struct {
	ID                  uuid.UUID        `json:"id" db:"id"`
	FunderID            uuid.UUID        `json:"funder_id" db:"funder_id"`
	NotificationType    NotificationType `json:"notification_type" db:"notification_type"`
	HashValue           *string          `json:"hash_value,omitempty" db:"hash_value"`
	HashLevel           *HashLevel       `json:"hash_level,omitempty" db:"hash_level"`
	PartyType           *PartyType       `json:"party_type,omitempty" db:"party_type"`
	TriggeredByFunderID *uuid.UUID       `json:"triggered_by_funder_id,omitempty" db:"triggered_by_funder_id"`
	Message             string           `json:"message" db:"message"`
	IsRead              bool             `json:"is_read" db:"is_read"`
	CreatedAt           time.Time        `json:"created_at" db:"created_at"`
}

// NotificationListRequest represents a request to list notifications
type NotificationListRequest struct {
	UnreadOnly bool `json:"unread_only,omitempty"`
	Limit      int  `json:"limit,omitempty"`
	Offset     int  `json:"offset,omitempty"`
}

// NotificationListResponse represents the response for listing notifications
type NotificationListResponse struct {
	Notifications []Notification `json:"notifications"`
	TotalCount    int            `json:"total_count"`
	UnreadCount   int            `json:"unread_count"`
}

// MarkNotificationReadRequest represents a request to mark notifications as read
type MarkNotificationReadRequest struct {
	NotificationIDs []uuid.UUID `json:"notification_ids" validate:"required,min=1"`
}
