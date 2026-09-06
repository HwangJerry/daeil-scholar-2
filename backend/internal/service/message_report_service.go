// message_report_service.go — Report validation and moderator resolution rules.
package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"strings"
	"unicode/utf8"
)

const maximumReportDetails = 1000

type MessageReportStore interface {
	Create(int, model.MessageReportRequest) (int64, error)
	List(string, int64) ([]model.MessageReport, error)
	Resolve(int64, int, model.MessageReportResolution) error
}

type MessageReportService struct{ Store MessageReportStore }

func (s *MessageReportService) Create(reporter int, request model.MessageReportRequest) (int64, error) {
	request.Details = strings.TrimSpace(request.Details)
	if reporter <= 0 || request.MessageID <= 0 || utf8.RuneCountInString(request.Details) > maximumReportDetails {
		return 0, &model.ValidationError{Msg: "신고 내용을 확인해주세요."}
	}
	switch request.Reason {
	case "harassment", "spam", "inappropriate", "other":
	default:
		return 0, &model.ValidationError{Msg: "신고 사유를 선택해주세요."}
	}
	return s.Store.Create(reporter, request)
}

func (s *MessageReportService) List(status string, before int64) ([]model.MessageReport, error) {
	if status == "" {
		status = "open"
	}
	if (status != "open" && status != "removed" && status != "dismissed") || before < 0 {
		return nil, &model.ValidationError{Msg: "올바른 신고 목록을 선택해주세요."}
	}
	return s.Store.List(status, before)
}

func (s *MessageReportService) Resolve(id int64, moderator int, resolution model.MessageReportResolution) error {
	resolution.Note = strings.TrimSpace(resolution.Note)
	if id <= 0 || moderator <= 0 || (resolution.Status != "removed" && resolution.Status != "dismissed") ||
		resolution.Note == "" || utf8.RuneCountInString(resolution.Note) > maximumReportDetails {
		return &model.ValidationError{Msg: "처리 결과와 사유를 입력해주세요."}
	}
	return s.Store.Resolve(id, moderator, resolution)
}
