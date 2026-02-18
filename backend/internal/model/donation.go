package model

// DonationSnapshot represents a row in DONATION_SNAPSHOT table.
type DonationSnapshot struct {
	DSSeq     int    `db:"DS_SEQ"`
	DSDate    string `db:"DS_DATE"`
	DSTotal   int64  `db:"DS_TOTAL"`
	ManualAdj int64  `db:"DS_MANUAL_ADJ"`
	DonorCnt  int    `db:"DS_DONOR_CNT"`
	Goal      int64  `db:"DS_GOAL"`
	RegDate   string `db:"REG_DATE"`
}

// DonationConfig represents a row in DONATION_CONFIG table.
type DonationConfig struct {
	DCSeq     int    `db:"DC_SEQ"`
	Goal      int64  `db:"DC_GOAL"`
	ManualAdj int64  `db:"DC_MANUAL_ADJ"`
	Note      string `db:"DC_NOTE"`
	IsActive  string `db:"IS_ACTIVE"`
	RegDate   string `db:"REG_DATE"`
	RegOper   int    `db:"REG_OPER"`
}

