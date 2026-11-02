// hall_of_fame_service.go — Public wall assembly and operator CRUD with audit logging.
package service

import (
	"errors"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	// ErrHallOfFameDuplicate means a live row already has this identity.
	ErrHallOfFameDuplicate = errors.New("hall of fame entry already exists")
	// ErrHallOfFameInvalidInput wraps operator input the service rejects.
	ErrHallOfFameInvalidInput = errors.New("invalid hall of fame input")
)

// HallOfFameActor identifies the operator for the audit log.
type HallOfFameActor struct {
	UsrSeq int
	IP     string
}

// logHallOfFameAudit starts an audit line. Names and hashes stay out of the logs;
// the entry id is enough to trace a change. The caller adds fields and calls Msg.
func logHallOfFameAudit(actor HallOfFameActor, action string) *zerolog.Event {
	return log.Info().Str("audit", "hall_of_fame").Str("action", action).Int("actorSeq", actor.UsrSeq).Str("ip", actor.IP)
}

const (
	hallOfFameDefaultPageSize = 50
	hallOfFameMaxPageSize     = 200
)

type hallOfFameRepository interface {
	ListPublic() ([]model.HallOfFameEntry, error)
	ListAdmin(filter model.HallOfFameAdminFilter) ([]model.HallOfFameEntry, int, []model.HallOfFameTierCount, error)
	FindActiveByID(id int64) (*model.HallOfFameEntry, error)
	FindByNaturalKey(naturalKey string) (*model.HallOfFameEntry, error)
	Insert(entry model.HallOfFameEntry) (int64, error)
	Save(entry model.HallOfFameEntry) error
	SoftDeleteManual(id int64) error
}

type HallOfFameService struct{ repo hallOfFameRepository }

func NewHallOfFameService(repo hallOfFameRepository) *HallOfFameService {
	return &HallOfFameService{repo: repo}
}

// GetPublic groups publishable rows by tier. Every tier is present even when empty.
func (s *HallOfFameService) GetPublic() (*model.HallOfFamePublicResponse, error) {
	entries, err := s.repo.ListPublic()
	if err != nil {
		return nil, err
	}
	return buildHallOfFamePublicResponse(entries), nil
}

func buildHallOfFamePublicResponse(entries []model.HallOfFameEntry) *model.HallOfFamePublicResponse {
	sorted := append([]model.HallOfFameEntry{}, entries...)
	sort.SliceStable(sorted, func(i, j int) bool { return hallOfFameDisplayLess(sorted[i], sorted[j]) })
	byTier := make(map[int][]model.HallOfFamePublicEntry, len(model.HallOfFameTiersDescending))
	for _, entry := range sorted {
		if entry.TierManWon == nil {
			continue
		}
		byTier[*entry.TierManWon] = append(byTier[*entry.TierManWon], model.HallOfFamePublicEntry{
			DisplayName: entry.DisplayName,
			Label:       hallOfFameLabel(entry.Kind, entry.Cohort, entry.Department, entry.RoleLabel),
		})
	}
	response := &model.HallOfFamePublicResponse{Tiers: make([]model.HallOfFamePublicTier, 0, len(model.HallOfFameTiersDescending))}
	for _, tier := range model.HallOfFameTiersDescending {
		tierEntries := byTier[tier]
		if tierEntries == nil {
			tierEntries = []model.HallOfFamePublicEntry{}
		}
		response.Tiers = append(response.Tiers, model.HallOfFamePublicTier{TierManWon: tier, Entries: tierEntries})
	}
	return response
}

// hallOfFameDisplayLess orders by cohort (missing last), then name. Hangul syllable
// code points follow 가나다 order, so plain string comparison is correct.
func hallOfFameDisplayLess(left, right model.HallOfFameEntry) bool {
	if (left.Cohort == nil) != (right.Cohort == nil) {
		return left.Cohort != nil
	}
	if left.Cohort != nil && *left.Cohort != *right.Cohort {
		return *left.Cohort < *right.Cohort
	}
	if left.DisplayName != right.DisplayName {
		return left.DisplayName < right.DisplayName
	}
	return left.ID < right.ID
}

func (s *HallOfFameService) ListAdmin(filter model.HallOfFameAdminFilter) (*model.HallOfFameAdminList, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Size < 1 {
		filter.Size = hallOfFameDefaultPageSize
	}
	filter.Size = min(filter.Size, hallOfFameMaxPageSize)
	entries, total, counts, err := s.repo.ListAdmin(filter)
	if err != nil {
		return nil, err
	}
	list := &model.HallOfFameAdminList{
		Items: make([]model.HallOfFameAdminEntry, 0, len(entries)), Total: total, Page: filter.Page,
		Size: filter.Size, TierCounts: counts,
	}
	for _, entry := range entries {
		list.Items = append(list.Items, toHallOfFameAdminEntry(entry))
	}
	return list, nil
}

func toHallOfFameAdminEntry(entry model.HallOfFameEntry) model.HallOfFameAdminEntry {
	return model.HallOfFameAdminEntry{HallOfFameEntry: entry, Label: hallOfFameLabel(entry.Kind, entry.Cohort, entry.Department, entry.RoleLabel)}
}

func cleanHallOfFameOptional(value *string) *string {
	if value == nil {
		return nil
	}
	cleaned := normalizeHallOfFameText(*value)
	if cleaned == "" {
		return nil
	}
	return &cleaned
}

func validateHallOfFameEntry(entry model.HallOfFameEntry) error {
	switch {
	case entry.DisplayName == "":
		return hallOfFameInputError("이름이 필요합니다.")
	case utf8.RuneCountInString(entry.DisplayName) > hallOfFameMaxNameRunes:
		return hallOfFameInputError("이름은 100자 이하여야 합니다.")
	case !entry.Kind.Valid():
		return hallOfFameInputError("종류 값이 올바르지 않습니다.")
	case entry.Cohort != nil && (*entry.Cohort <= 0 || *entry.Cohort > hallOfFameMaxCohort):
		return hallOfFameInputError("기수 값이 올바르지 않습니다.")
	case entry.Department != nil && utf8.RuneCountInString(*entry.Department) > hallOfFameMaxDeptRunes:
		return hallOfFameInputError("과는 30자 이하여야 합니다.")
	case entry.TierManWon != nil && !model.IsHallOfFameTier(*entry.TierManWon):
		return hallOfFameInputError("등급은 100/300/500/1000만원 중 하나여야 합니다.")
	case entry.ContactName != nil && utf8.RuneCountInString(*entry.ContactName) > hallOfFameMaxNameRunes:
		return hallOfFameInputError("담당자 이름은 100자 이하여야 합니다.")
	case entry.RoleLabel != nil && utf8.RuneCountInString(*entry.RoleLabel) > hallOfFameMaxRoleRunes:
		return hallOfFameInputError("직함은 30자 이하여야 합니다.")
	}
	return nil
}

func hallOfFameInputError(message string) error {
	return &hallOfFameValidationError{message: message}
}

type hallOfFameValidationError struct{ message string }

func (e *hallOfFameValidationError) Error() string { return e.message }
func (e *hallOfFameValidationError) Unwrap() error { return ErrHallOfFameInvalidInput }

// Create adds a manual row. A key that matches a soft-deleted row revives that row,
// since the UNIQUE natural_key also covers deleted rows.
func (s *HallOfFameService) Create(input model.HallOfFameCreateInput, actor HallOfFameActor) (*model.HallOfFameAdminEntry, error) {
	if input.Kind == "" {
		input.Kind = model.HallOfFameKindAlumni
	}
	entry := model.HallOfFameEntry{
		DisplayName: normalizeHallOfFameText(input.DisplayName), Kind: input.Kind, Cohort: input.Cohort,
		Department: cleanHallOfFameOptional(input.Department), TierManWon: input.TierManWon,
		ContactName: cleanHallOfFameOptional(input.ContactName), RoleLabel: cleanHallOfFameOptional(input.RoleLabel),
		ConsentPublic: input.ConsentPublic == nil || *input.ConsentPublic,
		Visible:       input.Visible == nil || *input.Visible,
		Source:        model.HallOfFameSourceManual,
	}
	if err := validateHallOfFameEntry(entry); err != nil {
		return nil, err
	}
	entry.NaturalKey = hallOfFameNaturalKey(entry.DisplayName, entry.Cohort, entry.Kind, nil)
	existing, err := s.repo.FindByNaturalKey(entry.NaturalKey)
	if err != nil {
		return nil, err
	}
	switch {
	case existing == nil:
		if entry.ID, err = s.repo.Insert(entry); err != nil {
			return nil, err
		}
	case existing.DeletedAt == nil:
		return nil, ErrHallOfFameDuplicate
	default:
		entry.ID = existing.ID
		if err := s.repo.Save(entry); err != nil {
			return nil, err
		}
	}
	logHallOfFameAudit(actor, "create").Int64("entryId", entry.ID).Msg("admin hall of fame change")
	return s.adminEntry(entry.ID)
}

func (s *HallOfFameService) adminEntry(id int64) (*model.HallOfFameAdminEntry, error) {
	stored, err := s.repo.FindActiveByID(id)
	if err != nil {
		return nil, err
	}
	result := toHallOfFameAdminEntry(*stored)
	return &result, nil
}

// Patch applies the present fields. Editing imported content flips the row to manual
// so later imports leave it alone; the visible/consent toggles do not, because
// imports never touch those and the row should keep receiving tier updates.
func (s *HallOfFameService) Patch(id int64, input model.HallOfFamePatchInput, actor HallOfFameActor) (*model.HallOfFameAdminEntry, error) {
	entry, err := s.repo.FindActiveByID(id)
	if err != nil {
		return nil, err
	}
	updated := *entry
	if input.DisplayName.Set {
		updated.DisplayName = ""
		if input.DisplayName.Value != nil {
			updated.DisplayName = normalizeHallOfFameText(*input.DisplayName.Value)
		}
	}
	if input.Kind.Set {
		updated.Kind = ""
		if input.Kind.Value != nil {
			updated.Kind = *input.Kind.Value
		}
	}
	if input.Cohort.Set {
		updated.Cohort = input.Cohort.Value
	}
	if input.Department.Set {
		updated.Department = cleanHallOfFameOptional(input.Department.Value)
	}
	if input.TierManWon.Set {
		updated.TierManWon = input.TierManWon.Value
	}
	if input.ContactName.Set {
		updated.ContactName = cleanHallOfFameOptional(input.ContactName.Value)
	}
	if input.RoleLabel.Set {
		updated.RoleLabel = cleanHallOfFameOptional(input.RoleLabel.Value)
	}
	if input.ConsentPublic.Set {
		if input.ConsentPublic.Value == nil {
			return nil, hallOfFameInputError("공개 동의 값이 필요합니다.")
		}
		updated.ConsentPublic = *input.ConsentPublic.Value
	}
	if input.Visible.Set {
		if input.Visible.Value == nil {
			return nil, hallOfFameInputError("노출 값이 필요합니다.")
		}
		updated.Visible = *input.Visible.Value
	}
	if err := validateHallOfFameEntry(updated); err != nil {
		return nil, err
	}
	contentChanged := len(hallOfFameRowChanges(*entry, model.HallOfFameImportRow{
		DisplayName: updated.DisplayName, Kind: updated.Kind, Cohort: updated.Cohort,
		Department: updated.Department, TierManWon: updated.TierManWon, ContactName: updated.ContactName, RoleLabel: updated.RoleLabel,
	})) > 0
	if contentChanged {
		updated.Source = model.HallOfFameSourceManual
	}
	if err := s.repo.Save(updated); err != nil {
		return nil, err
	}
	logHallOfFameAudit(actor, "update").Int64("entryId", id).Bool("contentChanged", contentChanged).
		Bool("visible", updated.Visible).Bool("consentPublic", updated.ConsentPublic).Msg("admin hall of fame change")
	return s.adminEntry(id)
}

// Delete soft-deletes a row and keeps it out of future imports.
func (s *HallOfFameService) Delete(id int64, actor HallOfFameActor) error {
	if err := s.repo.SoftDeleteManual(id); err != nil {
		return err
	}
	logHallOfFameAudit(actor, "delete").Int64("entryId", id).Msg("admin hall of fame change")
	return nil
}

// HallOfFameRejectionMessage extracts the operator-safe text of a validation error.
func HallOfFameRejectionMessage(err error) string {
	var validation *hallOfFameValidationError
	if errors.As(err, &validation) {
		return strings.TrimSpace(validation.message)
	}
	return "입력값이 올바르지 않습니다."
}
