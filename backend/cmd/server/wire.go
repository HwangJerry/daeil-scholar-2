// wire.go — Dependency injection: repository, service, and handler wiring
package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/handler"
	"github.com/dflh-saf/backend/internal/job"
	"github.com/dflh-saf/backend/internal/maintenance"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/observability"
	"github.com/dflh-saf/backend/internal/presenter"
	"github.com/dflh-saf/backend/internal/realtime"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/dflh-saf/social-auth/kakao"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

// deps holds all wired dependencies needed by the application lifecycle.
type deps struct {
	authService            *service.AuthService
	authRepo               *repository.AuthRepository
	handlers               handlers
	cacheStore             *cache.Cache
	donationRepo           *repository.DonationRepository
	donationJob            *job.DonationSnapshotJob
	sessionRepo            *repository.SessionRepository
	passwordResetRepo      *repository.PasswordResetRepository
	pgAuditLog             *service.PGAuditLogger
	emailQueue             chan model.EmailMessage
	emailService           *service.EmailService
	subscriptionBillingJob *job.SubscriptionBillingJob
	visitJob               *job.VisitAggregationJob
	pushOutboxWorker       *job.PushOutboxWorker
	maintenanceGate        *maintenance.Gate
}

// wireDeps creates all repositories, services, and handlers from config and DB.
func wireDeps(db *sqlx.DB, cfg *config.Config, logger zerolog.Logger, debugHook *observability.Hook) (*deps, error) {
	maintenanceGate, err := maintenance.NewRuntimeGate(cfg.Environment, cfg.Maintenance.SentinelPath, cfg.Maintenance.SmokeProofSHA256, cfg.Maintenance.SmokeAllowedPaths...)
	if err != nil {
		return nil, err
	}
	environment := strings.ToLower(strings.TrimSpace(cfg.Environment))
	if (environment == "prod" || environment == "production") &&
		(cfg.Maintenance.ReleaseBridgePath != maintenance.ProductionReleaseBridgePath || cfg.Maintenance.ReleaseOwnerUID != 0) {
		return nil, fmt.Errorf("production maintenance release bridge authority is invalid")
	}
	if err := maintenanceGate.ConfigureRelease(maintenance.ReleaseConfig{
		BridgePath:       cfg.Maintenance.ReleaseBridgePath,
		ProofSHA256:      cfg.Maintenance.ReleaseProofSHA256,
		ExpectedOwnerUID: cfg.Maintenance.ReleaseOwnerUID,
	}); err != nil {
		return nil, err
	}
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
	adminDisclosureRepo := repository.NewAdminDisclosureRepository(db)
	disclosureRepo := repository.NewDisclosureRepository(db)
	adminAdRepo := repository.NewAdminAdRepository(db)
	adminDonationRepo := repository.NewAdminDonationRepository(db)
	adminMemberRepo := repository.NewAdminMemberRepository(db)
	donateRepo := repository.NewDonateRepository(db)
	myDonationRepo := repository.NewMyDonationRepository(db)
	likeRepo := repository.NewLikeRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	passwordResetRepo := repository.NewPasswordResetRepository(db)
	visitRepo := repository.NewVisitRepository(db)
	pushOutboxRepo := repository.NewPushOutboxRepository(db)

	cacheStore := cache.New(5*time.Minute, 10*time.Minute)
	kakaoClient := kakao.NewClient(kakao.Config{
		ClientID:     cfg.Kakao.ClientID,
		ClientSecret: cfg.Kakao.ClientSecret,
		RedirectURI:  cfg.Kakao.RedirectURI,
	})

	realtimeHub := realtime.NewHub(logger)
	messageNotifier := service.NewRealtimeMessageNotifier(realtimeHub)
	pushTokenRepo := repository.NewMobileDeviceTokenRepository(db)
	pushPreferenceRepo := repository.NewPushPreferenceRepository(db)
	pushProviders := map[string]service.MobilePushProvider{}
	var apnsProvider service.MobilePushProvider
	if cfg.Push.APNsKeyID != "" && cfg.Push.APNSTeamID != "" && cfg.Push.APNsBundleID != "" &&
		(cfg.Push.APNsKeyPath != "" || cfg.Push.APNsKeyValue != "") {
		apnsProvider = service.NewAPNsPushProvider(cfg.Push, logger)
		pushProviders["ios"] = apnsProvider
	}
	if cfg.Push.FCMCredentialsJSON != "" || cfg.Push.FCMCredentialsFile != "" {
		fcmProvider, err := service.NewFCMPushProvider(context.Background(), cfg.Push, logger)
		if err != nil {
			return nil, err
		}
		pushProviders["android"] = fcmProvider
	}
	pushProvider := service.NewPlatformPushProvider(logger, pushProviders)
	pushService := service.NewMobilePushServiceWithOutboxAndPreferences(pushTokenRepo, pushPreferenceRepo, pushOutboxRepo, pushProvider, logger)
	pushOutboxWorker := job.NewPushOutboxWorker(pushOutboxRepo, pushTokenRepo, pushPreferenceRepo, apnsProvider, maintenanceGate, job.PushOutboxWorkerConfig{
		BatchSize:       cfg.Push.OutboxBatchSize,
		PollInterval:    cfg.Push.OutboxPollInterval,
		MaxAttempts:     cfg.Push.OutboxMaxAttempts,
		BaseBackoff:     cfg.Push.OutboxBaseBackoff,
		MaxBackoff:      cfg.Push.OutboxMaxBackoff,
		RecoveryTimeout: cfg.Push.OutboxRecoveryTimeout,
		RequestTimeout:  cfg.Push.OutboxRequestTimeout,
	}, logger)

	// Email infrastructure
	emailQueue := make(chan model.EmailMessage, 100)
	emailService := service.NewEmailService(cfg.SMTP, logger)

	authService := service.NewAuthService(authRepo, sessionRepo, cfg, cacheStore, kakaoClient, logger)
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
	adminNoticeSvc := service.NewAdminNoticeService(adminNoticeRepo, fileRepo, pushService)
	adminDisclosureSvc := service.NewAdminDisclosureService(adminDisclosureRepo, fileRepo)
	disclosureSvc := service.NewDisclosureService(disclosureRepo)
	adminAdSvc := service.NewAdminAdService(adminAdRepo)
	adminDonationSvc := service.NewAdminDonationService(adminDonationRepo, donationRepo)
	donationJob := job.NewDonationSnapshotJob(donationRepo, maintenanceGate, logger)
	adminDonationOrchestrator := service.NewDonationConfigOrchestrator(adminDonationSvc, donationService, donationJob)
	adminMemberSvc := service.NewAdminMemberService(adminMemberRepo)
	visitService := service.NewVisitService(visitRepo, cacheStore, cfg.VisitIPSalt, logger)
	visitJob := job.NewVisitAggregationJob(visitRepo, maintenanceGate, logger)
	adminDashboardSvc := service.NewAdminDashboardService(adminMemberSvc, adminNoticeSvc, adminAdSvc, donationService, visitService)
	fileStorage := service.NewFileStorageService(cfg.Upload.BasePath)
	imageResizer := service.NewImageResizeService(1200)
	fileRecordSvc := service.NewFileRecordService(fileRepo)
	uploadOrchestrator := service.NewUploadOrchestrator(fileStorage, imageResizer, fileRecordSvc)
	attachmentStorage := service.NewAttachmentStorageService(cfg.Upload.BasePath)
	attachmentUploadOrchestrator := service.NewAttachmentUploadOrchestrator(attachmentStorage, fileRecordSvc)
	profileUploadService := service.NewProfileUploadService(profileRepo, uploadOrchestrator)
	easypayService := service.NewEasyPayService(cfg.EasyPay)
	pgAuditLogger, err := service.NewPGAuditLogger(cfg.PGAuditLogPath, maintenanceGate.Active())
	if err != nil {
		return nil, err
	}

	likeService := service.NewLikeService(likeRepo, feedRepo)
	commentService := service.NewCommentService(commentRepo)
	myDonationService := service.NewMyDonationService(myDonationRepo)
	messageService := service.NewMessageService(messageRepo, profileRepo, messageNotifier, pushService)
	passwordResetService := service.NewPasswordResetService(passwordResetRepo, emailQueue, logger, cfg.Server.SiteBaseURL)
	passwordChangeSvc := service.NewPasswordChangeService(profileRepo)
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	subscriptionActivator := service.NewSubscriptionActivator(subscriptionRepo)
	donateService := service.NewDonateService(donateRepo, subscriptionActivator, easypayService, cacheStore, logger, pgAuditLogger)
	adminJobCatRepo := repository.NewAdminJobCategoryRepository(db)
	adminJobCatSvc := service.NewAdminJobCategoryService(adminJobCatRepo, cacheStore)
	historyRepo := repository.NewHistoryRepository(db)
	historySvc := service.NewHistoryService(historyRepo)

	subscriptionService := service.NewSubscriptionService(subscriptionRepo, donateService, easypayService, pgAuditLogger, cacheStore, logger)
	subscriptionBillingJob := job.NewSubscriptionBillingJob(subscriptionRepo, donateRepo, easypayService, pgAuditLogger, cacheStore, cfg.EasyPay, maintenanceGate, logger)

	feedPresenter := presenter.NewFeedPresenter()

	h := handlers{
		health:             handler.NewHealthHandler(db),
		auth:               handler.NewAuthHandler(authService, memberService, registrationService, profileService, cacheStore, cfg, logger),
		feed:               handler.NewFeedHandler(feedService, likeService, feedPresenter),
		like:               handler.NewLikeHandler(likeService),
		comment:            handler.NewCommentHandler(commentService),
		donation:           handler.NewDonationHandler(donationService),
		alumni:             handler.NewAlumniHandler(alumniService),
		profile:            handler.NewProfileHandler(profileService),
		profileUpload:      handler.NewProfileUploadHandler(profileUploadService),
		ad:                 handler.NewAdHandler(adService),
		adLike:             handler.NewAdLikeHandler(adLikeService),
		adComment:          handler.NewAdCommentHandler(adCommentService),
		adminNotice:        handler.NewAdminNoticeHandler(adminNoticeSvc, feedPresenter),
		adminDisclosure:    handler.NewAdminDisclosureHandler(adminDisclosureSvc, feedPresenter),
		disclosure:         handler.NewDisclosureHandler(disclosureSvc, feedPresenter),
		adminAd:            handler.NewAdminAdHandler(adminAdSvc),
		adminDonation:      handler.NewAdminDonationHandler(adminDonationOrchestrator),
		adminMember:        handler.NewAdminMemberHandler(adminMemberSvc),
		adminDashboard:     handler.NewAdminDashboardHandler(adminDashboardSvc),
		adminUpload:        handler.NewAdminUploadHandler(uploadOrchestrator, cfg.Upload.MaxFileSizeMB),
		adminAttachUpload:  handler.NewAdminAttachmentUploadHandler(attachmentUploadOrchestrator, cfg.Upload.MaxFileSizeMB),
		socialLinkPhoto:    handler.NewSocialLinkPhotoHandler(uploadOrchestrator, cacheStore, logger),
		myDonation:         handler.NewMyDonationHandler(myDonationService),
		message:            handler.NewMessageHandler(messageService),
		payment:            handler.NewPaymentHandler(donateService, cfg.EasyPay),
		subscription:       handler.NewSubscriptionHandler(subscriptionService, cfg.EasyPay),
		og:                 handler.NewOGHandler(ogService, cfg.Server.SiteBaseURL),
		sitemap:            handler.NewSitemapHandler(sitemapService, cfg.Server.SiteBaseURL),
		rss:                handler.NewRSSHandler(rssService, cfg.Server.SiteBaseURL),
		passwordReset:      handler.NewPasswordResetHandler(passwordResetService, logger),
		passwordChange:     handler.NewPasswordChangeHandler(passwordChangeSvc),
		badge:              handler.NewBadgeHandler(messageService, logger),
		adminJobCat:        handler.NewAdminJobCategoryHandler(adminJobCatSvc),
		history:            handler.NewHistoryHandler(historySvc),
		adminSubscription:  handler.NewAdminSubscriptionHandler(subscriptionBillingJob, logger),
		realtime:           handler.NewRealtimeHandler(realtimeHub, logger),
		visit:              handler.NewVisitHandler(visitService, logger, cfg.Server.IsSecure()),
		adminErrorReport:   handler.NewAdminErrorReportHandler(logger, debugHook),
		push:               handler.NewPushHandler(pushService),
		maintenanceRelease: handler.NewMaintenanceReleaseHandler(maintenanceGate, cfg.Maintenance.ReleaseDrainTimeout),
	}

	return &deps{
		authService:            authService,
		authRepo:               authRepo,
		handlers:               h,
		cacheStore:             cacheStore,
		donationRepo:           donationRepo,
		donationJob:            donationJob,
		sessionRepo:            sessionRepo,
		passwordResetRepo:      passwordResetRepo,
		pgAuditLog:             pgAuditLogger,
		emailQueue:             emailQueue,
		emailService:           emailService,
		subscriptionBillingJob: subscriptionBillingJob,
		visitJob:               visitJob,
		pushOutboxWorker:       pushOutboxWorker,
		maintenanceGate:        maintenanceGate,
	}, nil
}
