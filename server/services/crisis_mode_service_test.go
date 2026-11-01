package services

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"

	_ "modernc.org/sqlite"
)

type crisisTestFileCleanup struct {
	collectErr error
	collected  []string
	executed   int
}

func (f *crisisTestFileCleanup) CollectChannelFiles(context.Context, string) (*CleanupPlan, error) {
	return nil, nil
}
func (f *crisisTestFileCleanup) CollectServerFiles(context.Context, string) (*CleanupPlan, error) {
	return nil, nil
}
func (f *crisisTestFileCleanup) CollectUserFiles(context.Context, string) (*CleanupPlan, error) {
	return nil, nil
}
func (f *crisisTestFileCleanup) CollectUserMessageFiles(_ context.Context, userID string) (*CleanupPlan, error) {
	f.collected = append(f.collected, userID)
	if f.collectErr != nil {
		return nil, f.collectErr
	}
	return &CleanupPlan{}, nil
}
func (f *crisisTestFileCleanup) Execute(plan *CleanupPlan) {
	if plan != nil {
		f.executed++
	}
}

type crisisTestConnections struct{ disconnected []string }

func (c *crisisTestConnections) DisconnectUser(userID string) {
	c.disconnected = append(c.disconnected, userID)
}

func crisisTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func crisisExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

func TestCrisisModeSetSiteEnabledRevokesOnlyOrdinaryUsers(t *testing.T) {
	db := crisisTestDB(t)
	crisisExec(t, db, `CREATE TABLE crisis_mode_state (
		id INTEGER PRIMARY KEY, enabled INTEGER NOT NULL, site_enabled INTEGER NOT NULL, last_run_at DATETIME
	)`)
	crisisExec(t, db, `INSERT INTO crisis_mode_state (id, enabled, site_enabled) VALUES (1, 0, 1)`)
	crisisExec(t, db, `CREATE TABLE users (id TEXT PRIMARY KEY, is_platform_admin INTEGER NOT NULL, token_version INTEGER NOT NULL)`)
	crisisExec(t, db, `CREATE TABLE sessions (id TEXT PRIMARY KEY, user_id TEXT NOT NULL)`)
	crisisExec(t, db, `INSERT INTO users VALUES ('member-a', 0, 4), ('admin-a', 1, 7), ('member-b', 0, 2)`)
	crisisExec(t, db, `INSERT INTO sessions VALUES ('s-member-a', 'member-a'), ('s-admin-a', 'admin-a'), ('s-member-b', 'member-b')`)

	connections := &crisisTestConnections{}
	svc := NewCrisisModeService(db, &crisisTestFileCleanup{}, connections)

	if err := svc.SetSiteEnabled(context.Background(), false); err != nil {
		t.Fatal(err)
	}

	var siteEnabled int
	if err := db.QueryRow(`SELECT site_enabled FROM crisis_mode_state WHERE id = 1`).Scan(&siteEnabled); err != nil {
		t.Fatal(err)
	}
	if siteEnabled != 0 {
		t.Fatalf("site_enabled = %d, want 0", siteEnabled)
	}

	rows, err := db.Query(`SELECT id, token_version FROM users ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	gotVersions := map[string]int{}
	for rows.Next() {
		var id string
		var version int
		if err := rows.Scan(&id, &version); err != nil {
			t.Fatal(err)
		}
		gotVersions[id] = version
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	wantVersions := map[string]int{"admin-a": 7, "member-a": 5, "member-b": 3}
	if !reflect.DeepEqual(gotVersions, wantVersions) {
		t.Fatalf("token versions = %#v, want %#v", gotVersions, wantVersions)
	}

	var remainingSessions []string
	rows, err = db.Query(`SELECT user_id FROM sessions ORDER BY user_id`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		remainingSessions = append(remainingSessions, id)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(remainingSessions, []string{"admin-a"}) {
		t.Fatalf("remaining sessions = %#v, want only admin-a", remainingSessions)
	}
	if !reflect.DeepEqual(connections.disconnected, []string{"member-a", "member-b"}) {
		t.Fatalf("disconnected users = %#v", connections.disconnected)
	}

	if err := svc.SetSiteEnabled(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(connections.disconnected, []string{"member-a", "member-b"}) {
		t.Fatalf("restoring site unexpectedly disconnected users: %#v", connections.disconnected)
	}
}

func TestCrisisModeCleanNowDeletesOnlyConversationRowsAndRetainsReports(t *testing.T) {
	db := crisisTestDB(t)
	crisisExec(t, db, `CREATE TABLE users (id TEXT PRIMARY KEY)`)
	crisisExec(t, db, `INSERT INTO users VALUES ('u1'), ('u2')`)
	crisisExec(t, db, `CREATE TABLE messages (id TEXT PRIMARY KEY)`)
	crisisExec(t, db, `CREATE TABLE dm_messages (id TEXT PRIMARY KEY)`)
	crisisExec(t, db, `CREATE TABLE voice_messages (id TEXT PRIMARY KEY)`)
	crisisExec(t, db, `CREATE TABLE crisis_mode_state (
		id INTEGER PRIMARY KEY, enabled INTEGER NOT NULL, site_enabled INTEGER NOT NULL, last_run_at DATETIME
	)`)
	crisisExec(t, db, `INSERT INTO crisis_mode_state (id, enabled, site_enabled) VALUES (1, 0, 1)`)
	crisisExec(t, db, `CREATE TABLE reports (id TEXT PRIMARY KEY)`)
	crisisExec(t, db, `CREATE TABLE report_attachments (id TEXT PRIMARY KEY, report_id TEXT NOT NULL)`)
	crisisExec(t, db, `INSERT INTO messages VALUES ('m1'), ('m2')`)
	crisisExec(t, db, `INSERT INTO dm_messages VALUES ('d1')`)
	crisisExec(t, db, `INSERT INTO voice_messages VALUES ('v1'), ('v2'), ('v3')`)
	crisisExec(t, db, `INSERT INTO reports VALUES ('r1')`)
	crisisExec(t, db, `INSERT INTO report_attachments VALUES ('evidence1', 'r1')`)

	files := &crisisTestFileCleanup{}
	svc := NewCrisisModeService(db, files)
	result, err := svc.CleanNow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.ChannelMessages != 2 || result.DMMessages != 1 || result.VoiceMessages != 3 {
		t.Fatalf("cleanup counts = %+v, want channel=2 dm=1 voice=3", result)
	}
	if !reflect.DeepEqual(files.collected, []string{"u1", "u2"}) || files.executed != 2 {
		t.Fatalf("file cleanup collected=%v executed=%d", files.collected, files.executed)
	}

	for _, table := range []string{"messages", "dm_messages", "voice_messages"} {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Errorf("%s has %d rows after cleanup, want 0", table, count)
		}
	}
	var reports, evidence int
	if err := db.QueryRow(`SELECT COUNT(*) FROM reports`).Scan(&reports); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM report_attachments`).Scan(&evidence); err != nil {
		t.Fatal(err)
	}
	if reports != 1 || evidence != 1 {
		t.Fatalf("reports/evidence = %d/%d, want 1/1", reports, evidence)
	}
	var lastRun sql.NullString
	if err := db.QueryRow(`SELECT last_run_at FROM crisis_mode_state WHERE id = 1`).Scan(&lastRun); err != nil {
		t.Fatal(err)
	}
	if !lastRun.Valid {
		t.Fatal("successful cleanup did not stamp last_run_at")
	}
}

func TestCrisisModeCleanNowCollectionFailurePreservesMessages(t *testing.T) {
	db := crisisTestDB(t)
	crisisExec(t, db, `CREATE TABLE users (id TEXT PRIMARY KEY)`)
	crisisExec(t, db, `INSERT INTO users VALUES ('u1')`)
	crisisExec(t, db, `CREATE TABLE messages (id TEXT PRIMARY KEY)`)
	crisisExec(t, db, `CREATE TABLE dm_messages (id TEXT PRIMARY KEY)`)
	crisisExec(t, db, `CREATE TABLE voice_messages (id TEXT PRIMARY KEY)`)
	crisisExec(t, db, `CREATE TABLE crisis_mode_state (
		id INTEGER PRIMARY KEY, enabled INTEGER NOT NULL, site_enabled INTEGER NOT NULL, last_run_at DATETIME
	)`)
	crisisExec(t, db, `INSERT INTO crisis_mode_state (id, enabled, site_enabled) VALUES (1, 0, 1)`)
	crisisExec(t, db, `INSERT INTO messages VALUES ('m1')`)

	files := &crisisTestFileCleanup{collectErr: fmt.Errorf("storage unavailable")}
	svc := NewCrisisModeService(db, files)
	if _, err := svc.CleanNow(context.Background()); err == nil {
		t.Fatal("CleanNow succeeded despite file collection failure")
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM messages`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("message count = %d after failed collection, want 1", count)
	}
	if files.executed != 0 {
		t.Fatalf("executed file cleanup %d times after failed collection", files.executed)
	}
}
