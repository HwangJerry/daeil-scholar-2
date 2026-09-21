// notification_template_repo_test.go — SQL contract tests for notification template persistence.
package repository_test

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/jmoiron/sqlx"
)

func newTemplateRepo(t *testing.T) (*repository.NotificationTemplateRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	return repository.NewNotificationTemplateRepository(sqlx.NewDb(db, "sqlmock")), mock, func() { _ = db.Close() }
}

func templateColumns() []string {
	return []string{"NT_KEY", "NT_CHANNEL", "NT_TITLE", "NT_BODY", "NT_VERSION", "UPDATED_AT", "UPDATED_BY"}
}

func TestNotificationTemplateRepositoryListsAllTemplates(t *testing.T) {
	repo, mock, closeDB := newTemplateRepo(t)
	defer closeDB()
	updatedAt := time.Date(2026, 9, 21, 10, 0, 0, 0, time.Local)

	mock.ExpectQuery(`(?s)SELECT NT_KEY, NT_CHANNEL, NT_TITLE, NT_BODY, NT_VERSION, UPDATED_AT, UPDATED_BY.*FROM notification_templates.*ORDER BY NT_KEY ASC`).
		WillReturnRows(sqlmock.NewRows(templateColumns()).
			AddRow("push.notice.new", "push", "새 소식", "{subject}", 2, updatedAt, 7))

	templates, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll() error = %v", err)
	}
	if len(templates) != 1 || templates[0].Key != "push.notice.new" || templates[0].Version != 2 {
		t.Fatalf("ListAll() = %#v", templates)
	}
	if templates[0].UpdatedBy == nil || *templates[0].UpdatedBy != 7 {
		t.Fatalf("UpdatedBy = %#v", templates[0].UpdatedBy)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestNotificationTemplateRepositoryGetMissingRowIsNotAnError keeps a database
// that has not run migration 075 from failing every notification: the caller
// falls back to the catalog default.
func TestNotificationTemplateRepositoryGetMissingRowIsNotAnError(t *testing.T) {
	repo, mock, closeDB := newTemplateRepo(t)
	defer closeDB()

	mock.ExpectQuery(`(?s)FROM notification_templates.*WHERE NT_KEY = \?`).
		WithArgs("sms.phone_verification").
		WillReturnRows(sqlmock.NewRows(templateColumns()))

	template, err := repo.Get("sms.phone_verification")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if template != nil {
		t.Fatalf("Get() = %#v, want nil", template)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestNotificationTemplateRepositoryGetReturnsStoredRow(t *testing.T) {
	repo, mock, closeDB := newTemplateRepo(t)
	defer closeDB()

	mock.ExpectQuery(`(?s)FROM notification_templates.*WHERE NT_KEY = \?`).
		WithArgs("push.notice.new").
		WillReturnRows(sqlmock.NewRows(templateColumns()).
			AddRow("push.notice.new", "push", "새 소식", "{subject}", 4, time.Now(), nil))

	template, err := repo.Get("push.notice.new")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if template == nil || template.Version != 4 || template.Body != "{subject}" {
		t.Fatalf("Get() = %#v", template)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestNotificationTemplateRepositoryUpdateBumpsTheVersion(t *testing.T) {
	repo, mock, closeDB := newTemplateRepo(t)
	defer closeDB()

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT NT_VERSION.*FROM notification_templates.*WHERE NT_KEY = \?.*FOR UPDATE`).
		WithArgs("push.notice.new").
		WillReturnRows(sqlmock.NewRows([]string{"NT_VERSION"}).AddRow(4))
	mock.ExpectExec(`(?s)UPDATE notification_templates.*SET NT_TITLE = \?, NT_BODY = \?, NT_VERSION = NT_VERSION \+ 1`).
		WithArgs("새 소식", "{subject}", 7, "push.notice.new").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	version, err := repo.Update("push.notice.new", "새 소식", "{subject}", 7, 4)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if version != 5 {
		t.Fatalf("version = %d, want 5", version)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestNotificationTemplateRepositoryUpdateRejectsAStaleVersion proves a
// concurrent edit is reported rather than overwritten: nothing is written and
// the transaction rolls back.
func TestNotificationTemplateRepositoryUpdateRejectsAStaleVersion(t *testing.T) {
	repo, mock, closeDB := newTemplateRepo(t)
	defer closeDB()

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT NT_VERSION.*FOR UPDATE`).
		WithArgs("push.notice.new").
		WillReturnRows(sqlmock.NewRows([]string{"NT_VERSION"}).AddRow(6))
	mock.ExpectRollback()

	_, err := repo.Update("push.notice.new", "새 소식", "{subject}", 7, 4)

	if !errors.Is(err, repository.ErrNotificationTemplateConflict) {
		t.Fatalf("error = %v, want ErrNotificationTemplateConflict", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestNotificationTemplateRepositoryUpdateRequiresAPositiveVersion proves there
// is no bypass of the concurrency check: an omitted version is refused at the
// storage layer too, not only in the service.
func TestNotificationTemplateRepositoryUpdateRequiresAPositiveVersion(t *testing.T) {
	for _, expectedVersion := range []int{0, -1} {
		repo, mock, closeDB := newTemplateRepo(t)

		mock.ExpectBegin()
		mock.ExpectQuery(`(?s)SELECT NT_VERSION.*FOR UPDATE`).
			WithArgs("push.notice.new").
			WillReturnRows(sqlmock.NewRows([]string{"NT_VERSION"}).AddRow(6))
		mock.ExpectRollback()

		_, err := repo.Update("push.notice.new", "새 소식", "{subject}", 7, expectedVersion)

		if !errors.Is(err, repository.ErrNotificationTemplateConflict) {
			t.Fatalf("expectedVersion %d: error = %v, want ErrNotificationTemplateConflict", expectedVersion, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("expectedVersion %d: %v", expectedVersion, err)
		}
		closeDB()
	}
}

// TestNotificationTemplateRepositoryUpdateStoresAnAbsentEditorAsNull keeps a
// write without an administrator looking like the seeded rows rather than
// attributing it to member 0.
func TestNotificationTemplateRepositoryUpdateStoresAnAbsentEditorAsNull(t *testing.T) {
	repo, mock, closeDB := newTemplateRepo(t)
	defer closeDB()

	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT NT_VERSION.*FOR UPDATE`).
		WithArgs("push.notice.new").
		WillReturnRows(sqlmock.NewRows([]string{"NT_VERSION"}).AddRow(1))
	mock.ExpectExec(`(?s)UPDATE notification_templates`).
		WithArgs("새 소식", "{subject}", nil, "push.notice.new").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if _, err := repo.Update("push.notice.new", "새 소식", "{subject}", 0, 1); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
