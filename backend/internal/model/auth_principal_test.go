// auth_principal_test.go — Verifies the canonical authenticated principal JSON contract.
package model

import (
	"encoding/json"
	"testing"
)

func TestAuthUserJSONIncludesCanonicalAuthorizationState(t *testing.T) {
	user := AuthUser{
		USRSeq:    42,
		USRID:     "legacy-id",
		USRName:   "Member",
		Email:     "member@example.com",
		AdminRole: nil,
		Verification: AlumniVerification{
			Status: VerificationPending,
		},
	}

	encoded, err := json.Marshal(user)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["email"] != "member@example.com" {
		t.Fatalf("email = %#v", payload["email"])
	}
	if role, exists := payload["adminRole"]; !exists || role != nil {
		t.Fatalf("adminRole = %#v, exists = %v", role, exists)
	}
	verification, ok := payload["verification"].(map[string]interface{})
	if !ok || verification["status"] != string(VerificationPending) {
		t.Fatalf("verification = %#v", payload["verification"])
	}
}

func TestVerificationStatusRecognizesCanonicalStates(t *testing.T) {
	for _, status := range []VerificationStatus{
		VerificationUnsubmitted,
		VerificationPending,
		VerificationRejected,
		VerificationApproved,
		VerificationReapprovalPending,
	} {
		if !status.Valid() {
			t.Fatalf("status %q must be valid", status)
		}
	}
	if VerificationStatus("unknown").Valid() {
		t.Fatal("unknown verification status must be invalid")
	}
}
