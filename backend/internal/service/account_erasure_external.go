// account_erasure_external.go — Explicit evidence from a trusted external erasure processor.
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/dflh-saf/backend/internal/model"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// The processor must delete Sentry/logs/backups and resolve historical/indirect
// data before attesting. A missing integration is NEVER treated as success.
type HTTPErasureProcessor struct{ Endpoint, Token string }

func (p *HTTPErasureProcessor) EraseTargets(ctx context.Context, s model.ErasureExternalSubject) ([]model.ErasureTarget, error) {
	u, err := url.Parse(p.Endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || p.Token == "" {
		return nil, &model.ErasureBlocked{Code: "EXTERNAL_ERASURE_PROCESSOR_REQUIRED"}
	}
	body, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.Token)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req)
	if err != nil {
		return nil, &model.ErasureBlocked{Code: "EXTERNAL_ERASURE_PENDING"}
	}
	defer response.Body.Close()
	var result struct {
		Targets            []model.ErasureTarget `json:"targets"`
		RetentionRespected bool                  `json:"retentionRespected"`
		RequestID          int64                 `json:"requestId"`
		Complete           bool                  `json:"complete"`
		Backups            bool                  `json:"backupsErased"`
		External           bool                  `json:"externalDataErased"`
		Historical         bool                  `json:"historicalFilesErased"`
		Indirect           bool                  `json:"otherIdentifiersChecked"`
		Evidence           string                `json:"evidenceReference"`
	}
	if response.StatusCode != 200 || json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&result) != nil || result.RequestID != s.RequestID || !result.RetentionRespected {
		return nil, &model.ErasureBlocked{Code: "EXTERNAL_ERASURE_PENDING"}
	}
	if result.Targets != nil {
		return validateExternalTargets(s.RequiredTargets, result.Targets)
	}
	// Compatibility: old all-scope evidence remains valid. Partial booleans are
	// independently recorded only when the response includes review evidence.
	if strings.TrimSpace(result.Evidence) == "" || len(result.Evidence) > 200 {
		return nil, &model.ErasureBlocked{Code: "EXTERNAL_ERASURE_PENDING"}
	}
	flags := map[string]bool{"backups": result.Backups, "historical_files": result.Historical, "external_data": result.External, "other_identifiers": result.Indirect}
	targets := []model.ErasureTarget{}
	names := s.RequiredTargets
	if len(names) == 0 {
		names = model.ErasureTargetNames
	}
	for _, name := range names {
		status := "pending"
		evidence := ""
		if flags[name] && result.Complete {
			status = "complete"
			evidence = result.Evidence
		}
		targets = append(targets, model.ErasureTarget{Name: name, Status: status, Evidence: evidence})
	}
	return targets, nil
}

func (p *HTTPErasureProcessor) Erase(ctx context.Context, s model.ErasureExternalSubject) (string, error) {
	targets, err := p.EraseTargets(ctx, s)
	if err != nil {
		return "", err
	}
	for _, target := range targets {
		if !target.Verified() {
			return "", &model.ErasureBlocked{Code: "EXTERNAL_ERASURE_PENDING"}
		}
	}
	if len(targets) != len(model.ErasureTargetNames) {
		return "", &model.ErasureBlocked{Code: "EXTERNAL_ERASURE_PENDING"}
	}
	return targets[0].Evidence, nil
}
