package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/akinalp/mqvi/models"
	"github.com/akinalp/mqvi/pkg"
	"github.com/akinalp/mqvi/pkg/geoip"
	"github.com/akinalp/mqvi/pkg/password"
	"github.com/akinalp/mqvi/testutil"
	"github.com/akinalp/mqvi/testutil/dbtest"
)

func TestRegisterRejectsReservedUsernameWithoutConsumingInvite(t *testing.T) {
	ctx := context.Background()
	fx := dbtest.New(t)
	inviterID := fx.User("")

	code := "one-time-signup-code"
	codeHash := sha256.Sum256([]byte(code))
	codeHashHex := hex.EncodeToString(codeHash[:])
	if _, err := fx.DB.ExecContext(ctx, `
		INSERT INTO signup_invite_codes (code_hash, inviter_user_id)
		VALUES (?, ?)
	`, codeHashHex, inviterID); err != nil {
		t.Fatalf("insert invite code: %v", err)
	}
	if _, err := fx.DB.ExecContext(ctx, `
		INSERT INTO username_reservations (username, reserved_until)
		VALUES ('ReservedName', datetime('now', '+24 hours'))
	`); err != nil {
		t.Fatalf("insert username reservation: %v", err)
	}

	svc := NewAuthService(
		&testutil.MockUserRepo{},
		&testutil.MockSessionRepo{},
		&testutil.MockResetRepo{},
		&testutil.MockEventPublisher{},
		&testutil.MockEmailSender{},
		password.NoopChecker{},
		geoip.Noop{},
		testJWTSecret,
		testEncryptionKey,
		15, 90, 24,
		fx.DB,
	)

	_, err := svc.Register(ctx, &models.CreateUserRequest{
		Username:   "reservedname",
		Password:   testRegisterPassword,
		InviteCode: code,
	}, models.ClientInfo{})
	if !errors.Is(err, pkg.ErrAlreadyExists) {
		t.Fatalf("Register() error = %v, want ErrAlreadyExists", err)
	}

	var usedAt, usedByUserID *string
	if err := fx.DB.QueryRowContext(ctx, `
		SELECT used_at, used_by_user_id FROM signup_invite_codes WHERE code_hash = ?
	`, codeHashHex).Scan(&usedAt, &usedByUserID); err != nil {
		t.Fatalf("read invite code: %v", err)
	}
	if usedAt != nil || usedByUserID != nil {
		t.Fatal("invite code was consumed by a registration rejected for username reservation")
	}
}
