// donate_service.go — Business logic for donation order creation and payment confirmation
package service

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

type DonateService struct {
	repo    *repository.DonateRepository
	cache   *cache.Cache
	epSvc   *EasyPayService
	logger  zerolog.Logger
	pgAudit *PGAuditLogger
}

func NewDonateService(repo *repository.DonateRepository, epSvc *EasyPayService, cacheStore *cache.Cache, logger zerolog.Logger, pgAudit *PGAuditLogger) *DonateService {
	return &DonateService{repo: repo, epSvc: epSvc, cache: cacheStore, logger: logger, pgAudit: pgAudit}
}

func (s *DonateService) CreateOrder(user *model.AuthUser, req model.CreateOrderRequest, ip string, cfg config.EasyPayConfig) (*model.CreateOrderResponse, error) {
	if req.Amount < 10000 {
		return nil, errors.New("최소 기부 금액은 10,000원입니다")
	}
	validPayTypes := map[string]bool{"CARD": true, "BANK": true, "HP": true}
	if !validPayTypes[req.PayType] {
		return nil, errors.New("지원하지 않는 결제 수단입니다")
	}
	validGates := map[string]bool{"immediately": true, "profile": true}
	if !validGates[req.Gate] {
		return nil, errors.New("지원하지 않는 후원 유형입니다")
	}

	gateCode := "S"
	if req.Gate == "profile" {
		gateCode = "P"
	}

	orderSeq, err := s.repo.InsertOrder(user.USRSeq, gateCode, req.PayType, req.Amount, ip)
	if err != nil {
		return nil, fmt.Errorf("주문 생성 실패: %w", err)
	}

	if req.PayType == "BANK" {
		return &model.CreateOrderResponse{OrderSeq: int(orderSeq), PaymentParams: nil}, nil
	}

	spPayType := resolveEasyPayType(req.PayType, req.Gate)
	mallID := cfg.ImmediatelyMallID
	windowType := "iframe"
	productSuffix := "일시후원"
	if req.Gate == "profile" {
		mallID = cfg.ProfileMallID
		windowType = "submit"
		productSuffix = "정기후원"
	}

	orderNo := strconv.FormatInt(orderSeq, 10)
	params := model.PaymentParams{
		MallID:      mallID,
		OrderNo:     orderNo,
		ProductAmt:  strconv.Itoa(req.Amount),
		ProductName: fmt.Sprintf("%s님_%s_%s원", user.USRName, productSuffix, formatComma(req.Amount)),
		PayType:     spPayType,
		ReturnURL:   cfg.ReturnBaseURL + "/pg/easypay/return",
		RelayURL:    cfg.ReturnBaseURL + "/pg/easypay/relay",
		WindowType:  windowType,
		UserName:    user.USRName,
		MallName:    "대일외국어고등학교 장학회",
		Currency:    "00",
		Charset:     "UTF-8",
		LangFlag:    "KOR",
	}

	return &model.CreateOrderResponse{
		OrderSeq:      int(orderSeq),
		PaymentParams: &params,
	}, nil
}

func resolveEasyPayType(payType string, gate string) string {
	if gate == "profile" && payType == "CARD" {
		return "81"
	}
	switch payType {
	case "HP":
		return "31"
	default:
		return "11"
	}
}

func (s *DonateService) ConfirmPayment(orderNo string, encData string, sessionKey string, traceNo string, clientIP string) error {
	orderSeq, err := strconv.Atoi(orderNo)
	if err != nil {
		return fmt.Errorf("invalid order number: %w", err)
	}

	gateCode, err := s.repo.GetOrderGate(orderSeq)
	if err != nil {
		return fmt.Errorf("주문 게이트 조회 실패: %w", err)
	}
	gate := "immediately"
	if gateCode == "P" {
		gate = "profile"
	}

	// Idempotency check BEFORE calling ep_cli.Approve() to prevent duplicate charges
	orderPrice, paymentStatus, err := s.repo.GetOrderPrice(orderSeq)
	if err != nil {
		return fmt.Errorf("주문 조회 실패: %w", err)
	}
	if paymentStatus == "Y" {
		s.logger.Info().Str("orderNo", orderNo).Msg("order already paid (idempotent)")
		return nil
	}

	result, err := s.epSvc.Approve(model.ApproveRequest{
		OrderNo:     orderNo,
		EncryptData: encData,
		SessionKey:  sessionKey,
		TraceNo:     traceNo,
		ClientIP:    clientIP,
	}, gate)
	if err != nil {
		s.logger.Error().Err(err).Str("orderNo", orderNo).Msg("ep_cli approval failed")
		s.pgAudit.Log(orderNo, "approve_fail", nil, err)
		return err
	}
	if result.ResCode != "0000" {
		s.logger.Warn().Str("resCode", result.ResCode).Str("resMsg", result.ResMsg).Str("orderNo", orderNo).Msg("PG approval rejected")
		s.pgAudit.Log(orderNo, "approve_fail", result, nil)
		return fmt.Errorf("PG 승인 실패: %s (%s)", result.ResMsg, result.ResCode)
	}

	s.pgAudit.Log(orderNo, "approve_success", result, nil)

	approvedAmount, _ := strconv.Atoi(result.Amount)
	if orderPrice != approvedAmount {
		s.logger.Error().Int("orderPrice", orderPrice).Int("approvedAmount", approvedAmount).Str("orderNo", orderNo).Msg("amount mismatch")
		return fmt.Errorf("금액 불일치: 주문 %d, 승인 %d", orderPrice, approvedAmount)
	}

	pgData := &model.PGData{
		CNO:      result.CNO,
		ResCD:    result.ResCode,
		ResMsg:   result.ResMsg,
		Amount:   approvedAmount,
		NumCard:  result.CardNo,
		TranDate: result.TranDate,
		AuthNo:   result.AuthNo,
		PayType:  result.PayType,
		OSeq:     orderSeq,
	}

	// Wrap InsertPGData + UpdateOrderPayment in a single DB transaction
	tx, err := s.repo.DB.Beginx()
	if err != nil {
		s.logger.Error().Err(err).Str("orderNo", orderNo).Msg("failed to begin transaction")
		s.pgAudit.Log(orderNo, "db_tx_begin_fail", result, err)
		return fmt.Errorf("트랜잭션 시작 실패: %w", err)
	}
	defer tx.Rollback()

	pgSeq, err := s.repo.InsertPGDataTx(tx, pgData)
	if err != nil {
		s.logger.Error().Err(err).Str("orderNo", orderNo).Msg("failed to insert PG data")
		s.pgAudit.Log(orderNo, "db_insert_fail", result, err)
		return fmt.Errorf("PG 데이터 저장 실패: %w", err)
	}

	affected, err := s.repo.UpdateOrderPaymentTx(tx, orderSeq, approvedAmount, pgSeq, clientIP)
	if err != nil {
		s.logger.Error().Err(err).Str("orderNo", orderNo).Msg("failed to update order payment")
		s.pgAudit.Log(orderNo, "db_update_fail", result, err)
		return fmt.Errorf("주문 업데이트 실패: %w", err)
	}

	if err := tx.Commit(); err != nil {
		s.logger.Error().Err(err).Str("orderNo", orderNo).Msg("failed to commit transaction")
		s.pgAudit.Log(orderNo, "db_tx_commit_fail", result, err)
		return fmt.Errorf("트랜잭션 커밋 실패: %w", err)
	}

	if affected == 0 {
		s.logger.Info().Str("orderNo", orderNo).Msg("order already processed (idempotent)")
		return nil
	}

	s.cache.Delete("donation_summary")
	s.logger.Info().Str("orderNo", orderNo).Int("amount", approvedAmount).Msg("payment confirmed")
	return nil
}

func formatComma(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}
