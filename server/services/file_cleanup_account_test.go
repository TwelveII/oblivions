package services

import (
	"context"
	"testing"

	"github.com/akinalp/mqvi/repository"
	"github.com/akinalp/mqvi/testutil/dbtest"
)

func TestCollectUserFilesIncludesServerReportEvidenceOnTransferredServer(t *testing.T) {
	ctx := context.Background()
	fx := dbtest.New(t)
	reporterID := fx.User("reporter")
	successorID := fx.User("successor")
	serverID := fx.Server(dbtest.ServerSeed{
		ID:      "surviving-server",
		OwnerID: reporterID,
		Name:    "Surviving server",
	})

	if _, err := fx.DB.ExecContext(ctx, `
		INSERT INTO server_members (server_id, user_id, joined_at)
		VALUES (?, ?, '2020-01-01 00:00:00'),
		       (?, ?, '2021-01-01 00:00:00')
	`, serverID, reporterID, serverID, successorID); err != nil {
		t.Fatalf("insert server memberships: %v", err)
	}
	if _, err := fx.DB.ExecContext(ctx, `
		INSERT INTO server_reports (id, reporter_id, server_id, reason, description)
		VALUES ('report-1', ?, ?, 'abuse', 'Report evidence cleanup test')
	`, reporterID, serverID); err != nil {
		t.Fatalf("insert server report: %v", err)
	}
	const evidenceURL = "/api/files/server-reports/report-1/evidence.png"
	if _, err := fx.DB.ExecContext(ctx, `
		INSERT INTO server_report_attachments
			(id, server_report_id, filename, file_url, file_size, mime_type)
		VALUES ('evidence-1', 'report-1', 'evidence.png', ?, 1234, 'image/png')
	`, evidenceURL); err != nil {
		t.Fatalf("insert server report evidence: %v", err)
	}

	cleanup := &fileCleanupService{db: fx.DB}
	plan, err := cleanup.CollectUserFiles(ctx, reporterID)
	if err != nil {
		t.Fatalf("CollectUserFiles(): %v", err)
	}

	found := 0
	for _, ref := range plan.refs {
		if ref.URL == evidenceURL {
			found++
			if ref.UserID != reporterID {
				t.Fatalf("evidence quota owner = %q, want reporter %q", ref.UserID, reporterID)
			}
		}
	}
	if found != 1 {
		t.Fatalf("collected server report evidence %d times, want exactly once", found)
	}

	userRepo := repository.NewSQLiteUserRepo(fx.DB)
	if err := userRepo.HardDeleteUser(ctx, reporterID, false); err != nil {
		t.Fatalf("HardDeleteUser(): %v", err)
	}

	var reportCount, attachmentCount int
	if err := fx.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM server_reports WHERE id = 'report-1'`,
	).Scan(&reportCount); err != nil {
		t.Fatalf("count server reports: %v", err)
	}
	if err := fx.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM server_report_attachments WHERE id = 'evidence-1'`,
	).Scan(&attachmentCount); err != nil {
		t.Fatalf("count server report attachments: %v", err)
	}
	if reportCount != 0 || attachmentCount != 0 {
		t.Fatalf("report rows survived deletion: reports=%d attachments=%d", reportCount, attachmentCount)
	}

	var newOwner string
	if err := fx.DB.QueryRowContext(ctx,
		`SELECT owner_id FROM servers WHERE id = ?`, serverID,
	).Scan(&newOwner); err != nil {
		t.Fatalf("read transferred server owner: %v", err)
	}
	if newOwner != successorID {
		t.Fatalf("server owner = %q, want successor %q", newOwner, successorID)
	}
}
