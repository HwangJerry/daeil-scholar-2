// notification_template_service.go — Resolves, caches, and updates admin-editable notification texts.
package service

import (
	"database/sql"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

const (
	notificationTemplateCacheTTL    = 5 * time.Minute
	notificationTemplateCachePrefix = "notification_template:"
	notificationTemplateWarnPrefix  = "notification_template_warn:"
	// notificationTemplateErrorCacheTTL bounds how long a transient store
	// failure keeps the default in place. A database blip must not pin the
	// fallback for the full read TTL, but it must still absorb a push fan-out.
	notificationTemplateErrorCacheTTL = 30 * time.Second
)

var (
	ErrNotificationTemplateNotFound = errors.New("notification template not found")
	// ErrNotificationTemplateConflict is raised when the row changed while the
	// administrator was editing it.
	ErrNotificationTemplateConflict = repository.ErrNotificationTemplateConflict
)

// NotificationTemplateStore persists administrator overrides of the texts whose
// keys are fixed in the catalog.
type NotificationTemplateStore interface {
	ListAll() ([]model.NotificationTemplate, error)
	Get(key string) (*model.NotificationTemplate, error)
	Update(key, title, body string, updatedBy int, expectedVersion int) (int, error)
}

// RenderedTemplate is one notification's final text. Version is 0 only when the
// fallback path was used — the row was missing, unreadable, or unusable — and
// otherwise carries the stored row's version, including the seeded version 1.
type RenderedTemplate struct {
	Key     string
	Version int
	Title   string
	Body    string
}

// TemplateView is one row of the admin editing screen: the catalog metadata
// joined with whatever is stored.
type TemplateView struct {
	Key          string                                  `json:"key"`
	Channel      string                                  `json:"channel"`
	DisplayName  string                                  `json:"displayName"`
	Description  string                                  `json:"description"`
	Title        string                                  `json:"title"`
	Body         string                                  `json:"body"`
	DefaultTitle string                                  `json:"defaultTitle"`
	DefaultBody  string                                  `json:"defaultBody"`
	IsDefault    bool                                    `json:"isDefault"`
	Invalid      bool                                    `json:"invalid"`
	Version      int                                     `json:"version"`
	UpdatedAt    *time.Time                              `json:"updatedAt"`
	UpdatedBy    *int                                    `json:"updatedBy"`
	Placeholders []model.NotificationTemplatePlaceholder `json:"placeholders"`
	Limits       model.NotificationChannelLimits         `json:"limits"`
}

// UpdateNotificationTemplateInput is one administrator's submission.
type UpdateNotificationTemplateInput struct {
	Key   string
	Title string
	Body  string
	// ExpectedVersion is the version the administrator was editing. It is
	// mandatory: a submission without it is rejected rather than allowed to
	// overwrite a concurrent edit.
	ExpectedVersion int
	UpdatedBy       int
}

// NotificationTemplateService reads notification texts on the sending path and
// writes them on the admin path.
//
// The sending path never fails: every notification that could be sent with the
// catalog default is still sent when the store is unreachable or its content is
// unusable. Reads are cached because push fan-out renders the same template
// once per recipient shard.
type NotificationTemplateService struct {
	store  NotificationTemplateStore
	cache  *cache.Cache
	logger zerolog.Logger
	// generations counts invalidations per key. A read that started before an
	// update must not install its stale result afterwards, and the cache has no
	// compare-and-set of its own.
	generations sync.Map // key string -> *atomic.Uint64
}

func NewNotificationTemplateService(
	store NotificationTemplateStore,
	cacheStore *cache.Cache,
	logger zerolog.Logger,
) *NotificationTemplateService {
	return &NotificationTemplateService{store: store, cache: cacheStore, logger: logger}
}

// Catalog returns the fixed set of templates administrators may edit.
func (s *NotificationTemplateService) Catalog() []model.NotificationTemplateDefinition {
	return model.NotificationTemplateCatalog()
}

// Render produces the text for one notification. It never returns an error and
// never blocks a send: any problem degrades to the catalog default and is
// logged once per key per cache TTL. It is safe for concurrent use.
func (s *NotificationTemplateService) Render(key string, vars map[string]string) RenderedTemplate {
	definition, known := model.NotificationTemplateDefinitionFor(key)
	if !known {
		// An unknown key is a call-site bug, not a data problem; there is no
		// default to fall back to, so the caller gets empty text rather than a
		// message containing raw template syntax.
		s.warnOncePerTTL(key, errors.New("unknown notification template key"), "unknown notification template key")
		return RenderedTemplate{Key: key}
	}

	resolved := s.resolve(definition)
	return RenderedTemplate{
		Key:     key,
		Version: resolved.Version,
		Title:   renderTemplateText(resolved.Title, definition, vars),
		Body:    renderTemplateText(resolved.Body, definition, vars),
	}
}

// List joins the catalog with the stored overrides for the admin screen.
func (s *NotificationTemplateService) List() ([]TemplateView, error) {
	stored, err := s.store.ListAll()
	if err != nil {
		return nil, err
	}
	byKey := make(map[string]model.NotificationTemplate, len(stored))
	for _, template := range stored {
		byKey[template.Key] = template
	}

	catalog := model.NotificationTemplateCatalog()
	views := make([]TemplateView, 0, len(catalog))
	for _, definition := range catalog {
		var stored *model.NotificationTemplate
		if row, found := byKey[definition.Key]; found {
			stored = &row
		}
		views = append(views, buildTemplateView(definition, stored))
	}
	return views, nil
}

// Update validates and stores one administrator's edit, returning the row as it
// now stands. The cached copy is dropped so the next send uses the new text.
//
// The returned view is composed from the submission that was just committed
// rather than read back: a re-read could fail after a successful write — making
// a saved edit look rejected — or return a rival administrator's row as if it
// were this one's result.
func (s *NotificationTemplateService) Update(input UpdateNotificationTemplateInput) (*TemplateView, error) {
	definition, known := model.NotificationTemplateDefinitionFor(input.Key)
	if !known {
		return nil, ErrNotificationTemplateNotFound
	}
	if err := validateTemplateSubmission(definition, input); err != nil {
		return nil, err
	}

	version, err := s.store.Update(input.Key, input.Title, input.Body, input.UpdatedBy, input.ExpectedVersion)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// The catalog key exists but the seed row does not: the migration
			// has not been applied on this database.
			return nil, ErrNotificationTemplateNotFound
		}
		return nil, err
	}
	s.invalidate(input.Key)

	updatedBy := input.UpdatedBy
	saved := model.NotificationTemplate{
		Key:       input.Key,
		Channel:   definition.Channel,
		Title:     input.Title,
		Body:      input.Body,
		Version:   version,
		UpdatedAt: time.Now(),
		UpdatedBy: &updatedBy,
	}
	view := buildTemplateView(definition, &saved)
	return &view, nil
}

// resolvedTemplate is the stored-or-default text held in the cache, before
// placeholder substitution.
type resolvedTemplate struct {
	Title   string
	Body    string
	Version int
}

// resolve returns the text to render, reading through the cache.
//
// The generation captured before the read is re-checked before the result is
// stored: a render that read the old row while an administrator was saving
// would otherwise repopulate the cache after the update invalidated it and pin
// the stale text for a whole TTL.
func (s *NotificationTemplateService) resolve(definition model.NotificationTemplateDefinition) resolvedTemplate {
	cacheKey := notificationTemplateCachePrefix + definition.Key
	if s.cache != nil {
		if cached, found := s.cache.Get(cacheKey); found {
			if resolved, ok := cached.(resolvedTemplate); ok {
				return resolved
			}
		}
	}

	generation := s.generation(definition.Key)
	resolved, ttl := s.loadOrDefault(definition)
	if s.cache != nil && s.generation(definition.Key) == generation {
		s.cache.Set(cacheKey, resolved, ttl)
	}
	return resolved
}

// loadOrDefault reads one row and reports how long its result may be cached.
// A store failure is cached only briefly, because it is expected to clear on
// its own; every other outcome changes only through Update, which invalidates.
func (s *NotificationTemplateService) loadOrDefault(
	definition model.NotificationTemplateDefinition,
) (resolvedTemplate, time.Duration) {
	fallback := resolvedTemplate{Title: definition.DefaultTitle, Body: definition.DefaultBody}

	stored, err := s.store.Get(definition.Key)
	if err != nil {
		s.warnOncePerTTL(definition.Key, err, "notification template lookup failed; using default text")
		return fallback, notificationTemplateErrorCacheTTL
	}
	if stored == nil {
		s.warnOncePerTTL(definition.Key, nil, "notification template row missing; using default text")
		return fallback, notificationTemplateCacheTTL
	}
	// A channel that disagrees with the catalog means the row is not the
	// template this key stands for — a bad hand edit, or a seed that drifted —
	// and its text was never checked against the rules of the channel it would
	// actually be sent on.
	if stored.Channel != definition.Channel {
		s.warnOncePerTTL(definition.Key, nil, "stored notification template channel does not match the catalog; using default text")
		return fallback, notificationTemplateCacheTTL
	}
	// A row can become invalid without anyone editing it — a catalog change that
	// removes a placeholder, or a hand-edited database. Sending the default is
	// better than sending text with a broken token in it.
	if err := ValidateNotificationTemplate(definition, stored.Title, stored.Body); err != nil {
		s.warnOncePerTTL(definition.Key, err, "stored notification template is invalid; using default text")
		return fallback, notificationTemplateCacheTTL
	}
	return resolvedTemplate{Title: stored.Title, Body: stored.Body, Version: stored.Version}, notificationTemplateCacheTTL
}

// generation reads the current invalidation counter for one key.
func (s *NotificationTemplateService) generation(key string) uint64 {
	counter, _ := s.generations.LoadOrStore(key, &atomic.Uint64{})
	return counter.(*atomic.Uint64).Load()
}

func (s *NotificationTemplateService) invalidate(key string) {
	// The counter is bumped before the entries are dropped, so a read racing
	// this update sees the new generation and declines to cache its result.
	counter, _ := s.generations.LoadOrStore(key, &atomic.Uint64{})
	counter.(*atomic.Uint64).Add(1)
	if s.cache == nil {
		return
	}
	s.cache.Delete(notificationTemplateCachePrefix + key)
	s.cache.Delete(notificationTemplateWarnPrefix + key)
}

// validateTemplateSubmission collects every reason one administrator's edit is
// refused, so the screen can mark all the offending inputs at once.
func validateTemplateSubmission(
	definition model.NotificationTemplateDefinition,
	input UpdateNotificationTemplateInput,
) error {
	fields := make([]NotificationTemplateFieldError, 0)
	for _, err := range []error{
		ValidateNotificationTemplateVersion(input.ExpectedVersion),
		ValidateNotificationTemplate(definition, input.Title, input.Body),
	} {
		var validation *NotificationTemplateValidationError
		if errors.As(err, &validation) {
			fields = append(fields, validation.Fields...)
		}
	}
	if len(fields) == 0 {
		return nil
	}
	return &NotificationTemplateValidationError{Fields: fields}
}

// warnOncePerTTL keeps a broken template from filling the log with one line per
// notification: the same key is reported at most once per cache lifetime.
func (s *NotificationTemplateService) warnOncePerTTL(key string, cause error, message string) {
	if s.cache != nil {
		warnKey := notificationTemplateWarnPrefix + key
		if _, alreadyWarned := s.cache.Get(warnKey); alreadyWarned {
			return
		}
		s.cache.Set(warnKey, true, notificationTemplateCacheTTL)
	}
	s.logger.Warn().Err(cause).Str("templateKey", key).Msg(message)
}

// buildTemplateView merges one catalog entry with its stored row, if any.
func buildTemplateView(
	definition model.NotificationTemplateDefinition,
	stored *model.NotificationTemplate,
) TemplateView {
	view := TemplateView{
		Key:          definition.Key,
		Channel:      definition.Channel,
		DisplayName:  definition.DisplayName,
		Description:  definition.Description,
		Title:        definition.DefaultTitle,
		Body:         definition.DefaultBody,
		DefaultTitle: definition.DefaultTitle,
		DefaultBody:  definition.DefaultBody,
		IsDefault:    true,
		Placeholders: definition.Placeholders,
		Limits:       definition.Limits(),
	}
	if view.Placeholders == nil {
		view.Placeholders = []model.NotificationTemplatePlaceholder{}
	}

	if stored == nil {
		return view
	}
	row := *stored
	view.Title = row.Title
	view.Body = row.Body
	view.Version = row.Version
	// Migration 075 seeds every key, so the presence of a row says nothing about
	// whether anyone edited it: only text that still equals the catalog default
	// is reported as default.
	view.IsDefault = row.Title == definition.DefaultTitle && row.Body == definition.DefaultBody
	// Invalid marks a row the renderer refuses to use, so the screen does not
	// present unusable text as the message members are receiving.
	view.Invalid = ValidateNotificationTemplate(definition, row.Title, row.Body) != nil
	updatedAt := row.UpdatedAt
	view.UpdatedAt = &updatedAt
	view.UpdatedBy = row.UpdatedBy
	return view
}
