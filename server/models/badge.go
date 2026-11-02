package models

import "time"

// Badge represents a badge template that can be assigned to users.
// CreatedBy is nullable: ON DELETE SET NULL fires when the creating admin is deleted
// (migration 060) — badge stays as platform-level content.
type Badge struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon"`
	IconType  string    `json:"icon_type"` // "builtin" or "custom"
	Color1    string    `json:"color1"`
	Color2    *string   `json:"color2"` // nil = solid color, non-nil = gradient
	CreatedBy *string   `json:"created_by"` // nullable — null when creating admin was deleted
	CreatedAt time.Time `json:"created_at"`
}

// UserBadge represents a badge assigned to a specific user.
// AssignedBy is nullable for the same reason as Badge.CreatedBy.
type UserBadge struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	BadgeID    string    `json:"badge_id"`
	AssignedBy *string   `json:"assigned_by"` // nullable — null when assigning admin was deleted
	AssignedAt time.Time `json:"assigned_at"`
	Badge      *Badge    `json:"badge,omitempty"` // populated on read
}

// MaxBadgesPerUser is the maximum number of badges a user can have.
const MaxBadgesPerUser = 3

// CreateBadgeRequest is the payload for creating a new badge template.
type CreateBadgeRequest struct {
	Name     string  `json:"name"`
	Icon     string  `json:"icon"`
	IconType string  `json:"icon_type"`
	Color1   string  `json:"color1"`
	Color2   *string `json:"color2"`
}
