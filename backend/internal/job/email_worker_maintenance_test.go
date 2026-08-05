// email_worker_maintenance_test.go — Maintenance freeze tests for queued email delivery.
package job

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/maintenance"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/rs/zerolog"
)

type recordingEmailSender struct {
	messages []model.EmailMessage
}

func (s *recordingEmailSender) Send(message model.EmailMessage) error {
	s.messages = append(s.messages, message)
	return nil
}

func TestEmailWorkerPreservesQueueDuringMaintenance(t *testing.T) {
	sentinel := filepath.Join(t.TempDir(), "maintenance")
	if err := os.WriteFile(sentinel, []byte("active\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gate, err := maintenance.NewGate(sentinel, "")
	if err != nil {
		t.Fatal(err)
	}
	queue := make(chan model.EmailMessage, 1)
	queue <- model.EmailMessage{}
	worker := NewEmailWorker(queue, nil, gate, zerolog.Nop())
	worker.Start()
	defer worker.Stop()

	time.Sleep(25 * time.Millisecond)
	if got := len(queue); got != 1 {
		t.Fatalf("queued messages = %d, want 1", got)
	}
}

func TestEmailWorkerStopDrainsPendingQueue(t *testing.T) {
	sentinel := filepath.Join(t.TempDir(), "maintenance")
	gate, err := maintenance.NewGate(sentinel, "")
	if err != nil {
		t.Fatal(err)
	}
	queue := make(chan model.EmailMessage, 2)
	queue <- model.EmailMessage{Subject: "first"}
	queue <- model.EmailMessage{Subject: "second"}
	sender := &recordingEmailSender{}
	worker := NewEmailWorker(queue, sender, gate, zerolog.Nop())
	worker.Start()
	time.Sleep(25 * time.Millisecond)
	worker.Stop()

	if len(sender.messages) != 2 {
		t.Fatalf("drained messages = %d, want 2", len(sender.messages))
	}
}

func TestEmailWorkerStopDoesNotDrainAfterMaintenanceIsActive(t *testing.T) {
	sentinel := filepath.Join(t.TempDir(), "maintenance")
	if err := os.WriteFile(sentinel, []byte("active\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gate, err := maintenance.NewGate(sentinel, "")
	if err != nil {
		t.Fatal(err)
	}
	queue := make(chan model.EmailMessage, 1)
	queue <- model.EmailMessage{Subject: "must-stay-frozen"}
	sender := &recordingEmailSender{}
	worker := NewEmailWorker(queue, sender, gate, zerolog.Nop())
	worker.Start()
	time.Sleep(25 * time.Millisecond)
	worker.Stop()

	if len(sender.messages) != 0 {
		t.Fatalf("messages sent after maintenance activation = %d, want 0", len(sender.messages))
	}
	if got := len(queue); got != 1 {
		t.Fatalf("queued messages after frozen stop = %d, want 1", got)
	}
}
