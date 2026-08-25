package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteStore provides persistence for tokens and refresh profiles.
type SQLiteStore struct {
	mu     sync.Mutex
	db     *sql.DB
	DBPath string
}

const sqliteBusyRetryCount = 6

func sqliteBusy(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") || strings.Contains(msg, "sqlite_busy") || strings.Contains(msg, "busy")
}

func sqliteRetryDelay(attempt int) time.Duration {
	return time.Duration(100*(1<<attempt)) * time.Millisecond
}

// NewSQLiteStore opens (or creates) a SQLite database.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir for db: %w", err)
	}
	// DELETE journal mode is slower than WAL, but it is far more predictable on
	// Docker bind mounts and Windows-hosted volumes where WAL sidecar files can
	// cause token imports to appear flaky.
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=DELETE&_busy_timeout=15000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	s := &SQLiteStore{db: db, DBPath: dbPath}
	// The store already serializes operations. Limiting the driver to one
	// connection prevents sql.DB from creating a second connection that can
	// contend with a writer between two serialized calls.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLiteStore) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS tokens (
			id TEXT PRIMARY KEY,
			data TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS refresh_profiles (
			id TEXT PRIMARY KEY,
			data TEXT NOT NULL
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// --------------- Tokens ---------------

// LoadTokens reads all tokens from the database.
func (s *SQLiteStore) LoadTokens() ([]map[string]interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for attempt := 0; attempt < sqliteBusyRetryCount; attempt++ {
		rows, err := s.db.Query("SELECT data FROM tokens ORDER BY id")
		if err != nil {
			if sqliteBusy(err) && attempt+1 < sqliteBusyRetryCount {
				time.Sleep(sqliteRetryDelay(attempt))
				continue
			}
			return nil, err
		}
		var tokens []map[string]interface{}
		for rows.Next() {
			var raw string
			if err := rows.Scan(&raw); err != nil {
				continue
			}
			var item map[string]interface{}
			if err := json.Unmarshal([]byte(raw), &item); err != nil {
				continue
			}
			tokens = append(tokens, item)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			if sqliteBusy(err) && attempt+1 < sqliteBusyRetryCount {
				time.Sleep(sqliteRetryDelay(attempt))
				continue
			}
			return nil, err
		}
		return tokens, nil
	}
	return nil, fmt.Errorf("sqlite token read failed after retries")
}

// ReplaceTokens atomically replaces all tokens.
func (s *SQLiteStore) ReplaceTokens(tokens []map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for attempt := 0; attempt < sqliteBusyRetryCount; attempt++ {
		tx, err := s.db.Begin()
		if err != nil {
			if sqliteBusy(err) && attempt+1 < sqliteBusyRetryCount {
				time.Sleep(sqliteRetryDelay(attempt))
				continue
			}
			return err
		}
		failed := func(err error) error {
			_ = tx.Rollback()
			return err
		}
		if _, err := tx.Exec("DELETE FROM tokens"); err != nil {
			if sqliteBusy(err) && attempt+1 < sqliteBusyRetryCount {
				_ = tx.Rollback()
				time.Sleep(sqliteRetryDelay(attempt))
				continue
			}
			return failed(err)
		}
		stmt, err := tx.Prepare("INSERT OR REPLACE INTO tokens (id, data) VALUES (?, ?)")
		if err != nil {
			return failed(err)
		}
		writeErr := error(nil)
		for _, token := range tokens {
			id, _ := token["id"].(string)
			if id == "" {
				continue
			}
			raw, marshalErr := json.Marshal(token)
			if marshalErr != nil {
				continue
			}
			if _, execErr := stmt.Exec(id, string(raw)); execErr != nil {
				writeErr = execErr
				break
			}
		}
		_ = stmt.Close()
		if writeErr != nil {
			if sqliteBusy(writeErr) && attempt+1 < sqliteBusyRetryCount {
				_ = tx.Rollback()
				time.Sleep(sqliteRetryDelay(attempt))
				continue
			}
			return failed(writeErr)
		}
		if err := tx.Commit(); err != nil {
			if sqliteBusy(err) && attempt+1 < sqliteBusyRetryCount {
				time.Sleep(sqliteRetryDelay(attempt))
				continue
			}
			return err
		}
		return nil
	}
	return fmt.Errorf("sqlite token write failed after retries")
}

// --------------- Refresh Profiles ---------------

// LoadRefreshProfiles reads all refresh profiles from the database.
func (s *SQLiteStore) LoadRefreshProfiles() ([]map[string]interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query("SELECT data FROM refresh_profiles ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []map[string]interface{}
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		var item map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &item); err != nil {
			continue
		}
		profiles = append(profiles, item)
	}
	return profiles, nil
}

// ReplaceRefreshProfiles atomically replaces all refresh profiles.
func (s *SQLiteStore) ReplaceRefreshProfiles(profiles []map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM refresh_profiles"); err != nil {
		return err
	}
	stmt, err := tx.Prepare("INSERT OR REPLACE INTO refresh_profiles (id, data) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, profile := range profiles {
		id, _ := profile["id"].(string)
		if id == "" {
			continue
		}
		raw, err := json.Marshal(profile)
		if err != nil {
			continue
		}
		if _, err := stmt.Exec(id, string(raw)); err != nil {
			continue
		}
	}
	return tx.Commit()
}
