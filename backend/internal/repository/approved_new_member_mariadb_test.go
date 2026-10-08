package repository

import (
	"testing"
	"time"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/testsupport/mariadb"
)

func TestApprovedNewMemberCanBeFoundMessagedAndBlocked(t *testing.T) {
	db := mariadb.Start(t).NewDatabase(t, append(mariadb.ProdBaseline(t), mariadb.File("../../migrations/081_add_member_message_allowed.sql"))...).DB
	db.MustExec(`INSERT INTO WEO_MEMBER (USR_SEQ, USR_ID, USR_NAME, USR_PWD, USR_STATUS) VALUES
		(101, 'viewer', 'Viewer', 'x', 'CCC'), (202, 'new-member', 'New Member', 'x', 'BBB')`)
	db.MustExec(`INSERT INTO ALUMNI_VERIFICATION (USR_SEQ, STATUS, GRADUATION_YEAR, COHORT, DEPARTMENT, CREATED_AT, UPDATED_AT) VALUES
		(101, 'approved', 2000, '20', 'English', UTC_TIMESTAMP(), UTC_TIMESTAMP()),
		(202, 'pending', 2001, '21', 'English', UTC_TIMESTAMP(), UTC_TIMESTAMP())`)
	var updatedAt time.Time
	if err := db.Get(&updatedAt, `SELECT UPDATED_AT FROM ALUMNI_VERIFICATION WHERE USR_SEQ=202`); err != nil {
		t.Fatal(err)
	}
	if err := NewAdminMemberRepository(db).ApproveAlumniVerification(202, 101, updatedAt); err != nil {
		t.Fatal(err)
	}
	var legacyStatus string
	if err := db.Get(&legacyStatus, `SELECT USR_STATUS FROM WEO_MEMBER WHERE USR_SEQ=202`); err != nil {
		t.Fatal(err)
	}
	if legacyStatus != "BBB" {
		t.Fatalf("approval changed legacy status to %q", legacyStatus)
	}
	blocks := NewMemberBlockRepository(db)
	messages := NewMessageRepository(db)
	alumni := NewAlumniRepository(db)
	for _, check := range []struct {
		name string
		fn   func(int) (bool, error)
	}{{"block target", blocks.IsApprovedAlumni}, {"message recipient", messages.IsApprovedAlumni}} {
		ok, err := check.fn(202)
		if err != nil || !ok {
			t.Errorf("%s: approved BBB member unavailable: %v", check.name, err)
		}
	}
	detail, err := alumni.GetDetail(101, 202)
	if err != nil || detail == nil {
		t.Errorf("approved new member detail: %v / %v", detail, err)
	}
	_, total, err := alumni.Search(model.AlumniSearchParams{})
	if err != nil || total != 2 {
		t.Errorf("search total=%d, want 2: %v", total, err)
	}
	filters, err := alumni.GetFilters()
	if err != nil || filters == nil || len(filters.GraduationYears) != 2 {
		t.Errorf("new member missing from filters: %+v / %v", filters, err)
	}
	names, total, err := alumni.GetWidgetPreview()
	if err != nil || total != 2 || len(names) != 2 {
		t.Errorf("new member missing from preview: %+v / %d / %v", names, total, err)
	}
	db.MustExec(`INSERT INTO ALUMNI_MESSAGE (AM_SENDER_SEQ, AM_RECVR_SEQ, AM_CONTENT, AM_CLIENT_MESSAGE_ID, REG_DATE)
		VALUES (202,101,'hello','new-member-message',UTC_TIMESTAMP())`)
	conversations, err := messages.GetConversations(101, nil, 0, 10)
	if err != nil || len(conversations) != 1 || !conversations[0].RecipientAvailable {
		t.Errorf("approved new member conversation unavailable: %+v / %v", conversations, err)
	}
	state, err := blocks.Block(101, 202)
	if err != nil || state == nil || !state.BlockedByMe {
		t.Errorf("block approved new member: %+v / %v", state, err)
	}
	// Including BBB must not make withdrawn or unapproved accounts eligible.
	for _, change := range []string{
		`UPDATE WEO_MEMBER SET USR_STATUS='AAA' WHERE USR_SEQ=202`,
		`UPDATE WEO_MEMBER SET USR_STATUS='BBB' WHERE USR_SEQ=202; UPDATE ALUMNI_VERIFICATION SET STATUS='pending' WHERE USR_SEQ=202`,
	} {
		db.MustExec(change)
		for _, check := range []func(int) (bool, error){blocks.IsApprovedAlumni, messages.IsApprovedAlumni} {
			ok, err := check(202)
			if err != nil || ok {
				t.Errorf("ineligible member accepted: %v / %v", ok, err)
			}
		}
		if _, err := blocks.Block(101, 202); err != ErrMemberBlockTargetNotApproved {
			t.Errorf("block ineligible member: %v", err)
		}
	}
}
