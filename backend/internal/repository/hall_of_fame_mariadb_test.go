// hall_of_fame_mariadb_test.go — Runs migration 086 and every repository query on pinned MariaDB 10.1.38.
package repository

import (
	"errors"
	"os"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/testsupport/mariadb"
	"github.com/jmoiron/sqlx"
)

func TestHallOfFameRepositoryOnMariaDB101(t *testing.T) {
	if os.Getenv("DFLH_DOCKER_TESTS") != "1" {
		t.Skip("set DFLH_DOCKER_TESTS=1 for pinned MariaDB 10.1.38")
	}
	database := mariadb.Start(t).NewDatabase(t, mariadb.File("../../migrations/086_create_hall_of_fame_entry.sql"))
	repo := NewHallOfFameRepository(database.DB)
	tier, cohort, batch := 1000, 15, "batch-1"
	hash, role := "abc", "교장"
	entry := model.HallOfFameEntry{NaturalKey: "key-1", DisplayName: "김하나", Kind: model.HallOfFameKindAlumni, Cohort: &cohort,
		TierManWon: &tier, RoleLabel: &role, PhoneHash: &hash, ConsentPublic: true, Visible: true, Source: model.HallOfFameSourceExcel, SourceBatchID: &batch}

	err := repo.RunInTransaction(func(tx *sqlx.Tx) error {
		if err := repo.InsertBatchTx(tx, model.HallOfFameImportBatch{ID: batch, FileSHA256: "f", UploadedBy: 1, SummaryJSON: "{}"}); err != nil {
			return err
		}
		return repo.InsertTx(tx, entry)
	})
	if err != nil {
		t.Fatal(err)
	}
	if committed, err := repo.FileAlreadyCommitted("f"); err != nil || !committed {
		t.Fatalf("committed: %v %v", committed, err)
	}
	if _, err := repo.Insert(entry); !errors.Is(err, ErrHallOfFameDuplicateKey) {
		t.Fatalf("duplicate: %v", err)
	}
	public, err := repo.ListPublic()
	if err != nil || len(public) != 1 || !public[0].Visible || !public[0].ConsentPublic || *public[0].PhoneHash != hash || *public[0].RoleLabel != role {
		t.Fatalf("public: %+v %v", public, err)
	}
	stored := public[0]

	visible := true
	rows, total, counts, err := repo.ListAdmin(model.HallOfFameAdminFilter{Tier: &tier, Kind: "alumni", Query: "15기", Visible: &visible, Page: 1, Size: 50})
	if err != nil || total != 1 || len(rows) != 1 || len(counts) != 1 || counts[0].Count != 1 {
		t.Fatalf("admin: %+v %d %+v %v", rows, total, counts, err)
	}
	if rows, total, _, err = repo.ListAdmin(model.HallOfFameAdminFilter{Query: "100%", Page: 1, Size: 50}); err != nil || total != 0 || len(rows) != 0 {
		t.Fatalf("escaped search: %v", err)
	}

	stored.ConsentPublic = false
	if err := repo.Save(stored); err != nil {
		t.Fatal(err)
	}
	if public, err = repo.ListPublic(); err != nil || len(public) != 0 {
		t.Fatalf("consent off must hide: %+v %v", public, err)
	}

	err = repo.RunInTransaction(func(tx *sqlx.Tx) error {
		locked, err := repo.ListAllForImportTx(tx)
		if err != nil || len(locked) != 1 {
			t.Fatalf("locked: %+v %v", locked, err)
		}
		return repo.SoftDeleteImportedTx(tx, []int64{stored.ID})
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.FindActiveByID(stored.ID); !errors.Is(err, ErrHallOfFameEntryNotFound) {
		t.Fatalf("deleted: %v", err)
	}
	found, err := repo.FindByNaturalKey("key-1")
	if err != nil || found == nil || found.DeletedAt == nil || found.Source != model.HallOfFameSourceExcel {
		t.Fatalf("tombstone: %+v %v", found, err)
	}
	if err := repo.Save(*found); err != nil {
		t.Fatal(err)
	}
	if err := repo.SoftDeleteManual(found.ID); err != nil {
		t.Fatal(err)
	}
	found, _ = repo.FindByNaturalKey("key-1")
	if found.Source != model.HallOfFameSourceManual || found.DeletedAt == nil {
		t.Fatalf("manual tombstone: %+v", found)
	}
	if err := repo.SoftDeleteManual(found.ID); !errors.Is(err, ErrHallOfFameEntryNotFound) {
		t.Fatalf("double delete: %v", err)
	}
}
