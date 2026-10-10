package repository

import (
	"database/sql"
	"errors"
	"github.com/dflh-saf/backend/internal/model"
)

// SignupPhotoRetirement is an unlink capability minted only after metadata
// retirement commits. Its private path cannot be supplied by arbitrary callers.
type SignupPhotoRetirement struct{ local string }

func (receipt *SignupPhotoRetirement) Erase(erase func(string) error) error {
	if receipt == nil || receipt.local == "" {
		return errors.New("invalid signup photo retirement")
	}
	return erase(receipt.local)
}

// DiscardSignupProfileUpload is the one-attempt repository adapter. Production
// callers retain a committed receipt for bounded retries after row retirement.
func (r *FileRepository) DiscardSignupProfileUpload(id int, url, origin string, erase func(string) error) error {
	receipt, err := r.RetireSignupProfileUpload(id, url, origin)
	if err != nil || receipt == nil {
		return err
	}
	return receipt.Erase(erase)
}

// RetireSignupProfileUpload first locks the exact tracked upload PK. Default
// REPEATABLE READ establishes the consistent snapshot only after that lock is
// obtained, so committed managed claimers are visible without broad write locks.
// Commit errors (including ambiguous outcomes) never grant disk unlink permission.
func (r *FileRepository) RetireSignupProfileUpload(id int, url, origin string) (*SignupPhotoRetirement, error) {
	tx, err := r.DB.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var file struct {
		Gate string `db:"F_GATE"`
		Join int    `db:"F_JOIN_SEQ"`
		URL  string `db:"URL_PATH"`
	}
	err = tx.Get(&file, `SELECT COALESCE(F_GATE,'') AS F_GATE,COALESCE(F_JOIN_SEQ,0) AS F_JOIN_SEQ,CONCAT(FILE_PATH,'/',FILE_NAME) AS URL_PATH FROM WEO_FILES WHERE F_SEQ=? FOR UPDATE`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if file.Gate != "PR" || file.Join != 0 || file.URL != url {
		return nil, &model.ErasureBlocked{Code: "UPLOAD_OWNERSHIP_CHANGED"}
	}
	local, external, err := model.ErasureFilePath(url, origin)
	if err != nil {
		return nil, err
	}
	if external {
		return nil, nil
	}
	schema, err := readErasureSchema(tx)
	if err != nil {
		return nil, err
	}
	if schema["ALUMNI_UPLOAD_OWNER"] != nil {
		var owners int
		if err = tx.Get(&owners, `SELECT COUNT(*) FROM ALUMNI_UPLOAD_OWNER WHERE F_SEQ=?`, id); err != nil {
			return nil, err
		}
		if owners > 0 {
			return nil, &model.ErasureBlocked{Code: "FILE_STILL_REFERENCED"}
		}
	}
	if err = scanSurvivingFileReferences(tx, schema, local, origin, nil, id, false, signupPhotoReferenceCandidate(local)); err != nil {
		return nil, err
	}
	if _, err = tx.Exec(`DELETE FROM WEO_FILES WHERE F_SEQ=?`, id); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &SignupPhotoRetirement{local: local}, nil
}
