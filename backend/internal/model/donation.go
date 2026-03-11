package model

// DonationSnapshot represents a row in DONATION_SNAPSHOT table.
type DonationSnapshot struct {
	DSSeq     int    `db:"DS_SEQ"        json:"dsSeq"`
	DSDate    string `db:"DS_DATE"       json:"dsDate"`
	DSTotal   int64  `db:"DS_TOTAL"      json:"dsTotal"`
	ManualAdj int64  `db:"DS_MANUAL_ADJ" json:"dsManualAdj"`
	DonorCnt  int    `db:"DS_DONOR_CNT"  json:"dsDonorCnt"`
	Goal      int64  `db:"DS_GOAL"       json:"dsGoal"`
	RegDate   string `db:"REG_DATE"      json:"regDate"`
}

// DonationConfig represents a row in DONATION_CONFIG table.
type DonationConfig struct {
	DCSeq     int    `db:"DC_SEQ"        json:"dcSeq"`
	Goal      int64  `db:"DC_GOAL"       json:"dcGoal"`
	ManualAdj int64  `db:"DC_MANUAL_ADJ" json:"dcManualAdj"`
	Note      string `db:"DC_NOTE"       json:"dcNote"`
	IsActive  string `db:"IS_ACTIVE"     json:"isActive"`
	RegDate   string `db:"REG_DATE"      json:"regDate"`
	RegOper   int    `db:"REG_OPER"      json:"regOper"`
}

