package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/akinalp/mqvi/pkg"
	"github.com/akinalp/mqvi/testutil/dbtest"
)

// Real migrations rather than a hand-written subset: these tests touch sessions and reset tokens,
// whose cascade behaviour only means anything against the schema that actually ships.
func newUserRepoTestDB(t *testing.T) *sql.DB {
	t.Helper()
	return dbtest.New(t).DB
}

func TestSQLiteUserRepo_UpdatePasswordConflictDoesNotRotate(t *testing.T) {
	ctx := context.Background()
	db := newUserRepoTestDB(t)
	repo := NewSQLiteUserRepo(db)
	if _, err := db.ExecContext(ctx, `INSERT INTO users (id, username, password_hash, token_version) VALUES ('u1', 'u1', 'old-hash', 0)`); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	_, err := repo.UpdatePassword(ctx, "u1", "stale-hash", "new-hash")
	if !errors.Is(err, pkg.ErrConflict) {
		t.Fatalf("got %v want ErrConflict", err)
	}

	var passwordHash string
	var tokenVersion int
	if err := db.QueryRowContext(ctx, `SELECT password_hash, token_version FROM users WHERE id = 'u1'`).Scan(&passwordHash, &tokenVersion); err != nil {
		t.Fatalf("select user: %v", err)
	}
	if passwordHash != "old-hash" || tokenVersion != 0 {
		t.Fatalf("got password_hash=%q token_version=%d", passwordHash, tokenVersion)
	}
}

func TestSQLiteUserRepo_ResetPasswordWithTokenClearsSiblingTokensAndSessions(t *testing.T) {
	ctx := context.Background()
	db := newUserRepoTestDB(t)
	repo := NewSQLiteUserRepo(db)
	expiresAt := time.Now().Add(time.Hour)
	if _, err := db.ExecContext(ctx, `INSERT INTO users (id, username, password_hash, token_version) VALUES ('u1', 'u1', 'old-hash', 0)`); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO sessions (id, user_id, token_hash, expires_at) VALUES ('s1', 'u1', 'h-s1', ?)`, expiresAt); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at) VALUES ('rt1', 'u1', 'h1', ?), ('rt2', 'u1', 'h2', ?)`, expiresAt, expiresAt); err != nil {
		t.Fatalf("insert reset tokens: %v", err)
	}

	newTV, err := repo.ResetPasswordWithToken(ctx, "u1", "rt1", "new-hash")
	if err != nil {
		t.Fatalf("ResetPasswordWithToken: %v", err)
	}
	if newTV != 1 {
		t.Fatalf("newTV got %d want 1", newTV)
	}

	var passwordHash string
	var tokenVersion int
	if err := db.QueryRowContext(ctx, `SELECT password_hash, token_version FROM users WHERE id = 'u1'`).Scan(&passwordHash, &tokenVersion); err != nil {
		t.Fatalf("select user: %v", err)
	}
	if passwordHash != "new-hash" || tokenVersion != 1 {
		t.Fatalf("got password_hash=%q token_version=%d", passwordHash, tokenVersion)
	}

	var resetCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM password_reset_tokens WHERE user_id = 'u1'`).Scan(&resetCount); err != nil {
		t.Fatalf("count reset tokens: %v", err)
	}
	if resetCount != 0 {
		t.Fatalf("reset token count got %d want 0", resetCount)
	}

	var sessionCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions WHERE user_id = 'u1'`).Scan(&sessionCount); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if sessionCount != 0 {
		t.Fatalf("session count got %d want 0", sessionCount)
	}
}

// The tombstone keeps the user row, so push_tokens' ON DELETE CASCADE never fires; without the
// explicit delete a deleted account's phone keeps its registration.
func TestSQLiteUserRepo_HardDeleteShouldRemoveThePushTokens(t *testing.T) {
	ctx := context.Background()
	db := newUserRepoTestDB(t)
	repo := NewSQLiteUserRepo(db)
	if _, err := db.ExecContext(ctx, `INSERT INTO users (id, username, password_hash) VALUES ('u1', 'u1', 'x')`); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO push_tokens (user_id, token, platform) VALUES ('u1', 'fcm-1', 'android')`); err != nil {
		t.Fatalf("insert push token: %v", err)
	}

	if err := repo.HardDeleteUser(ctx, "u1", false); err != nil {
		t.Fatalf("hard delete: %v", err)
	}

	var left int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM push_tokens WHERE user_id = 'u1'`).Scan(&left); err != nil {
		t.Fatalf("count: %v", err)
	}
	if left != 0 {
		t.Fatal("push token survived the account's hard delete")
	}
}

// Signup invite references must not block account tombstoning.
func TestSQLiteUserRepo_HardDeleteCleansSignupInviteReferences(t *testing.T) {
	ctx := context.Background()
	db := newUserRepoTestDB(t)
	repo := NewSQLiteUserRepo(db)

	_, err := db.ExecContext(ctx, `
		INSERT INTO users (id, username, password_hash)
		VALUES ('u1', 'u1', 'x'), ('u2', 'u2', 'x'), ('inviter', 'inviter', 'x')
	`)
	if err != nil {
		t.Fatalf("insert users: %v", err)
	}

	usedAt := time.Now().UTC().Truncate(time.Second)
	_, err = db.ExecContext(ctx, `
		INSERT INTO signup_invite_codes (id, code_hash, inviter_user_id, used_at, used_by_user_id)
		VALUES ('code1', 'hash1', 'inviter', ?, 'u1')
	`, usedAt)
	if err != nil {
		t.Fatalf("insert signup invite: %v", err)
	}

	if _, err := db.ExecContext(ctx,
		`UPDATE users SET invited_by_user_id = 'u1' WHERE id = 'u2'`,
	); err != nil {
		t.Fatalf("set invited_by reference: %v", err)
	}

	if err := repo.HardDeleteUser(ctx, "u1", false); err != nil {
		t.Fatalf("hard delete: %v", err)
	}

	var usedBy sql.NullString
	var inviterID string
	var storedUsedAt time.Time
	if err := db.QueryRowContext(ctx, `
		SELECT inviter_user_id, used_by_user_id, used_at
		FROM signup_invite_codes WHERE id = 'code1'
	`).Scan(&inviterID, &usedBy, &storedUsedAt); err != nil {
		t.Fatalf("read signup invite: %v", err)
	}
	if inviterID != "inviter" || usedBy.Valid {
		t.Fatalf("invite references: inviter=%q used_by=%v", inviterID, usedBy)
	}
	if !storedUsedAt.Equal(usedAt) {
		t.Fatalf("used_at changed: got %v want %v", storedUsedAt, usedAt)
	}

	var invitedBy sql.NullString
	if err := db.QueryRowContext(ctx,
		`SELECT invited_by_user_id FROM users WHERE id = 'u2'`,
	).Scan(&invitedBy); err != nil {
		t.Fatalf("read invited_by_user_id: %v", err)
	}
	if invitedBy.Valid {
		t.Fatalf("invited_by_user_id was not cleared: %v", invitedBy)
	}
}

func TestSQLiteUserRepo_HardDeleteCleansDMDeliveries(t *testing.T) {
	ctx := context.Background()
	db := newUserRepoTestDB(t)
	repo := NewSQLiteUserRepo(db)

	_, err := db.ExecContext(ctx, `
		INSERT INTO users (id, username, password_hash)
		VALUES ('u1', 'u1', 'x'), ('u2', 'u2', 'x')
	`)
	if err != nil {
		t.Fatalf("insert users: %v", err)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO dm_channels (id, user1_id, user2_id)
		VALUES ('dm1', 'u1', 'u2')
	`)
	if err != nil {
		t.Fatalf("insert DM channel: %v", err)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO dm_deliveries
			(user_id, dm_channel_id, last_delivered_at)
		VALUES ('u1', 'dm1', CURRENT_TIMESTAMP)
	`)
	if err != nil {
		t.Fatalf("insert delivery cursor: %v", err)
	}

	if err := repo.HardDeleteUser(ctx, "u1", false); err != nil {
		t.Fatalf("hard delete: %v", err)
	}

	var cursors, channels int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM dm_deliveries WHERE user_id = 'u1'`,
	).Scan(&cursors); err != nil {
		t.Fatalf("count delivery cursors: %v", err)
	}
	if cursors != 0 {
		t.Fatalf("delivery cursors remaining: %d", cursors)
	}

	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM dm_channels WHERE id = 'dm1'`,
	).Scan(&channels); err != nil {
		t.Fatalf("count DM channels: %v", err)
	}
	if channels != 1 {
		t.Fatalf("DM channel count got %d want 1", channels)
	}
}
