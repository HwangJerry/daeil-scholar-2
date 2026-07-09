package service

import (
	"errors"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

type fakeAdminNoticeStore struct {
	insertSeq   int
	insertErr   error
	insertCalls int
}

func (f *fakeAdminNoticeStore) GetNotices(page, size int, keyword string) ([]model.AdminNoticeRow, int, error) {
	return nil, 0, nil
}

func (f *fakeAdminNoticeStore) GetNoticeForEdit(seq int) (*model.NoticeDetail, error) {
	return nil, nil
}

func (f *fakeAdminNoticeStore) InsertNotice(n *model.AdminNoticeInsert) (int, error) {
	f.insertCalls++
	if f.insertErr != nil {
		return 0, f.insertErr
	}
	return f.insertSeq, nil
}

func (f *fakeAdminNoticeStore) UpdateNotice(seq int, n *model.AdminNoticeInsert) error {
	return nil
}

func (f *fakeAdminNoticeStore) DeleteNotice(seq int) error {
	return nil
}

func (f *fakeAdminNoticeStore) TogglePin(seq int) error {
	return nil
}

func (f *fakeAdminNoticeStore) CountNotices() (int, error) {
	return 0, nil
}

type fakeAdminNoticeFileStore struct {
	attachErr   error
	attachCalls int
	noticeSeq   int
	fileSeqs    []int
}

func (f *fakeAdminNoticeFileStore) GetAttachmentsByNotice(noticeSeq int) ([]model.FileRecord, error) {
	return nil, nil
}

func (f *fakeAdminNoticeFileStore) AttachFilesToNotice(noticeSeq int, fileSeqs []int) error {
	f.attachCalls++
	f.noticeSeq = noticeSeq
	f.fileSeqs = append([]int(nil), fileSeqs...)
	return f.attachErr
}

func (f *fakeAdminNoticeFileStore) ReconcileAttachments(noticeSeq int, keepFSeqs []int) error {
	return nil
}

func (f *fakeAdminNoticeFileStore) SoftDeleteFilesByJoin(noticeSeq int) error {
	return nil
}

type fakePostPushNotifier struct {
	calls     int
	authorSeq int
	postSeq   int
	subject   string
}

func (f *fakePostPushNotifier) NotifyPostPublished(authorSeq int, postSeq int, subject string) {
	f.calls++
	f.authorSeq = authorSeq
	f.postSeq = postSeq
	f.subject = subject
}

func TestAdminNoticeServiceCreateNotifiesPostPublishedAfterSuccess(t *testing.T) {
	repo := &fakeAdminNoticeStore{insertSeq: 555}
	fileRepo := &fakeAdminNoticeFileStore{}
	notifier := &fakePostPushNotifier{}
	svc := &AdminNoticeService{repo: repo, fileRepo: fileRepo, notifier: notifier}

	seq, err := svc.Create("공지 제목", "공지 내용", "관리자", 99, "Y", []int{10, 11})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seq != 555 {
		t.Fatalf("unexpected seq: got %d want 555", seq)
	}
	if notifier.calls != 1 {
		t.Fatalf("expected one push call, got %d", notifier.calls)
	}
	if notifier.authorSeq != 99 || notifier.postSeq != 555 || notifier.subject != "공지 제목" {
		t.Fatalf("unexpected push args: %#v", notifier)
	}
	if fileRepo.attachCalls != 1 || fileRepo.noticeSeq != 555 {
		t.Fatalf("expected attachments reconciled before push, got %#v", fileRepo)
	}
}

func TestAdminNoticeServiceCreateDoesNotNotifyWhenInsertFails(t *testing.T) {
	repo := &fakeAdminNoticeStore{insertErr: errors.New("insert failed")}
	fileRepo := &fakeAdminNoticeFileStore{}
	notifier := &fakePostPushNotifier{}
	svc := &AdminNoticeService{repo: repo, fileRepo: fileRepo, notifier: notifier}

	_, err := svc.Create("공지 제목", "공지 내용", "관리자", 99, "Y", []int{10})
	if err == nil {
		t.Fatal("expected insert error")
	}
	if notifier.calls != 0 {
		t.Fatalf("expected no push call, got %d", notifier.calls)
	}
	if fileRepo.attachCalls != 0 {
		t.Fatalf("expected no attachment call, got %d", fileRepo.attachCalls)
	}
}

func TestAdminNoticeServiceCreateDoesNotNotifyWhenAttachFilesFails(t *testing.T) {
	repo := &fakeAdminNoticeStore{insertSeq: 555}
	fileRepo := &fakeAdminNoticeFileStore{attachErr: errors.New("attach failed")}
	notifier := &fakePostPushNotifier{}
	svc := &AdminNoticeService{repo: repo, fileRepo: fileRepo, notifier: notifier}

	seq, err := svc.Create("공지 제목", "공지 내용", "관리자", 99, "Y", []int{10})
	if err == nil {
		t.Fatal("expected attachment error")
	}
	if seq != 555 {
		t.Fatalf("expected inserted seq returned on attachment failure, got %d", seq)
	}
	if notifier.calls != 0 {
		t.Fatalf("expected no push call, got %d", notifier.calls)
	}
}
