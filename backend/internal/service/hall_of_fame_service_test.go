// hall_of_fame_service_test.go — Public response shape and operator edit rules.
package service

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
)

type fakeHallOfFameCrudRepo struct {
	entry   *model.HallOfFameEntry
	byKey   *model.HallOfFameEntry
	saved   *model.HallOfFameEntry
	deleted int64
}

func (f *fakeHallOfFameCrudRepo) ListPublic() ([]model.HallOfFameEntry, error) { return nil, nil }
func (f *fakeHallOfFameCrudRepo) ListAdmin(model.HallOfFameAdminFilter) ([]model.HallOfFameEntry, int, []model.HallOfFameTierCount, error) {
	return nil, 0, nil, nil
}
func (f *fakeHallOfFameCrudRepo) FindActiveByID(int64) (*model.HallOfFameEntry, error) {
	if f.entry == nil {
		return nil, repository.ErrHallOfFameEntryNotFound
	}
	return f.entry, nil
}
func (f *fakeHallOfFameCrudRepo) FindByNaturalKey(string) (*model.HallOfFameEntry, error) {
	return f.byKey, nil
}
func (f *fakeHallOfFameCrudRepo) Insert(entry model.HallOfFameEntry) (int64, error) {
	entry.ID = 1
	f.entry = &entry
	return 1, nil
}
func (f *fakeHallOfFameCrudRepo) Save(entry model.HallOfFameEntry) error {
	f.saved, f.entry = &entry, &entry
	return nil
}
func (f *fakeHallOfFameCrudRepo) SoftDeleteManual(id int64) error {
	f.deleted = id
	return nil
}

func hallOfFameTestEntry(id int64, name string, cohort, tier *int) model.HallOfFameEntry {
	return model.HallOfFameEntry{ID: id, DisplayName: name, Kind: model.HallOfFameKindAlumni, Cohort: cohort, TierManWon: tier,
		Source: model.HallOfFameSourceExcel, Visible: true, ConsentPublic: true}
}

func TestPublicResponseOrderingAndEmptyTiers(t *testing.T) {
	one, two, tier := 1, 2, 500
	phone := "secret-hash"
	teacher := hallOfFameTestEntry(4, "다교사", nil, &tier)
	teacher.Kind, teacher.PhoneHash = model.HallOfFameKindTeacher, &phone
	response := buildHallOfFamePublicResponse([]model.HallOfFameEntry{
		teacher,
		hallOfFameTestEntry(1, "나둘", &two, &tier),
		hallOfFameTestEntry(2, "라넷", &one, &tier),
		hallOfFameTestEntry(3, "가셋", &one, &tier),
	})
	if len(response.Tiers) != 4 {
		t.Fatalf("tiers: %+v", response.Tiers)
	}
	for index, want := range []int{1000, 500, 300, 100} {
		if response.Tiers[index].TierManWon != want || response.Tiers[index].Entries == nil {
			t.Fatalf("tier %d: %+v", index, response.Tiers[index])
		}
	}
	names := make([]string, 0, 4)
	for _, entry := range response.Tiers[1].Entries {
		names = append(names, entry.DisplayName)
	}
	if strings.Join(names, ",") != "가셋,라넷,나둘,다교사" {
		t.Fatalf("order: %v", names)
	}
	encoded, _ := json.Marshal(response)
	body := string(encoded)
	for _, forbidden := range []string{"count", "Count", "total", "kind", "phone", "secret-hash", "amount", "teacher"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("public JSON contains %q: %s", forbidden, body)
		}
	}
	if !strings.Contains(string(encoded), `"entries":[]`) {
		t.Fatalf("empty tiers must be []: %s", encoded)
	}
}

func TestPatchFlipsSourceOnlyForContentEdits(t *testing.T) {
	tier := 100
	repo := &fakeHallOfFameCrudRepo{}
	entry := hallOfFameTestEntry(1, "김하나", nil, &tier)
	repo.entry = &entry
	service := NewHallOfFameService(repo)

	var toggle model.HallOfFamePatchInput
	if err := json.Unmarshal([]byte(`{"visible":false}`), &toggle); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Patch(1, toggle, hallOfFameTestActor); err != nil {
		t.Fatal(err)
	}
	if repo.saved.Source != model.HallOfFameSourceExcel || repo.saved.Visible {
		t.Fatalf("toggle: %+v", repo.saved)
	}

	var edit model.HallOfFamePatchInput
	if err := json.Unmarshal([]byte(`{"tierManWon":null,"cohort":15}`), &edit); err != nil {
		t.Fatal(err)
	}
	if !edit.TierManWon.Set || edit.TierManWon.Value != nil {
		t.Fatalf("explicit null must be set: %+v", edit.TierManWon)
	}
	if _, err := service.Patch(1, edit, hallOfFameTestActor); err != nil {
		t.Fatal(err)
	}
	if repo.saved.Source != model.HallOfFameSourceManual || repo.saved.TierManWon != nil || *repo.saved.Cohort != 15 {
		t.Fatalf("edit: %+v", repo.saved)
	}
}

func TestPatchRejectsInvalidTierAndMissingEntry(t *testing.T) {
	tier := 100
	entry := hallOfFameTestEntry(1, "김하나", nil, &tier)
	service := NewHallOfFameService(&fakeHallOfFameCrudRepo{entry: &entry})
	var input model.HallOfFamePatchInput
	_ = json.Unmarshal([]byte(`{"tierManWon":5000}`), &input)
	if _, err := service.Patch(1, input, hallOfFameTestActor); !errors.Is(err, ErrHallOfFameInvalidInput) {
		t.Fatalf("err = %v", err)
	}
	if _, err := NewHallOfFameService(&fakeHallOfFameCrudRepo{}).Patch(1, input, hallOfFameTestActor); !errors.Is(err, repository.ErrHallOfFameEntryNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestCreateHonoursTogglesAndRoleLabel(t *testing.T) {
	repo := &fakeHallOfFameCrudRepo{}
	consent, role := false, "명예교장"
	created, err := NewHallOfFameService(repo).Create(model.HallOfFameCreateInput{
		DisplayName: "김하나", Kind: model.HallOfFameKindTeacher, ConsentPublic: &consent, RoleLabel: &role,
	}, hallOfFameTestActor)
	if err != nil {
		t.Fatal(err)
	}
	if created.ConsentPublic || !created.Visible || created.Label != role {
		t.Fatalf("created: %+v", created)
	}
}

func TestPatchRoleLabelCountsAsContentEdit(t *testing.T) {
	tier := 100
	repo := &fakeHallOfFameCrudRepo{}
	entry := hallOfFameTestEntry(1, "김하나", nil, &tier)
	repo.entry = &entry
	var input model.HallOfFamePatchInput
	_ = json.Unmarshal([]byte(`{"roleLabel":"교장"}`), &input)
	if _, err := NewHallOfFameService(repo).Patch(1, input, hallOfFameTestActor); err != nil {
		t.Fatal(err)
	}
	if repo.saved.Source != model.HallOfFameSourceManual || *repo.saved.RoleLabel != "교장" {
		t.Fatalf("saved: %+v", repo.saved)
	}
}

func TestCreateIsManualAndRefusesLiveDuplicates(t *testing.T) {
	repo := &fakeHallOfFameCrudRepo{}
	created, err := NewHallOfFameService(repo).Create(model.HallOfFameCreateInput{DisplayName: " 김하나 "}, hallOfFameTestActor)
	if err != nil {
		t.Fatal(err)
	}
	if created.Source != model.HallOfFameSourceManual || created.DisplayName != "김하나" || created.Kind != model.HallOfFameKindAlumni {
		t.Fatalf("created: %+v", created)
	}
	live := model.HallOfFameEntry{ID: 9}
	if _, err := NewHallOfFameService(&fakeHallOfFameCrudRepo{byKey: &live}).Create(model.HallOfFameCreateInput{DisplayName: "김하나"}, hallOfFameTestActor); !errors.Is(err, ErrHallOfFameDuplicate) {
		t.Fatalf("err = %v", err)
	}
	if _, err := NewHallOfFameService(repo).Create(model.HallOfFameCreateInput{DisplayName: ""}, hallOfFameTestActor); !errors.Is(err, ErrHallOfFameInvalidInput) {
		t.Fatalf("err = %v", err)
	}
}
