// notification_template_service_test.go — Rendering, fallback, caching, and update behaviour.
package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

// fakeTemplateStore records how often the service reaches the database so the
// caching behaviour can be observed.
type fakeTemplateStore struct {
	rows       map[string]model.NotificationTemplate
	getErr     error
	listErr    error
	updateErr  error
	newVersion int
	getCalls   int
	lastUpdate service.UpdateNotificationTemplateInput
	// onGet runs inside Get, which lets a test interleave an administrator's
	// save with a render that has already started reading.
	onGet func()
}

func (s *fakeTemplateStore) ListAll() ([]model.NotificationTemplate, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	templates := make([]model.NotificationTemplate, 0, len(s.rows))
	for _, row := range s.rows {
		templates = append(templates, row)
	}
	return templates, nil
}

func (s *fakeTemplateStore) Get(key string) (*model.NotificationTemplate, error) {
	s.getCalls++
	if hook := s.onGet; hook != nil {
		s.onGet = nil
		hook()
	}
	if s.getErr != nil {
		return nil, s.getErr
	}
	row, found := s.rows[key]
	if !found {
		return nil, nil
	}
	return &row, nil
}

func (s *fakeTemplateStore) Update(key, title, body string, updatedBy, expectedVersion int) (int, error) {
	s.lastUpdate = service.UpdateNotificationTemplateInput{
		Key: key, Title: title, Body: body, UpdatedBy: updatedBy, ExpectedVersion: expectedVersion,
	}
	if s.updateErr != nil {
		return 0, s.updateErr
	}
	row := s.rows[key]
	row.Key, row.Title, row.Body = key, title, body
	row.Version = s.newVersion
	if row.Channel == "" {
		// NT_CHANNEL is written by the seed and never updated, so a row that
		// exists always carries it; the fake must not invent a blank channel.
		if definition, known := model.NotificationTemplateDefinitionFor(key); known {
			row.Channel = definition.Channel
		}
	}
	if s.rows == nil {
		s.rows = map[string]model.NotificationTemplate{}
	}
	s.rows[key] = row
	return s.newVersion, nil
}

func newTemplateService(store service.NotificationTemplateStore) *service.NotificationTemplateService {
	templates, _ := newTemplateServiceWithCache(store)
	return templates
}

// newTemplateServiceWithCache hands the test the same cache the service uses,
// so a TTL can be inspected instead of waited out.
func newTemplateServiceWithCache(
	store service.NotificationTemplateStore,
) (*service.NotificationTemplateService, *cache.Cache) {
	cacheStore := cache.New(5*time.Minute, 10*time.Minute)
	return service.NewNotificationTemplateService(store, cacheStore, zerolog.Nop()), cacheStore
}

// expireNotificationTemplateCache simulates waiting `elapsed`: the cached entry
// is dropped only when its own TTL would have run out by then, so a test can
// tell the short failure TTL from the normal read TTL without sleeping.
func expireNotificationTemplateCache(t *testing.T, cacheStore *cache.Cache, key string, elapsed time.Duration) {
	t.Helper()
	cacheKey := "notification_template:" + key
	item, found := cacheStore.Items()[cacheKey]
	if !found {
		t.Fatalf("nothing cached for %s", key)
	}
	if time.Until(time.Unix(0, item.Expiration)) > elapsed {
		return
	}
	cacheStore.Delete(cacheKey)
}

func TestRenderUsesStoredTextAndSubstitutesPlaceholders(t *testing.T) {
	store := &fakeTemplateStore{rows: map[string]model.NotificationTemplate{
		model.NotificationTemplateNoticeNew: {
			Key: model.NotificationTemplateNoticeNew, Channel: model.NotificationChannelPush,
			Title: "공지", Body: "{subject} 을(를) 확인하세요.", Version: 3,
		},
	}}

	rendered := newTemplateService(store).Render(
		model.NotificationTemplateNoticeNew,
		map[string]string{"subject": "총회 안내"},
	)

	if rendered.Title != "공지" || rendered.Body != "총회 안내 을(를) 확인하세요." {
		t.Fatalf("rendered = %#v", rendered)
	}
	if rendered.Version != 3 {
		t.Fatalf("Version = %d, want the stored version 3", rendered.Version)
	}
}

// TestRenderTreatsMissingVariableAsEmpty keeps raw template syntax out of a
// member's notification when a caller omits an optional value.
func TestRenderTreatsMissingVariableAsEmpty(t *testing.T) {
	rendered := newTemplateService(&fakeTemplateStore{}).Render(
		model.NotificationTemplateMessagePreviewOff, nil,
	)

	if rendered.Title != "" {
		t.Fatalf("Title = %q, want the unset {senderName} to render empty", rendered.Title)
	}
	if rendered.Body != "새 메시지가 도착했습니다." {
		t.Fatalf("Body = %q", rendered.Body)
	}
}

func TestRenderFallsBackToTheCatalogDefault(t *testing.T) {
	invalidRow := map[string]model.NotificationTemplate{
		model.NotificationTemplateVerificationApproved: {
			Key:     model.NotificationTemplateVerificationApproved,
			Channel: model.NotificationChannelPush,
			Title:   "동문 인증 결과",
			Body:    "결과: {typo}",
			Version: 9,
		},
	}
	tests := []struct {
		name  string
		store *fakeTemplateStore
	}{
		{name: "missing row", store: &fakeTemplateStore{}},
		{name: "database error", store: &fakeTemplateStore{getErr: errors.New("connection refused")}},
		{name: "invalid stored value", store: &fakeTemplateStore{rows: invalidRow}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rendered := newTemplateService(test.store).Render(model.NotificationTemplateVerificationApproved, nil)

			definition, _ := model.NotificationTemplateDefinitionFor(model.NotificationTemplateVerificationApproved)
			if rendered.Body != definition.DefaultBody || rendered.Title != definition.DefaultTitle {
				t.Fatalf("rendered = %#v, want the catalog default", rendered)
			}
			if rendered.Version != 0 {
				t.Fatalf("Version = %d, want 0 to mark the default", rendered.Version)
			}
		})
	}
}

func TestRenderUnknownKeyReturnsEmptyTextWithoutPanicking(t *testing.T) {
	rendered := newTemplateService(&fakeTemplateStore{}).Render("push.does.not.exist", map[string]string{"a": "b"})

	if rendered.Title != "" || rendered.Body != "" {
		t.Fatalf("rendered = %#v, want empty text for an unknown key", rendered)
	}
}

func TestRenderCachesTheResolvedTemplate(t *testing.T) {
	store := &fakeTemplateStore{}
	templates := newTemplateService(store)

	for range 5 {
		templates.Render(model.NotificationTemplateVerificationRejected, nil)
	}

	if store.getCalls != 1 {
		t.Fatalf("getCalls = %d, want 1: the push fan-out must not query per recipient", store.getCalls)
	}
}

func TestUpdateInvalidatesTheCachedTemplate(t *testing.T) {
	store := &fakeTemplateStore{newVersion: 2}
	templates := newTemplateService(store)
	templates.Render(model.NotificationTemplateNoticeNew, map[string]string{"subject": "이전"})

	saved, err := templates.Update(service.UpdateNotificationTemplateInput{
		Key: model.NotificationTemplateNoticeNew, Title: "새 공지", Body: "{subject}", UpdatedBy: 7, ExpectedVersion: 1,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if saved.Version != 2 {
		t.Fatalf("Version = %d, want 2", saved.Version)
	}

	rendered := templates.Render(model.NotificationTemplateNoticeNew, map[string]string{"subject": "총회"})
	if rendered.Title != "새 공지" || rendered.Body != "총회" {
		t.Fatalf("rendered = %#v, want the newly saved text", rendered)
	}
}

func TestUpdateRejectsUnknownKey(t *testing.T) {
	_, err := newTemplateService(&fakeTemplateStore{}).Update(service.UpdateNotificationTemplateInput{
		Key: "push.nope", Title: "제목", Body: "본문",
	})

	if !errors.Is(err, service.ErrNotificationTemplateNotFound) {
		t.Fatalf("error = %v, want ErrNotificationTemplateNotFound", err)
	}
}

func TestUpdateRejectsInvalidTextBeforeTouchingTheStore(t *testing.T) {
	store := &fakeTemplateStore{newVersion: 2}

	_, err := newTemplateService(store).Update(service.UpdateNotificationTemplateInput{
		Key: model.NotificationTemplatePhoneVerificationSMS, Body: "인증번호 {cod}",
	})

	var validation *service.NotificationTemplateValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error = %v, want a validation error", err)
	}
	if store.lastUpdate.Key != "" {
		t.Fatal("an invalid submission must not reach the database")
	}
}

func TestUpdateReportsAVersionConflict(t *testing.T) {
	store := &fakeTemplateStore{updateErr: repository.ErrNotificationTemplateConflict}

	_, err := newTemplateService(store).Update(service.UpdateNotificationTemplateInput{
		Key: model.NotificationTemplateNoticeNew, Title: "새 소식", Body: "{subject}", ExpectedVersion: 4,
	})

	if !errors.Is(err, service.ErrNotificationTemplateConflict) {
		t.Fatalf("error = %v, want ErrNotificationTemplateConflict", err)
	}
	if store.lastUpdate.ExpectedVersion != 4 {
		t.Fatalf("ExpectedVersion = %d, want the administrator's version to reach the store", store.lastUpdate.ExpectedVersion)
	}
}

func TestListJoinsTheCatalogWithStoredOverrides(t *testing.T) {
	updatedBy := 11
	store := &fakeTemplateStore{rows: map[string]model.NotificationTemplate{
		model.NotificationTemplateNoticeNew: {
			Key: model.NotificationTemplateNoticeNew, Channel: model.NotificationChannelPush,
			Title: "공지 도착", Body: "{subject}", Version: 5,
			UpdatedAt: time.Date(2026, 9, 21, 9, 0, 0, 0, time.Local), UpdatedBy: &updatedBy,
		},
	}}

	views, err := newTemplateService(store).List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(views) != len(model.NotificationTemplateCatalog()) {
		t.Fatalf("len(views) = %d, want the whole catalog", len(views))
	}

	byKey := make(map[string]service.TemplateView, len(views))
	for _, view := range views {
		byKey[view.Key] = view
	}

	overridden := byKey[model.NotificationTemplateNoticeNew]
	if overridden.Title != "공지 도착" || overridden.IsDefault || overridden.Version != 5 {
		t.Fatalf("overridden view = %#v", overridden)
	}
	if overridden.UpdatedBy == nil || *overridden.UpdatedBy != updatedBy || overridden.UpdatedAt == nil {
		t.Fatalf("audit fields missing: %#v", overridden)
	}
	if overridden.DefaultBody != "{subject}" || len(overridden.Placeholders) != 1 {
		t.Fatalf("catalog metadata missing: %#v", overridden)
	}

	untouched := byKey[model.NotificationTemplatePhoneVerificationSMS]
	if !untouched.IsDefault || untouched.Body != untouched.DefaultBody {
		t.Fatalf("untouched view = %#v, want the catalog default", untouched)
	}
	if untouched.Limits.MaxBodyEUCKRBytes != model.SMSTemplateMaxEUCKRBytes {
		t.Fatalf("SMS limits missing from the view: %#v", untouched.Limits)
	}
	if byKey[model.NotificationTemplateVerificationApproved].Limits.MaxBodyRunes != model.PushTemplateMaxBodyRunes {
		t.Fatal("push limits missing from the view")
	}
}

func TestListMarksAStoredTemplateTheRendererRefuses(t *testing.T) {
	store := &fakeTemplateStore{rows: map[string]model.NotificationTemplate{
		model.NotificationTemplateNoticeNew: {
			Key: model.NotificationTemplateNoticeNew, Channel: model.NotificationChannelPush,
			Title: "공지", Body: "제목 없음", Version: 2,
		},
	}}

	views, err := newTemplateService(store).List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	for _, view := range views {
		if view.Key != model.NotificationTemplateNoticeNew {
			continue
		}
		if !view.Invalid {
			t.Fatal("a stored body missing its required placeholder must be flagged as invalid")
		}
		return
	}
	t.Fatal("the overridden template is missing from the listing")
}

// TestRenderDoesNotCacheAResultThatLostTheRaceWithAnUpdate covers the
// check-then-set window: a render that read the old row must not install it
// after the administrator's save already invalidated the cache, or the stale
// text would be sent for a full TTL.
func TestRenderDoesNotCacheAResultThatLostTheRaceWithAnUpdate(t *testing.T) {
	store := &fakeTemplateStore{newVersion: 2, rows: map[string]model.NotificationTemplate{
		model.NotificationTemplateNoticeNew: {
			Key: model.NotificationTemplateNoticeNew, Channel: model.NotificationChannelPush,
			Title: "이전 제목", Body: "{subject}", Version: 1,
		},
	}}
	templates := newTemplateService(store)
	// The save lands while the first render is inside its store read.
	store.onGet = func() {
		if _, err := templates.Update(service.UpdateNotificationTemplateInput{
			Key: model.NotificationTemplateNoticeNew, Title: "새 제목", Body: "{subject}",
			UpdatedBy: 3, ExpectedVersion: 1,
		}); err != nil {
			t.Errorf("Update() during render: %v", err)
		}
	}

	templates.Render(model.NotificationTemplateNoticeNew, map[string]string{"subject": "총회"})

	rendered := templates.Render(model.NotificationTemplateNoticeNew, map[string]string{"subject": "총회"})
	if rendered.Title != "새 제목" {
		t.Fatalf("Title = %q, want the saved text: a losing read pinned the stale row", rendered.Title)
	}
}

func TestUpdateRequiresTheEditedVersion(t *testing.T) {
	store := &fakeTemplateStore{newVersion: 2}

	_, err := newTemplateService(store).Update(service.UpdateNotificationTemplateInput{
		Key: model.NotificationTemplateNoticeNew, Title: "새 소식", Body: "{subject}", ExpectedVersion: 0,
	})

	fields := fieldErrors(t, err)
	if !hasFieldError(fields, "expectedVersion") {
		t.Fatalf("fields = %#v, want an expectedVersion error", fields)
	}
	if store.lastUpdate.Key != "" {
		t.Fatal("a submission without a version must not reach the database")
	}
}

// TestRenderRetriesSoonAfterAStoreFailure keeps a transient outage from pinning
// the default for the whole read TTL.
func TestRenderRetriesSoonAfterAStoreFailure(t *testing.T) {
	store := &fakeTemplateStore{getErr: errors.New("connection refused")}
	templates, cacheStore := newTemplateServiceWithCache(store)

	templates.Render(model.NotificationTemplateNoticeNew, nil)
	store.getErr = nil
	store.rows = map[string]model.NotificationTemplate{
		model.NotificationTemplateNoticeNew: {
			Key: model.NotificationTemplateNoticeNew, Channel: model.NotificationChannelPush,
			Title: "복구된 제목", Body: "{subject}", Version: 3,
		},
	}

	// The failure is cached for 30s rather than the 5 minute read TTL; expiry is
	// asserted through the service's own cache rather than by sleeping.
	expireNotificationTemplateCache(t, cacheStore, model.NotificationTemplateNoticeNew, 45*time.Second)

	rendered := templates.Render(model.NotificationTemplateNoticeNew, map[string]string{"subject": "총회"})
	if rendered.Title != "복구된 제목" {
		t.Fatalf("Title = %q, want the row read after the outage cleared", rendered.Title)
	}
}

// TestRenderKeepsAnInvalidRowCachedForTheFullTTL is the counterpart: that
// fallback only changes through Update, which invalidates, so it must not be
// re-read every 30 seconds.
func TestRenderKeepsAnInvalidRowCachedForTheFullTTL(t *testing.T) {
	store := &fakeTemplateStore{rows: map[string]model.NotificationTemplate{
		model.NotificationTemplateNoticeNew: {
			Key: model.NotificationTemplateNoticeNew, Channel: model.NotificationChannelPush,
			Title: "공지", Body: "치환 항목 없음", Version: 2,
		},
	}}
	templates, cacheStore := newTemplateServiceWithCache(store)

	templates.Render(model.NotificationTemplateNoticeNew, nil)
	expireNotificationTemplateCache(t, cacheStore, model.NotificationTemplateNoticeNew, 45*time.Second)
	templates.Render(model.NotificationTemplateNoticeNew, nil)

	if store.getCalls != 1 {
		t.Fatalf("getCalls = %d, want 1: an invalid row must not be re-read on the short TTL", store.getCalls)
	}
}

// TestRenderRejectsARowWhoseChannelDrifted catches a row that is not the
// template its key stands for; its text was never checked against the rules of
// the channel it would be sent on.
func TestRenderRejectsARowWhoseChannelDrifted(t *testing.T) {
	store := &fakeTemplateStore{rows: map[string]model.NotificationTemplate{
		model.NotificationTemplateNoticeNew: {
			Key: model.NotificationTemplateNoticeNew, Channel: model.NotificationChannelSMS,
			Title: "공지", Body: "{subject}", Version: 4,
		},
	}}

	rendered := newTemplateService(store).Render(model.NotificationTemplateNoticeNew, map[string]string{"subject": "총회"})

	definition, _ := model.NotificationTemplateDefinitionFor(model.NotificationTemplateNoticeNew)
	if rendered.Title != definition.DefaultTitle || rendered.Version != 0 {
		t.Fatalf("rendered = %#v, want the catalog default for a channel mismatch", rendered)
	}
}

// TestIsDefaultTracksTheTextNotTheRow matters because migration 075 seeds every
// key: a row exists from the start, so only its content can say whether an
// administrator changed anything.
func TestIsDefaultTracksTheTextNotTheRow(t *testing.T) {
	definition, _ := model.NotificationTemplateDefinitionFor(model.NotificationTemplateNoticeNew)
	store := &fakeTemplateStore{rows: map[string]model.NotificationTemplate{
		model.NotificationTemplateNoticeNew: {
			Key: model.NotificationTemplateNoticeNew, Channel: model.NotificationChannelPush,
			Title: definition.DefaultTitle, Body: definition.DefaultBody, Version: 1,
		},
	}}

	view := viewForKey(t, newTemplateService(store), model.NotificationTemplateNoticeNew)
	if !view.IsDefault {
		t.Fatal("a seeded row still holding the default text must report IsDefault")
	}
	if view.Version != 1 {
		t.Fatalf("Version = %d, want the seeded version to stay visible", view.Version)
	}
}

// TestRenderReportsTheSeededVersion pins the corrected contract: 0 marks the
// fallback path, not "unedited".
func TestRenderReportsTheSeededVersion(t *testing.T) {
	definition, _ := model.NotificationTemplateDefinitionFor(model.NotificationTemplateNoticeNew)
	store := &fakeTemplateStore{rows: map[string]model.NotificationTemplate{
		model.NotificationTemplateNoticeNew: {
			Key: model.NotificationTemplateNoticeNew, Channel: model.NotificationChannelPush,
			Title: definition.DefaultTitle, Body: definition.DefaultBody, Version: 1,
		},
	}}

	rendered := newTemplateService(store).Render(model.NotificationTemplateNoticeNew, map[string]string{"subject": "총회"})

	if rendered.Version != 1 {
		t.Fatalf("Version = %d, want the seeded row's version 1", rendered.Version)
	}
}

// viewForKey picks one row out of the admin list, which is the only read path
// now that the write returns the saved row itself.
func viewForKey(t *testing.T, templates *service.NotificationTemplateService, key string) service.TemplateView {
	t.Helper()
	views, err := templates.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, view := range views {
		if view.Key == key {
			return view
		}
	}
	t.Fatalf("key %q is not in the admin list", key)
	return service.TemplateView{}
}

// The saved row is composed from the committed submission, not read back: a
// re-read could fail after a successful write, or return a rival edit.
func TestUpdateReturnsTheSavedRowWithoutASecondRead(t *testing.T) {
	store := &fakeTemplateStore{newVersion: 7}
	definition, _ := model.NotificationTemplateDefinitionFor(model.NotificationTemplateNoticeNew)

	saved, err := newTemplateService(store).Update(service.UpdateNotificationTemplateInput{
		Key: model.NotificationTemplateNoticeNew, Title: "새 공지", Body: "{subject}",
		UpdatedBy: 9, ExpectedVersion: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if store.getCalls != 0 {
		t.Fatalf("getCalls = %d, want the write to compose its own answer", store.getCalls)
	}
	if saved.Version != 7 || saved.Title != "새 공지" || saved.Body != "{subject}" {
		t.Fatalf("saved = %#v", saved)
	}
	if saved.UpdatedBy == nil || *saved.UpdatedBy != 9 {
		t.Fatalf("UpdatedBy = %v, want the operator who saved", saved.UpdatedBy)
	}
	if saved.UpdatedAt == nil || saved.UpdatedAt.IsZero() {
		t.Fatal("UpdatedAt must be set so the screen can show when the edit landed")
	}
	if saved.Invalid || saved.IsDefault {
		t.Fatalf("saved = %#v, want a valid, non-default row", saved)
	}
	if saved.DefaultBody != definition.DefaultBody || saved.Channel != definition.Channel {
		t.Fatalf("saved = %#v, want the catalog metadata carried through", saved)
	}
}
