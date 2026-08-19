package model

// DonationSnapshot represents a row in DONATION_SNAPSHOT table.
type DonationSnapshot struct {
	DSSeq     int    `db:"DS_SEQ"        json:"dsSeq"`
	DSDate    string `db:"DS_DATE"       json:"dsDate"`
	DSTotal   int64  `db:"DS_TOTAL"      json:"dsTotal"`
	ManualAdj int64  `db:"DS_MANUAL_ADJ" json:"dsManualAdj"`
	DonorCnt  int    `db:"DS_DONOR_CNT"  json:"dsDonorCnt"`
	Goal      int64  `db:"DS_GOAL"       json:"dsGoal"`
	Overwrite string `db:"DS_OVERWRITE"  json:"dsOverwrite"`
	RegDate   string `db:"REG_DATE"      json:"regDate"`
}

// DonationConfig represents a row in DONATION_CONFIG table.
type DonationConfig struct {
	DCSeq           int    `db:"DC_SEQ"                json:"dcSeq"`
	Goal            int64  `db:"DC_GOAL"               json:"dcGoal"`
	ManualAdj       int64  `db:"DC_MANUAL_ADJ"         json:"dcManualAdj"`
	ManualDonorCnt  int    `db:"DC_MANUAL_DONOR_CNT"   json:"dcManualDonorCnt"`
	TierSproutMin   int64  `db:"DC_TIER_SPROUT_MIN"    json:"dcTierSproutMin"`
	TierSaplingMin  int64  `db:"DC_TIER_SAPLING_MIN"   json:"dcTierSaplingMin"`
	TierTreeMin     int64  `db:"DC_TIER_TREE_MIN"      json:"dcTierTreeMin"`
	TierBloomingMin int64  `db:"DC_TIER_BLOOMING_MIN"  json:"dcTierBloomingMin"`
	TierFruitingMin int64  `db:"DC_TIER_FRUITING_MIN"  json:"dcTierFruitingMin"`
	Note            string `db:"DC_NOTE"               json:"dcNote"`
	Overwrite       string `db:"DC_OVERWRITE"          json:"dcOverwrite"`
	IsActive        string `db:"IS_ACTIVE"             json:"isActive"`
	RegDate         string `db:"REG_DATE"              json:"regDate"`
	RegOper         int    `db:"REG_OPER"              json:"regOper"`
}

type DonationDonor struct {
	Name       string `db:"O_DONOR_NAME" json:"name"`
	Cohort     string `db:"O_DONOR_COHORT" json:"cohort"`
	Department string `db:"O_DONOR_DEPARTMENT" json:"department"`
	Phone      string `db:"O_DONOR_PHONE" json:"phone"`
}

type DonationOrderInput struct {
	Source            string        `json:"source"`
	AccountUsrSeq     *int          `json:"accountUsrSeq"`
	AccountUsrSeqSet  bool          `json:"-"`
	TransactionNumber *string       `json:"transactionNumber"`
	DonationDate      string        `json:"donationDate"`
	Donor             DonationDonor `json:"donor"`
	DonationType      string        `json:"donationType"`
	GrossAmount       int64         `json:"grossAmount"`
	RefundedAmount    int64         `json:"refundedAmount"`
	Status            string        `json:"status"`
	PaymentMethod     string        `json:"paymentMethod"`
	Memo              *string       `json:"memo"`
	LastEditedAt      string        `json:"lastEditedAt,omitempty"`
}

type NormalizedDonationOrder struct {
	DonationOrderInput
	NetReceivedAmount int64
	CompositeKey      string
	LegacyGate        string
	LegacyStatus      string
	LegacyPayment     string
}

type DonationOrder struct {
	OrderSeq          int64         `json:"orderSeq"`
	AccountUsrSeq     *int          `json:"accountUsrSeq"`
	Source            string        `json:"source"`
	TransactionNumber *string       `json:"transactionNumber"`
	DonationDate      string        `json:"donationDate"`
	Donor             DonationDonor `json:"donor"`
	DonationType      string        `json:"donationType"`
	GrossAmount       int64         `json:"grossAmount"`
	RefundedAmount    int64         `json:"refundedAmount"`
	NetReceivedAmount int64         `json:"netReceivedAmount"`
	Status            string        `json:"status"`
	PaymentMethod     string        `json:"paymentMethod"`
	Memo              *string       `json:"memo"`
	LastEditedBy      int           `json:"lastEditedBy"`
	LastEditedAt      string        `json:"lastEditedAt"`
	LastEditedIP      string        `json:"lastEditedIp"`
}

type DonationOrderFilters struct {
	Name              string
	Phone             string
	TransactionNumber string
	Source            string
	Status            string
	DonationType      string
}

type DonationOrderPage struct {
	Items []*DonationOrder `json:"items"`
	Total int              `json:"total"`
	Page  int              `json:"page"`
	Size  int              `json:"size"`
}
