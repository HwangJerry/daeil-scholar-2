// subscription_service.go — Business logic for recurring donation subscriptions
package service

import (
	"errors"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

// SubscriptionService manages recurring donation subscription lifecycle.
type SubscriptionService struct {
	repo      *repository.SubscriptionRepository
	donateSvc *DonateService
	cache     *cache.Cache
	logger    zerolog.Logger
}

// NewSubscriptionService creates a new SubscriptionService.
func NewSubscriptionService(
	repo *repository.SubscriptionRepository,
	donateSvc *DonateService,
	cacheStore *cache.Cache,
	logger zerolog.Logger,
) *SubscriptionService {
	return &SubscriptionService{
		repo:      repo,
		donateSvc: donateSvc,
		cache:     cacheStore,
		logger:    logger,
	}
}

// CreateSubscription validates input, creates a donation order for PG setup, and inserts a subscription record.
func (s *SubscriptionService) CreateSubscription(
	user *model.AuthUser,
	req model.CreateSubscriptionRequest,
	ip string,
	cfg config.EasyPayConfig,
) (*model.CreateOrderResponse, error) {
	if req.Amount < 10000 {
		return nil, errors.New("최소 정기후원 금액은 10,000원입니다")
	}
	if req.PayType != "CARD" {
		return nil, errors.New("정기후원은 카드 결제만 가능합니다")
	}

	existing, err := s.repo.GetByUser(user.USRSeq)
	if err != nil {
		return nil, errors.New("구독 조회 실패")
	}
	if existing != nil {
		return nil, errors.New("이미 활성화된 정기후원이 있습니다")
	}

	orderReq := model.CreateOrderRequest{
		Amount:  req.Amount,
		PayType: req.PayType,
		Gate:    "profile",
	}
	orderResp, err := s.donateSvc.CreateOrder(user, orderReq, ip, cfg)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	sub := &model.Subscription{
		USRSeq:    user.USRSeq,
		Amount:    req.Amount,
		PayType:   req.PayType,
		Status:    "active",
		StartDate: now,
		NextBill:  now.AddDate(0, 1, 0),
	}
	subSeq, err := s.repo.Insert(sub)
	if err != nil {
		s.logger.Error().Err(err).Int("usrSeq", user.USRSeq).Msg("subscription insert failed")
		return nil, errors.New("정기후원 등록 실패")
	}
	s.logger.Info().Int64("subSeq", subSeq).Int("usrSeq", user.USRSeq).Msg("subscription created")

	return orderResp, nil
}

// GetMySubscription returns the user's active subscription, or nil if none.
func (s *SubscriptionService) GetMySubscription(usrSeq int) (*model.Subscription, error) {
	return s.repo.GetByUser(usrSeq)
}

// CancelSubscription verifies ownership and cancels the subscription.
func (s *SubscriptionService) CancelSubscription(usrSeq, subSeq int) error {
	sub, err := s.repo.GetByUser(usrSeq)
	if err != nil {
		return errors.New("구독 조회 실패")
	}
	if sub == nil || sub.SubSeq != subSeq {
		return errors.New("해당 정기후원을 찾을 수 없습니다")
	}

	if err := s.repo.UpdateStatus(subSeq, "cancelled"); err != nil {
		return errors.New("정기후원 취소 실패")
	}
	s.logger.Info().Int("subSeq", subSeq).Int("usrSeq", usrSeq).Msg("subscription cancelled")
	return nil
}
