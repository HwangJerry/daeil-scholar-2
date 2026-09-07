// donation-retention — Apply reviewed per-order legal decisions from a local JSON file.
package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"os"
)

func main() {
	file := flag.String("file", "", "reviewed JSON decision array; dates use RFC3339 midnight")
	apply := flag.Bool("apply", false, "persist validated decisions; default validates only")
	order := flag.Int("inspect-order", 0, "read-only: print the current source fingerprint for a reviewed order")
	flag.Parse()
	if *order > 0 {
		if err := inspect(*order); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := run(*file, *apply); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(file string, apply bool) error {
	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("open decision file failed")
	}
	defer f.Close()
	var decisions []model.DonationRetentionDecision
	decoder := json.NewDecoder(f)
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&decisions); err != nil {
		return fmt.Errorf("invalid decision JSON")
	}
	seen := map[int]bool{}
	for _, d := range decisions {
		snapshot, err := hex.DecodeString(d.SourceFingerprint)
		if err != nil || len(snapshot) != 32 {
			return fmt.Errorf("order %d requires its reviewed sourceFingerprint", d.OrderID)
		}
		if seen[d.OrderID] {
			return fmt.Errorf("duplicate order decision")
		}
		seen[d.OrderID] = true
		if err = service.ValidateDonationRetention(d); err != nil {
			return err
		}
	}
	if !apply {
		fmt.Printf("Validated %d decisions; no changes.\n", len(decisions))
		return nil
	}
	db, err := repository.NewDB(config.Load().DB)
	if err != nil {
		return fmt.Errorf("database connection failed")
	}
	defer db.Close()
	repo := &repository.AccountDeletionRequestRepository{DB: db}
	for _, d := range decisions {
		if err = repo.SaveDonationRetention(d); err != nil {
			return fmt.Errorf("decision write failed for order %d; previous decisions may have committed", d.OrderID)
		}
	}
	fmt.Printf("Stored %d reviewed decisions.\n", len(decisions))
	return nil
}

func inspect(id int) error {
	db, err := repository.NewDB(config.Load().DB)
	if err != nil {
		return fmt.Errorf("database connection failed")
	}
	defer db.Close()
	repo := &repository.AccountDeletionRequestRepository{DB: db}
	fingerprint, err := repo.DonationRetentionSource(id)
	if err != nil {
		return fmt.Errorf("order snapshot unavailable")
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{"orderId": id, "sourceFingerprint": fingerprint})
}
