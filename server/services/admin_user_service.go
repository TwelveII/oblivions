// Package services — AdminUserService: platform-level user management.
//
// Handles platform-wide ban and hard delete (distinct from server-scoped MemberService.BanUser).
// Email notifications are optional — sent if reason is provided and user has an email.
// Email errors do not roll back the action.
package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/akinalp/mqvi/database"
	"github.com/akinalp/mqvi/models"
	"github.com/akinalp/mqvi/pkg"
	"github.com/akinalp/mqvi/pkg/email"
	"github.com/akinalp/mqvi/repository"
	"github.com/akinalp/mqvi/ws"
)

// AdminUserService handles platform-level user ban and deletion.
type AdminUserService interface {
	PlatformBanUser(ctx context.Context, adminUserID, targetUserID, reason string, deleteMessages bool) error
	PlatformUnbanUser(ctx context.Context, adminUserID, targetUserID string) error
	// SoftDeleteUser marks the user soft-deleted (recoverable for 30 days).
	SoftDeleteUser(ctx context.Context, adminUserID, targetUserID, reason string) error
	// HardDeleteUser anonymizes the user (tombstone). Files cleaned, quota released.
	HardDeleteUser(ctx context.Context, adminUserID, targetUserID, reason string) error
	// RestoreUser un-soft-deletes a user (admin override).
	RestoreUser(ctx context.Context, adminUserID, targetUserID string) error
	SetPlatformAdmin(ctx context.Context, adminUserID, targetUserID string, isAdmin bool) error
	// ExpireSoftDeletedUser tombstones a user whose 30-day soft-delete window has elapsed.
	// Called only by the embedded cleanup worker — no admin actor, no email
	// notification (the user already received one when they soft-deleted), no
	// platform-admin guard (cleanup target is by definition a soft-deleted user,
	// not a live admin). Must verify deleted_at + TTL defensively in case the
	// caller passes a stale ID.
	ExpireSoftDeletedUser(ctx context.Context, targetUserID string) error
}

type adminUserService struct {
	db          *sql.DB
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	serverRepo  repository.ServerRepository
	// Broadcaster (not just ClientManager) — needed for BroadcastToServer when
	// hard-deleting a user with owned servers.
	hub         ws.BroadcastAndManage
	voiceKit    VoiceDisconnecter // ISP defined in member_service.go
	emailSender email.EmailSender // optional, nil = no emails
	fileCleanup FileCleanupService
}

func NewAdminUserService(
	db *sql.DB,
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
	serverRepo repository.ServerRepository,
	hub ws.BroadcastAndManage,
	voiceKit VoiceDisconnecter,
	emailSender email.EmailSender,
	fileCleanup FileCleanupService,
) AdminUserService {
	return &adminUserService{
		db:          db,
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		serverRepo:  serverRepo,
		hub:         hub,
		voiceKit:    voiceKit,
		emailSender: emailSender,
		fileCleanup: fileCleanup,
	}
}

func (s *adminUserService) PlatformBanUser(ctx context.Context, adminUserID, targetUserID, reason string, deleteMessages bool) error {
	if adminUserID == targetUserID {
		return fmt.Errorf("%w: cannot ban yourself", pkg.ErrBadRequest)
	}

	target, err := s.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return fmt.Errorf("target user not found: %w", err)
	}

	if target.IsPlatformAdmin {
		return fmt.Errorf("%w: cannot ban a platform admin", pkg.ErrForbidden)
	}

	if target.IsPlatformBanned {
		return fmt.Errorf("%w: user is already banned", pkg.ErrBadRequest)
	}

	// Collect file refs BEFORE any mutations so a failed collect doesn't leave partial state
	var msgPlan *CleanupPlan
	if deleteMessages {
		var collectErr error
		msgPlan, collectErr = s.fileCleanup.CollectUserMessageFiles(ctx, targetUserID)
		if collectErr != nil {
			return fmt.Errorf("failed to collect user message files: %w", collectErr)
		}
	}

	// Atomic: user flag + durable ban record in one transaction
	var banEmail string
	if target.Email != nil {
		banEmail = *target.Email
	}
	if err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		txRepo := repository.NewSQLiteUserRepo(tx)
		if err := txRepo.PlatformBan(ctx, targetUserID, reason, adminUserID); err != nil {
			return fmt.Errorf("failed to ban user: %w", err)
		}
		if err := txRepo.InsertPlatformBan(ctx, banEmail, target.Username, targetUserID, reason, adminUserID); err != nil {
			return fmt.Errorf("failed to insert platform ban record: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	if deleteMessages {
		if err := s.userRepo.DeleteAllMessagesByUser(ctx, targetUserID); err != nil {
			return fmt.Errorf("failed to delete user messages: %w", err)
		}
		s.fileCleanup.Execute(msgPlan)
	}

	s.voiceKit.DisconnectUser(targetUserID)
	s.hub.DisconnectUser(targetUserID)

	// Best-effort email notification
	if reason != "" && target.Email != nil && s.emailSender != nil {
		if emailErr := s.emailSender.SendPlatformBanNotification(ctx, *target.Email, reason); emailErr != nil {
			log.Printf("[admin] failed to send ban notification email to %s: %v", targetUserID, emailErr)
		}
	}

	return nil
}

func (s *adminUserService) PlatformUnbanUser(ctx context.Context, adminUserID, targetUserID string) error {
	if adminUserID == targetUserID {
		return fmt.Errorf("%w: cannot unban yourself", pkg.ErrBadRequest)
	}

	// User row may be missing (hard-deleted in old code path) or anonymized
	// (tombstone, is_hard_deleted=1). Either way the only meaningful artifact
	// of the ban that survives is the platform_bans table — that's what we clear.
	target, userErr := s.userRepo.GetByID(ctx, targetUserID)
	userExists := userErr == nil
	if userErr != nil && !errors.Is(userErr, pkg.ErrNotFound) {
		return fmt.Errorf("failed to look up user: %w", userErr)
	}

	// For a live (non-tombstone) user, require the user-level ban flag to be set.
	// For tombstones, the flag is residual — only platform_bans existence matters.
	if userExists && !target.IsHardDeleted && !target.IsPlatformBanned {
		return fmt.Errorf("%w: user is not banned", pkg.ErrBadRequest)
	}

	if !userExists || target.IsHardDeleted {
		// Verify a platform_bans record actually exists for this user ID
		banned, checkErr := s.userRepo.IsPlatformBannedByUserID(ctx, targetUserID)
		if checkErr != nil {
			return fmt.Errorf("failed to check platform ban: %w", checkErr)
		}
		if !banned {
			return fmt.Errorf("%w: no ban record found", pkg.ErrNotFound)
		}
	}

	// Atomic: clear user flag (if user exists) + remove durable ban record
	return database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		txRepo := repository.NewSQLiteUserRepo(tx)
		if userExists {
			if err := txRepo.PlatformUnban(ctx, targetUserID); err != nil {
				return fmt.Errorf("failed to unban user: %w", err)
			}
		}
		if err := txRepo.DeletePlatformBan(ctx, targetUserID); err != nil {
			return fmt.Errorf("failed to delete platform ban record: %w", err)
		}
		return nil
	})
}

// SoftDeleteUser marks the user soft-deleted (recoverable for 30 days).
// Disconnects sessions/realtime so the user is logged out immediately.
func (s *adminUserService) SoftDeleteUser(ctx context.Context, adminUserID, targetUserID, reason string) error {
	if adminUserID == targetUserID {
		return fmt.Errorf("%w: cannot delete yourself", pkg.ErrBadRequest)
	}

	target, err := s.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return fmt.Errorf("target user not found: %w", err)
	}

	if target.IsPlatformAdmin {
		return fmt.Errorf("%w: cannot delete a platform admin", pkg.ErrForbidden)
	}

	if target.DeletedAt != nil {
		return fmt.Errorf("%w: user is already deleted", pkg.ErrBadRequest)
	}

	if err := s.userRepo.SoftDelete(ctx, targetUserID, true); err != nil {
		return fmt.Errorf("failed to soft delete user: %w", err)
	}

	// Invalidate refresh tokens — auth middleware now also rejects via GetActiveByID,
	// but keeping sessions around contradicts the "logged out" guarantee and bloats the
	// table. Best-effort: log on failure so soft-delete still succeeds.
	if err := s.sessionRepo.DeleteByUserID(ctx, targetUserID); err != nil {
		log.Printf("[admin] failed to delete sessions for soft-deleted user %s: %v", targetUserID, err)
	}

	s.voiceKit.DisconnectUser(targetUserID)
	s.hub.DisconnectUser(targetUserID)

	if reason != "" && target.Email != nil && s.emailSender != nil {
		if emailErr := s.emailSender.SendAccountDeleteNotification(ctx, *target.Email, reason); emailErr != nil {
			log.Printf("[admin] failed to send soft-delete notification email to %s: %v", targetUserID, emailErr)
		}
	}

	log.Printf("[admin] admin %s soft-deleted user %s (%s)", adminUserID, targetUserID, target.Username)
	return nil
}

// RestoreUser un-soft-deletes a user (admin override). Tombstones (is_hard_deleted=1)
// are not restorable.
func (s *adminUserService) RestoreUser(ctx context.Context, adminUserID, targetUserID string) error {
	if adminUserID == targetUserID {
		return fmt.Errorf("%w: cannot restore yourself", pkg.ErrBadRequest)
	}

	target, err := s.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return fmt.Errorf("target user not found: %w", err)
	}

	if target.DeletedAt == nil {
		return fmt.Errorf("%w: user is not deleted", pkg.ErrBadRequest)
	}

	if target.IsHardDeleted {
		return fmt.Errorf("%w: tombstone users cannot be restored", pkg.ErrForbidden)
	}

	if err := s.userRepo.Restore(ctx, targetUserID); err != nil {
		return fmt.Errorf("failed to restore user: %w", err)
	}

	log.Printf("[admin] admin %s restored user %s (%s)", adminUserID, targetUserID, target.Username)
	return nil
}

// HardDeleteUser permanently deletes a user and all associated data.
// Email notification is sent BEFORE deletion (email address is lost after delete).
func (s *adminUserService) HardDeleteUser(ctx context.Context, adminUserID, targetUserID, reason string) error {
	if adminUserID == targetUserID {
		return fmt.Errorf("%w: cannot delete yourself", pkg.ErrBadRequest)
	}

	target, err := s.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return fmt.Errorf("target user not found: %w", err)
	}

	if target.IsPlatformAdmin {
		return fmt.Errorf("%w: cannot delete a platform admin", pkg.ErrForbidden)
	}

	// Send email BEFORE deletion (address is lost after)
	if reason != "" && target.Email != nil && s.emailSender != nil {
		if emailErr := s.emailSender.SendAccountDeleteNotification(ctx, *target.Email, reason); emailErr != nil {
			log.Printf("[admin] failed to send delete notification email to %s: %v", targetUserID, emailErr)
		}
	}

	// Phase 1: collect ALL user file refs (including owned server files)
	plan, collectErr := s.fileCleanup.CollectUserFiles(ctx, targetUserID)
	if collectErr != nil {
		return fmt.Errorf("failed to collect user files: %w", collectErr)
	}

	// Phase 1b: collect owned server IDs so we can broadcast server_delete
	// to their members before the cascade DELETE wipes server_members rows
	// (BroadcastToServer needs the membership table to resolve recipients).
	ownedServerIDs, ownedErr := s.serverRepo.ListActiveServerIDsByOwner(ctx, targetUserID)
	if ownedErr != nil {
		return fmt.Errorf("failed to list owned servers: %w", ownedErr)
	}
	for _, sid := range ownedServerIDs {
		s.hub.BroadcastToServer(sid, ws.Event{
			Op:   ws.OpServerDelete,
			Data: map[string]string{"id": sid},
		})
	}

	// Disconnect realtime connections before DB delete to avoid race conditions
	s.voiceKit.DisconnectUser(targetUserID)
	s.hub.DisconnectUser(targetUserID)

	// Phase 2: DB delete (CASCADE removes user data, owned servers, etc.)
	if err := s.userRepo.HardDeleteUser(ctx, targetUserID, true); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Phase 3: delete files from disk + release other users' quota
	s.fileCleanup.Execute(plan)

	return nil
}

// ExpireSoftDeletedUser is the worker-driven path. Reuses the same collect →
// broadcast → repo.HardDeleteUser → execute sequence as the admin variant but
// skips admin actor checks, email notification (already sent at soft-delete),
// and the platform-admin guard (a platform admin cannot reach this path because
// they cannot soft-delete themselves). Defensive TTL check ensures the caller
// did not race a restore.
func (s *adminUserService) ExpireSoftDeletedUser(ctx context.Context, targetUserID string) error {
	target, err := s.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return fmt.Errorf("expire target user: %w", err)
	}
	if target.DeletedAt == nil {
		return fmt.Errorf("%w: user is not soft-deleted", pkg.ErrBadRequest)
	}
	if target.IsHardDeleted {
		return fmt.Errorf("%w: user is already a tombstone", pkg.ErrBadRequest)
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -models.SoftDeleteTTLDays)
	if target.DeletedAt.After(cutoff) {
		return fmt.Errorf("%w: soft-delete TTL not yet elapsed (deleted_at=%s)", pkg.ErrBadRequest, target.DeletedAt.Format(time.RFC3339))
	}

	plan, collectErr := s.fileCleanup.CollectUserFiles(ctx, targetUserID)
	if collectErr != nil {
		return fmt.Errorf("collect user files: %w", collectErr)
	}

	ownedServerIDs, ownedErr := s.serverRepo.ListActiveServerIDsByOwner(ctx, targetUserID)
	if ownedErr != nil {
		return fmt.Errorf("list owned servers: %w", ownedErr)
	}
	for _, sid := range ownedServerIDs {
		s.hub.BroadcastToServer(sid, ws.Event{
			Op:   ws.OpServerDelete,
			Data: map[string]string{"id": sid},
		})
	}

	// Defensive: a soft-deleted user is offline by definition, but disconnect
	// keeps the contract identical to the admin path.
	s.voiceKit.DisconnectUser(targetUserID)
	s.hub.DisconnectUser(targetUserID)

	if err := s.userRepo.HardDeleteUser(ctx, targetUserID, target.DeletedByAdmin); err != nil {
		return fmt.Errorf("tombstone user: %w", err)
	}

	s.fileCleanup.Execute(plan)
	return nil
}

func (s *adminUserService) SetPlatformAdmin(ctx context.Context, adminUserID, targetUserID string, isAdmin bool) error {
	if adminUserID == targetUserID {
		return fmt.Errorf("%w: cannot modify your own admin status", pkg.ErrBadRequest)
	}

	if _, err := s.userRepo.GetByID(ctx, targetUserID); err != nil {
		return fmt.Errorf("target user not found: %w", err)
	}

	if err := s.userRepo.SetPlatformAdmin(ctx, targetUserID, isAdmin); err != nil {
		return fmt.Errorf("failed to update admin status: %w", err)
	}

	return nil
}
