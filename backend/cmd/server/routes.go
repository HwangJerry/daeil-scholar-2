// routes.go — HTTP route registration for the API server
package main

import (
	"net/http"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/handler"
	mw "github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/observability"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

const defaultMaxBodySizeBytes int64 = 2 << 20

// handlers holds all HTTP handler instances for route registration.
type handlers struct {
	health              *handler.HealthHandler
	auth                *handler.AuthHandler
	feed                *handler.FeedHandler
	like                *handler.LikeHandler
	comment             *handler.CommentHandler
	donation            *handler.DonationHandler
	alumni              *handler.AlumniHandler
	profile             *handler.ProfileHandler
	profileUpload       *handler.ProfileUploadHandler
	bannerAd            *handler.BannerAdHandler
	adminNotice         *handler.AdminNoticeHandler
	adminDisclosure     *handler.AdminDisclosureHandler
	disclosure          *handler.DisclosureHandler
	adminBannerAd       *handler.AdminBannerAdHandler
	adminDonation       *handler.AdminDonationHandler
	adminDonationImport *handler.AdminDonationImportHandler
	adminMember         *handler.AdminMemberHandler
	adminDashboard      *handler.AdminDashboardHandler
	adminUpload         *handler.AdminUploadHandler
	adminAttachUpload   *handler.AdminAttachmentUploadHandler
	socialLinkPhoto     *handler.SocialLinkPhotoHandler
	personalDonation    *handler.PersonalDonationHandler
	message             *handler.MessageHandler
	memberBlock         *handler.MemberBlockHandler
	push                *handler.PushHandler
	payment             *handler.PaymentHandler
	subscription        *handler.SubscriptionHandler
	og                  *handler.OGHandler
	sitemap             *handler.SitemapHandler
	rss                 *handler.RSSHandler
	passwordReset       *handler.PasswordResetHandler
	passwordChange      *handler.PasswordChangeHandler
	badge               *handler.BadgeHandler
	adminJobCat         *handler.AdminJobCategoryHandler
	history             *handler.HistoryHandler
	realtime            *handler.RealtimeHandler
	adminSubscription   *handler.AdminSubscriptionHandler
	visit               *handler.VisitHandler
	adminErrorReport    *handler.AdminErrorReportHandler
}

// registerRoutes creates a chi.Router with all middleware and API routes.
func registerRoutes(h handlers, authService *service.AuthService, cacheStore *cache.Cache, allowedOrigins []string, cfg *config.Config, logger zerolog.Logger, debugHook *observability.Hook) chi.Router {
	router := chi.NewRouter()
	router.Use(mw.Recoverer(logger, debugHook))
	router.Use(mw.RequestLogger(logger))
	router.Use(mw.CORSMiddleware(allowedOrigins))

	// Static file servers (dev: proxied from Vite/Nginx; prod: handled by Nginx alias)
	router.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.Upload.BasePath))))
	router.Handle("/files/*", http.StripPrefix("/files/", http.FileServer(http.Dir(cfg.Upload.LegacyPath))))

	// Bot-facing OG and sitemap endpoints (no CSRF, read-only)
	router.Get("/og/post/{seq}", h.og.GetPostOG)
	router.Get("/sitemap.xml", h.sitemap.GetSitemap)
	router.Get("/rss.xml", h.rss.GetRSS)

	// DISABLED 2026-04-28: external donation redirect (dangled — see /home/jerryhwang/.claude/plans/drifting-gliding-hopcroft.md).
	// registerPGRoutes(router, h)
	registerAPIRoutes(router, h, authService, cacheStore, allowedOrigins, cfg)

	return router
}

// registerPGRoutes registers PG callback routes (no CSRF, no auth).
func registerPGRoutes(router chi.Router, h handlers) {
	router.Route("/pg", func(r chi.Router) {
		r.Post("/easypay/relay", h.payment.EasyPayRelay)
		r.Get("/easypay/relay", h.payment.EasyPayRelay)
		r.Post("/easypay/return", h.payment.EasyPayReturn)
	})
}

// registerAPIRoutes registers all /api/* routes with CSRF protection.
func registerAPIRoutes(router chi.Router, h handlers, authService *service.AuthService, cacheStore *cache.Cache, allowedOrigins []string, cfg *config.Config) {
	router.Group(func(r chi.Router) {
		r.Use(mw.MaxBodySize(defaultMaxBodySizeBytes))
		r.Use(mw.CSRFMiddleware(allowedOrigins))

		registerPublicRoutes(r, h, cacheStore)
		registerAuthRoutes(r, h, authService)
		registerOptionalAuthRoutes(r, h, authService)
		registerAdminRoutes(r, h, authService, cfg)
	})

	// The preview handler applies the configured upload limit with MaxBytesReader.
	// Keep it outside the default API body limit so only this upload endpoint can
	// accept a body larger than 2 MiB.
	router.Group(func(r chi.Router) {
		r.Use(mw.CSRFMiddleware(allowedOrigins))
		r.Use(mw.AuthMiddleware(authService))
		r.Use(mw.AdminAuthMiddleware)
		registerAdminDonationImportPreviewRoute(r, h.adminDonationImport)
	})
}

// registerPublicRoutes registers unauthenticated public endpoints.
func registerPublicRoutes(r chi.Router, h handlers, cacheStore *cache.Cache) {
	r.Get("/api/health", h.health.Check)
	r.Get("/api/feed/hero", h.feed.GetHero)
	r.Get("/api/donation/summary", h.donation.GetSummary)
	r.Get("/api/auth/kakao", h.auth.KakaoLogin)
	r.Get("/api/auth/kakao/callback", h.auth.KakaoCallback)
	r.With(mw.LoginRateLimiter(cacheStore)).Post("/api/auth/kakao/mobile", h.auth.KakaoMobileLogin)
	r.With(mw.LoginRateLimiter(cacheStore)).Post("/api/auth/apple/challenge", h.auth.AppleChallenge)
	r.With(mw.LoginRateLimiter(cacheStore)).Post("/api/auth/apple/mobile", h.auth.AppleMobileLogin)
	r.Post("/api/auth/apple/notifications", h.auth.AppleServerNotification)
	r.With(mw.LoginRateLimiter(cacheStore)).Post("/api/auth/mobile/login", h.auth.MobileLogin)
	r.With(mw.LoginRateLimiter(cacheStore)).Post("/api/auth/refresh", h.auth.Refresh)
	r.With(mw.LoginRateLimiter(cacheStore)).Post("/api/auth/login", h.auth.Login)
	r.With(mw.LoginRateLimiter(cacheStore)).Post("/api/auth/register", h.auth.Register)
	r.Get("/api/auth/check-id", h.auth.CheckID)
	r.Get("/api/auth/check-phone", h.auth.CheckPhone)
	r.Get("/api/auth/check-email", h.auth.CheckEmail)
	r.With(mw.LoginRateLimiter(cacheStore)).Post("/api/auth/social/link", h.auth.SocialLink)
	r.Get("/api/auth/social/link/prefill", h.auth.SocialLinkPrefill)
	r.Post("/api/auth/social/link/photo", h.socialLinkPhoto.Upload)
	r.With(mw.LoginRateLimiter(cacheStore)).Post("/api/auth/kakao/link", h.auth.KakaoLink)
	r.Get("/api/history", h.history.GetGrouped)
	r.Get("/api/public/job-categories", h.alumni.GetJobCategories)
	r.With(mw.LoginRateLimiter(cacheStore)).Post("/api/auth/password/reset-request", h.passwordReset.RequestReset)
	r.Post("/api/auth/password/reset-confirm", h.passwordReset.ConfirmReset)
	r.Get("/api/auth/password/validate-token", h.passwordReset.ValidateToken)
}

// registerAuthRoutes registers endpoints that require authentication.
func registerAuthRoutes(r chi.Router, h handlers, authService *service.AuthService) {
	r.Group(func(r chi.Router) {
		r.Use(mw.AuthMiddleware(authService))
		r.Get("/api/auth/me", h.auth.Me)
		r.Get("/api/auth/account/connections", h.auth.GetAccountConnections)
		r.Post("/api/auth/identities/link/{provider}", h.auth.LinkIdentity)
		r.Delete("/api/auth/social/{provider}", h.auth.DisconnectSocial)
		r.Post("/api/auth/logout", h.auth.Logout)
		r.Post("/api/auth/logout/all", h.auth.LogoutAll)
		r.Delete("/api/auth/account", h.auth.DeleteAccount)
		r.Get("/api/alumni/verification", h.profile.GetAlumniVerification)
		r.Put("/api/alumni/verification", h.profile.PutAlumniVerification)
		r.With(mw.ApprovedAlumniMiddleware).Get("/api/alumni", h.alumni.Search)
		r.With(mw.ApprovedAlumniMiddleware).Get("/api/alumni/filters", h.alumni.GetFilters)
		r.With(mw.ApprovedAlumniMiddleware).Get("/api/alumni/{userSeq}", h.alumni.GetDetail)
		r.With(mw.ApprovedAlumniMiddleware).Get("/api/alumni/widget", h.alumni.GetWidgetPreview)
		r.Get("/api/profile", h.profile.GetProfile)
		r.Put("/api/profile", h.profile.UpdateProfile)
		r.Post("/api/profile/photo", h.profileUpload.UploadPhoto)
		r.Post("/api/profile/bizcard", h.profileUpload.UploadBizCard)
		r.Post("/api/profile/password", h.passwordChange.ChangePassword)
		// DISABLED 2026-04-28: external donation redirect (dangled — see /home/jerryhwang/.claude/plans/drifting-gliding-hopcroft.md).
		// r.Post("/api/donation/orders", h.payment.CreateOrder)
		r.Get("/api/donation/my", h.personalDonation.GetMyDonations)
		r.Post("/api/feed/{seq}/like", h.like.ToggleLike)
		r.Post("/api/feed/{seq}/comments", h.comment.CreateComment)
		r.Delete("/api/feed/{seq}/comments/{cSeq}", h.comment.DeleteComment)
		// DISABLED 2026-04-28: external donation redirect (dangled — see /home/jerryhwang/.claude/plans/drifting-gliding-hopcroft.md).
		// r.Post("/api/donation/subscription", h.subscription.CreateSubscription)
		// r.Get("/api/donation/subscription", h.subscription.GetMySubscription)
		// r.Delete("/api/donation/subscription", h.subscription.CancelSubscription)
		r.With(mw.ApprovedAlumniMiddleware).Post("/api/messages", h.message.Send)
		r.With(mw.ApprovedAlumniMiddleware).Get("/api/messages/inbox", h.message.GetInbox)
		r.With(mw.ApprovedAlumniMiddleware).Get("/api/messages/outbox", h.message.GetOutbox)
		r.With(mw.ApprovedAlumniMiddleware).Put("/api/messages/{seq}/read", h.message.MarkAsRead)
		r.With(mw.ApprovedAlumniMiddleware).Delete("/api/messages/{seq}", h.message.Delete)
		r.With(mw.ApprovedAlumniMiddleware).Get("/api/messages/conversations", h.message.GetConversations)
		r.With(mw.ApprovedAlumniMiddleware).Get("/api/messages/conversations/{userSeq}", h.message.GetConversationMessages)
		r.With(mw.ApprovedAlumniMiddleware).Put("/api/messages/conversations/{userSeq}/read", h.message.MarkConversationRead)
		r.With(mw.ApprovedAlumniMiddleware).Get("/api/blocks", h.memberBlock.List)
		r.With(mw.ApprovedAlumniMiddleware).Get("/api/blocks/{userSeq}", h.memberBlock.Get)
		r.With(mw.ApprovedAlumniMiddleware).Put("/api/blocks/{userSeq}", h.memberBlock.Put)
		r.With(mw.ApprovedAlumniMiddleware).Delete("/api/blocks/{userSeq}", h.memberBlock.Delete)
		r.With(mw.ApprovedAlumniMiddleware).Post("/api/push/device/register", h.push.RegisterDevice)
		r.With(mw.ApprovedAlumniMiddleware).Post("/api/push/device/unregister", h.push.UnregisterDevice)
		r.With(mw.ApprovedAlumniMiddleware).Get("/api/push/preferences", h.push.GetPreferences)
		r.With(mw.ApprovedAlumniMiddleware).Put("/api/push/preferences", h.push.PutPreferences)
		r.With(mw.ApprovedAlumniMiddleware).Get("/api/badges", h.badge.GetBadges)
		r.With(mw.ApprovedAlumniMiddleware).Get("/api/messages/stream", h.realtime.Stream)
	})
}

// registerOptionalAuthRoutes registers endpoints that work with or without auth.
func registerOptionalAuthRoutes(r chi.Router, h handlers, authService *service.AuthService) {
	r.Group(func(r chi.Router) {
		r.Use(mw.OptionalAuthMiddleware(authService))
		r.Get("/api/feed", h.feed.GetFeed)
		r.Get("/api/feed/{seq}", h.feed.GetDetail)
		r.Get("/api/feed/{seq}/siblings", h.feed.GetSiblings)
		r.Get("/api/feed/{seq}/comments", h.comment.ListComments)
		r.Get("/api/disclosure", h.disclosure.GetList)
		r.Get("/api/disclosure/{seq}", h.disclosure.GetDetail)
		r.Get("/api/banner-ad/active", h.bannerAd.GetActive)
		r.Post("/api/banner-ad/{bnSeq}/view", h.bannerAd.TrackView)
		r.Post("/api/banner-ad/{bnSeq}/click", h.bannerAd.TrackClick)
		r.Post("/api/visit/beacon", h.visit.Beacon)
	})
}

// registerAdminRoutes registers admin-only endpoints.
func registerAdminRoutes(r chi.Router, h handlers, authService *service.AuthService, cfg *config.Config) {
	r.Route("/api/admin", func(r chi.Router) {
		r.Use(mw.AuthMiddleware(authService))
		r.Use(mw.AdminAuthMiddleware)
		r.Get("/dashboard", h.adminDashboard.Dashboard)
		r.Get("/stats/active-users", h.adminDashboard.ActiveUsers)
		r.Get("/feed", h.adminNotice.List)
		r.Get("/feed/{seq}", h.adminNotice.Detail)
		r.Post("/feed", h.adminNotice.Create)
		r.Put("/feed/{seq}", h.adminNotice.Update)
		r.Delete("/feed/{seq}", h.adminNotice.Delete)
		r.Put("/feed/{seq}/pin", h.adminNotice.TogglePin)
		r.Get("/disclosure", h.adminDisclosure.List)
		r.Get("/disclosure/{seq}", h.adminDisclosure.Detail)
		r.Post("/disclosure", h.adminDisclosure.Create)
		r.Put("/disclosure/{seq}", h.adminDisclosure.Update)
		r.Delete("/disclosure/{seq}", h.adminDisclosure.Delete)
		r.Post("/upload", h.adminUpload.Upload)
		r.Post("/upload/attachment", h.adminAttachUpload.Upload)
		r.Get("/banner-ad", h.adminBannerAd.List)
		r.Post("/banner-ad", h.adminBannerAd.Create)
		r.Put("/banner-ad/{seq}", h.adminBannerAd.Update)
		r.Delete("/banner-ad/{seq}", h.adminBannerAd.Delete)
		r.Get("/banner-ad/stats", h.adminBannerAd.Stats)
		r.Get("/banner-ad/{bnSeq}", h.adminBannerAd.Detail)
		r.Get("/donation/config", h.adminDonation.GetConfig)
		r.Put("/donation/config", h.adminDonation.UpdateConfig)
		r.Get("/donation/history", h.adminDonation.History)
		r.Route("/donation", func(donationRouter chi.Router) {
			registerAdminDonationRoutes(donationRouter, h.adminDonation, h.adminDonationImport)
		})
		r.Get("/member", h.adminMember.List)
		r.Get("/member/{seq}", h.adminMember.Detail)
		r.Put("/member/{seq}", h.adminMember.Update)
		r.Get("/member/stats", h.adminMember.Stats)
		r.Get("/alumni-verifications", h.adminMember.ListAlumniVerifications)
		r.Get("/alumni-verifications/{userSeq}", h.adminMember.GetAlumniVerificationDetail)
		r.Post("/alumni-verifications/{userSeq}/approve", h.adminMember.ApproveAlumniVerification)
		r.Post("/alumni-verifications/{userSeq}/reject", h.adminMember.RejectAlumniVerification)
		r.Get("/job-category", h.adminJobCat.List)
		r.Post("/job-category", h.adminJobCat.Create)
		r.Post("/job-category/reorder", h.adminJobCat.Reorder)
		r.Put("/job-category/{seq}", h.adminJobCat.Update)
		r.Delete("/job-category/{seq}", h.adminJobCat.Delete)
		r.Post("/errors/report", h.adminErrorReport.Report)
		r.Get("/history", h.history.AdminList)
		r.Post("/history", h.history.AdminCreate)
		r.Put("/history/{seq}", h.history.AdminUpdate)
		r.Delete("/history/{seq}", h.history.AdminDelete)

		// DISABLED 2026-04-28: external donation redirect (dangled — see /home/jerryhwang/.claude/plans/drifting-gliding-hopcroft.md).
		// Manual subscription billing trigger — only mounted in dev so production cannot
		// accidentally fire an out-of-cycle charge run.
		// if cfg.Environment == "dev" {
		// 	r.Post("/subscription/run-billing", h.adminSubscription.RunBilling)
		// }
	})
}

func registerAdminDonationRoutes(router chi.Router, donationHandler *handler.AdminDonationHandler, importHandler *handler.AdminDonationImportHandler) {
	router.Get("/orders", donationHandler.ListOrders)
	router.Get("/orders/{orderSeq}", donationHandler.GetOrder)
	router.Post("/orders", donationHandler.CreateOrder)
	router.Put("/orders/{orderSeq}", donationHandler.UpdateOrder)
	router.Post("/import/commit", importHandler.Commit)
}

func registerAdminDonationImportPreviewRoute(router chi.Router, importHandler *handler.AdminDonationImportHandler) {
	router.Post("/api/admin/donation/import/preview", importHandler.Preview)
}
