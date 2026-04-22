// wire.go — Dependency injection: repository, service, and handler wiring
package main

import (
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/handler"
	"github.com/dflh-saf/backend/internal/job"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/presenter"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

// deps holds all wired dependencies needed by the application lifecycle.
type deps struct {
	authService       *service.AuthService
	handlers          handlers
	cacheStore        *cache.Cache
	donationRepo      *repository.DonationRepository
	donationJob       *job.DonationSnapshotJob
	sessionRepo       *repository.SessionRepository
	passwordResetRepo *repository.PasswordResetRepository
	pgAuditLog        *service.PGAuditLogger
	emailQueue        chan model.EmailMessage
	emailService      *service.EmailService
}

// wireDeps creates all repositories, services, and handlers from config and DB.
func wireDeps(db *sqlx.DB, cfg *config.Config, logger zerolog.Logger) (*deps, error) {
	authRepo := repository.NewAuthRepository(db)
	feedRepo := repository.NewFeedRepository(db)
	donationRepo := repository.NewDonationRepository(db)
	alumniRepo := repository.NewAlumniRepository(db)
	profileRepo := repository.NewProfileRepository(db)
	adRepo := repository.NewAdRepository(db)
	adLikeRepo := repository.NewAdLikeRepository(db)
	adCommentRepo := repository.NewAdCommentRepository(db)
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
	passwordResetRepo := repository.NewPasswordResetRepository(db)

	cacheStore := cache.New(5*time.Minute, 10*time.Minute)

	// Email infrastructure
	emailQueue := make(chan model.EmailMessage, 100)
	emailService := service.NewEmailService(cfg.SMTP, logger)

	authService := service.NewAuthService(authRepo, sessionRepo, cfg, cacheStore, logger)
	memberService := service.NewMemberService(authRepo)
	registrationService := service.NewRegistrationService(authRepo, profileRepo)
	feedService := service.NewFeedService(feedRepo, adRepo, cacheStore)
	ogService := service.NewOGService(feedRepo)
	sitemapService := service.NewSitemapService(feedRepo, cacheStore)
	rssService := service.NewRSSService(feedRepo, cacheStore)
	donationService := service.NewDonationService(donationRepo, cacheStore)
	alumniService := service.NewAlumniService(alumniRepo, cacheStore)
	profileService := service.NewProfileService(profileRepo)
	adService := service.NewAdService(adRepo)
	adLikeService := service.NewAdLikeService(adLikeRepo)
	adCommentService := service.NewAdCommentService(adCommentRepo)
	adminNoticeSvc := service.NewAdminNoticeService(adminNoticeRepo)
	adminAdSvc := service.NewAdminAdService(adminAdRepo)
	adminDonationSvc := service.NewAdminDonationService(adminDonationRepo, donationRepo)
	donationJob := job.NewDonationSnapshotJob(donationRepo, logger)
	adminDonationOrchestrator := service.NewDonationConfigOrchestrator(adminDonationSvc, donationService, donationJob)
	adminMemberSvc := service.NewAdminMemberService(adminMemberRepo)
	adminDashboardSvc := service.NewAdminDashboardService(adminMemberSvc, adminNoticeSvc, adminAdSvc, donationService)
	fileStorage := service.NewFileStorageService(cfg.Upload.BasePath)
	imageResizer := service.NewImageResizeService(1200)
	fileRecordSvc := service.NewFileRecordService(fileRepo)
	uploadOrchestrator := service.NewUploadOrchestrator(fileStorage, imageResizer, fileRecordSvc)
	profileUploadService := service.NewProfileUploadService(profileRepo, uploadOrchestrator)
	easypayService := service.NewEasyPayService(cfg.EasyPay)
	pgAuditLogger, err := service.NewPGAuditLogger(cfg.PGAuditLogPath)
	if err != nil {
		return nil, err
	}

	likeService := service.NewLikeService(likeRepo, feedRepo)
	commentService := service.NewCommentService(commentRepo)
	myDonationService := service.NewMyDonationService(myDonationRepo)
	messageService := service.NewMessageService(messageRepo, profileRepo)
	passwordResetService := service.NewPasswordResetService(passwordResetRepo, emailQueue, logger, cfg.Server.SiteBaseURL)
	passwordChangeSvc := service.NewPasswordChangeService(profileRepo)
	donateService := service.NewDonateService(donateRepo, easypayService, cacheStore, logger, pgAuditLogger)
	adminJobCatRepo := repository.NewAdminJobCategoryRepository(db)
	adminJobCatSvc := service.NewAdminJobCategoryService(adminJobCatRepo, cacheStore)

	subscriptionRepo := repository.NewSubscriptionRepository(db)
	subscriptionService := service.NewSubscriptionService(subscriptionRepo, donateService, cacheStore, logger)

	feedPresenter := presenter.NewFeedPresenter()

	h := handlers{
		health:         handler.NewHealthHandler(db),
		auth:           handler.NewAuthHandler(authService, memberService, registrationService, cacheStore, cfg, logger),
		feed:           handler.NewFeedHandler(feedService, likeService, feedPresenter),
		like:           handler.NewLikeHandler(likeService),
		comment:        handler.NewCommentHandler(commentService),
		donation:       handler.NewDonationHandler(donationService),
		alumni:         handler.NewAlumniHandler(alumniService),
		profile:        handler.NewProfileHandler(profileService),
		profileUpload:  handler.NewProfileUploadHandler(profileUploadService),
		ad:             handler.NewAdHandler(adService),
		adLike:         handler.NewAdLikeHandler(adLikeService),
		adComment:      handler.NewAdCommentHandler(adCommentService),
		adminNotice:    handler.NewAdminNoticeHandler(adminNoticeSvc, feedPresenter),
		adminAd:        handler.NewAdminAdHandler(adminAdSvc),
		adminDonation:  handler.NewAdminDonationHandler(adminDonationOrchestrator),
		adminMember:    handler.NewAdminMemberHandler(adminMemberSvc),
		adminDashboard: handler.NewAdminDashboardHandler(adminDashboardSvc),
		adminUpload:    handler.NewAdminUploadHandler(uploadOrchestrator, cfg.Upload.MaxFileSizeMB),
		myDonation:     handler.NewMyDonationHandler(myDonationService),
		message:        handler.NewMessageHandler(messageService),
		payment:        handler.NewPaymentHandler(donateService, cfg.EasyPay),
		subscription:   handler.NewSubscriptionHandler(subscriptionService, cfg.EasyPay),
		og:             handler.NewOGHandler(ogService, cfg.Server.SiteBaseURL),
		sitemap:        handler.NewSitemapHandler(sitemapService, cfg.Server.SiteBaseURL),
		rss:            handler.NewRSSHandler(rssService, cfg.Server.SiteBaseURL),
		passwordReset:  handler.NewPasswordResetHandler(passwordResetService, logger),
		passwordChange: handler.NewPasswordChangeHandler(passwordChangeSvc),
		badge:          handler.NewBadgeHandler(messageService, logger),
		adminJobCat:    handler.NewAdminJobCategoryHandler(adminJobCatSvc),
	}

	return &deps{
		authService:       authService,
		handlers:          h,
		cacheStore:        cacheStore,
		donationRepo:      donationRepo,
		donationJob:       donationJob,
		sessionRepo:       sessionRepo,
		passwordResetRepo: passwordResetRepo,
		pgAuditLog:        pgAuditLogger,
		emailQueue:        emailQueue,
		emailService:      emailService,
	}, nil
}
