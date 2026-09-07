// erasure_historical_files_test.go — Reject unsafe plans before reaching the database.
package service

import (
	"context"
	"github.com/dflh-saf/backend/internal/model"
	"os"
	"path/filepath"
	"testing"
)

type historicalQueueSpy struct{ called bool }

func (s *historicalQueueSpy) QueueHistoricalErasureFiles(model.ErasureHistoricalFilePlan, bool) error {
	s.called = true
	return nil
}

func TestHistoricalFilePlanRequiresReviewedCanonicalInventory(t *testing.T) {
	good := model.ErasureHistoricalFilePlan{RequestID: 1, UserSeq: 42, Evidence: "review-42", OwnershipVerified: true, RetentionRespected: true, Files: []string{"/files/profile/old.jpg"}}
	if err := ValidateHistoricalErasurePlan(good); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"/files/../private", "/files/%2e%2e/private", "https://app.example.org/files/old.jpg", "/files/old.jpg?x=1", "/files/old.jpg#x", "/files//old.jpg", "/files/", "/other/old.jpg", "/files/old\n.jpg"} {
		plan := good
		plan.Files = []string{raw}
		if ValidateHistoricalErasurePlan(plan) == nil {
			t.Fatalf("unsafe path accepted: %q", raw)
		}
	}
	for _, mutate := range []func(*model.ErasureHistoricalFilePlan){
		func(p *model.ErasureHistoricalFilePlan) { p.OwnershipVerified = false },
		func(p *model.ErasureHistoricalFilePlan) { p.RetentionRespected = false },
		func(p *model.ErasureHistoricalFilePlan) { p.Evidence = "private person@example.org" },
		func(p *model.ErasureHistoricalFilePlan) { p.Files = append(p.Files, p.Files[0]) },
		func(p *model.ErasureHistoricalFilePlan) { p.Files = nil },
	} {
		plan := good
		mutate(&plan)
		spy := &historicalQueueSpy{}
		if QueueReviewedHistoricalFiles(spy, nil, plan, true) == nil || spy.called {
			t.Fatal("unreviewed plan reached storage")
		}
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(filepath.Join(root, "profile"), 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "profile", "old.jpg")
	if err = os.WriteFile(file, []byte("synthetic"), 0600); err != nil {
		t.Fatal(err)
	}
	spy := &historicalQueueSpy{}
	if err = QueueReviewedHistoricalFiles(spy, &AccountErasureFiles{LegacyRoot: root}, good, false); err != nil || !spy.called {
		t.Fatal("valid dry-run rejected", err)
	}
	if _, err = os.Stat(file); err != nil {
		t.Fatal("inspection deleted a file")
	}
}

func TestErasureFilesMissingRootAndAncestorSymlinkAreNotSuccess(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	missing := &AccountErasureFiles{UploadRoot: filepath.Join(base, "missing")}
	if missing.EraseURL("/uploads/old.jpg") == nil {
		t.Fatal("missing storage accepted as proof")
	}
	physical := filepath.Join(base, "physical")
	if err = os.MkdirAll(filepath.Join(physical, "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(physical, filepath.Join(base, "alias")); err != nil {
		t.Fatal(err)
	}
	linked := &AccountErasureFiles{UploadRoot: filepath.Join(base, "alias", "nested")}
	if linked.EraseURL("/uploads/missing.jpg") == nil {
		t.Fatal("symlink ancestor accepted")
	}
	verified := &AccountErasureFiles{UploadRoot: filepath.Join(physical, "nested")}
	if err = verified.EraseURL("/uploads/missing.jpg"); err != nil {
		t.Fatal("verified root missing leaf should allow retry", err)
	}
}

// This exercises real filesystem deletion through the worker; external success
// is deliberately unavailable and must not turn file success into completion.
func TestWorkerDeletesRealHistoricalFileWithoutFalseExternalCompletion(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "old.jpg")
	other := filepath.Join(root, "other.jpg")
	for _, name := range []string{file, other} {
		if err = os.WriteFile(name, []byte("synthetic"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	store := &erasureStoreFake{active: true, work: model.ErasureWork{RequestID: 1, UserSeq: 42, Stage: "database_erased"}, files: []model.ErasureFile{{ID: 1, URL: "/files/old.jpg"}}}
	storage := &AccountErasureFiles{LegacyRoot: filepath.Join(root, "wrong-root")}
	worker := &AutomaticErasureService{Store: store, Files: storage}
	if err = worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(store.files) != 1 || store.completed != 0 {
		t.Fatal("missing root lost queued work")
	}
	storage.LegacyRoot = root
	if err = worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(file); !os.IsNotExist(err) {
		t.Fatal("queued file survived", err)
	}
	if _, err = os.Stat(other); err != nil {
		t.Fatal("unrelated file lost", err)
	}
	if len(store.files) != 0 || store.completed != 0 || store.databaseCalls != 0 || store.retry != "EXTERNAL_ERASURE_PROCESSOR_REQUIRED" {
		t.Fatal("file success bypassed external verification")
	}
}

func TestLegacyUploadAliasesUseVerifiedPhysicalRoot(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	storage := &AccountErasureFiles{LegacyRoot: root}
	for _, prefix := range []string{"/upload/", "/old/upload/", "/files/"} {
		file := filepath.Join(root, "legacy.jpg")
		if err = os.WriteFile(file, []byte("synthetic legacy"), 0600); err != nil {
			t.Fatal(err)
		}
		if err = storage.EraseURL(prefix + "legacy.jpg"); err != nil {
			t.Fatal(prefix, err)
		}
		if _, err = os.Stat(file); !os.IsNotExist(err) {
			t.Fatal("legacy file survived")
		}
	}
	for _, raw := range []string{"/upload/../other.jpg", "/old/upload/%2e%2e/other.jpg"} {
		if storage.EraseURL(raw) == nil {
			t.Fatal("legacy alias bypassed traversal guard")
		}
	}
}
