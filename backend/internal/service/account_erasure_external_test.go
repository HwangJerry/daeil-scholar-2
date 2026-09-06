// account_erasure_external_test.go — Acknowledgement must prove all required external scopes.
package service

import (
	"context"
	"encoding/json"
	"github.com/dflh-saf/backend/internal/model"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExternalErasureRejectsPartialOrUnrelatedConfirmation(t *testing.T) {
	reply := map[string]interface{}{"requestId": int64(17), "complete": true, "retentionRespected": true, "backupsErased": true, "externalDataErased": true, "historicalFilesErased": true, "otherIdentifiersChecked": true, "evidenceReference": "test-proof"}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer fixture-secret" || r.Method != "POST" {
			t.Error("missing authenticated POST")
		}
		var subject model.ErasureExternalSubject
		if json.NewDecoder(r.Body).Decode(&subject) != nil || subject.RequestID != 17 {
			t.Error("incorrect subject")
		}
		json.NewEncoder(w).Encode(reply)
	}))
	defer server.Close()
	old := http.DefaultTransport
	http.DefaultTransport = server.Client().Transport
	defer func() { http.DefaultTransport = old }()
	processor := &HTTPErasureProcessor{Endpoint: server.URL, Token: "fixture-secret"}
	subject := model.ErasureExternalSubject{RequestID: 17, UserSeq: 42}
	if proof, err := processor.Erase(context.Background(), subject); err != nil || proof != "test-proof" {
		t.Fatal(proof, err)
	}
	for _, field := range []string{"complete", "retentionRespected", "backupsErased", "externalDataErased", "historicalFilesErased", "otherIdentifiersChecked"} {
		reply[field] = false
		if _, err := processor.Erase(context.Background(), subject); err == nil {
			t.Fatal("partial result accepted", field)
		}
		reply[field] = true
	}
	reply["requestId"] = 18
	if _, err := processor.Erase(context.Background(), subject); err == nil {
		t.Fatal("other request accepted")
	}
	processor.Endpoint = "http://insecure.example"
	if _, err := processor.Erase(context.Background(), subject); err == nil {
		t.Fatal("insecure processor accepted")
	}
}
