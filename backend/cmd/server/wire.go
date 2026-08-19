// wire.go — Dependency injection: repository, service, and handler wiring
package main

import (
	"context"
	"time"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/handler"
	"github.com/dflh-saf/backend/internal/job"
	"github.com/dflh-saf/backend/internal/model"
	"github.com/dflh-saf/backend/internal/observability"
	"github.com/dflh-saf/backend/internal/presenter"
	"github.com/dflh-saf/backend/internal/push"
	"github.com/dflh-saf/backend/internal/realtime"
	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

// deps holds all wired dependencies needed by the application lifecycle.
type deps struct {
	authService            *service.AuthService
	handlers               handlers
	cacheStore             *cache.Cache
	donationRepo           *repository.DonationRepository
	donationJob            *job.DonationSnapshotJob
	sessionRepo            *repository.SessionRepository
	passwordResetRepo      *repository.PasswordResetRepository
	authRepo               *repository.AuthRepository
	pgAuditLog             *service.PGAuditLogger
	emailQueue             chan model.EmailMessage
	emailService           *service.EmailService
	subscriptionBillingJob *job.SubscriptionBillingJob
	visitJob               *job.VisitAggregationJob
	blockedMessageCleanup  *job.BlockedMessageCleanupJob
	pushDelivery           *service.PushDeliveryNotifier
	socialRevocationWorker *job.SocialRevocationWorker
}

// wireDeps creates all repositories, services, and handlers from config and DB.
func wireDeps(db *sqlx.DB, cfg *config.Config, logger zerolog.Logger, debugHook *observability.Hook) (*deps, error) {
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
	donationImportRepo := repository.NewDonationImportRepository(db)
	adminMemberRepo := repository.NewAdminMemberRepository(db)
	donateRepo := repository.NewDonateRepository(db)
	personalDonationRepo := repository.NewPersonalDonationRepository(db)
	likeRepo := repository.NewLikeRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	memberBlockRepo := repository.NewMemberBlockRepository(db)
	pushRepo := repository.NewPushRepository(db)
	passwordResetRepo := repository.NewPasswordResetRepository(db)
	identityRepo := repository.NewIdentityRepository(db)
	credentialRepo := repository.NewCredentialRepository(db)
	signupRepo := repository.NewSignupRepository(db)
	passwordMutationRepo := repository.NewPasswordMutationRepository(db)
	visitRepo := repository.NewVisitRepository(db)
	canonicalPasswordReady, err := repository.CanonicalPasswordWriteReady(db)
	if err != nil {
		return nil, err
	}
	phoneClaimsReady, err := repository.PhoneClaimsWriteReady(db)
	if err != nil {
		return nil, err
	}
	if phoneClaimsReady {
		authRepo.EnablePhoneClaims()
		profileRepo.EnablePhoneClaims()
	} else {
		authRepo.EnablePhoneClaimAutoDetection()
		profileRepo.EnablePhoneClaimAutoDetection()
	}

	cacheStore := cache.New(5*time.Minute, 10*time.Minute)
	socialLinkTokens := service.NewSocialLinkTokenStore(cacheStore)

	realtimeHub := realtime.NewHub(logger)
	var messageNotifier service.MessageNotifier = service.NewRealtimeMessageNotifier(realtimeHub)
	var pushDelivery *service.PushDeliveryNotifier
	if cfg.Push.Enabled {
		pushSender, err := push.NewSender(context.Background(), cfg.Push, nil)
		if err != nil {
			return nil, err
		}
		pushDelivery = service.NewPushDeliveryNotifier(pushRepo, pushSender, logger)
		messageNotifier = service.NewCompositeMessageNotifier(messageNotifier, pushDelivery)
	}

	// Email infrastructure
	emailQueue := make(chan model.EmailMessage, 100)
	emailService := service.NewEmailService(cfg.SMTP, logger)

	authService := service.NewAuthService(authRepo, sessionRepo, cfg, cacheStore, logger)
	socialRevocationAppleVerifier := service.NewAppleIdentityVerifier(authService, cfg.Apple)
	socialRevocationVault, socialRevocationVaultErr := service.NewSocialCredentialVault(cfg.Apple.CredentialEncryptionKey)
	// TODO: make the poll interval configurable; 1 minute is a reasonable default for now.
	socialRevocationWorker := job.NewSocialRevocationWorker(
		authRepo, authService, socialRevocationAppleVerifier, socialRevocationVault, socialRevocationVaultErr,
		time.Minute, logger,
	)
	memberService := service.NewMemberService(authRepo)
	registrationService := service.NewRegistrationService(authRepo, profileRepo)
	if canonicalPasswordReady {
		canonicalPasswordService := service.NewCanonicalPasswordService(identityRepo, credentialRepo)
		memberService = service.NewMemberServiceWithPasswordCredentials(authRepo, canonicalPasswordService)
		registrationService = service.NewTransactionalRegistrationService(authRepo, profileRepo, signupRepo)
	}
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
	adminNoticeSvc := service.NewAdminNoticeService(adminNoticeRepo, fileRepo)
	adminDisclosureSvc := service.NewAdminDisclosureService(adminDisclosureRepo, fileRepo)
	disclosureSvc := service.NewDisclosureService(disclosureRepo)
	adminAdSvc := service.NewAdminAdService(adminAdRepo)
	adminDonationSvc := service.NewAdminDonationService(adminDonationRepo, donationRepo)
	donationImportSvc := service.NewDonationImportService(donationImportRepo, adminDonationSvc)
	donationJob := job.NewDonationSnapshotJob(donationRepo, logger)
	adminDonationOrchestrator := service.NewDonationConfigOrchestrator(adminDonationSvc, donationService, donationJob)
	adminMemberSvc := service.NewAdminMemberService(adminMemberRepo)
	visitService := service.NewVisitService(visitRepo, cacheStore, cfg.VisitIPSalt, logger)
	visitJob := job.NewVisitAggregationJob(visitRepo, logger)
	adminDashboardSvc := service.NewAdminDashboardService(adminMemberSvc, adminNoticeSvc, adminAdSvc, donationService, visitService)
	fileStorage := service.NewFileStorageService(cfg.Upload.BasePath)
	imageResizer := service.NewImageResizeService(1200)
	fileRecordSvc := service.NewFileRecordService(fileRepo)
	uploadOrchestrator := service.NewUploadOrchestrator(fileStorage, imageResizer, fileRecordSvc)
	attachmentStorage := service.NewAttachmentStorageService(cfg.Upload.BasePath)
	attachmentUploadOrchestrator := service.NewAttachmentUploadOrchestrator(attachmentStorage, fileRecordSvc)
	profileUploadService := service.NewProfileUploadService(profileRepo, uploadOrchestrator)
	easypayService := service.NewEasyPayService(cfg.EasyPay)
	pgAuditLogger, err := service.NewPGAuditLogger(cfg.PGAuditLogPath)
	if err != nil {
		return nil, err
	}

	likeService := service.NewLikeService(likeRepo, feedRepo)
	commentService := service.NewCommentService(commentRepo)
	personalDonationService := service.NewPersonalDonationService(personalDonationRepo)
	messageService := service.NewMessageService(messageRepo, profileRepo, messageNotifier)
	memberBlockService := service.NewMemberBlockService(memberBlockRepo)
	pushService := service.NewPushService(pushRepo)
	blockedMessageCleanup := job.NewBlockedMessageCleanupJob(memberBlockRepo, logger)
	passwordResetService := service.NewPasswordResetService(passwordResetRepo, emailQueue, logger, cfg.Server.SiteBaseURL)
	passwordChangeSvc := service.NewPasswordChangeService(profileRepo)
	if canonicalPasswordReady {
		passwordResetService = service.NewAtomicPasswordResetService(passwordResetRepo, emailQueue, logger, cfg.Server.SiteBaseURL)
		passwordChangeSvc = service.NewAtomicPasswordChangeService(passwordMutationRepo)
	}
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	subscriptionActivator := service.NewSubscriptionActivator(subscriptionRepo)
	donateService := service.NewDonateService(donateRepo, subscriptionActivator, easypayService, cacheStore, logger, pgAuditLogger)
	adminJobCatRepo := repository.NewAdminJobCategoryRepository(db)
	adminJobCatSvc := service.NewAdminJobCategoryService(adminJobCatRepo, cacheStore)
	historyRepo := repository.NewHistoryRepository(db)
	historySvc := service.NewHistoryService(historyRepo)

	subscriptionService := service.NewSubscriptionService(subscriptionRepo, donateService, easypayService, pgAuditLogger, cacheStore, logger)
	subscriptionBillingJob := job.NewSubscriptionBillingJob(subscriptionRepo, donateRepo, easypayService, pgAuditLogger, cacheStore, cfg.EasyPay, logger)

	feedPresenter := presenter.NewFeedPresenter()

	h := handlers{
		health:              handler.NewHealthHandler(db),
		auth:                handler.NewAuthHandler(authService, memberService, registrationService, profileService, cacheStore, socialLinkTokens, cfg, logger),
		feed:                handler.NewFeedHandler(feedService, likeService, feedPresenter),
		like:                handler.NewLikeHandler(likeService),
		comment:             handler.NewCommentHandler(commentService),
		donation:            handler.NewDonationHandler(donationService),
		alumni:              handler.NewAlumniHandler(alumniService),
		profile:             handler.NewProfileHandler(profileService),
		profileUpload:       handler.NewProfileUploadHandler(profileUploadService),
		ad:                  handler.NewAdHandler(adService),
		adLike:              handler.NewAdLikeHandler(adLikeService),
		adComment:           handler.NewAdCommentHandler(adCommentService),
		adminNotice:         handler.NewAdminNoticeHandler(adminNoticeSvc, feedPresenter),
		adminDisclosure:     handler.NewAdminDisclosureHandler(adminDisclosureSvc, feedPresenter),
		disclosure:          handler.NewDisclosureHandler(disclosureSvc, feedPresenter),
		adminAd:             handler.NewAdminAdHandler(adminAdSvc),
		adminDonation:       handler.NewAdminDonationHandler(adminDonationOrchestrator),
		adminDonationImport: handler.NewAdminDonationImportHandler(donationImportSvc, cfg.Upload.MaxFileSizeMB),
		adminMember:         handler.NewAdminMemberHandler(adminMemberSvc),
		adminDashboard:      handler.NewAdminDashboardHandler(adminDashboardSvc),
		adminUpload:         handler.NewAdminUploadHandler(uploadOrchestrator, cfg.Upload.MaxFileSizeMB),
		adminAttachUpload:   handler.NewAdminAttachmentUploadHandler(attachmentUploadOrchestrator, cfg.Upload.MaxFileSizeMB),
		socialLinkPhoto:     handler.NewSocialLinkPhotoHandler(uploadOrchestrator, socialLinkTokens, logger),
		personalDonation:    handler.NewPersonalDonationHandler(personalDonationService),
		message:             handler.NewMessageHandler(messageService),
		memberBlock:         handler.NewMemberBlockHandler(memberBlockService),
		push:                handler.NewPushHandler(pushService),
		payment:             handler.NewPaymentHandler(donateService, cfg.EasyPay),
		subscription:        handler.NewSubscriptionHandler(subscriptionService, cfg.EasyPay),
		og:                  handler.NewOGHandler(ogService, cfg.Server.SiteBaseURL),
		sitemap:             handler.NewSitemapHandler(sitemapService, cfg.Server.SiteBaseURL),
		rss:                 handler.NewRSSHandler(rssService, cfg.Server.SiteBaseURL),
		passwordReset:       handler.NewPasswordResetHandler(passwordResetService, logger),
		passwordChange:      handler.NewPasswordChangeHandler(passwordChangeSvc),
		badge:               handler.NewBadgeHandler(messageService, logger),
		adminJobCat:         handler.NewAdminJobCategoryHandler(adminJobCatSvc),
		history:             handler.NewHistoryHandler(historySvc),
		adminSubscription:   handler.NewAdminSubscriptionHandler(subscriptionBillingJob, logger),
		realtime:            handler.NewRealtimeHandler(realtimeHub, logger),
		visit:               handler.NewVisitHandler(visitService, logger, cfg.Server.IsSecure()),
		adminErrorReport:    handler.NewAdminErrorReportHandler(logger, debugHook),
	}

	return &deps{
		authService:            authService,
		handlers:               h,
		cacheStore:             cacheStore,
		donationRepo:           donationRepo,
		donationJob:            donationJob,
		sessionRepo:            sessionRepo,
		passwordResetRepo:      passwordResetRepo,
		authRepo:               authRepo,
		pgAuditLog:             pgAuditLogger,
		emailQueue:             emailQueue,
		emailService:           emailService,
		subscriptionBillingJob: subscriptionBillingJob,
		visitJob:               visitJob,
		blockedMessageCleanup:  blockedMessageCleanup,
		pushDelivery:           pushDelivery,
		socialRevocationWorker: socialRevocationWorker,
	}, nil
}
