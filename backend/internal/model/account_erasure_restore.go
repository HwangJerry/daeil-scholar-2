// account_erasure_restore.go — Erased members that a backup restore could bring back.
package model

// ErasureRestoreGuard records a completed erasure while retained backups may
// still contain the member. Present reports whether the member row exists again.
type ErasureRestoreGuard struct {
	RequestID int64 `db:"REQUEST_ID"`
	UserSeq   int   `db:"USR_SEQ"`
	Present   bool  `db:"PRESENT"`
}
