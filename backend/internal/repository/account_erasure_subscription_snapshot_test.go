package repository

import (
	"github.com/dflh-saf/backend/internal/testsupport/mariadb"
	"testing"
)

func TestSubscriptionReviewUsesCurrentLockedSnapshot(t *testing.T) {
	db := mariadb.Start(t).NewDatabase(t, mariadb.Statement(`
 CREATE TABLE WEO_MEMBER(USR_SEQ INT PRIMARY KEY,USR_STATUS CHAR(3)) ENGINE=InnoDB;
 CREATE TABLE SUBSCRIPTION(SUB_SEQ INT PRIMARY KEY,USR_SEQ INT,STATUS VARCHAR(20),BILLING_KEY VARCHAR(64)) ENGINE=InnoDB;
 CREATE TABLE ALUMNI_ERASURE_SUBSCRIPTION_REVIEW(REQUEST_ID BIGINT,SUB_SEQ INT,SOURCE_FINGERPRINT CHAR(64),PROVIDER_CLOSURE_OUTCOME VARCHAR(16),EVIDENCE_REFERENCE VARCHAR(200),OPERATOR_SEQ INT,PRIMARY KEY(REQUEST_ID,SUB_SEQ)) ENGINE=InnoDB;
 INSERT INTO WEO_MEMBER VALUES(100,'AAA');
 INSERT INTO SUBSCRIPTION VALUES(1,100,'failed',NULL);
 `)).DB
	first, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	schema, err := readErasureSchema(first)
	if err != nil {
		t.Fatal(err)
	}
	items, err := subscriptionErasureReviews(first, schema, 7, 100)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Commit(); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO ALUMNI_ERASURE_SUBSCRIPTION_REVIEW VALUES(7,1,?,'failed','synthetic-provider-closure',999)`, items[0].SourceFingerprint); err != nil {
		t.Fatal(err)
	}
	stale, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	defer stale.Rollback()
	var status string
	if err := stale.Get(&status, `SELECT USR_STATUS FROM WEO_MEMBER WHERE USR_SEQ=100`); err != nil {
		t.Fatal(err)
	}
	// A billing callback commits after the erasure transaction established its snapshot.
	if _, err := db.Exec(`UPDATE SUBSCRIPTION SET STATUS='active',BILLING_KEY='synthetic-live-key' WHERE SUB_SEQ=1`); err != nil {
		t.Fatal(err)
	}
	items, err = subscriptionErasureReviews(stale, schema, 7, 100)
	if err != nil {
		t.Fatal(err)
	}
	if items[0].CanReview || items[0].Reviewed || !items[0].HasBillingKey || items[0].Status != "active" {
		t.Fatal("older repeatable-read snapshot authorized erasing an active billing reference")
	}
}
