// donation-retention — Apply reviewed per-order legal decisions from a local JSON file.
package main

import (
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
	flag.Parse()
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
