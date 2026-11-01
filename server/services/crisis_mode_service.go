package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"
)

// CrisisModeState is the administrator-visible state of the independent conversation-retention
// worker. Reports and their evidence are intentionally retained; only chat messages and their
// message attachments are removed.
type CrisisModeState struct {
	Enabled   bool       `json:"enabled"`
	LastRunAt *time.Time `json:"last_run_at"`
}

type CrisisModeCleanResult struct {
	ChannelMessages int64 `json:"channel_messages"`
	DMMessages      int64 `json:"dm_messages"`
	VoiceMessages   int64 `json:"voice_messages"`
}

// CrisisModeService owns the deliberately destructive conversation-retention path. It is
// separate from CleanupService so routine orphan/session cleanup can never wipe conversations.
type CrisisModeService interface {
	Start()
	Stop()
	GetState(ctx context.Context) (*CrisisModeState, error)
	SetEnabled(ctx context.Context, enabled bool) error
	CleanNow(ctx context.Context) (*CrisisModeCleanResult, error)
}

type crisisModeService struct {
	db         *sql.DB
	fileCleanup FileCleanupService
	mu         sync.Mutex
	cancel     context.CancelFunc
	done       chan struct{}
}

func NewCrisisModeService(db *sql.DB, fileCleanup FileCleanupService) CrisisModeService {
	return &crisisModeService{db: db, fileCleanup: fileCleanup}
}

func (s *crisisModeService) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.done = make(chan struct{})
	go s.loop(ctx)
}

func (s *crisisModeService) Stop() {
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	s.cancel, s.done = nil, nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
		<-done
	}
}

func (s *crisisModeService) loop(ctx context.Context) {
	defer close(s.done)
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			state, err := s.GetState(ctx)
			if err != nil {
				log.Printf("[crisis] state read failed: %v", err)
				continue
			}
			if !state.Enabled || (state.LastRunAt != nil && time.Since(*state.LastRunAt) < 24*time.Hour) {
				continue
			}
			runCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
			if _, err := s.CleanNow(runCtx); err != nil {
				log.Printf("[crisis] scheduled conversation cleanup failed: %v", err)
			}
			cancel()
		}
	}
}

func (s *crisisModeService) GetState(ctx context.Context) (*CrisisModeState, error) {
	var enabled int
	var last sql.NullTime
	if err := s.db.QueryRowContext(ctx, `SELECT enabled, last_run_at FROM crisis_mode_state WHERE id = 1`).Scan(&enabled, &last); err != nil {
		return nil, fmt.Errorf("read crisis mode state: %w", err)
	}
	state := &CrisisModeState{Enabled: enabled == 1}
	if last.Valid {
		t := last.Time.UTC()
		state.LastRunAt = &t
	}
	return state, nil
}

func (s *crisisModeService) SetEnabled(ctx context.Context, enabled bool) error {
	value := 0
	if enabled {
		value = 1
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE crisis_mode_state
		SET enabled = ?, last_run_at = CASE WHEN ? = 1 THEN CURRENT_TIMESTAMP ELSE last_run_at END
		WHERE id = 1`, value, value)
	if err != nil {
		return fmt.Errorf("update crisis mode state: %w", err)
	}
	return nil
}

// CleanNow collects all message-file references before the transaction, deletes all conversation
// rows atomically, then removes files/releases quota. If the DB transaction fails, no files are
// deleted. This does not delete report evidence, accounts, channels, or server configuration.
func (s *crisisModeService) CleanNow(ctx context.Context) (*CrisisModeCleanResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.QueryContext(ctx, `SELECT id FROM users`)
	if err != nil {
		return nil, fmt.Errorf("list users for crisis cleanup: %w", err)
	}
	var userIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan user for crisis cleanup: %w", err)
		}
		userIDs = append(userIDs, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate users for crisis cleanup: %w", err)
	}
	rows.Close()

	var plans []*CleanupPlan
	for _, userID := range userIDs {
		plan, err := s.fileCleanup.CollectUserMessageFiles(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("collect message files for user %s: %w", userID, err)
		}
		plans = append(plans, plan)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin crisis cleanup transaction: %w", err)
	}
	defer tx.Rollback()

	result := &CrisisModeCleanResult{}
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM messages`).Scan(&result.ChannelMessages); err != nil {
		return nil, fmt.Errorf("count channel messages: %w", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM dm_messages`).Scan(&result.DMMessages); err != nil {
		return nil, fmt.Errorf("count DM messages: %w", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM voice_messages`).Scan(&result.VoiceMessages); err != nil {
		return nil, fmt.Errorf("count voice messages: %w", err)
	}
	for _, table := range []string{"messages", "dm_messages", "voice_messages"} {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return nil, fmt.Errorf("delete %s: %w", table, err)
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE crisis_mode_state SET last_run_at = CURRENT_TIMESTAMP WHERE id = 1`); err != nil {
		return nil, fmt.Errorf("stamp crisis cleanup: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit crisis cleanup: %w", err)
	}
	for _, plan := range plans {
		s.fileCleanup.Execute(plan)
	}
	log.Printf("[crisis] conversation cleanup complete: channel=%d dm=%d voice=%d", result.ChannelMessages, result.DMMessages, result.VoiceMessages)
	return result, nil
}
