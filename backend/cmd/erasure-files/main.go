// erasure-files — Queue an operator-reviewed inventory; dry-run is the default.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"io"
	"os"
	"time"
)

const maxPlanBytes = 2 << 20
const lockTimeout = 10 * time.Second

func main() {
	file := flag.String("file", "", "private reviewed historical-file JSON")
	apply := flag.Bool("apply", false, "enqueue verified paths; default checks without changing data")
	flag.Parse()
	if err := run(*file, *apply); err != nil {
		// SQL/OS errors can contain private paths and credentials.
		fmt.Fprintln(os.Stderr, "Historical file plan rejected; verify request state, ownership, storage roots and database access. No completion was recorded.")
		os.Exit(1)
	}
}

func readPlan(file string) (model.ErasureHistoricalFilePlan, error) {
	var plan model.ErasureHistoricalFilePlan
	f, err := os.Open(file)
	if err != nil {
		return plan, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return plan, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxPlanBytes || info.Mode().Perm()&0077 != 0 {
		return plan, fmt.Errorf("plan must be a private regular file")
	}
	decoder := json.NewDecoder(io.LimitReader(f, maxPlanBytes+1))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&plan); err != nil {
		return plan, err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return plan, fmt.Errorf("unexpected trailing data")
	}
	return plan, service.ValidateHistoricalErasurePlan(plan)
}

func run(file string, apply bool) error {
	plan, err := readPlan(file)
	if err != nil {
		return err
	}
	cfg := config.Load()
	db, err := repository.NewDB(cfg.DB)
	if err != nil {
		return err
	}
	defer db.Close()
	repo := &repository.AccountDeletionRequestRepository{DB: db}
	ctx, cancel := context.WithTimeout(context.Background(), lockTimeout)
	defer cancel()
	release, locked, err := repo.ErasureLock(ctx)
	if err != nil {
		return err
	}
	if !locked {
		return fmt.Errorf("erasure worker busy; retry")
	}
	defer release()
	files := &service.AccountErasureFiles{UploadRoot: cfg.Upload.BasePath, LegacyRoot: cfg.AccountErasure.LegacyRoot, SiteOrigin: cfg.Server.SiteBaseURL}
	if err = service.QueueReviewedHistoricalFiles(repo, files, plan, apply); err != nil {
		return err
	}
	if apply {
		fmt.Printf("Queued %d reviewed files. The worker deletes them; historical scope is not marked complete.\n", len(plan.Files))
	} else {
		fmt.Printf("Validated %d reviewed files against request, references and storage. No changes.\n", len(plan.Files))
	}
	return nil
}
