package database

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no CGO)
)

// recoverableErrors lists error patterns that can be safely skipped
// when re-running a partially applied migration (e.g. "duplicate column name").
var recoverableErrors = []string{
	"duplicate column name",
}

type DB struct {
	Conn *sql.DB
}

// New opens a SQLite connection and runs pending migrations.
func New(dbPath string, migrationsFS fs.FS) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// foreign_keys=on (off by default in SQLite), journal_mode=WAL for concurrent r/w,
	// busy_timeout=5000ms lets concurrent writers wait instead of returning SQLITE_BUSY immediately.
	conn, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Connection pool settings for SQLite WAL mode:
	// - MaxOpenConns=4 allows concurrent reads (WAL serializes writes internally)
	// - MaxIdleConns=2 keeps warm connections ready
	// - ConnMaxLifetime=0 means connections are reused indefinitely
	conn.SetMaxOpenConns(4)
	conn.SetMaxIdleConns(2)
	conn.SetConnMaxLifetime(0)

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{Conn: conn}

	if err := db.runMigrations(migrationsFS); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("[database] connected and migrations applied")
	return db, nil
}

func (db *DB) Close() error {
	return db.Conn.Close()
}

// runMigrations applies SQL files from migrationsFS in alphabetical order.
// Uses schema_migrations table to track which files have been applied.
func (db *DB) runMigrations(migrationsFS fs.FS) error {
	if _, err := db.Conn.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	entries, err := fs.ReadDir(migrationsFS, ".")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var sqlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			sqlFiles = append(sqlFiles, entry.Name())
		}
	}

	sort.Strings(sqlFiles)

	applied := make(map[string]bool)
	rows, err := db.Conn.Query("SELECT filename FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("failed to query schema_migrations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("failed to scan migration row: %w", err)
		}
		applied[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to iterate migration rows: %w", err)
	}

	// Bootstrap: if schema_migrations is empty but tables already exist,
	// mark all migrations as applied to avoid re-running ALTER TABLE etc.
	if len(applied) == 0 {
		var tableCount int
		if err := db.Conn.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users'",
		).Scan(&tableCount); err != nil {
			return fmt.Errorf("failed to check existing tables: %w", err)
		}

		if tableCount > 0 {
			for _, file := range sqlFiles {
				if _, err := db.Conn.Exec(
					"INSERT INTO schema_migrations (filename) VALUES (?)", file,
				); err != nil {
					return fmt.Errorf("failed to bootstrap migration %s: %w", file, err)
				}
				applied[file] = true
			}
			log.Printf("[database] bootstrapped %d existing migrations", len(sqlFiles))
			return nil
		}
	}

	for _, file := range sqlFiles {
		if applied[file] {
			continue
		}

		content, err := fs.ReadFile(migrationsFS, file)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", file, err)
		}

		// Run each migration in a transaction so partial application is impossible.
		// If any statement fails, the entire migration is rolled back.
		if err := db.execMigrationInTx(file, string(content)); err != nil {
			return err
		}

		log.Printf("[database] migration applied: %s", file)
	}

	return nil
}

// execMigrationInTx runs a migration file inside a single transaction.
// PRAGMA statements are executed outside the transaction (SQLite requires this),
// all other statements run inside the tx. On success, the migration filename
// is recorded in schema_migrations within the same tx.
func (db *DB) execMigrationInTx(filename, content string) (err error) {
	statements := splitStatements(content)

	// Phase 1: execute PRAGMA statements outside the transaction.
	// SQLite PRAGMAs like journal_mode and foreign_keys cannot run inside a transaction.
	var nonPragma []string
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if isPragma(stmt) {
			if _, execErr := db.Conn.Exec(stmt); execErr != nil {
				return fmt.Errorf("failed to execute PRAGMA in %s: %w", filename, execErr)
			}
		} else {
			nonPragma = append(nonPragma, stmt)
		}
	}

	// Phase 2: execute remaining statements in a transaction.
	tx, err := db.Conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin migration tx for %s: %w", filename, err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	for i, stmt := range nonPragma {
		if _, execErr := tx.Exec(stmt); execErr != nil {
			errMsg := execErr.Error()
			recoverable := false
			for _, pattern := range recoverableErrors {
				if strings.Contains(errMsg, pattern) {
					recoverable = true
					break
				}
			}

			if recoverable {
				log.Printf("[database] %s: statement %d skipped (recoverable: %s)", filename, i+1, errMsg)
				continue
			}

			return fmt.Errorf("failed to execute migration %s (statement %d): %w", filename, i+1, execErr)
		}
	}

	if _, err = tx.Exec("INSERT INTO schema_migrations (filename) VALUES (?)", filename); err != nil {
		return fmt.Errorf("failed to record migration %s: %w", filename, err)
	}

	return tx.Commit()
}

// isPragma returns true if the statement is a PRAGMA command.
func isPragma(stmt string) bool {
	upper := strings.ToUpper(strings.TrimSpace(stmt))
	return strings.HasPrefix(upper, "PRAGMA ")
}

// splitStatements splits SQL by semicolons, respecting string literals,
// BEGIN...END blocks (for triggers), and -- line comments.
func splitStatements(sql string) []string {
	var statements []string
	var current strings.Builder
	inString := false
	beginDepth := 0

	for i := 0; i < len(sql); i++ {
		ch := sql[i]

		// Skip -- line comments (outside strings): advance to end of line
		if !inString && ch == '-' && i+1 < len(sql) && sql[i+1] == '-' {
			for i < len(sql) && sql[i] != '\n' {
				i++
			}
			// Write the newline to preserve line structure
			if i < len(sql) {
				current.WriteByte('\n')
			}
			continue
		}

		if ch == '\'' {
			if inString && i+1 < len(sql) && sql[i+1] == '\'' {
				current.WriteByte(ch)
				current.WriteByte(sql[i+1])
				i++
				continue
			}
			inString = !inString
		}

		if !inString {
			if matchKeyword(sql, i, "BEGIN") {
				beginDepth++
			}
			if matchKeyword(sql, i, "END") && beginDepth > 0 {
				beginDepth--
			}
		}

		if ch == ';' && !inString && beginDepth == 0 {
			s := strings.TrimSpace(current.String())
			if s != "" {
				statements = append(statements, s)
			}
			current.Reset()
			continue
		}

		current.WriteByte(ch)
	}

	s := strings.TrimSpace(current.String())
	if s != "" {
		statements = append(statements, s)
	}

	return statements
}

// matchKeyword checks for a case-insensitive keyword at the given position
// with word-boundary checks on both sides.
func matchKeyword(sql string, pos int, keyword string) bool {
	if pos+len(keyword) > len(sql) {
		return false
	}
	if pos > 0 && isIdentChar(sql[pos-1]) {
		return false
	}
	for j := 0; j < len(keyword); j++ {
		c := sql[pos+j]
		if c >= 'a' && c <= 'z' {
			c -= 32
		}
		if c != keyword[j] {
			return false
		}
	}
	afterIdx := pos + len(keyword)
	if afterIdx < len(sql) && isIdentChar(sql[afterIdx]) {
		return false
	}
	return true
}

func isIdentChar(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '_'
}
