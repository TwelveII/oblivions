package repository

import (
	"context"
	"testing"

	"github.com/akinalp/mqvi/testutil/dbtest"
)

func TestSQLiteUserRepo_HardDeleteTransfersOwnedServersAndDeletesEmptyOnes(t *testing.T) {
	ctx := context.Background()
	db := dbtest.New(t).DB
	repo := NewSQLiteUserRepo(db)

	owner := "server-owner"
	firstMember := "first-member"
	secondMember := "second-member"
	emptyOwner := "empty-owner"
	for _, user := range []struct{ id, username string }{
		{owner, "server_owner"},
		{firstMember, "first_member"},
		{secondMember, "second_member"},
		{emptyOwner, "empty_owner"},
	} {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO users (id, username, password_hash) VALUES (?, ?, 'hash')`,
			user.id, user.username,
		); err != nil {
			t.Fatalf("insert user %s: %v", user.id, err)
		}
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO servers (id, name, owner_id, e2ee_enabled)
		VALUES ('surviving-server', 'Surviving', ?, 0),
		       ('empty-server', 'Empty', ?, 0)
	`, owner, emptyOwner); err != nil {
		t.Fatalf("insert servers: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO server_members (server_id, user_id, joined_at)
		VALUES
			('surviving-server', ?, '2020-01-01 00:00:00'),
			('surviving-server', ?, '2021-01-01 00:00:00'),
			('surviving-server', ?, '2022-01-01 00:00:00'),
			('empty-server', ?, '2020-01-01 00:00:00')
	`, owner, firstMember, secondMember, emptyOwner); err != nil {
		t.Fatalf("insert server memberships: %v", err)
	}

	if err := repo.HardDeleteUser(ctx, owner, false); err != nil {
		t.Fatalf("hard delete server owner: %v", err)
	}
	if err := repo.HardDeleteUser(ctx, emptyOwner, false); err != nil {
		t.Fatalf("hard delete empty server owner: %v", err)
	}

	var newOwner string
	if err := db.QueryRowContext(ctx,
		`SELECT owner_id FROM servers WHERE id = 'surviving-server'`,
	).Scan(&newOwner); err != nil {
		t.Fatalf("read transferred server owner: %v", err)
	}
	if newOwner != firstMember {
		t.Fatalf("server transferred to %q, want earliest member %q", newOwner, firstMember)
	}

	var ownerMemberships, emptyServerCount int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM server_members WHERE user_id = ?`, owner,
	).Scan(&ownerMemberships); err != nil {
		t.Fatalf("count deleted owner's memberships: %v", err)
	}
	if ownerMemberships != 0 {
		t.Fatalf("deleted owner's memberships remain: %d", ownerMemberships)
	}
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM servers WHERE id = 'empty-server'`,
	).Scan(&emptyServerCount); err != nil {
		t.Fatalf("count empty server: %v", err)
	}
	if emptyServerCount != 0 {
		t.Fatal("server with no remaining members survived owner deletion")
	}
}
