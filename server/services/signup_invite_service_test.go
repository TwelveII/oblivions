package services

import (
	"context"
	"errors"
	"testing"

	"github.com/akinalp/mqvi/testutil/dbtest"
)

func TestSignupInviteQuotaAndRevocation(t *testing.T) {
	ctx := context.Background()
	fx := dbtest.New(t)
	inviterID := fx.User("inviter")
	svc := NewSignupInviteService(fx.DB)

	created, err := svc.Create(ctx, inviterID, 0)
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}
	if created.Code == "" {
		t.Fatal("expected raw invite code")
	}

	var storedHash string
	if err := fx.DB.QueryRow(
		`SELECT code_hash FROM signup_invite_codes WHERE id = ?`,
		created.ID,
	).Scan(&storedHash); err != nil {
		t.Fatal(err)
	}
	if storedHash == created.Code || storedHash != hashSignupCode(created.Code) {
		t.Fatal("expected only the SHA-256 hash to be stored")
	}

	if _, err := svc.Create(ctx, inviterID, 0); !errors.Is(err, ErrSignupInviteQuota) {
		t.Fatalf("expected quota error, got %v", err)
	}

	if err := svc.Revoke(ctx, inviterID, created.ID); err != nil {
		t.Fatalf("revoke invite: %v", err)
	}

	if _, err := svc.Create(ctx, inviterID, 0); !errors.Is(err, ErrSignupInviteQuota) {
		t.Fatalf("revoking must not restore lifetime quota; got %v", err)
	}

	var count int
	if err := fx.DB.QueryRow(
		`SELECT invite_codes_created FROM users WHERE id = ?`,
		inviterID,
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected lifetime count 1, got %d", count)
	}
}

func TestSignupInviteExpiryOptions(t *testing.T) {
	ctx := context.Background()
	fx := dbtest.New(t)
	inviterID := fx.User("inviter")
	svc := NewSignupInviteService(fx.DB)

	for _, days := range []int{0, 1, 7, 30} {
		if err := svc.AdminSetLimit(ctx, inviterID, days+1); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Create(ctx, inviterID, days); err != nil {
			t.Fatalf("expiry %d days: %v", days, err)
		}
	}

	if _, err := svc.Create(ctx, inviterID, 2); err == nil {
		t.Fatal("expected unsupported expiry to fail")
	}
}


func TestSignupInviteList_ShouldHideUsedCodesButKeepUnusedCodes(t *testing.T) {
	ctx := context.Background()
	fx := dbtest.New(t)
	inviterID := fx.User("inviter")
	svc := NewSignupInviteService(fx.DB)
	if err := svc.AdminSetLimit(ctx, inviterID, 2); err != nil {
		t.Fatal(err)
	}

	used, err := svc.Create(ctx, inviterID, 0)
	if err != nil {
		t.Fatalf("create used invite: %v", err)
	}
	available, err := svc.Create(ctx, inviterID, 0)
	if err != nil {
		t.Fatalf("create available invite: %v", err)
	}
	if _, err := fx.DB.ExecContext(ctx, `
		UPDATE signup_invite_codes SET used_at = CURRENT_TIMESTAMP, used_by_user_id = ?
		WHERE id = ?
	`, fx.User("invitee"), used.ID); err != nil {
		t.Fatalf("mark invite used: %v", err)
	}

	listed, err := svc.List(ctx, inviterID)
	if err != nil {
		t.Fatalf("list invites: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != available.ID {
		t.Fatalf("listed invites = %+v; want only unused invite %s (used %s)", listed, available.ID, used.ID)
	}
}
