package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/akinalp/mqvi/models"
	"github.com/akinalp/mqvi/pkg"
	"github.com/akinalp/mqvi/testutil/dbtest"
)

func TestSQLiteUserRepo_CreateRejectsReservedUsername(t *testing.T) {
	ctx := context.Background()
	db := dbtest.New(t).DB
	repo := NewSQLiteUserRepo(db)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO username_reservations (username, reserved_until)
		VALUES ('HeldName', datetime('now', '+24 hours'))
	`); err != nil {
		t.Fatalf("insert reservation: %v", err)
	}

	user := &models.User{
		Username: "heldname",
		Status:   models.UserStatusOnline,
		Language: "en",
	}
	err := repo.Create(ctx, user)
	if !errors.Is(err, pkg.ErrAlreadyExists) {
		t.Fatalf("Create() error = %v, want ErrAlreadyExists", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE username = 'heldname' COLLATE NOCASE`).Scan(&count); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 0 {
		t.Fatalf("reserved username was created: %d matching users", count)
	}
}

func TestSQLiteUserRepo_UpdateRejectsReservedUsernameAtomically(t *testing.T) {
	ctx := context.Background()
	db := dbtest.New(t).DB
	repo := NewSQLiteUserRepo(db)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO users (id, username, password_hash)
		VALUES ('u1', 'original', 'hash')
	`); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO username_reservations (username, reserved_until)
		VALUES ('HeldName', datetime('now', '+24 hours'))
	`); err != nil {
		t.Fatalf("insert reservation: %v", err)
	}

	user := &models.User{ID: "u1", Username: "heldname"}
	err := repo.Update(ctx, user)
	if !errors.Is(err, pkg.ErrAlreadyExists) {
		t.Fatalf("Update() error = %v, want ErrAlreadyExists", err)
	}

	var username string
	if err := db.QueryRowContext(ctx, `SELECT username FROM users WHERE id = 'u1'`).Scan(&username); err != nil {
		t.Fatalf("read username: %v", err)
	}
	if username != "original" {
		t.Fatalf("username changed despite reservation: %q", username)
	}
}

func TestSQLiteUserRepo_ExpiredUsernameReservationDoesNotBlock(t *testing.T) {
	ctx := context.Background()
	db := dbtest.New(t).DB
	repo := NewSQLiteUserRepo(db)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO username_reservations (username, reserved_until)
		VALUES ('ReleasedName', datetime('now', '-1 hour'))
	`); err != nil {
		t.Fatalf("insert expired reservation: %v", err)
	}

	reserved, err := repo.IsUsernameReserved(ctx, "releasedname")
	if err != nil {
		t.Fatalf("IsUsernameReserved(): %v", err)
	}
	if reserved {
		t.Fatal("expired reservation still blocks the username")
	}
}

func TestSQLiteUserRepo_HardDeleteReservesUsernameFor24Hours(t *testing.T) {
	ctx := context.Background()
	db := dbtest.New(t).DB
	repo := NewSQLiteUserRepo(db)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO users (id, username, password_hash)
		VALUES ('u1', 'OriginalName', 'hash')
	`); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	before := time.Now().UTC()
	if err := repo.HardDeleteUser(ctx, "u1", false); err != nil {
		t.Fatalf("HardDeleteUser(): %v", err)
	}
	after := time.Now().UTC()

	reserved, err := repo.IsUsernameReserved(ctx, "originalname")
	if err != nil {
		t.Fatalf("IsUsernameReserved(): %v", err)
	}
	if !reserved {
		t.Fatal("deleted user's original username was not reserved")
	}

	var reservedUntil time.Time
	if err := db.QueryRowContext(ctx, `
		SELECT reserved_until FROM username_reservations
		WHERE username = 'originalname' COLLATE NOCASE
	`).Scan(&reservedUntil); err != nil {
		t.Fatalf("read reservation expiry: %v", err)
	}
	if reservedUntil.Before(before.Add(23*time.Hour+59*time.Minute)) ||
		reservedUntil.After(after.Add(24*time.Hour+time.Minute)) {
		t.Fatalf("reservation expiry %v is not approximately 24 hours after deletion", reservedUntil)
	}
}
