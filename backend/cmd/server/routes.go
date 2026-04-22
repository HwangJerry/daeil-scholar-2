// routes.go — HTTP route registration for the API server
package main

import (
	"net/http"

	"github.com/dflh-saf/backend/internal/config"
	"github.com/dflh-saf/backend/internal/handler"
	mw "github.com/dflh-saf/backend/internal/middleware"
	"github.com/dflh-saf/backend/internal/service"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
)

// handlers holds all HTTP handler instances for route registration.
type handlers struct {
	health          *handler.HealthHandler
	auth            *handler.AuthHandler
	feed            *handler.FeedHandler
	like            *handler.LikeHandler
	comment         *handler.CommentHandler
	donation        *handler.DonationHandler
	alumni          *handler.AlumniHandler
	profile         *handler.ProfileHandler
	profileUpload   *handler.ProfileUploadHandler
	ad              *handler.AdHandler
	adLike          *handler.AdLikeHandler
	adComment       *handler.AdCommentHandler
	adminNotice     *handler.AdminNoticeHandler
	adminAd         *handler.AdminAdHandler
	adminDonation   *handler.AdminDonationHandler
	adminMember     *handler.AdminMemberHandler
	adminDashboard  *handler.AdminDashboardHandler
	adminUpload     *handler.AdminUploadHandler
	myDonation      *handler.MyDonationHandler
	message         *handler.MessageHandler
	payment         *handler.PaymentHandler
	subscription    *handler.SubscriptionHandler
	og              *handler.OGHandler
	sitemap         *handler.SitemapHandler
	rss             *handler.RSSHandler
	passwordReset   *handler.PasswordResetHandler
	passwordChange  *handler.PasswordChangeHandler
	badge           *handler.BadgeHandler
	adminJobCat     *handler.AdminJobCategoryHandler
}

// registerRoutes creates a chi.Router with all middleware and API routes.
func registerRoutes(h handlers, authService *service.AuthService, cacheStore *cache.Cache, allowedOrigins []string, uploadCfg config.UploadConfig, logger zerolog.Logger) chi.Router {
	router := chi.NewRouter()
	router.Use(chimw.Recoverer)
	router.Use(mw.RequestLogger(logger))
	router.Use(mw.CORSMiddleware(allowedOrigins))
	router.Use(mw.MaxBodySize(1 << 20))

	// Static file servers (dev: proxied from Vite/Nginx; prod: handled by Nginx alias)
	router.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadCfg.BasePath))))
	router.Handle("/files/*", http.StripPrefix("/files/", http.FileServer(http.Dir(uploadCfg.LegacyPath))))

	// Bot-facing OG and sitemap endpoints (no CSRF, read-only)
	router.Get("/og/post/{seq}", h.og.GetPostOG)
	router.Get("/sitemap.xml", h.sitemap.GetSitemap)
	router.Get("/rss.xml", h.rss.GetRSS)

	registerPGRoutes(router, h)
	registerAPIRoutes(router, h, authService, cacheStore, allowedOrigins)

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
func registerAPIRoutes(router chi.Router, h handlers, authService *service.AuthService, cacheStore *cache.Cache, allowedOrigins []string) {
	router.Group(func(r chi.Router) {
		r.Use(mw.CSRFMiddleware(allowedOrigins))

		registerPublicRoutes(r, h, cacheStore)
		registerAuthRoutes(r, h, authService)
		registerOptionalAuthRoutes(r, h, authService)
		registerAdminRoutes(r, h, authService)
	})
}

// registerPublicRoutes registers unauthenticated public endpoints.
func registerPublicRoutes(r chi.Router, h handlers, cacheStore *cache.Cache) {
	r.Get("/api/health", h.health.Check)
	r.Get("/api/feed/hero", h.feed.GetHero)
	r.Get("/api/donation/summary", h.donation.GetSummary)
	r.Get("/api/auth/kakao", h.auth.KakaoLogin)
	r.Get("/api/auth/kakao/callback", h.auth.KakaoCallback)
	r.With(mw.LoginRateLimiter(cacheStore)).Post("/api/auth/login", h.auth.Login)
	r.With(mw.LoginRateLimiter(cacheStore)).Post("/api/auth/register", h.auth.Register)
	r.Get("/api/auth/check-id", h.auth.CheckID)
	r.Get("/api/auth/check-phone", h.auth.CheckPhone)
	r.Get("/api/auth/check-email", h.auth.CheckEmail)
	r.Post("/api/auth/social/link", h.auth.SocialLink)
	r.Post("/api/auth/kakao/link", h.auth.KakaoLink)
	r.Get("/api/alumni/widget", h.alumni.GetWidgetPreview)
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
		r.Post("/api/auth/logout", h.auth.Logout)
		r.Get("/api/alumni", h.alumni.Search)
		r.Get("/api/alumni/filters", h.alumni.GetFilters)
		r.Get("/api/profile", h.profile.GetProfile)
		r.Put("/api/profile", h.profile.UpdateProfile)
		r.Post("/api/profile/photo", h.profileUpload.UploadPhoto)
		r.Post("/api/profile/bizcard", h.profileUpload.UploadBizCard)
		r.Post("/api/profile/password", h.passwordChange.ChangePassword)
		r.Post("/api/donation/orders", h.payment.CreateOrder)
		r.Get("/api/donation/my", h.myDonation.GetMyDonations)
		r.Post("/api/feed/{seq}/like", h.like.ToggleLike)
		r.Post("/api/feed/{seq}/comments", h.comment.CreateComment)
		r.Delete("/api/feed/{seq}/comments/{cSeq}", h.comment.DeleteComment)
		r.Post("/api/ad/{maSeq}/like", h.adLike.ToggleLike)
		r.Post("/api/ad/{maSeq}/comments", h.adComment.CreateComment)
		r.Delete("/api/ad/{maSeq}/comments/{acSeq}", h.adComment.DeleteComment)
		r.Post("/api/donation/subscription", h.subscription.CreateSubscription)
		r.Get("/api/donation/subscription", h.subscription.GetMySubscription)
		r.Delete("/api/donation/subscription", h.subscription.CancelSubscription)
		r.Post("/api/messages", h.message.Send)
		r.Get("/api/messages/inbox", h.message.GetInbox)
		r.Get("/api/messages/outbox", h.message.GetOutbox)
		r.Put("/api/messages/{seq}/read", h.message.MarkAsRead)
		r.Delete("/api/messages/{seq}", h.message.Delete)
		r.Get("/api/messages/conversations", h.message.GetConversations)
		r.Get("/api/messages/conversations/{userSeq}", h.message.GetConversationMessages)
		r.Put("/api/messages/conversations/{userSeq}/read", h.message.MarkConversationRead)
		r.Get("/api/badges", h.badge.GetBadges)
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
		r.Post("/api/ad/{maSeq}/view", h.ad.TrackView)
		r.Post("/api/ad/{maSeq}/click", h.ad.TrackClick)
		r.Get("/api/ad/{maSeq}/comments", h.adComment.ListComments)
	})
}

// registerAdminRoutes registers admin-only endpoints.
func registerAdminRoutes(r chi.Router, h handlers, authService *service.AuthService) {
	r.Route("/api/admin", func(r chi.Router) {
		r.Use(mw.AuthMiddleware(authService))
		r.Use(mw.AdminAuthMiddleware)
		r.Get("/dashboard", h.adminDashboard.Dashboard)
		r.Get("/feed", h.adminNotice.List)
		r.Get("/feed/{seq}", h.adminNotice.Detail)
		r.Post("/feed", h.adminNotice.Create)
		r.Put("/feed/{seq}", h.adminNotice.Update)
		r.Delete("/feed/{seq}", h.adminNotice.Delete)
		r.Put("/feed/{seq}/pin", h.adminNotice.TogglePin)
		r.Post("/upload", h.adminUpload.Upload)
		r.Get("/ad", h.adminAd.List)
		r.Post("/ad", h.adminAd.Create)
		r.Put("/ad/{seq}", h.adminAd.Update)
		r.Delete("/ad/{seq}", h.adminAd.Delete)
		r.Get("/ad/stats", h.adminAd.Stats)
		r.Get("/donation/config", h.adminDonation.GetConfig)
		r.Put("/donation/config", h.adminDonation.UpdateConfig)
		r.Get("/donation/history", h.adminDonation.History)
		r.Get("/donation/orders", h.adminDonation.ListOrders)
		r.Put("/donation/orders/{seq}", h.adminDonation.UpdateOrder)
		r.Get("/member", h.adminMember.List)
		r.Get("/member/{seq}", h.adminMember.Detail)
		r.Put("/member/{seq}", h.adminMember.Update)
		r.Get("/member/stats", h.adminMember.Stats)
		r.Get("/job-category", h.adminJobCat.List)
		r.Post("/job-category", h.adminJobCat.Create)
		r.Put("/job-category/{seq}", h.adminJobCat.Update)
		r.Delete("/job-category/{seq}", h.adminJobCat.Delete)
	})
}
