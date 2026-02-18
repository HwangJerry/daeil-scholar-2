// Thumbnail backfill — sets THUMBNAIL_URL from WEO_FILES for notices missing one
package main

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
)

func BackfillThumbnails(db *sqlx.DB) {
	rows, err := db.Queryx(`
		SELECT b.SEQ, wf.FILE_PATH, wf.FILE_NAME
		FROM WEO_BOARDBBS b
		JOIN (
			SELECT F_JOIN_SEQ, MIN(F_SEQ) AS FIRST_SEQ
			FROM WEO_FILES
			WHERE F_GATE = 'BB' AND OPEN_YN = 'Y'
			  AND (FILE_NAME LIKE '%.jpg' OR FILE_NAME LIKE '%.jpeg' OR FILE_NAME LIKE '%.png' OR FILE_NAME LIKE '%.webp')
			GROUP BY F_JOIN_SEQ
		) f ON b.SEQ = f.F_JOIN_SEQ
		JOIN WEO_FILES wf ON wf.F_SEQ = f.FIRST_SEQ
		WHERE b.GATE = 'NOTICE' AND b.THUMBNAIL_URL IS NULL
	`)
	if err != nil {
		log.Printf("thumbnail query error: %v", err)
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var seq int
		var filePath, fileName string
		if err := rows.Scan(&seq, &filePath, &fileName); err != nil {
			continue
		}
		thumb := fmt.Sprintf("/files/%s/%s", filePath, fileName)
		if _, err := db.Exec(`UPDATE WEO_BOARDBBS SET THUMBNAIL_URL = ? WHERE SEQ = ?`, thumb, seq); err != nil {
			log.Printf("failed to update thumbnail for SEQ %d: %v", seq, err)
			continue
		}
		count++
	}
	log.Printf("backfilled %d thumbnails", count)
}
