package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"strings"
	"testing"
)

func TestInvalidReportsDoNotReachStorage(t *testing.T) {
	s := &MessageReportService{} // Any accidental storage call panics.
	for _, request := range []model.MessageReportRequest{
		{MessageID: 0, Reason: "spam"},
		{MessageID: 1, Reason: "unrecognized"},
		{MessageID: 1, Reason: "spam", Details: strings.Repeat("가", 1001)},
	} {
		if _, err := s.Create(42, request); err == nil {
			t.Fatal("invalid report accepted")
		}
	}
	for _, request := range []model.MessageReportResolution{
		{Status: "open", Note: "invalid transition"},
		{Status: "removed", Note: " "},
		{Status: "dismissed", Note: strings.Repeat("가", 1001)},
	} {
		if err := s.Resolve(1, 7, request); err == nil {
			t.Fatal("invalid resolution accepted")
		}
	}
}
