package core

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

func OpenDatabase(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(6)
	db.SetMaxIdleConns(6)
	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000; PRAGMA foreign_keys=ON;"); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// withImmediateTx keeps write ownership short and retries SQLite's normal
// cross-process busy condition. Neither PHP nor Go holds a transaction while
// FFmpeg or file I/O is running.
func withImmediateTx(ctx context.Context, db *sql.DB, fn func(*sql.Conn) error) error {
	var lastErr error
	for attempt := 0; attempt < 8; attempt++ {
		conn, err := db.Conn(ctx)
		if err != nil {
			return err
		}
		_, err = conn.ExecContext(ctx, "BEGIN IMMEDIATE")
		if err == nil {
			committed := false
			func() {
				defer func() {
					if !committed {
						_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
					}
				}()
				err = fn(conn)
				if err == nil {
					_, err = conn.ExecContext(ctx, "COMMIT")
					committed = err == nil
				}
			}()
			conn.Close()
			if err == nil {
				return nil
			}
		} else {
			conn.Close()
		}
		lastErr = err
		if !isBusy(err) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 40 * time.Millisecond):
		}
	}
	return fmt.Errorf("sqlite remained busy after retries: %w", lastErr)
}

func isBusy(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "database is locked") || strings.Contains(message, "database is busy")
}

//go:embed schema.sql
var schema string
