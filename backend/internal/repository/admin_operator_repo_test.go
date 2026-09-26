package repository

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/jmoiron/sqlx"
)

func newAdminOperatorRepoMock(t *testing.T) (*AdminOperatorRepository, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return NewAdminOperatorRepository(sqlx.NewDb(db, "sqlmock")), mock
}

func expectMemberStatus(mock sqlmock.Sqlmock, seq int, status string) {
	mock.ExpectQuery(`SELECT USR_STATUS FROM WEO_MEMBER WHERE USR_SEQ = \? FOR UPDATE`).WithArgs(seq).
		WillReturnRows(sqlmock.NewRows([]string{"USR_STATUS"}).AddRow(status))
}

func expectAdminRoleLocks(mock sqlmock.Sqlmock, target int, roots []int, current string) {
	rootRows := sqlmock.NewRows([]string{"USR_SEQ"})
	for _, seq := range roots {
		rootRows.AddRow(seq)
	}
	mock.ExpectQuery(`SELECT USR_SEQ FROM ALUMNI_ADMIN_ROLE WHERE ADMIN_ROLE = 'root'[\s\S]*FOR UPDATE`).WillReturnRows(rootRows)
	currentRows := sqlmock.NewRows([]string{"ADMIN_ROLE"})
	if current != "" {
		currentRows.AddRow(current)
	}
	mock.ExpectQuery(`SELECT ADMIN_ROLE FROM ALUMNI_ADMIN_ROLE WHERE USR_SEQ = \? FOR UPDATE`).WithArgs(target).WillReturnRows(currentRows)
}

func TestSetOperatorRoleGrantsOperatorToActiveMember(t *testing.T) {
	repo, mock := newAdminOperatorRepoMock(t)
	mock.ExpectBegin()
	expectMemberStatus(mock, 42, "CCC")
	expectAdminRoleLocks(mock, 42, []int{1}, "")
	mock.ExpectExec(`INSERT INTO ALUMNI_ADMIN_ROLE`).WithArgs(42, "operator", 1, 1).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := repo.SetOperatorRole(1, 42, model.AdminRoleOperator); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSetOperatorRoleRefusesInactiveOrMissingMember(t *testing.T) {
	for _, status := range []string{"AAA", "ABA", "ACA"} {
		repo, mock := newAdminOperatorRepoMock(t)
		mock.ExpectBegin()
		expectMemberStatus(mock, 42, status)
		mock.ExpectRollback()
		if err := repo.SetOperatorRole(1, 42, model.AdminRoleOperator); !errors.Is(err, ErrOperatorTargetIneligible) {
			t.Fatalf("status %s: error = %v", status, err)
		}
	}

	repo, mock := newAdminOperatorRepoMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT USR_STATUS FROM WEO_MEMBER`).WithArgs(404).WillReturnRows(sqlmock.NewRows([]string{"USR_STATUS"}))
	mock.ExpectRollback()
	if err := repo.SetOperatorRole(1, 404, model.AdminRoleOperator); !errors.Is(err, ErrOperatorTargetIneligible) {
		t.Fatalf("missing member: error = %v", err)
	}
}

func TestSetOperatorRoleRefusesDemotingTheOnlyRoot(t *testing.T) {
	repo, mock := newAdminOperatorRepoMock(t)
	mock.ExpectBegin()
	expectMemberStatus(mock, 42, "ZZZ")
	expectAdminRoleLocks(mock, 42, []int{42}, "root")
	mock.ExpectRollback()

	if err := repo.SetOperatorRole(1, 42, model.AdminRoleOperator); !errors.Is(err, ErrLastRootAdmin) {
		t.Fatalf("error = %v, want ErrLastRootAdmin", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSetOperatorRoleDemotesRootWhenAnotherRootRemains(t *testing.T) {
	repo, mock := newAdminOperatorRepoMock(t)
	mock.ExpectBegin()
	expectMemberStatus(mock, 42, "ZZZ")
	expectAdminRoleLocks(mock, 42, []int{1, 42}, "root")
	mock.ExpectExec(`INSERT INTO ALUMNI_ADMIN_ROLE`).WithArgs(42, "operator", 1, 1).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	if err := repo.SetOperatorRole(1, 42, model.AdminRoleOperator); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRevokeOperatorRemovesRoleAndGuardsLastRoot(t *testing.T) {
	repo, mock := newAdminOperatorRepoMock(t)
	mock.ExpectBegin()
	expectAdminRoleLocks(mock, 42, []int{1}, "operator")
	mock.ExpectExec(`DELETE FROM ALUMNI_ADMIN_ROLE WHERE USR_SEQ = \?`).WithArgs(42).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.RevokeOperator(42); err != nil {
		t.Fatal(err)
	}

	repo, mock = newAdminOperatorRepoMock(t)
	mock.ExpectBegin()
	expectAdminRoleLocks(mock, 42, []int{42}, "root")
	mock.ExpectRollback()
	if err := repo.RevokeOperator(42); !errors.Is(err, ErrLastRootAdmin) {
		t.Fatalf("last root: error = %v", err)
	}

	repo, mock = newAdminOperatorRepoMock(t)
	mock.ExpectBegin()
	expectAdminRoleLocks(mock, 42, []int{1}, "")
	mock.ExpectRollback()
	if err := repo.RevokeOperator(42); !errors.Is(err, ErrOperatorNotFound) {
		t.Fatalf("no role: error = %v", err)
	}
}
