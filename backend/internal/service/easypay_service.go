// easypay_service.go — EasyPay PG gateway approval via ep_cli binary execution
package service

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/model"
)

type EasyPayService struct {
	cfg config.EasyPayConfig
}

func NewEasyPayService(cfg config.EasyPayConfig) *EasyPayService {
	return &EasyPayService{cfg: cfg}
}

func (s *EasyPayService) Approve(req model.ApproveRequest, gate string) (*model.ApproveResult, error) {
	binPath := filepath.Join(s.cfg.BinBase, gate, "bin", "linux_64", "ep_cli")
	certFile := filepath.Join(s.cfg.BinBase, gate, "mobile", "cert", "pg_cert.pem")
	logDir := filepath.Join(s.cfg.BinBase, gate, "mobile", "log")

	mallID := s.cfg.ImmediatelyMallID
	if gate == "profile" {
		mallID = s.cfg.ProfileMallID
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := fmt.Sprintf(
		"order_no=%s,cert_file=%s,mall_id=%s,tr_cd=00101000,gw_url=%s,gw_port=%s,enc_data=%s,snd_key=%s,trace_no=%s,cust_ip=%s,log_dir=%s,log_level=1",
		req.OrderNo, certFile, mallID,
		s.cfg.GatewayURL, s.cfg.GatewayPort,
		req.EncryptData, req.SessionKey, req.TraceNo, req.ClientIP, logDir,
	)

	cmd := exec.CommandContext(ctx, binPath, "-h", args)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ep_cli execution failed: %w", err)
	}

	return parseEasyPayResponse(string(output)), nil
}

func parseEasyPayResponse(raw string) *model.ApproveResult {
	fields := strings.Split(raw, "\x1F")
	m := make(map[string]string, len(fields))
	for _, field := range fields {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) == 2 {
			m[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return &model.ApproveResult{
		ResCode:      m["res_cd"],
		ResMsg:       m["res_msg"],
		CNO:          m["cno"],
		Amount:       m["amount"],
		AuthNo:       m["auth_no"],
		TranDate:     m["tran_date"],
		CardNo:       m["card_no"],
		PayType:      m["pay_type"],
		IssuerName:   m["issuer_nm"],
		AcquirerName: m["acquirer_nm"],
	}
}
