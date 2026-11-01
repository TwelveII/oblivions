package services

import (
	"context"
	"strings"
	"testing"

	"github.com/akinalp/mqvi/pkg/files"
)

func TestCollectUserFilesPreservesSharedAssetsWhenOwnedServerWillTransfer(t *testing.T) {
	db := orphanTestDB(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO users (id, username, password_hash) VALUES ('u2', 'member', 'x')
	`); err != nil {
		t.Fatalf("insert member: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO server_members (server_id, user_id) VALUES ('s1', 'u1'), ('s1', 'u2')
	`); err != nil {
		t.Fatalf("insert memberships: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO channels (id, name, type, server_id) VALUES ('chan1', 'general', 'text', 's1')
	`); err != nil {
		t.Fatalf("insert channel: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		UPDATE servers SET icon_url = '/uploads/server-icon.png', banner_url = '/uploads/server-banner.png' WHERE id = 's1'
	`); err != nil {
		t.Fatalf("set server assets: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO messages (id, channel_id, user_id, content) VALUES
			('owner-msg', 'chan1', 'u1', 'remove me'),
			('member-msg', 'chan1', 'u2', 'keep me')
	`); err != nil {
		t.Fatalf("insert messages: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO attachments (id, message_id, filename, file_url, file_size) VALUES
			('owner-file', 'owner-msg', 'owner.txt', '/uploads/owner.txt', 10),
			('member-file', 'member-msg', 'member.txt', '/uploads/member.txt', 20)
	`); err != nil {
		t.Fatalf("insert attachments: %v", err)
	}

	svc := &fileCleanupService{db: db, fileDeleter: files.NewLocator(t.TempDir(), "")}
	plan, err := svc.CollectUserFiles(ctx, "u1")
	if err != nil {
		t.Fatalf("collect user files: %v", err)
	}

	var collected []string
	for _, ref := range plan.refs {
		collected = append(collected, ref.URL)
	}
	for _, url := range plan.urls {
		collected = append(collected, url)
	}
	joined := strings.Join(collected, "\n")
	if !strings.Contains(joined, "/uploads/owner.txt") {
		t.Fatalf("former owner's attachment was not collected: %s", joined)
	}
	for _, preserved := range []string{"/uploads/member.txt", "/uploads/server-icon.png", "/uploads/server-banner.png"} {
		if strings.Contains(joined, preserved) {
			t.Fatalf("shared server/member asset would be deleted despite ownership transfer: %s", preserved)
		}
	}
}
