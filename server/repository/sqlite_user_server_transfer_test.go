package repository

import (
	"context"
	"testing"

	"github.com/akinalp/mqvi/testutil/dbtest"
)

func TestSQLiteUserRepo_HardDeleteTransfersOwnedServerAndPreservesOtherMembersContent(t *testing.T) {
	ctx := context.Background()
	db := dbtest.New(t).DB
	repo := NewSQLiteUserRepo(db)

	_, err := db.ExecContext(ctx, `
		INSERT INTO users (id, username, password_hash) VALUES
			('owner', 'owner', 'x'),
			('member1', 'member1', 'x'),
			('member2', 'member2', 'x')
	`)
	if err != nil { t.Fatalf("insert users: %v", err) }

	_, err = db.ExecContext(ctx, `
		INSERT INTO servers (id, name, owner_id) VALUES ('srv1', 'Community', 'owner')
	`)
	if err != nil { t.Fatalf("insert server: %v", err) }

	_, err = db.ExecContext(ctx, `
		INSERT INTO server_members (server_id, user_id, joined_at) VALUES
			('srv1', 'owner', '2026-01-01 00:00:00'),
			('srv1', 'member1', '2026-01-02 00:00:00'),
			('srv1', 'member2', '2026-01-03 00:00:00')
	`)
	if err != nil { t.Fatalf("insert members: %v", err) }

	_, err = db.ExecContext(ctx, `
		INSERT INTO channels (id, name, type, server_id) VALUES ('ch1', 'general', 'text', 'srv1')
	`)
	if err != nil { t.Fatalf("insert channel: %v", err) }
	_, err = db.ExecContext(ctx, `
		INSERT INTO messages (id, channel_id, user_id, content) VALUES
			('owner-msg', 'ch1', 'owner', 'remove this'),
			('member-msg', 'ch1', 'member1', 'keep this')
	`)
	if err != nil { t.Fatalf("insert messages: %v", err) }
	_, err = db.ExecContext(ctx, `
		INSERT INTO attachments (id, message_id, filename, file_url) VALUES
			('owner-file', 'owner-msg', 'owner.txt', '/uploads/owner.txt'),
			('member-file', 'member-msg', 'member.txt', '/uploads/member.txt')
	`)
	if err != nil { t.Fatalf("insert attachments: %v", err) }

	if err := repo.HardDeleteUser(ctx, "owner", false); err != nil {
		t.Fatalf("hard delete owner: %v", err)
	}

	var ownerID string
	if err := db.QueryRowContext(ctx, `SELECT owner_id FROM servers WHERE id = 'srv1'`).Scan(&ownerID); err != nil {
		t.Fatalf("server should survive: %v", err)
	}
	if ownerID != "member1" {
		t.Fatalf("successor = %q, want oldest member member1", ownerID)
	}

	var deletedMembers, ownerMessages, ownerAttachments, memberMessages, memberAttachments int
	checks := []struct {
		query string
		dest  *int
	}{
		{`SELECT COUNT(*) FROM server_members WHERE server_id = 'srv1' AND user_id = 'owner'`, &deletedMembers},
		{`SELECT COUNT(*) FROM messages WHERE user_id = 'owner'`, &ownerMessages},
		{`SELECT COUNT(*) FROM attachments WHERE id = 'owner-file'`, &ownerAttachments},
		{`SELECT COUNT(*) FROM messages WHERE id = 'member-msg'`, &memberMessages},
		{`SELECT COUNT(*) FROM attachments WHERE id = 'member-file'`, &memberAttachments},
	}
	for _, check := range checks {
		if err := db.QueryRowContext(ctx, check.query).Scan(check.dest); err != nil {
			t.Fatalf("query %q: %v", check.query, err)
		}
	}
	if deletedMembers != 0 || ownerMessages != 0 || ownerAttachments != 0 {
		t.Fatalf("former owner's data remains: membership=%d messages=%d attachments=%d", deletedMembers, ownerMessages, ownerAttachments)
	}
	if memberMessages != 1 || memberAttachments != 1 {
		t.Fatalf("other member's content was lost: messages=%d attachments=%d", memberMessages, memberAttachments)
	}
}

func TestSQLiteUserRepo_HardDeleteRemovesOwnedServerWithNoOtherMembers(t *testing.T) {
	ctx := context.Background()
	db := dbtest.New(t).DB
	repo := NewSQLiteUserRepo(db)

	if _, err := db.ExecContext(ctx, `INSERT INTO users (id, username, password_hash) VALUES ('owner', 'owner', 'x')`); err != nil {
		t.Fatalf("insert owner: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO servers (id, name, owner_id) VALUES ('empty-srv', 'Empty', 'owner')`); err != nil {
		t.Fatalf("insert server: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO server_members (server_id, user_id) VALUES ('empty-srv', 'owner')`); err != nil {
		t.Fatalf("insert member: %v", err)
	}

	if err := repo.HardDeleteUser(ctx, "owner", false); err != nil {
		t.Fatalf("hard delete owner: %v", err)
	}

	var servers, users int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM servers WHERE id = 'empty-srv'`).Scan(&servers); err != nil {
		t.Fatalf("count server: %v", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE id = 'owner'`).Scan(&users); err != nil {
		t.Fatalf("count user: %v", err)
	}
	if servers != 0 || users != 0 {
		t.Fatalf("empty server/account remain: servers=%d users=%d", servers, users)
	}
}
