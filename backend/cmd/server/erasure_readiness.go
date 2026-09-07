// erasure_readiness.go — Read-only validation usable before replacing the live binary.
package main

import (
	"fmt"
	"github.com/dflh-saf/backend/internal/config"
	"net/url"
	"os"
	"path/filepath"
)

func validateErasureRuntime(cfg *config.Config) error {
	if err := cfg.AccountErasure.Validate(); err != nil {
		return err
	}
	if !cfg.AccountErasure.RequestsEnabled && !cfg.AccountErasure.WorkerEnabled {
		return nil
	}
	u, err := url.Parse(cfg.Server.SiteBaseURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return fmt.Errorf("SITE_BASE_URL must be an HTTPS origin")
	}
	for _, root := range []string{cfg.Upload.BasePath, cfg.AccountErasure.LegacyRoot} {
		if !filepath.IsAbs(root) {
			return fmt.Errorf("erasure storage root must be absolute")
		}
		resolved, err := filepath.EvalSymlinks(root)
		if err != nil || resolved != filepath.Clean(root) {
			return fmt.Errorf("erasure storage root must exist without symlink ancestors")
		}
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			return fmt.Errorf("erasure storage root must be a directory")
		}
	}
	return nil
}
