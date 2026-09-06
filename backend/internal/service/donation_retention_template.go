// donation_retention_template.go — Apply a confirmed organizational ledger policy to each donation date.
package service

import (
	"fmt"
	"github.com/dflh-saf/backend/internal/model"
	"strings"
	"time"
)

// Enable only after confirming Article 51 applicability, fiscal year and the
// separate retention of complete receipt originals (including any required IDs).
func LedgerRetentionTemplate(confirmed, receiptOriginalsSeparate bool, yearEndMonth int, evidence string) func(int, time.Time) (model.DonationRetentionDecision, error) {
	if !confirmed || !receiptOriginalsSeparate || yearEndMonth < 1 || yearEndMonth > 12 || strings.TrimSpace(evidence) == "" || len(evidence) > 200 {
		return nil
	}
	return func(id int, donationDate time.Time) (model.DonationRetentionDecision, error) {
		if donationDate.IsZero() || donationDate.After(time.Now()) {
			return model.DonationRetentionDecision{}, fmt.Errorf("unverified donation date")
		}
		year := donationDate.Year()
		if int(donationDate.Month()) > yearEndMonth {
			year++
		}
		end := time.Date(year, time.Month(yearEndMonth)+1, 0, 0, 0, 0, 0, time.UTC)
		until := retentionAnniversary(end, 10)
		return model.DonationRetentionDecision{OrderID: id, Basis: "ledger_10y", BasisDate: &end, Until: &until, Evidence: evidence}, nil
	}
}
