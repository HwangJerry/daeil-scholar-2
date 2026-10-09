package repository

import (
	"database/sql"
	"errors"
	"github.com/jmoiron/sqlx"
	"path"
	"regexp"
	"strings"
)

var ErrManagedUploadUnavailable = errors.New("managed upload no longer exists or does not match")
var privateSignupPhotoName = regexp.MustCompile(`^[0-9a-f]{24}\.(?:jpg|jpeg|png|gif|webp)$`)

// LockManagedUpload uses the primary key shared with orphan cleanup. Acquire
// this before account/owner locks; a cleanup winner makes a waiting claim fail.
func lockManagedUpload(tx *sqlx.Tx, id int, raw string) error {
	var existing string
	err := tx.Get(&existing, `SELECT CONCAT(FILE_PATH,'/',FILE_NAME) FROM WEO_FILES WHERE F_SEQ=? FOR UPDATE`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrManagedUploadUnavailable
	}
	if err != nil {
		return err
	}
	if existing != raw {
		return ErrManagedUploadUnavailable
	}
	return nil
}

// Provider URLs and old/unmanaged paths are outside this new-upload protocol.
// Lookup is nonlocking; re-read by PK with FOR UPDATE observes a cleanup winner.
func lockPrivateSignupPhoto(tx *sqlx.Tx, raw string) error {
	if !strings.HasPrefix(raw, "/uploads/profile/") || !privateSignupPhotoName.MatchString(path.Base(raw)) {
		return nil
	}
	if raw != "/uploads/profile/"+path.Base(raw) {
		return ErrManagedUploadUnavailable
	}
	var id int
	err := tx.Get(&id, `SELECT F_SEQ FROM WEO_FILES WHERE FILE_PATH='/uploads/profile' AND FILE_NAME=?`, path.Base(raw))
	if errors.Is(err, sql.ErrNoRows) {
		return ErrManagedUploadUnavailable
	}
	if err != nil {
		return err
	}
	return lockManagedUpload(tx, id, raw)
}
