// comment_report_service.go — Report validation and moderator resolution rules.
package service

import (
	"github.com/dflh-saf/backend/internal/model"
	"strings"
	"unicode/utf8"
)

type CommentReportStore interface {
	Create(int, model.CommentReportRequest) (int64, error)
	List(string, int64) ([]model.CommentReport, error)
	Resolve(int64, int, model.CommentReportResolution) error
}

type CommentReportService struct {
	Store          CommentReportStore
	InvalidateFeed func()
}

func (s *CommentReportService) Create(reporter int, request model.CommentReportRequest) (int64, error) {
	request.Details = strings.TrimSpace(request.Details)
	if reporter <= 0 || request.PostID <= 0 || request.CommentID <= 0 || utf8.RuneCountInString(request.Details) > maximumReportDetails {
		return 0, &model.ValidationError{Msg: "신고 내용을 확인해주세요."}
	}
	switch request.Reason {
	case "harassment", "spam", "inappropriate", "other":
	default:
		return 0, &model.ValidationError{Msg: "신고 사유를 선택해주세요."}
	}
	return s.Store.Create(reporter, request)
}

func (s *CommentReportService) List(status string, before int64) ([]model.CommentReport, error) {
	if status == "" {
		status = "open"
	}
	if (status != "open" && status != "removed" && status != "dismissed") || before < 0 {
		return nil, &model.ValidationError{Msg: "올바른 신고 목록을 선택해주세요."}
	}
	return s.Store.List(status, before)
}

func (s *CommentReportService) Resolve(id int64, moderator int, resolution model.CommentReportResolution) error {
	resolution.Note = strings.TrimSpace(resolution.Note)
	if id <= 0 || moderator <= 0 || (resolution.Status != "removed" && resolution.Status != "dismissed") ||
		resolution.Note == "" || utf8.RuneCountInString(resolution.Note) > maximumReportDetails {
		return &model.ValidationError{Msg: "처리 결과와 사유를 입력해주세요."}
	}
	err := s.Store.Resolve(id, moderator, resolution)
	if err == nil && resolution.Status == "removed" && s.InvalidateFeed != nil {
		s.InvalidateFeed()
	}
	return err
}
