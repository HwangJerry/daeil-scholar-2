// debugagent.go — zerolog Hook that forwards Error/Fatal events to the
// external Debug Agent Pipeline. Disabled when DebugAgent.Endpoint is empty.
package observability

import (
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	debugagent "github.com/cherryberryyogurt/debug-agent-go-client"
	"github.com/dflh-saf/backend/internal/config"
	"github.com/rs/zerolog"
)

const breadcrumbCap = 50

type breadcrumb struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

// Hook is a zerolog.Hook implementation that ships Error/Fatal events to a
// Debug Agent gateway via ReportErrorAsync. Info/Debug/Warn events are stored
// in a ring buffer and attached as breadcrumbs to the next error report.
type Hook struct {
	reporter    *debugagent.Reporter
	environment string
	mu          sync.Mutex
	ring        [breadcrumbCap]breadcrumb
	head        int // next write position
	count       int // filled slots, capped at breadcrumbCap
}

// NewHook constructs a Hook from config. Returns nil when the agent is
// disabled (empty endpoint), so callers can no-op via a simple nil check.
func NewHook(cfg config.DebugAgentConfig) *Hook {
	if !cfg.Enabled() {
		return nil
	}
	return &Hook{
		reporter:    debugagent.NewReporter(cfg.Endpoint, cfg.Project, cfg.Secret),
		environment: cfg.Environment,
	}
}

// drainBreadcrumbs returns buffered events oldest-first and resets the buffer.
func (h *Hook) drainBreadcrumbs() []breadcrumb {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := h.count
	if n == 0 {
		return nil
	}
	out := make([]breadcrumb, n)
	start := (h.head - n + breadcrumbCap*2) % breadcrumbCap
	for i := 0; i < n; i++ {
		out[i] = h.ring[(start+i)%breadcrumbCap]
	}
	h.head = 0
	h.count = 0
	return out
}

// Run implements zerolog.Hook.
// Info/Debug/Warn are buffered as breadcrumbs.
// Error/Fatal flush the buffer and send a report with the breadcrumb trail.
func (h *Hook) Run(_ *zerolog.Event, level zerolog.Level, msg string) {
	if h == nil {
		return
	}

	if level < zerolog.ErrorLevel {
		// Buffer as breadcrumb
		bc := breadcrumb{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Level:     level.String(),
			Message:   msg,
		}
		h.mu.Lock()
		h.ring[h.head] = bc
		h.head = (h.head + 1) % breadcrumbCap
		if h.count < breadcrumbCap {
			h.count++
		}
		h.mu.Unlock()
		return
	}

	logLevel := "error"
	if level == zerolog.FatalLevel {
		logLevel = "fatal"
	}
	meta := map[string]interface{}{
		"environment": h.environment,
		"service":     "backend",
		"breadcrumbs": h.drainBreadcrumbs(),
	}
	h.reporter.ReportErrorAsync(
		logLevel,
		msg,
		string(debug.Stack()),
		"",
		meta,
		nil,
		"",
		"",
	)
}

// ReportPanic forwards a recovered panic with its stack trace and arbitrary
// request metadata. Caller is responsible for capturing runtime/debug.Stack().
func (h *Hook) ReportPanic(recovered interface{}, stack []byte, meta map[string]interface{}) {
	if h == nil {
		return
	}
	if meta == nil {
		meta = map[string]interface{}{}
	}
	meta["environment"] = h.environment
	meta["service"] = "backend"
	meta["breadcrumbs"] = h.drainBreadcrumbs()
	h.reporter.ReportErrorAsync(
		"fatal",
		fmt.Sprintf("panic: %v", recovered),
		string(stack),
		"",
		meta,
		nil,
		"",
		"",
	)
}
