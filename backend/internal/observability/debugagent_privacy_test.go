// debugagent_privacy_test.go — Inspect outgoing reporter payloads without a remote service.
package observability

import (
	"encoding/json"
	"github.com/rs/zerolog"
	"strings"
	"testing"
)

type capturedErrorReporter struct{ reports []map[string]interface{} }

func (r *capturedErrorReporter) ReportErrorAsync(level, message, stack, screen string, meta, device map[string]interface{}, session, version string) {
	r.reports = append(r.reports, map[string]interface{}{"level": level, "message": message, "stack": stack, "screen": screen, "meta": meta, "device": device, "session": session, "version": version})
}
func TestDebugAgentTransportDropsCallerPersonalData(t *testing.T) {
	marker := "private-member@example.test"
	reporter := &capturedErrorReporter{}
	hook := &Hook{reporter: reporter, environment: "production"}
	hook.Run(nil, zerolog.InfoLevel, marker)
	hook.Run(nil, zerolog.ErrorLevel, marker)
	hook.ReportFrontendError()
	hook.ReportBackendPanic([]byte("runtime-stack-frame"))
	payload, err := json.Marshal(reporter.reports)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), marker) {
		t.Fatal("caller data forwarded")
	}
	if len(reporter.reports) != 3 || !strings.Contains(string(payload), "runtime-stack-frame") {
		t.Fatal("safe diagnostics lost")
	}
}
