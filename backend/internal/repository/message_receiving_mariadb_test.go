// message_receiving_mariadb_test.go — Real MariaDB receiving, search, insert guard and unblock round trips.
package repository

import (
	"errors"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/testsupport/mariadb"
	"path/filepath"
	"testing"
)

func TestMessageReceivingAndBlockManagementOnMariaDB101(t *testing.T) {
	schema := append(mariadb.ProdBaseline(t), mariadb.File(filepath.Join("..", "..", "migrations", "081_add_member_message_allowed.sql")))
	db := mariadb.Start(t).NewDatabase(t, schema...).DB
	db.MustExec(`INSERT INTO WEO_MEMBER (USR_SEQ, USR_ID, USR_NAME, USR_PWD, USR_STATUS) VALUES (101,'me','요청 동문','x','CCC'),(202,'peer','상대 동문','x','CCC')`)
	db.MustExec(`INSERT INTO ALUMNI_VERIFICATION (USR_SEQ,STATUS,COHORT,DEPARTMENT,CREATED_AT,UPDATED_AT) VALUES (101,'approved','18','영어',UTC_TIMESTAMP(),UTC_TIMESTAMP()),(202,'approved','20','일본어',UTC_TIMESTAMP(),UTC_TIMESTAMP())`)
	prefs := &MessagePreferencesRepository{DB: db}
	initial, err := prefs.Get(202)
	if err != nil || !initial.MessageAllowed {
		t.Fatalf("default: %v %v", initial, err)
	}
	messages := NewMessageRepository(db)
	first, err := messages.AcceptMessage(101, 202, "before", "이전 쪽지")
	if err != nil {
		t.Fatal(err)
	}
	if err := prefs.Save(202, model.MessagePreferences{MessageAllowed: false}); err != nil {
		t.Fatal(err)
	}
	allowed, err := messages.CanReceiveMessages(202)
	if err != nil || allowed {
		t.Fatalf("opt-out: %v %v", allowed, err)
	}
	_, err = messages.AcceptMessage(101, 202, "during", "거부할 쪽지")
	var rejection *model.MessageSendRejection
	if !errors.As(err, &rejection) || rejection.Code != model.MessageSendReceivingDisabled {
		t.Fatalf("insert guard: %v", err)
	}
	replay, err := messages.FindAcceptedMessage(101, "before")
	if err != nil || replay.MessageID != first.MessageID {
		t.Fatalf("replay: %v %v", replay, err)
	}
	alumni := NewAlumniRepository(db)
	_, total, err := alumni.Search(model.AlumniSearchParams{})
	if err != nil || total != 2 {
		t.Fatalf("directory: total=%d err=%v", total, err)
	}
	_, total, err = alumni.Search(model.AlumniSearchParams{MessageRecipientsOnly: true})
	if err != nil || total != 1 {
		t.Fatalf("recipient search: total=%d err=%v", total, err)
	}
	detail, err := alumni.GetDetail(101, 202)
	if err != nil || detail == nil || detail.MessageAllowed {
		t.Fatalf("detail: %v %v", detail, err)
	}
	history, total, err := messages.GetInbox(202, 1, 20)
	if err != nil || total != 1 || len(history) != 1 {
		t.Fatalf("history: %d %v", total, err)
	}
	if err := prefs.Save(202, model.MessagePreferences{MessageAllowed: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := messages.AcceptMessage(101, 202, "after", "다시 허용"); err != nil {
		t.Fatal(err)
	}
	blocks := NewMemberBlockRepository(db)
	if _, err := blocks.Block(101, 202); err != nil {
		t.Fatal(err)
	}
	list, err := blocks.List(101)
	if err != nil || len(list) != 1 || list[0].Name != "상대 동문" || list[0].Cohort != "20" {
		t.Fatalf("blocked list: %v %v", list, err)
	}
	if _, err := blocks.Unblock(101, 202); err != nil {
		t.Fatal(err)
	}
	list, err = blocks.List(101)
	if err != nil || len(list) != 0 {
		t.Fatalf("unblock: %v %v", list, err)
	}
}
