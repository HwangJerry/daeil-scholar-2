// account_erasure_preview_rows.go — Bounded, masked row samples and a digest over every affected row.
package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

const (
	previewRowLimit   = 20
	previewValueRunes = 120
	maskSecret        = "secret"
	maskPrivate       = "private"
)

// Credentials and secrets are never shown, even to root operators.
var previewSecretColumn = regexp.MustCompile(`(?i)(PASS|PWD|TOKEN|SECRET|HASH|CREDENTIAL|CIPHER|SALT|NONCE|OTP|SESSION_ID|_KEY$|^KEY$)`)

// Private correspondence between members is summarized by length only.
var previewPrivateContent = map[string]bool{
	"ALUMNI_MESSAGE.AM_CONTENT":              true,
	"ALUMNI_COMMENT_REPORT.CONTENT_SNAPSHOT": true,
	"ALUMNI_COMMENT_REPORT.DETAILS":          true,
	"ALUMNI_COMMENT_REPORT.MODERATOR_NOTE":   true,
	"ALUMNI_MESSAGE_REPORT.CONTENT_SNAPSHOT": true,
	"ALUMNI_MESSAGE_REPORT.DETAILS":          true,
}

// previewQuery is one table's merged erasure predicate and planned action.
type previewQuery struct {
	table  string
	action string
	where  string
	args   []interface{}
	after  []anonymizedColumn
}

// previewTable carries the displayed sample plus a hash of every affected row.
type previewTable struct {
	model.ErasurePreviewTable
	rowHashes []string
}

func readPreviewTable(tx *sqlx.Tx, q previewQuery) (previewTable, error) {
	t := previewTable{ErasurePreviewTable: model.ErasurePreviewTable{
		Table: q.table, Action: q.action, MaskedColumns: []string{}, ChangedColumns: []string{}, Rows: []model.ErasurePreviewRow{},
	}}
	if !deletionSQLIdentifier.MatchString(q.table) {
		return t, ErrDeletionIncomplete
	}
	// Tables and predicates are code-owned constants, never HTTP input.
	rows, err := tx.Queryx(fmt.Sprintf("SELECT * FROM `%s` WHERE %s", q.table, q.where), q.args...)
	if err != nil {
		return t, err
	}
	defer rows.Close()
	if t.Columns, err = rows.Columns(); err != nil {
		return t, err
	}
	changed := map[int]anonymizedColumn{}
	masks := make([]string, len(t.Columns))
	for i, name := range t.Columns {
		for _, c := range q.after {
			if c.column == name {
				changed[i] = c
				t.ChangedColumns = append(t.ChangedColumns, name)
			}
		}
		if previewPrivateContent[q.table+"."+name] {
			masks[i] = maskPrivate
		} else if previewSecretColumn.MatchString(name) {
			masks[i] = maskSecret
		}
		if masks[i] != "" {
			t.MaskedColumns = append(t.MaskedColumns, name)
		}
	}
	for rows.Next() {
		values, err := rows.SliceScan()
		if err != nil {
			return t, err
		}
		raw := make([]*string, len(values))
		for i, value := range values {
			raw[i] = previewRawValue(value)
		}
		t.rowHashes = append(t.rowHashes, previewRowHash(raw))
		t.Count++
		if len(t.Rows) >= previewRowLimit {
			continue
		}
		row := model.ErasurePreviewRow{Before: previewDisplay(raw, masks)}
		if len(changed) > 0 {
			after := append([]*string(nil), raw...)
			for i, c := range changed {
				value := fmt.Sprint(c.value)
				after[i] = &value
			}
			row.After = previewDisplay(after, masks)
		}
		t.Rows = append(t.Rows, row)
	}
	return t, rows.Err()
}

func previewRawValue(value interface{}) *string {
	var s string
	switch v := value.(type) {
	case nil:
		return nil
	case []byte:
		s = string(v)
	case time.Time:
		s = v.UTC().Format(time.RFC3339)
	default:
		s = fmt.Sprint(v)
	}
	return &s
}

// previewRowHash length-prefixes each value so NULL, "" and joined values differ.
func previewRowHash(values []*string) string {
	h := sha256.New()
	for _, v := range values {
		if v == nil {
			h.Write([]byte("N;"))
			continue
		}
		fmt.Fprintf(h, "V%d:%s;", len(*v), *v)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func previewDisplay(raw []*string, masks []string) []*string {
	out := make([]*string, len(raw))
	for i, v := range raw {
		if v == nil {
			continue
		}
		var s string
		switch {
		case masks[i] == maskSecret:
			s = "[보안값 비표시]"
		case masks[i] == maskPrivate:
			s = fmt.Sprintf("[쪽지 내용 비표시 · %d자]", utf8.RuneCountInString(*v))
		case !utf8.ValidString(*v):
			s = fmt.Sprintf("[이진 데이터 %d바이트]", len(*v))
		case utf8.RuneCountInString(*v) > previewValueRunes:
			s = string([]rune(*v)[:previewValueRunes]) + "…"
		default:
			s = *v
		}
		out[i] = &s
	}
	return out
}

// previewDigest is independent of row order but covers every affected row,
// not only the displayed sample, and every file queued for unlinking.
func previewDigest(tables []previewTable, files []string) string {
	h := sha256.New()
	for _, t := range tables {
		hashes := append([]string(nil), t.rowHashes...)
		sort.Strings(hashes)
		fmt.Fprintf(h, "T%s|%s|%s|%d\n", t.Table, t.Action, strings.Join(t.ChangedColumns, ","), len(hashes))
		for _, hash := range hashes {
			fmt.Fprintf(h, "%s\n", hash)
		}
	}
	sorted := append([]string(nil), files...)
	sort.Strings(sorted)
	for _, file := range sorted {
		fmt.Fprintf(h, "F%d:%s\n", len(file), file)
	}
	return hex.EncodeToString(h.Sum(nil))
}
