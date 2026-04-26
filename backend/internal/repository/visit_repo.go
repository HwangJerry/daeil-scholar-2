// visit_repo.go — Persistence for visit tracking (WEO_VISIT_DAILY, WEO_VISIT_SUMMARY)
package repository

import (
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

type VisitRepository struct {
	DB *sqlx.DB
}

func NewVisitRepository(db *sqlx.DB) *VisitRepository {
	return &VisitRepository{DB: db}
}

// UpsertDaily records one beacon hit. Same-day anon→login transitions promote
// VD_USR_SEQ without double-counting because the PK is (VD_DATE, VD_VISITOR_ID).
func (r *VisitRepository) UpsertDaily(ts time.Time, visitorID string, usrSeq int, uaHash, ipHash string) error {
	_, err := r.DB.Exec(`
		INSERT INTO WEO_VISIT_DAILY
			(VD_DATE, VD_VISITOR_ID, VD_USR_SEQ, VD_FIRST_TS, VD_LAST_TS, VD_HITS, VD_UA_HASH, VD_IP_HASH)
		VALUES (DATE(?), ?, ?, ?, ?, 1, ?, ?)
		ON DUPLICATE KEY UPDATE
			VD_LAST_TS = VALUES(VD_LAST_TS),
			VD_HITS    = VD_HITS + 1,
			VD_USR_SEQ = IF(VD_USR_SEQ = 0 AND VALUES(VD_USR_SEQ) > 0, VALUES(VD_USR_SEQ), VD_USR_SEQ)
	`, ts, visitorID, usrSeq, ts, ts, uaHash, ipHash)
	return err
}

// CountDAU returns distinct-visitor totals for a single day:
// total, logged-in member, anonymous (derived).
func (r *VisitRepository) CountDAU(date time.Time) (total, member, anon uint32, err error) {
	var row struct {
		Total  uint32 `db:"total"`
		Member uint32 `db:"member"`
	}
	err = r.DB.Get(&row, `
		SELECT
			COUNT(DISTINCT VD_VISITOR_ID) AS total,
			COUNT(DISTINCT CASE WHEN VD_USR_SEQ > 0 THEN VD_VISITOR_ID END) AS member
		FROM WEO_VISIT_DAILY
		WHERE VD_DATE = DATE(?)
	`, date)
	if err != nil {
		return 0, 0, 0, err
	}
	anonCount := uint32(0)
	if row.Total > row.Member {
		anonCount = row.Total - row.Member
	}
	return row.Total, row.Member, anonCount, nil
}

// CountMAU returns the 30-day rolling distinct-visitor count ending on endDate.
func (r *VisitRepository) CountMAU(endDate time.Time) (uint32, error) {
	var c uint32
	err := r.DB.Get(&c, `
		SELECT COUNT(DISTINCT VD_VISITOR_ID)
		FROM WEO_VISIT_DAILY
		WHERE VD_DATE BETWEEN DATE_SUB(DATE(?), INTERVAL 29 DAY) AND DATE(?)
	`, endDate, endDate)
	return c, err
}

// SumPageViews sums VD_HITS across all visitors for a single day.
func (r *VisitRepository) SumPageViews(date time.Time) (uint32, error) {
	var c uint32
	err := r.DB.Get(&c, `
		SELECT COALESCE(SUM(VD_HITS), 0)
		FROM WEO_VISIT_DAILY WHERE VD_DATE = DATE(?)
	`, date)
	return c, err
}

// UpsertSummary writes (or overwrites) the pre-aggregated row for a day.
func (r *VisitRepository) UpsertSummary(s model.VisitSummary) error {
	_, err := r.DB.Exec(`
		INSERT INTO WEO_VISIT_SUMMARY
			(VS_DATE, VS_DAU_TOTAL, VS_DAU_MEMBER, VS_DAU_ANON, VS_MAU_TOTAL, VS_PAGEVIEWS, REG_DATE)
		VALUES (DATE(?), ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			VS_DAU_TOTAL  = VALUES(VS_DAU_TOTAL),
			VS_DAU_MEMBER = VALUES(VS_DAU_MEMBER),
			VS_DAU_ANON   = VALUES(VS_DAU_ANON),
			VS_MAU_TOTAL  = VALUES(VS_MAU_TOTAL),
			VS_PAGEVIEWS  = VALUES(VS_PAGEVIEWS),
			REG_DATE      = VALUES(REG_DATE)
	`, s.Date, s.DAUTotal, s.DAUMember, s.DAUAnon, s.MAUTotal, s.PageViews, s.RegDate)
	return err
}

// GetSummaryRange returns summary rows within [from, to] inclusive, ordered ascending.
func (r *VisitRepository) GetSummaryRange(from, to time.Time) ([]model.VisitSummary, error) {
	var rows []model.VisitSummary
	err := r.DB.Select(&rows, `
		SELECT VS_DATE, VS_DAU_TOTAL, VS_DAU_MEMBER, VS_DAU_ANON, VS_MAU_TOTAL, VS_PAGEVIEWS, REG_DATE
		FROM WEO_VISIT_SUMMARY
		WHERE VS_DATE BETWEEN DATE(?) AND DATE(?)
		ORDER BY VS_DATE ASC
	`, from, to)
	return rows, err
}

// HasSummary reports whether a row already exists for the given day.
func (r *VisitRepository) HasSummary(date time.Time) (bool, error) {
	var c int
	err := r.DB.Get(&c, `SELECT COUNT(*) FROM WEO_VISIT_SUMMARY WHERE VS_DATE = DATE(?)`, date)
	return c > 0, err
}

// DeleteDailyBefore prunes WEO_VISIT_DAILY rows older than the cutoff (exclusive).
func (r *VisitRepository) DeleteDailyBefore(cutoff time.Time) (int64, error) {
	res, err := r.DB.Exec(`DELETE FROM WEO_VISIT_DAILY WHERE VD_DATE < DATE(?)`, cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
