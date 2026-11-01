package repository

import (
	"context"
	"testing"

	"github.com/akinalp/mqvi/testutil/dbtest"
)

func TestSQLiteUserRepo_HardDeletePreservesPlatformBan(t *testing.T) {
	ctx := context.Background()
	db := dbtest.New(t).DB
	repo := NewSQLiteUserRepo(db)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO users (id, username, password_hash)
		VALUES ('banned-user', 'blockeduser', 'hash'),
		       ('moderator', 'moderator', 'hash')
	`); err != nil {
		t.Fatalf("insert users: %v", err)
	}
	if err := repo.InsertPlatformBan(
		ctx,
		"blocked@example.test",
		"blockeduser",
		"banned-user",
		"abuse",
		"moderator",
	); err != nil {
		t.Fatalf("insert platform ban: %v", err)
	}

	if err := repo.HardDeleteUser(ctx, "banned-user", false); err != nil {
		t.Fatalf("hard delete banned account: %v", err)
	}

	emailBanned, err := repo.IsEmailPlatformBanned(ctx, "blocked@example.test")
	if err != nil {
		t.Fatalf("check email ban: %v", err)
	}
	if !emailBanned {
		t.Fatal("email platform ban was erased by account deletion")
	}

	usernameBanned, err := repo.IsUsernamePlatformBanned(ctx, "BLOCKEDUSER")
	if err != nil {
		t.Fatalf("check username ban: %v", err)
	}
	if !usernameBanned {
		t.Fatal("username platform ban was erased by account deletion")
	}

	userIDBanned, err := repo.IsPlatformBannedByUserID(ctx, "banned-user")
	if err != nil {
		t.Fatalf("check user ID ban: %v", err)
	}
	if !userIDBanned {
		t.Fatal("user ID platform ban was erased by account deletion")
	}

	// The issuer ID is intentionally retained as historical moderation attribution.
	if err := repo.HardDeleteUser(ctx, "moderator", false); err != nil {
		t.Fatalf("hard delete moderator: %v", err)
	}
	emailBanned, err = repo.IsEmailPlatformBanned(ctx, "blocked@example.test")
	if err != nil {
		t.Fatalf("recheck email ban: %v", err)
	}
	if !emailBanned {
		t.Fatal("platform ban was erased when its issuing moderator was deleted")
	}

	var preservedIssuer int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM platform_bans
		WHERE email = 'blocked@example.test' AND banned_by = 'moderator'
	`).Scan(&preservedIssuer); err != nil {
		t.Fatalf("check historical moderator attribution: %v", err)
	}
	if preservedIssuer != 1 {
		t.Fatalf("expected one preserved ban with historical issuer ID, got %d", preservedIssuer)
	}
}
