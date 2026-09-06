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

func (p *HTTPErasureProcessor) Erase(ctx context.Context, s model.ErasureExternalSubject) (string, error) {
	u, err := url.Parse(p.Endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || p.Token == "" {
		return "", &model.ErasureBlocked{Code: "EXTERNAL_ERASURE_PROCESSOR_REQUIRED"}
	}
	body, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+p.Token)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req)
	if err != nil {
		return "", &model.ErasureBlocked{Code: "EXTERNAL_ERASURE_PENDING"}
	}
	defer response.Body.Close()
	var result struct {
		RetentionRespected bool   `json:"retentionRespected"`
		RequestID          int64  `json:"requestId"`
		Complete           bool   `json:"complete"`
		Backups            bool   `json:"backupsErased"`
		External           bool   `json:"externalDataErased"`
		Historical         bool   `json:"historicalFilesErased"`
		Indirect           bool   `json:"otherIdentifiersChecked"`
		Evidence           string `json:"evidenceReference"`
	}
	if response.StatusCode != 200 || json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&result) != nil || result.RequestID != s.RequestID || !result.Complete || !result.RetentionRespected || !result.Backups || !result.External || !result.Historical || !result.Indirect || strings.TrimSpace(result.Evidence) == "" || len(result.Evidence) > 200 {
		return "", &model.ErasureBlocked{Code: "EXTERNAL_ERASURE_PENDING"}
	}
	return result.Evidence, nil
}
