package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PGAuditLogger writes JSON-per-line audit entries for PG payment events.
// Each write is fsync'd for durability — financial reconciliation depends on this log.
type PGAuditLogger struct {
	mu   sync.Mutex
	file *os.File
}

// PGAuditEntry represents a single audit log entry.
type PGAuditEntry struct {
	Timestamp string      `json:"ts"`
	OrderNo   string      `json:"order_no"`
	Event     string      `json:"event"` // "approve_success", "approve_fail", "db_insert_fail", "db_update_fail"
	RawData   interface{} `json:"data"`
	Error     string      `json:"error,omitempty"`
}

// NewPGAuditLogger opens (or creates) the audit log file at the given path.
func NewPGAuditLogger(path string) (*PGAuditLogger, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("pg audit: create dir %s: %w", dir, err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("pg audit: open %s: %w", path, err)
	}
	return &PGAuditLogger{file: f}, nil
}

// Log writes a single audit entry as JSON + newline, then fsyncs.
func (l *PGAuditLogger) Log(orderNo string, event string, data interface{}, err error) {
	entry := PGAuditEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		OrderNo:   orderNo,
		Event:     event,
		RawData:   data,
	}
	if err != nil {
		entry.Error = err.Error()
	}

	b, _ := json.Marshal(entry)
	b = append(b, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()
	l.file.Write(b)
	l.file.Sync()
}

// Close closes the underlying file.
func (l *PGAuditLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}
