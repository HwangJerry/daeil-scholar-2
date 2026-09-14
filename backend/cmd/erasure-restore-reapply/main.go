// erasure-restore-reapply — After restoring a backup, erase members whose deletion completed in the backup window.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
)

func main() {
	apply := flag.Bool("apply", false, "erase restored members; the default only reports")
	flag.Parse()
	if err := run(*apply); err != nil {
		var blocked *model.ErasureBlocked
		if errors.As(err, &blocked) {
			fmt.Fprintln(os.Stderr, "Restore re-erasure stopped for review: "+blocked.Code)
		} else {
			// SQL and OS errors can contain private paths and credentials.
			fmt.Fprintln(os.Stderr, "Restore re-erasure stopped; check database access, the restore guard table and storage roots.")
		}
		os.Exit(1)
	}
}

func run(apply bool) error {
	cfg := config.Load()
	db, err := repository.NewDB(cfg.DB)
	if err != nil {
		return err
	}
	defer db.Close()
	repo := &repository.AccountDeletionRequestRepository{DB: db, SiteOrigin: cfg.Server.SiteBaseURL}
	files := &service.AccountErasureFiles{UploadRoot: cfg.Upload.BasePath, LegacyRoot: cfg.AccountErasure.LegacyRoot, SiteOrigin: cfg.Server.SiteBaseURL}
	guards, err := repo.RestoreGuards()
	if err != nil {
		return err
	}
	present := 0
	for _, guard := range guards {
		if guard.Present {
			present++
		}
	}
	fmt.Printf("guarded erasures: %d, restored members present: %d\n", len(guards), present)
	if !apply || present == 0 {
		return nil
	}
	for _, guard := range guards {
		if !guard.Present {
			continue
		}
		paths, err := repo.ReapplyErasureAfterRestore(guard.UserSeq)
		if err != nil {
			return err
		}
		for _, path := range paths {
			if err := files.EraseURL(path); err != nil {
				return err
			}
		}
		fmt.Printf("request %d: erased again, files removed %d\n", guard.RequestID, len(paths))
	}
	return nil
}
