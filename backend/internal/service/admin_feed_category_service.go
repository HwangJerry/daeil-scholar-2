// admin_feed_category_service.go — Business rules for admin-managed feed post categories
package service

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/patrickmn/go-cache"
)

// FeedCategoryRuleError is a refused category change. Message is the Korean
// text the admin UI shows as-is.
type FeedCategoryRuleError struct {
	Code    string
	Message string
}

func (e *FeedCategoryRuleError) Error() string { return e.Code }

var (
	ErrFeedCategoryNameInvalid   = &FeedCategoryRuleError{"INVALID_NAME", "카테고리 이름은 1~8자로 입력하세요."}
	ErrFeedCategoryNameDuplicate = &FeedCategoryRuleError{"DUPLICATE_NAME", "이미 있는 카테고리 이름입니다."}
	ErrFeedCategoryLimit         = &FeedCategoryRuleError{"CATEGORY_LIMIT", "카테고리는 최대 6개까지 만들 수 있습니다."}
	ErrFeedCategoryDefaultDelete = &FeedCategoryRuleError{"DEFAULT_CATEGORY_DELETE", "기본 카테고리는 삭제할 수 없습니다."}
	ErrFeedCategoryDefaultHide   = &FeedCategoryRuleError{"DEFAULT_CATEGORY_HIDE", "기본 카테고리는 숨길 수 없습니다."}
	ErrFeedCategoryOpenInvalid   = &FeedCategoryRuleError{"INVALID_OPEN_YN", "앱 탭 노출 값이 올바르지 않습니다."}
	ErrFeedCategoryOrderInvalid  = &FeedCategoryRuleError{"INVALID_ORDER", "모든 카테고리를 한 번씩 포함한 순서를 보내 주세요."}
	ErrFeedCategoryMoveInvalid   = &FeedCategoryRuleError{"INVALID_MOVE_TARGET", "옮길 카테고리를 다시 선택하세요."}
	ErrFeedCategoryUnknown       = &FeedCategoryRuleError{"UNKNOWN_CATEGORY", "존재하지 않는 카테고리입니다."}
)

// ErrFeedCategoryNotFound is returned when the addressed category does not exist.
var ErrFeedCategoryNotFound = errors.New("feed_category_not_found")

// FeedCategoryHasPostsError refuses deleting a category that still has posts
// when no category to move them to was given.
type FeedCategoryHasPostsError struct {
	PostCount int
}

func (e *FeedCategoryHasPostsError) Error() string { return "feed_category_has_posts" }

type AdminFeedCategoryService struct {
	repo  repository.AdminFeedCategoryStore
	cache *cache.Cache
}

func NewAdminFeedCategoryService(repo repository.AdminFeedCategoryStore, cacheStore *cache.Cache) *AdminFeedCategoryService {
	return &AdminFeedCategoryService{repo: repo, cache: cacheStore}
}

func (s *AdminFeedCategoryService) List() ([]model.AdminFeedCategory, error) {
	return s.repo.GetAll()
}

func (s *AdminFeedCategoryService) Create(req model.AdminFeedCategoryUpsert) (*model.AdminFeedCategory, error) {
	name, err := normalizeFeedCategoryName(req.Name)
	if err != nil {
		return nil, err
	}
	openYN, err := normalizeFeedCategoryOpenYN(req.OpenYN, "Y")
	if err != nil {
		return nil, err
	}
	all, err := s.repo.GetAll()
	if err != nil {
		return nil, err
	}
	if len(all) >= model.FeedCategoryMaxCount {
		return nil, ErrFeedCategoryLimit
	}
	if hasFeedCategoryName(all, name, 0) {
		return nil, ErrFeedCategoryNameDuplicate
	}
	seq, err := s.repo.Insert(name, openYN)
	if err != nil {
		return nil, err
	}
	s.invalidateFeed()
	return s.repo.GetBySeq(seq)
}

func (s *AdminFeedCategoryService) Update(seq int, req model.AdminFeedCategoryUpsert) (*model.AdminFeedCategory, error) {
	current, err := s.repo.GetBySeq(seq)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, ErrFeedCategoryNotFound
	}
	name, err := normalizeFeedCategoryName(req.Name)
	if err != nil {
		return nil, err
	}
	openYN, err := normalizeFeedCategoryOpenYN(req.OpenYN, current.OpenYN)
	if err != nil {
		return nil, err
	}
	if current.IsDefault == "Y" && openYN == "N" {
		return nil, ErrFeedCategoryDefaultHide
	}
	all, err := s.repo.GetAll()
	if err != nil {
		return nil, err
	}
	if hasFeedCategoryName(all, name, seq) {
		return nil, ErrFeedCategoryNameDuplicate
	}
	if err := s.repo.Update(seq, name, openYN); err != nil {
		return nil, err
	}
	s.invalidateFeed()
	return s.repo.GetBySeq(seq)
}

// Reorder requires every existing category seq exactly once.
func (s *AdminFeedCategoryService) Reorder(seqs []int) error {
	all, err := s.repo.GetAll()
	if err != nil {
		return err
	}
	if !isFeedCategoryPermutation(all, seqs) {
		return ErrFeedCategoryOrderInvalid
	}
	if err := s.repo.Reorder(seqs); err != nil {
		return err
	}
	s.invalidateFeed()
	return nil
}

// Delete removes a non-default category. Its posts move to moveToSeq (another
// existing category) in the same transaction; without moveToSeq a category that
// still has posts is refused with FeedCategoryHasPostsError.
func (s *AdminFeedCategoryService) Delete(seq int, moveToSeq *int) error {
	current, err := s.repo.GetBySeq(seq)
	if err != nil {
		return err
	}
	if current == nil {
		return ErrFeedCategoryNotFound
	}
	if current.IsDefault == "Y" {
		return ErrFeedCategoryDefaultDelete
	}
	if moveToSeq != nil {
		if *moveToSeq == seq {
			return ErrFeedCategoryMoveInvalid
		}
		target, err := s.repo.GetBySeq(*moveToSeq)
		if err != nil {
			return err
		}
		if target == nil {
			return ErrFeedCategoryMoveInvalid
		}
	}
	postCount, err := s.repo.Delete(seq, moveToSeq)
	if err != nil {
		return err
	}
	if postCount > 0 {
		return &FeedCategoryHasPostsError{PostCount: postCount}
	}
	s.invalidateFeed()
	return nil
}

// invalidateFeed drops the cached hero, which carries the category name.
func (s *AdminFeedCategoryService) invalidateFeed() {
	if s.cache != nil {
		s.cache.Delete(feedHeroCacheKey)
	}
}

func normalizeFeedCategoryName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if n := utf8.RuneCountInString(name); n < 1 || n > model.FeedCategoryNameMaxRunes {
		return "", ErrFeedCategoryNameInvalid
	}
	return name, nil
}

// normalizeFeedCategoryOpenYN accepts Y or N; an empty value means fallback.
func normalizeFeedCategoryOpenYN(raw, fallback string) (string, error) {
	switch raw {
	case "":
		return fallback, nil
	case "Y", "N":
		return raw, nil
	default:
		return "", ErrFeedCategoryOpenInvalid
	}
}

// hasFeedCategoryName reports a case-insensitive name clash with any category
// other than exceptSeq.
func hasFeedCategoryName(all []model.AdminFeedCategory, name string, exceptSeq int) bool {
	for _, cat := range all {
		if cat.Seq != exceptSeq && strings.EqualFold(strings.TrimSpace(cat.Name), name) {
			return true
		}
	}
	return false
}

func isFeedCategoryPermutation(all []model.AdminFeedCategory, seqs []int) bool {
	if len(seqs) != len(all) {
		return false
	}
	remaining := make(map[int]bool, len(all))
	for _, cat := range all {
		remaining[cat.Seq] = true
	}
	for _, seq := range seqs {
		if !remaining[seq] {
			return false
		}
		delete(remaining, seq)
	}
	return true
}
