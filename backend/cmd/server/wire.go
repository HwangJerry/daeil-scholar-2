// wire.go — Dependency injection: repository, service, and handler wiring
package main

import (
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/handler"
	"github.com/dflh-saf/backend/internal/presenter"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

// deps holds all wired dependencies needed by the application lifecycle.
type deps struct {
	authService  *service.AuthService
	handlers     handlers
	donationRepo *repository.DonationRepository
	sessionRepo  *repository.SessionRepository
	pgAuditLog   *service.PGAuditLogger
}

// wireDeps creates all repositories, services, and handlers from config and DB.
func wireDeps(db *sqlx.DB, cfg *config.Config, logger zerolog.Logger) (*deps, error) {
	authRepo := repository.NewAuthRepository(db)
	feedRepo := repository.NewFeedRepository(db)
	donationRepo := repository.NewDonationRepository(db)
	alumniRepo := repository.NewAlumniRepository(db)
	profileRepo := repository.NewProfileRepository(db)
	adRepo := repository.NewAdRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	fileRepo := repository.NewFileRepository(db)
	adminNoticeRepo := repository.NewAdminNoticeRepository(db)
	adminAdRepo := repository.NewAdminAdRepository(db)
	adminDonationRepo := repository.NewAdminDonationRepository(db)
	adminMemberRepo := repository.NewAdminMemberRepository(db)
	donateRepo := repository.NewDonateRepository(db)
	myDonationRepo := repository.NewMyDonationRepository(db)
	likeRepo := repository.NewLikeRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	messageRepo := repository.NewMessageRepository(db)

	cacheStore := cache.New(5*time.Minute, 10*time.Minute)

	authService := service.NewAuthService(authRepo, sessionRepo, cfg, cacheStore, logger)
	memberService := service.NewMemberService(authRepo)
	feedService := service.NewFeedService(feedRepo, adRepo, cacheStore)
	donationService := service.NewDonationService(donationRepo, cacheStore)
	alumniService := service.NewAlumniService(alumniRepo, cacheStore)
	profileService := service.NewProfileService(profileRepo)
	adService := service.NewAdService(adRepo)
	adminNoticeSvc := service.NewAdminNoticeService(adminNoticeRepo)
	adminAdSvc := service.NewAdminAdService(adminAdRepo)
	adminDonationSvc := service.NewAdminDonationService(adminDonationRepo, donationRepo)
	adminMemberSvc := service.NewAdminMemberService(adminMemberRepo)
	adminDashboardSvc := service.NewAdminDashboardService(adminMemberSvc, adminNoticeSvc, adminAdSvc, donationService)
	fileStorage := service.NewFileStorageService(cfg.Upload.BasePath)
	imageResizer := service.NewImageResizeService(1200)
	fileRecordSvc := service.NewFileRecordService(fileRepo)
	uploadOrchestrator := service.NewUploadOrchestrator(fileStorage, imageResizer, fileRecordSvc)
	easypayService := service.NewEasyPayService(cfg.EasyPay)
	pgAuditLogger, err := service.NewPGAuditLogger(cfg.PGAuditLogPath)
	if err != nil {
		return nil, err
	}
	likeService := service.NewLikeService(likeRepo, feedRepo)
	commentService := service.NewCommentService(commentRepo)
	myDonationService := service.NewMyDonationService(myDonationRepo)
	messageService := service.NewMessageService(messageRepo, profileRepo)
	donateService := service.NewDonateService(donateRepo, easypayService, cacheStore, logger, pgAuditLogger)
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	subscriptionService := service.NewSubscriptionService(subscriptionRepo, donateService, cacheStore, logger)

	feedPresenter := presenter.NewFeedPresenter()

	h := handlers{
		health:         handler.NewHealthHandler(db),
		auth:           handler.NewAuthHandler(authService, memberService, cacheStore, cfg, logger),
		feed:           handler.NewFeedHandler(feedService, likeService, feedPresenter),
		like:           handler.NewLikeHandler(likeService),
		comment:        handler.NewCommentHandler(commentService),
		donation:       handler.NewDonationHandler(donationService),
		alumni:         handler.NewAlumniHandler(alumniService),
		profile:        handler.NewProfileHandler(profileService),
		ad:             handler.NewAdHandler(adService),
		adminNotice:    handler.NewAdminNoticeHandler(adminNoticeSvc, feedPresenter),
		adminAd:        handler.NewAdminAdHandler(adminAdSvc),
		adminDonation:  handler.NewAdminDonationHandler(adminDonationSvc),
		adminMember:    handler.NewAdminMemberHandler(adminMemberSvc),
		adminDashboard: handler.NewAdminDashboardHandler(adminDashboardSvc),
		adminUpload:    handler.NewAdminUploadHandler(uploadOrchestrator, cfg.Upload.MaxFileSizeMB),
		myDonation:     handler.NewMyDonationHandler(myDonationService),
		message:        handler.NewMessageHandler(messageService),
		payment:        handler.NewPaymentHandler(donateService, cfg.EasyPay),
		subscription:   handler.NewSubscriptionHandler(subscriptionService, cfg.EasyPay),
	}

	return &deps{
		authService:  authService,
		handlers:     h,
		donationRepo: donationRepo,
		sessionRepo:  sessionRepo,
		pgAuditLog:   pgAuditLogger,
	}, nil
}
