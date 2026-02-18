// File repository — WEO_FILES insert for uploaded files
package repository

import (
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type FileRepository struct {
	DB *sqlx.DB
}

func NewFileRepository(db *sqlx.DB) *FileRepository {
	return &FileRepository{DB: db}
}

func (r *FileRepository) InsertFile(f *model.FileRecord) (int, error) {
	res, err := r.DB.Exec(`
		INSERT INTO WEO_FILES (F_GATE, F_JOIN_SEQ, TYPE_NAME, FILE_NAME, FILE_SIZE, FILE_PATH, FILE_ORG_NAME, OPEN_YN, REG_DATE)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'Y', NOW())
	`, f.FGate, f.FJoinSeq, f.TypeName, f.FileName, f.FileSize, f.FilePath, f.FileOrgName)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}
