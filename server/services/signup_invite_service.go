package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/akinalp/mqvi/database"
)

var (
	ErrSignupInviteQuota   = errors.New("signup invite creation quota reached")
	ErrSignupInviteInvalid = errors.New("invalid or unavailable signup invite")
)

type SignupInvite struct {
	ID        string     `json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

type CreatedSignupInvite struct {
	SignupInvite
	Code string `json:"code"`
}

type SignupInviteService interface {
	Create(ctx context.Context, inviterID string, expiryDays int) (*CreatedSignupInvite, error)
	List(ctx context.Context, inviterID string) ([]SignupInvite, error)
	Revoke(ctx context.Context, inviterID, inviteID string) error
	AdminSetLimit(ctx context.Context, userID string, limit int) error
	AdminListRelationships(ctx context.Context) ([]SignupInviteRelationship, error)
}

type SignupInviteRelationship struct {
	UserID             string  `json:"user_id"`
	Username           string  `json:"username"`
	InvitedByUserID    *string `json:"invited_by_user_id"`
	InvitedByUsername  *string `json:"invited_by_username"`
	Invitees           int     `json:"invitees"`
	InviteLimit        int     `json:"invite_limit"`
	InviteCodesCreated int     `json:"invite_codes_created"`
}

type signupInviteService struct {
	db *sql.DB
}

func NewSignupInviteService(db *sql.DB) SignupInviteService {
	return &signupInviteService{db: db}
}

func hashSignupCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

func (s *signupInviteService) Create(ctx context.Context, inviterID string, expiryDays int) (*CreatedSignupInvite, error) {
	if expiryDays != 0 && expiryDays != 1 && expiryDays != 7 && expiryDays != 30 {
		return nil, fmt.Errorf("expiry must be 0, 1, 7, or 30 days")
	}

	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return nil, fmt.Errorf("generate signup invite: %w", err)
	}
	code := hex.EncodeToString(raw[:])
	hash := hashSignupCode(code)

	var result CreatedSignupInvite
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			UPDATE users
			SET invite_codes_created = invite_codes_created + 1
			WHERE id = ?
			  AND deleted_at IS NULL
			  AND invite_codes_created < invite_limit
		`, inviterID)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return ErrSignupInviteQuota
		}

		var expires any
		var expiry *time.Time
		if expiryDays > 0 {
			t := time.Now().UTC().Add(time.Duration(expiryDays) * 24 * time.Hour)
			expires = t
			expiry = &t
		}

		err = tx.QueryRowContext(ctx, `
			INSERT INTO signup_invite_codes (code_hash, inviter_user_id, expires_at)
			VALUES (?, ?, ?)
			RETURNING id, created_at
		`, hash, inviterID, expires).Scan(&result.ID, &result.CreatedAt)
		if err != nil {
			return err
		}
		result.ExpiresAt = expiry
		return nil
	})
	if err != nil {
		return nil, err
	}

	result.Code = code
	return &result, nil
}

func (s *signupInviteService) List(ctx context.Context, inviterID string) ([]SignupInvite, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, created_at, expires_at, used_at, revoked_at
		FROM signup_invite_codes
		WHERE inviter_user_id = ?
		ORDER BY created_at DESC
	`, inviterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]SignupInvite, 0)
	for rows.Next() {
		var item SignupInvite
		if err := rows.Scan(&item.ID, &item.CreatedAt, &item.ExpiresAt, &item.UsedAt, &item.RevokedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *signupInviteService) Revoke(ctx context.Context, inviterID, inviteID string) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE signup_invite_codes
		SET revoked_at = CURRENT_TIMESTAMP
		WHERE id = ? AND inviter_user_id = ?
		  AND used_at IS NULL AND revoked_at IS NULL
	`, inviteID, inviterID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrSignupInviteInvalid
	}
	return nil
}

func (s *signupInviteService) AdminSetLimit(ctx context.Context, userID string, limit int) error {
	if limit < 0 || limit > 10000 {
		return fmt.Errorf("invite limit must be between 0 and 10000")
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE users SET invite_limit = ?
		WHERE id = ? AND deleted_at IS NULL
	`, limit, userID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *signupInviteService) AdminListRelationships(ctx context.Context) ([]SignupInviteRelationship, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.id, u.username, u.invited_by_user_id, inviter.username,
		       (SELECT COUNT(*) FROM users child WHERE child.invited_by_user_id = u.id),
		       u.invite_limit, u.invite_codes_created
		FROM users u
		LEFT JOIN users inviter ON inviter.id = u.invited_by_user_id
		WHERE u.deleted_at IS NULL AND u.is_hard_deleted = 0
		ORDER BY u.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]SignupInviteRelationship, 0)
	for rows.Next() {
		var item SignupInviteRelationship
		if err := rows.Scan(&item.UserID, &item.Username, &item.InvitedByUserID,
			&item.InvitedByUsername, &item.Invitees, &item.InviteLimit,
			&item.InviteCodesCreated); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
