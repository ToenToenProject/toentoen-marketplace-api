package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
	"toentoen-backend/internal/domain/account"
	"toentoen-backend/internal/domain/auth"
	"toentoen-backend/internal/domain/customer/business_profile"
	"toentoen-backend/internal/domain/customer/explore"
	"toentoen-backend/internal/domain/customer/interaction"
	"toentoen-backend/internal/domain/customer/my_activity"
	"toentoen-backend/internal/domain/location"
	"toentoen-backend/internal/domain/realtime"
	"toentoen-backend/internal/domain/signal"

	"toentoen-backend/internal/constants"
	"toentoen-backend/internal/domain/business_chat"
	"toentoen-backend/internal/domain/customer/product"
	"toentoen-backend/internal/domain/customer/profile"
	customer_promotion "toentoen-backend/internal/domain/customer/promotion"
	"toentoen-backend/internal/domain/customer/report"
	"toentoen-backend/internal/domain/customer/review"
	"toentoen-backend/internal/domain/merchant/admin"
	"toentoen-backend/internal/domain/merchant/analytics"
	"toentoen-backend/internal/domain/merchant/business"
	"toentoen-backend/internal/domain/merchant/catalog"
	"toentoen-backend/internal/domain/merchant/post"
	merchant_promotion "toentoen-backend/internal/domain/merchant/promotion"
	"toentoen-backend/internal/domain/merchant/transfer"
	"toentoen-backend/internal/domain/meta"
	"toentoen-backend/internal/domain/notification"
	"toentoen-backend/internal/domain/shared/models"
	support_chat "toentoen-backend/internal/domain/support/chat"
	support_feedback "toentoen-backend/internal/domain/support/feedback"
	support_ticket "toentoen-backend/internal/domain/support/ticket"
	"toentoen-backend/internal/domain/webhook"

	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/shared/middlewares"
	"toentoen-backend/internal/domain/upload"
	"toentoen-backend/pkg/database"
	"toentoen-backend/pkg/meilisearch"
	"toentoen-backend/pkg/storage"
	"toentoen-backend/pkg/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/websocket/v2"
	"github.com/joho/godotenv"
)

func main() {
	// 1. Config & Database
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	config.InitConfig()
	database.ConnectDB()
	config.ConnectRedis()
	meilisearch.ConnectMeilisearch()
	storage.InitS3Client()
	config.InitFirebase()

	// Auto-sync businesses and products to Meilisearch on startup.
	go func() {
		meilisearch.InitMeilisearchIndices()
		if errSyn := meilisearch.SyncSynonymsFromDB(database.DB); errSyn != nil {
			log.Printf("⚠️ Meilisearch auto-sync synonyms error: %v", errSyn)
		}
		countBiz, err := meilisearch.SyncAllBusinesses(database.DB)
		if err != nil {
			log.Printf("⚠️ Meilisearch auto-sync businesses error: %v", err)
		} else {
			log.Printf("🔍 Meilisearch auto-synced %d businesses successfully", countBiz)
		}

		countProd, errProd := meilisearch.SyncAllProducts(database.DB)
		if errProd != nil {
			log.Printf("⚠️ Meilisearch auto-sync products error: %v", errProd)
		} else {
			log.Printf("🔍 Meilisearch auto-synced %d products successfully", countProd)
		}

		// Periodic reconciliation (every 10m) of public visibility following KYC status changes.
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if _, err := meilisearch.SyncAllBusinesses(database.DB); err != nil {
				log.Printf("⚠️ Periodic KYC visibility Meilisearch sync error: %v", err)
			}
		}
	}()

	wsHub := realtime.NewHub(config.RDB)
	go wsHub.Run()

	// Live location service.
	locationStore := location.NewRedisLocationStore(config.RDB)
	locationService := location.NewLocationService(locationStore, wsHub, database.DB)
	locationWSHandler := location.NewWSLocationHandler(locationService)
	locationHTTPHandler := location.NewHTTPLocationHandler(locationService)

	// Register WebSocket handler for live location updates.
	wsHub.RegisterMessageHandler("location", locationWSHandler)

	// Sync merchant locations from PostgreSQL to Redis every 15 minutes.
	locationService.StartPeriodicSync(15 * time.Minute)

	// 2. Repositories, Services, and Handlers
	authRepo := auth.NewAuthRepository(database.DB)
	authService := auth.NewAuthService(authRepo)
	authHandler := auth.NewAuthHandler(authService)

	profileRepo := account.NewProfileRepository(database.DB)
	profileService := account.NewProfileService(profileRepo)
	profileHandler := account.NewProfileHandler(profileService)

	// Notification service.
	notifRepo := notification.NewNotificationRepository(database.DB)
	notifService := notification.NewNotificationService(notifRepo, wsHub)
	notifHandler := notification.NewNotificationHandler(notifService)
	profileService.SetNotificationService(notifService)

	businessRepo := business.NewBusinessRepository(database.DB)
	business.StartScheduleReminderWorker(businessRepo, notifService) // Jalankan worker pengingat jadwal & auto-close
	business.StartStoreStatusSyncWorker(database.DB)                 // Jalankan worker evaluasi buka/tutup toko per menit ke Meilisearch
	businessService := business.NewBusinessService(businessRepo, notifService)
	businessHandler := business.NewBusinessHandler(businessService)

	productRepo := catalog.NewProductRepository(database.DB)
	productService := catalog.NewProductService(productRepo, notifService)
	productHandler := catalog.NewProductHandler(productService)

	packageRepo := catalog.NewPackageRepository(database.DB)
	packageService := catalog.NewPackageService(packageRepo, notifService)
	packageHandler := catalog.NewPackageHandler(packageService)

	productGroupRepo := catalog.NewProductGroupRepository(database.DB)
	productGroupService := catalog.NewProductGroupService(productGroupRepo, notifService)
	productGroupHandler := catalog.NewProductGroupHandler(productGroupService)

	// Business admin service.
	businessAdminRepo := admin.NewBusinessAdminRepository(database.DB)
	businessAdminService := admin.NewBusinessAdminService(businessAdminRepo, businessRepo, notifService)
	businessAdminHandler := admin.NewBusinessAdminHandler(businessAdminService)
	admin.StartGracePeriodWorker(businessAdminRepo, 30*time.Minute)

	productInteractionRepo := interaction.NewProductInteractionRepository(database.DB)
	productInteractionService := interaction.NewProductInteractionService(productInteractionRepo, businessRepo, notifService)
	productInteractionHandler := interaction.NewProductInteractionHandler(productInteractionService)

	postInteractionRepo := interaction.NewPostInteractionRepository(database.DB)
	postInteractionService := interaction.NewPostInteractionService(postInteractionRepo, businessRepo, notifService)
	postInteractionHandler := interaction.NewPostInteractionHandler(postInteractionService)

	// Promotion service.
	promotionRepo := merchant_promotion.NewPromotionRepository(database.DB)
	promotionService := merchant_promotion.NewPromotionService(promotionRepo, notifService)
	promotionHandler := merchant_promotion.NewPromotionHandler(promotionService)
	customerPromotionHandler := customer_promotion.NewCustomerPromotionHandler(promotionService, promotionRepo)
	merchant_promotion.StartPromotionWorker(database.DB, notifService)

	businessPostRepo := post.NewBusinessPostRepository(database.DB)
	businessPostService := post.NewBusinessPostService(businessPostRepo, productRepo, promotionRepo, notifService)
	businessPostHandler := post.NewBusinessPostHandler(businessPostService)

	// Demand Signal V2 service with real-time hub and notifications.
	signalRepo := signal.NewSignalRepository(database.DB)
	signal.StartSignalExpiryWorker(signalRepo) // Background scheduler auto-expire overdue signals
	signalService := signal.NewSignalService(signalRepo, businessRepo, notifService, wsHub)
	customerSignalHandler := signal.NewCustomerHandler(signalService)
	merchantSignalHandler := signal.NewMerchantHandler(signalService)
	wsHandler := realtime.NewWSHandler(wsHub)

	analyticsRepo := analytics.NewAnalyticsRepository(database.DB)
	analyticsService := analytics.NewAnalyticsService(analyticsRepo, productRepo)
	analyticsHandler := analytics.NewAnalyticsHandler(analyticsService)

	reviewRepo := review.NewReviewRepository(database.DB)
	reviewService := review.NewReviewService(reviewRepo, businessRepo, notifService)
	reviewHandler := review.NewReviewHandler(reviewService, productInteractionService)

	exploreRepo := explore.NewExploreRepository(database.DB)
	exploreService := explore.NewExploreService(exploreRepo)
	exploreHandler := explore.NewExploreHandler(exploreService)

	businessProfileRepo := business_profile.NewRepository()
	businessProfileService := business_profile.NewService(businessProfileRepo, analyticsService)
	businessProfileHandler := business_profile.NewHandler(businessProfileService)

	reportRepo := report.NewRepository()
	reportService := report.NewService(reportRepo, notifService)
	reportHandler := report.NewHandler(reportService)

	productDetailRepo := product.NewProductDetailRepository(database.DB)
	productDetailService := product.NewProductDetailService(productDetailRepo)
	productDetailHandler := product.NewProductDetailHandler(productDetailService)

	publicProfileRepo := profile.NewPublicProfileRepository(database.DB)
	publicProfileService := profile.NewPublicProfileService(publicProfileRepo)
	publicProfileHandler := profile.NewPublicProfileHandler(publicProfileService)

	myActivityRepo := my_activity.NewMyActivityRepository(database.DB)
	myActivityService := my_activity.NewMyActivityService(myActivityRepo)
	myActivityHandler := my_activity.NewMyActivityHandler(myActivityService)

	uploadHandler := upload.NewHandler()

	metaRepo := meta.NewMetaRepository(database.DB)
	metaService := meta.NewMetaService(metaRepo)
	metaHandler := meta.NewMetaHandler(metaService)

	supportChatRepo := support_chat.NewSupportChatRepository(database.DB)
	supportChatService := support_chat.NewSupportChatService(supportChatRepo, wsHub, notifService)
	supportChatHandler := support_chat.NewSupportChatHandler(supportChatService)

	supportTicketRepo := support_ticket.NewSupportTicketRepository(database.DB)
	supportTicketService := support_ticket.NewSupportTicketService(supportTicketRepo)
	supportTicketHandler := support_ticket.NewSupportTicketHandler(supportTicketService)

	feedbackRepo := support_feedback.NewFeedbackRepository(database.DB)
	feedbackService := support_feedback.NewFeedbackService(feedbackRepo)
	feedbackHandler := support_feedback.NewFeedbackHandler(feedbackService)

	businessChatRepo := business_chat.NewBusinessChatRepository(database.DB)
	businessChatService := business_chat.NewBusinessChatService(businessChatRepo, wsHub, notifService)
	businessChatHandler := business_chat.NewBusinessChatHandler(businessChatService)

	businessTransferRepo := transfer.NewBusinessTransferRepository(database.DB)
	businessTransferService := transfer.NewBusinessTransferService(businessTransferRepo, notifService)
	businessTransferHandler := transfer.NewBusinessTransferHandler(businessTransferService)

	webhookHandler := webhook.NewWebhookHandler()

	// 3. HTTP Server & Routing Setup
	app := fiber.New(fiber.Config{
		ProxyHeader:             fiber.HeaderXForwardedFor,
		EnableTrustedProxyCheck: true,
		TrustedProxies: []string{
			"127.0.0.1",
			"::1",
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
		},
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			var e *fiber.Error
			if errors.As(err, &e) {
				code = e.Code
			}
			msg := "Terjadi kesalahan pada server"
			respCode := constants.InternalServerErr
			if code == fiber.StatusNotFound {
				msg = "Rute tidak ditemukan"
				respCode = constants.DataNotFound
			} else if code == fiber.StatusMethodNotAllowed {
				msg = "Metode permintaan tidak diizinkan"
				respCode = constants.InvalidParam
			}
			return utils.SendResponse(c, code, respCode, msg, nil)
		},
	})

	// Security & Core Middlewares
	app.Use(recover.New())
	app.Use(helmet.New())
	app.Use(logger.New())

	// CORS Configuration
	corsConfig := cors.Config{
		AllowOrigins:     os.Getenv("FRONTEND_URL"),
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Requested-With, X-Internal-Secret",
		AllowMethods:     "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD",
		AllowCredentials: true,
	}

	// Fallback default for FRONTEND_URL.
	if corsConfig.AllowOrigins == "" {
		corsConfig.AllowOrigins = "*"
		// Fiber v2 panics when AllowOrigins="*" is combined with AllowCredentials=true.
		corsConfig.AllowCredentials = false
	}

	app.Use(cors.New(corsConfig))

	// Resolves client IP using Fiber's trusted proxy check (prevents header spoofing).
	getClientIP := func(c *fiber.Ctx) string {
		return c.IP()
	}

	// Global rate limiter.
	rateLimitMax := 1000
	if envMax := os.Getenv("RATE_LIMIT_MAX"); envMax != "" {
		if parsedMax, err := strconv.Atoi(envMax); err == nil && parsedMax > 0 {
			rateLimitMax = parsedMax
		}
	}

	app.Use(limiter.New(limiter.Config{
		Max:        rateLimitMax,
		Expiration: 1 * time.Minute,
		Next: func(c *fiber.Ctx) bool {
			// Optional rate limit bypass for local development.
			if os.Getenv("ENABLE_RATE_LIMITER") == "false" {
				return true
			}
			// Bypass rate limiter for CORS OPTIONS preflight requests.
			if c.Method() == fiber.MethodOptions {
				return true
			}
			// Rate-limit /api/v1 and webhook endpoints; ignore unrelated 404 bot scans.
			if !strings.HasPrefix(c.Path(), "/api/v1") && !strings.HasPrefix(c.Path(), "/webhook/") {
				return true
			}
			return false
		},
		KeyGenerator: getClientIP,
		LimitReached: func(c *fiber.Ctx) error {
			clientIP := getClientIP(c)
			log.Printf("⚠️ [RateLimiter Global 429] IP: %s terblokir di %s %s", clientIP, c.Method(), c.Path())
			return utils.SendResponse(c, fiber.StatusTooManyRequests, "429", "Terlalu banyak request. Silakan coba lagi nanti.", nil)
		},
	}))

	// Static assets.
	app.Static("/assets", "./assets")

		// Webhook routes (Meta WhatsApp Cloud API)
	app.Get("/webhook/whatsapp", webhookHandler.VerifyWhatsAppWebhook)
	app.Post("/webhook/whatsapp", webhookHandler.HandleWhatsAppWebhookEvent)

	api := app.Group("/api/v1")
	api.Get("/webhook/whatsapp", webhookHandler.VerifyWhatsAppWebhook)
	api.Post("/webhook/whatsapp", webhookHandler.HandleWhatsAppWebhookEvent)

		// Legal & compliance routes (Google Play Store)
	registerLegalRoutes(app, api)

		// Public metadata routes (Categories, Tags)
	metaGroup := api.Group("/meta")
	metaGroup.Get("/categories", metaHandler.GetCategories)
	metaGroup.Get("/tags", metaHandler.GetTags)
	metaGroup.Get("/store-types", metaHandler.GetStoreTypes)
	metaGroup.Get("/occupations", metaHandler.GetOccupations)
	metaGroup.Get("/report-categories", reportHandler.GetReportCategories)

		// WebSocket route
	api.Get("/ws", realtime.UpgradeCheck, websocket.New(wsHandler.Handle))

		// Upload routes (S3 presigned URLs)
	uploadGroup := api.Group("/upload", middlewares.RequireAuth)
	uploadGroup.Post("/presign", uploadHandler.GeneratePresignURL)
	uploadGroup.Get("/limits", uploadHandler.GetUploadLimits)

		// Authentication routes
	api.Post("/logout", auth.Logout)

	authLimiterMax := 60
	if envAuthMax := os.Getenv("AUTH_RATE_LIMIT_MAX"); envAuthMax != "" {
		if parsedAuthMax, err := strconv.Atoi(envAuthMax); err == nil && parsedAuthMax > 0 {
			authLimiterMax = parsedAuthMax
		}
	}

	authLimiter := limiter.New(limiter.Config{
		Max:        authLimiterMax,
		Expiration: 1 * time.Minute,
		Next: func(c *fiber.Ctx) bool {
			if os.Getenv("ENABLE_RATE_LIMITER") == "false" {
				return true
			}
			return c.Method() == fiber.MethodOptions
		},
		KeyGenerator: getClientIP,
		LimitReached: func(c *fiber.Ctx) error {
			clientIP := getClientIP(c)
			log.Printf("⚠️ [RateLimiter Auth 429] IP: %s terblokir di %s %s", clientIP, c.Method(), c.Path())
			return utils.SendResponse(c, fiber.StatusTooManyRequests, "429", "Terlalu banyak percobaan autentikasi. Silakan coba lagi setelah 1 menit.", nil)
		},
	})

	authGroup := api.Group("/auth", authLimiter)
	authGroup.Get("/check-referral", authHandler.CheckReferralCode)
	authGroup.Post("/register", authHandler.Register)
	authGroup.Post("/verify-register-otp", authHandler.VerifyRegisterOTP)
	authGroup.Post("/login", authHandler.Login)
	authGroup.Post("/verify-login-otp", authHandler.VerifyLoginOTP)
	authGroup.Post("/forgot-password", authHandler.ForgotPassword)
	authGroup.Post("/verify-forgot-otp", authHandler.VerifyForgotOTP)
	authGroup.Post("/reset-password", authHandler.ResetPassword)
	authGroup.Post("/email/send-otp", authHandler.SendEmailOTP)
	authGroup.Post("/email/verify-otp", authHandler.VerifyEmailOTP)
	authGroup.Post("/phone/send-otp", authHandler.SendPhoneOTP)
	authGroup.Post("/phone/verify-otp", authHandler.VerifyPhoneOTP)

	// Customer business profile & bookmark routes
	customerBusinessGroup := api.Group("/customer/businesses")
	customerBusinessGroup.Get("/bookmarks", middlewares.OptionalAuth, businessProfileHandler.GetBookmarks)
	customerBusinessGroup.Get("/:business_id", middlewares.OptionalAuth, businessProfileHandler.GetProfile)
	customerBusinessGroup.Post("/:business_id/bookmark", middlewares.RequireAuth, middlewares.RequireEmailVerified, businessProfileHandler.ToggleBookmark)
	customerBusinessGroup.Post("/:business_id/like", middlewares.RequireAuth, middlewares.RequireEmailVerified, businessProfileHandler.ToggleLike)
	customerBusinessGroup.Post("/:business_id/report", middlewares.RequireAuth, middlewares.RequireEmailVerified, reportHandler.CreateReport)

	// Products
	customerBusinessGroup.Get("/:business_id/product-groups", businessProfileHandler.GetProductGroups)
	customerBusinessGroup.Get("/:business_id/products", businessProfileHandler.GetProducts)
	customerBusinessGroup.Get("/:business_id/product-reviews/summary", productInteractionHandler.GetBusinessProductReviewsSummary)
	customerBusinessGroup.Get("/:business_id/posts", middlewares.OptionalAuth, businessProfileHandler.GetPosts)

	// User profile routes
	protected := api.Group("/account", middlewares.RequireAuth)
	protected.Get("/profile/me", profileHandler.GetMyProfile)
	protected.Get("/kyc/status", profileHandler.GetKYCStatus)
	protected.Post("/kyc/submit", middlewares.RequireEmailVerified, profileHandler.SubmitKYC)
	protected.Post("/kyc/consent", middlewares.RequireEmailVerified, profileHandler.SubmitConsent)
	protected.Put("/profile/customer", middlewares.AllowRoles("customer"), profileHandler.UpdateProfileCustomer)
	protected.Put("/profile/owner", middlewares.AllowRoles("owner"), profileHandler.UpdateProfileOwner)
	protected.Put("/profile/admin", middlewares.AllowRoles("business_admin"), profileHandler.UpdateProfileBusinessAdmin)
	protected.Post("/password/change", authHandler.ChangePassword)
	protected.Post("/email/change/request", authHandler.RequestEmailChange)
	protected.Post("/email/change/verify", authHandler.VerifyEmailChange)
	protected.Post("/phone/change/request", authHandler.RequestPhoneChange)
	protected.Post("/phone/change/verify", authHandler.VerifyPhoneChange)
	protected.Post("/delete/request", profileHandler.RequestDeleteOTP)
	protected.Delete("/delete/execute", profileHandler.VerifyDeleteOTPAndExecute)

	// Push notification token registration and removal.
	protected.Post("/device-token", profileHandler.RegisterDeviceToken)
	protected.Delete("/device-token", profileHandler.DeleteDeviceToken)

	// User notifications.
	directNotifGroup := api.Group("/notifications", middlewares.RequireAuth)
	directNotifGroup.Get("/", notifHandler.GetMyNotifications)
	directNotifGroup.Get("/unread-count", notifHandler.GetUnreadCount)
	directNotifGroup.Put("/read-all", notifHandler.MarkAllAsRead)
	directNotifGroup.Put("/:id/read", notifHandler.MarkAsRead)
	directNotifGroup.Delete("/:id", notifHandler.DeleteNotification)
	directNotifGroup.Delete("/", notifHandler.DeleteAllNotifications)

	// Internal webhook for toentoen-internal-operations (WebSocket & push notification triggers).
	api.Post("/internal/notifications", middlewares.RequireInternalSecret, notifHandler.SendInternalNotification)

	// My Activity
	myActivityGroup := protected.Group("/my-activity")
	myActivityGroup.Get("/summary", myActivityHandler.GetMyProfileSummary)
	myActivityGroup.Get("/comments", myActivityHandler.GetMyComments)
	myActivityGroup.Get("/reviews", myActivityHandler.GetMyBusinessReviews)
	myActivityGroup.Get("/bookmarks", myActivityHandler.GetMyBookmarks)
	myActivityGroup.Get("/wishlist", myActivityHandler.GetMyWishlists)

	api.Post("/customer/products/:product_id/wishlist", middlewares.RequireAuth, middlewares.RequireEmailVerified, myActivityHandler.ToggleWishlist)
	api.Get("/customer/wishlist", middlewares.RequireAuth, middlewares.RequireEmailVerified, myActivityHandler.GetMyWishlists)

	// Demand Signal V2 customer routes
	customerSignalGroup := api.Group("/customer/signals", middlewares.RequireAuth, middlewares.RequireEmailVerified, middlewares.AllowRoles("customer"))
	customerSignalGroup.Post("/", customerSignalHandler.CreateSignal)
	customerSignalGroup.Get("/", customerSignalHandler.GetMySignals)
	customerSignalGroup.Get("/:signal_id", customerSignalHandler.GetSignalDetail)
	customerSignalGroup.Put("/:signal_id", customerSignalHandler.UpdateSignal)
	customerSignalGroup.Delete("/:signal_id", customerSignalHandler.DeleteSignal)
	customerSignalGroup.Patch("/:signal_id/close", customerSignalHandler.CloseSignal)
	customerSignalGroup.Patch("/:signal_id/complete", customerSignalHandler.CompleteSignal)

	customerConvGroup := api.Group("/customer/conversations", middlewares.RequireAuth, middlewares.RequireEmailVerified)
	customerConvGroup.Get("/", customerSignalHandler.GetMyConversations)
	customerConvGroup.Get("/:conversation_id/messages", customerSignalHandler.GetMessages)
	customerConvGroup.Post("/:conversation_id/messages", customerSignalHandler.SendMessage)

	// Unified interaction routes (Reports, Likes, Comments, Reviews)
	api.Get("/reports/categories", reportHandler.GetReportCategories)
	adminReportRoles := middlewares.AllowRoles("platform_admin", "super_admin", "admin", "superadmin")
	reportsGroup := api.Group("/reports", middlewares.RequireAuth, middlewares.RequireEmailVerified)
	reportsGroup.Post("/", reportHandler.CreateReport)
	reportsGroup.Get("/", adminReportRoles, reportHandler.GetAllReports)
	reportsGroup.Get("/my", reportHandler.GetMyReports)
	reportsGroup.Get("/:report_id", reportHandler.GetReportDetail)
	reportsGroup.Delete("/:report_id", reportHandler.DeleteReport)
	reportsGroup.Patch("/:report_id/status", adminReportRoles, reportHandler.UpdateReportStatus)

	commentsGroup := api.Group("/comments", middlewares.RequireAuth, middlewares.RequireEmailVerified)
	commentsGroup.Post("/:comment_id/like", productInteractionHandler.ToggleUnifiedCommentLike)
	commentsGroup.Delete("/:comment_id", productInteractionHandler.DeleteUnifiedComment)
	commentsGroup.Post("/:comment_id/report", reportHandler.CreateReport)

	reviewsGroup := api.Group("/reviews")
	reviewsGroup.Post("/:review_id/like", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.ToggleUnifiedReviewLike)
	reviewsGroup.Post("/:review_id/replies", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.AddReviewReply)
	reviewsGroup.Post("/:review_id/report", middlewares.RequireAuth, middlewares.RequireEmailVerified, reportHandler.CreateReport)

	publicProductRoutes := api.Group("/products")
	publicProductRoutes.Get("/:product_id", middlewares.OptionalAuth, productDetailHandler.GetProductDetail)

	// Product review & comment routes
	publicProductRoutes.Post("/:product_id/reviews", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.AddProductReview)
	publicProductRoutes.Get("/:product_id/reviews", middlewares.OptionalAuth, productInteractionHandler.GetProductReviews)
	publicProductRoutes.Get("/:product_id/reviews/smart", middlewares.OptionalAuth, productInteractionHandler.GetSmartDisplayProductReviews)
	publicProductRoutes.Get("/reviews/detail/:review_id", middlewares.OptionalAuth, productInteractionHandler.GetProductReviewDetail)
	publicProductRoutes.Put("/reviews/:review_id", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.UpdateProductReview)
	publicProductRoutes.Delete("/reviews/:review_id", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.DeleteProductReview)
	publicProductRoutes.Post("/:product_id/comments", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.AddProductComment)
	publicProductRoutes.Put("/:product_id/comments/:comment_id", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.EditProductComment)
	publicProductRoutes.Delete("/:product_id/comments/:comment_id", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.DeleteProductComment)
	publicProductRoutes.Get("/:product_id/comments", middlewares.OptionalAuth, productInteractionHandler.GetProductComments)

	publicPostRoutes := api.Group("/posts")
	publicPostRoutes.Get("/business/:business_id", middlewares.OptionalAuth, businessProfileHandler.GetPosts)
	publicPostRoutes.Post("/:post_id/like", middlewares.RequireAuth, middlewares.RequireEmailVerified, postInteractionHandler.ToggleLike)
	publicPostRoutes.Post("/:post_id/comments", middlewares.RequireAuth, middlewares.RequireEmailVerified, postInteractionHandler.AddPostComment)
	publicPostRoutes.Put("/:post_id/comments/:comment_id", middlewares.RequireAuth, middlewares.RequireEmailVerified, postInteractionHandler.EditPostComment)
	publicPostRoutes.Delete("/:post_id/comments/:comment_id", middlewares.RequireAuth, middlewares.RequireEmailVerified, postInteractionHandler.DeletePostComment)
	publicPostRoutes.Get("/:post_id/comments", middlewares.OptionalAuth, postInteractionHandler.GetPostComments)
	publicPostRoutes.Post("/:post_id/report", middlewares.RequireAuth, middlewares.RequireEmailVerified, reportHandler.CreateReport)

	// Business owner and admin routes
	// Admin management (Owner & Supervisor).
	adminGroup := api.Group("/business/:business_id/admins", middlewares.RequireAuth, middlewares.AllowRoles("owner", "business_admin"), middlewares.BusinessAccessGuard())
	adminGroup.Post("/invite", businessAdminHandler.InviteAdmin)
	adminGroup.Post("/assignments/:assignment_id/approve", middlewares.AllowRoles("owner"), businessAdminHandler.RespondOwnerApproval)
	adminGroup.Get("/", businessAdminHandler.GetAdminTree)
	adminGroup.Patch("/:assignment_id", businessAdminHandler.UpdateAdminAssignment)
	adminGroup.Delete("/:assignment_id", businessAdminHandler.TerminateAdmin)
	adminGroup.Post("/:assignment_id/cancel-termination", businessAdminHandler.CancelTermination)
	adminGroup.Get("/:assignment_id/permissions", businessAdminHandler.GetPermissions)
	adminGroup.Patch("/:assignment_id/permissions", businessAdminHandler.UpdatePermissions)

	// Candidate admin personal actions (invitations, accept/decline, resignation).
	workerGroup := api.Group("/my-jobs", middlewares.RequireAuth, middlewares.AllowRoles("business_admin"))
	workerGroup.Get("/", businessAdminHandler.GetMyAssignments)
	workerGroup.Post("/:assignment_id/respond", businessAdminHandler.RespondToInvitation)
	workerGroup.Post("/:assignment_id/resign", businessAdminHandler.ResignFromBusiness)
	workerGroup.Post("/:assignment_id/cancel-resign", businessAdminHandler.CancelTermination)

	// Owner business management.
	ownerGroup := api.Group("/owner", middlewares.RequireAuth, middlewares.AllowRoles("owner"))
	ownerGroup.Get("/admins", businessAdminHandler.GetAdminsByOwner)
	ownerGroup.Get("/admins/:assignment_id/permissions", businessAdminHandler.GetPermissions)
	ownerGroup.Patch("/admins/:assignment_id/permissions", businessAdminHandler.UpdatePermissions)

	// Shared admin and owner access protected by BusinessAccessGuard.
	sharedAccess := middlewares.AllowRoles("owner", "business_admin")

	// Smart dispatcher: public store profile with active promotions for guests, customers, and merchants.
	businessProfileDispatcher := func(c *fiber.Ctx) error {
		role, _ := c.Locals("role").(string)
		accountID, _ := c.Locals("account_id").(string)
		businessID := strings.TrimSpace(c.Params("business_id"))

		if (role == "owner" || role == "business_admin") && accountID != "" && businessID != "" {
			var isOwner bool
			var b models.Business
			if err := database.DB.Select("account_id").Where("business_id = ?", businessID).First(&b).Error; err == nil {
				if b.AccountID == accountID {
					isOwner = true
				}
			}
			if isOwner {
				return businessHandler.GetBusinessDetail(c)
			}
			var adminCount int64
			if err := database.DB.Table("business_admins").
				Where("business_id = ? AND account_id = ? AND status = 'active'", businessID, accountID).
				Count(&adminCount).Error; err == nil && adminCount > 0 {
				return businessHandler.GetBusinessDetail(c)
			}
		}

		// Fallback to public profile with active promotions.
		return businessProfileHandler.GetProfile(c)
	}

	api.Get("/business/:business_id", middlewares.OptionalAuth, businessProfileDispatcher)

	// Business root group (authenticated).
	businessGroup := api.Group("/business", middlewares.RequireAuth)

	// Store list (Owner & Business Admin).
	businessGroup.Get("/", sharedAccess, businessHandler.GetMyBusinesses)

	// Owner-only store creation/deletion (allowed pre-KYC; stores remain private until KYC approval).
	businessGroup.Post("/", middlewares.RequireEmailVerified, middlewares.AllowRoles("owner"), businessHandler.CreateBusiness)
	businessGroup.Delete("/:business_id", middlewares.RequireEmailVerified, middlewares.AllowRoles("owner"), businessHandler.DeleteBusiness)

	// Shared store scope guarded by BusinessAccessGuard.
	sharedBusinessScope := businessGroup.Group("/:business_id", sharedAccess, middlewares.BusinessAccessGuard())
	sharedBusinessScope.Get("/", businessHandler.GetBusinessDetail)
	sharedBusinessScope.Patch("/", businessHandler.UpdateBusiness)
	// Store open toggle requires KYC approval; close/decline is permitted without KYC.
	sharedBusinessScope.Patch("/toggle-status", businessHandler.ToggleStoreStatus)
	sharedBusinessScope.Patch("/schedule-confirmation", businessHandler.ConfirmSchedule)
	// Live streaming strictly requires verified KYC status.
	sharedBusinessScope.Patch("/toggle-live", middlewares.RequireKYC, businessHandler.ToggleLiveStatus)

	// Admin permissions scoped to business.
	sharedBusinessScope.Get("/admins/:admin_id/permissions", businessAdminHandler.GetPermissions)
	sharedBusinessScope.Patch("/admins/:admin_id/permissions", businessAdminHandler.UpdatePermissions)
	sharedBusinessScope.Patch("/admins/:admin_id/notification-toggle", businessAdminHandler.ToggleNotification)

	// Products (static routes registered before :product_id to prevent route shadowing).
	productRoutes := sharedBusinessScope.Group("/products")
	productRoutes.Patch("/bulk-status", productHandler.BulkUpdateStatus)
	productRoutes.Delete("/bulk-delete", productHandler.BulkDelete)
	productRoutes.Post("/", productHandler.CreateProduct)
	productRoutes.Get("/", productHandler.GetFlatProducts)
	productRoutes.Get("/:product_id", productHandler.GetProductDetail)
	productRoutes.Put("/:product_id", productHandler.UpdateProduct)
	productRoutes.Delete("/:product_id", productHandler.DeleteProduct)

	// Standalone product packages.
	packageRoutes := sharedBusinessScope.Group("/packages")
	packageRoutes.Post("/calculate", packageHandler.CalculatePackagePrice)
	packageRoutes.Post("/", packageHandler.CreatePackage)
	packageRoutes.Get("/", packageHandler.GetPackages)
	packageRoutes.Get("/:package_id", packageHandler.GetPackageDetail)
	packageRoutes.Put("/:package_id", packageHandler.UpdatePackage)
	packageRoutes.Delete("/:package_id", packageHandler.DeletePackage)

	// Product groups.
	productGroupRoutes := sharedBusinessScope.Group("/product-groups")
	productGroupRoutes.Patch("/reorder", productGroupHandler.ReorderGroups)
	productGroupRoutes.Patch("/:product_group_id/contents", productGroupHandler.UpdateContents)
	productGroupRoutes.Post("/", productGroupHandler.CreateGroup)
	productGroupRoutes.Get("/", productGroupHandler.GetGroups)
	productGroupRoutes.Put("/:product_group_id", productGroupHandler.UpdateGroup)
	productGroupRoutes.Delete("/:product_group_id", productGroupHandler.DeleteGroup)

	// Merchant posts (management allowed pre-KYC; hidden from public feeds until KYC approval).
	postRoutes := sharedBusinessScope.Group("/posts")
	postRoutes.Post("/", businessPostHandler.CreatePost)
	postRoutes.Get("/", businessPostHandler.GetPosts)
	postRoutes.Get("/:post_id", businessPostHandler.GetPostDetail)
	postRoutes.Patch("/:post_id", businessPostHandler.UpdatePost)
	postRoutes.Delete("/:post_id", businessPostHandler.DeletePost)
	postRoutes.Patch("/:post_id/promotion/stop", businessPostHandler.StopPostPromotion)
	postRoutes.Patch("/:post_id/promotion/extend", businessPostHandler.ExtendPostPromotion)

	// Merchant promotions (management allowed pre-KYC; public broadcast requires KYC approval).
	promoRoutes := sharedBusinessScope.Group("/promotions")
	promoRoutes.Get("/eligible-products", promotionHandler.GetEligibleProducts)
	promoRoutes.Post("/", promotionHandler.CreatePromotion)
	promoRoutes.Get("/", promotionHandler.GetPromotions)
	promoRoutes.Get("/:promotion_id", promotionHandler.GetPromotionDetail)
	promoRoutes.Patch("/:promotion_id/extend", promotionHandler.ExtendPromotion)
	promoRoutes.Patch("/:promotion_id/stop", promotionHandler.StopPromotion)
	promoRoutes.Post("/:promotion_id/push", middlewares.RequireKYC, promotionHandler.PushPromotionNotification)

	// Demand Signal V2 merchant routes
	merchantSignalGroup := api.Group("/merchant/signals", middlewares.RequireAuth, sharedAccess)
	merchantSignalGroup.Get("/explore", merchantSignalHandler.ExploreSignals)
	merchantSignalGroup.Get("/:signal_id", merchantSignalHandler.GetSignalDetail)
	merchantSignalGroup.Get("/:signal_id/product-suggestions", merchantSignalHandler.GetProductSuggestions)
	merchantSignalGroup.Post("/:signal_id/reply", merchantSignalHandler.ReplyToSignal)

	merchantConvGroup := api.Group("/merchant/conversations", middlewares.RequireAuth, sharedAccess)
	merchantConvGroup.Get("/", merchantSignalHandler.GetMyConversations)
	merchantConvGroup.Get("/:conversation_id/messages", merchantSignalHandler.GetMessages)
	merchantConvGroup.Post("/:conversation_id/messages", merchantSignalHandler.SendMessage)

	// Analytics.
	analyticsRoutes := sharedBusinessScope.Group("/analytics")
	analyticsRoutes.Get("/daily", analyticsHandler.GetDailyAnalytics)
	analyticsRoutes.Get("/history", analyticsHandler.GetHistoryAnalytics)
	analyticsRoutes.Get("/explore", analyticsHandler.GetExploreAnalytics)

	// Business review routes
	reviewGroup := api.Group("/reviews")
	reviewGroup.Post("/business/:business_id", middlewares.RequireAuth, middlewares.RequireEmailVerified, reviewHandler.CreateReview)
	reviewGroup.Put("/business/:review_id", middlewares.RequireAuth, middlewares.RequireEmailVerified, reviewHandler.UpdateReview)
	reviewGroup.Delete("/business/:review_id", middlewares.RequireAuth, middlewares.RequireEmailVerified, reviewHandler.DeleteReview)
	reviewGroup.Get("/business/:business_id/smart", middlewares.OptionalAuth, reviewHandler.GetSmartDisplayReviews)
	reviewGroup.Get("/business/:business_id", middlewares.OptionalAuth, reviewHandler.GetAllReviews)
	reviewGroup.Get("/business/detail/:review_id", middlewares.OptionalAuth, reviewHandler.GetReviewDetail)

	// Customer explore & discovery routes
	customerExploreGroup := api.Group("/customer/explore")
	customerExploreGroup.Get("/businesses", middlewares.OptionalAuth, exploreHandler.SearchBusinesses)
	customerExploreGroup.Post("/search-click", middlewares.OptionalAuth, exploreHandler.RecordSearchClick)
	customerExploreGroup.Get("/analytics/zero-results", middlewares.RequireAuth, middlewares.AllowRoles("platform_admin", "super_admin"), exploreHandler.GetZeroResultSearches)
	customerExploreGroup.Get("/synonyms", exploreHandler.ListSearchSynonyms)
	customerExploreGroup.Post("/synonyms", middlewares.RequireAuth, middlewares.AllowRoles("platform_admin", "super_admin"), exploreHandler.UpsertSearchSynonym)
	customerExploreGroup.Get("/products", func(c *fiber.Ctx) error {
		if c.Query("scope") == "" {
			c.Request().URI().QueryArgs().Set("scope", "product")
		}
		return exploreHandler.SearchBusinesses(c)
	})
	customerExploreGroup.Get("/timeline", middlewares.OptionalAuth, exploreHandler.GetTimeline)
	customerExploreGroup.Post("/feed/events", middlewares.OptionalAuth, exploreHandler.RecordFeedEvents)
	customerExploreGroup.Post("/sync", middlewares.RequireAuth, middlewares.AllowRoles("platform_admin", "super_admin"), func(c *fiber.Ctx) error {
		meilisearch.InitMeilisearchIndices()
		_ = meilisearch.SyncSynonymsFromDB(database.DB)
		count, err := meilisearch.SyncAllBusinesses(database.DB)
		if err != nil {
			log.Printf("Meilisearch sync error: %v", err)
			return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal sinkronisasi Meilisearch", nil)
		}
		return utils.SendResponse(c, fiber.StatusOK, constants.Success, fmt.Sprintf("Berhasil menyinkronkan %d bisnis ke Meilisearch", count), fiber.Map{
			"total_synced": count,
		})
	})

	// Internal reconciliation hook for immediate store visibility update on KYC status change.
	api.Post("/internal/kyc/:account_id/sync", middlewares.RequireInternalSecret, func(c *fiber.Ctx) error {
		accountID := strings.TrimSpace(c.Params("account_id"))
		if accountID == "" {
			return utils.SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, "account_id wajib diisi", nil)
		}
		syncedCount, err := meilisearch.SyncOwnerBusinesses(database.DB, accountID)
		if err != nil {
			log.Printf("Internal KYC sync error for account %s: %v", accountID, err)
			return utils.SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, "Gagal menyinkronkan bisnis owner", nil)
		}
		if locationService != nil {
			go func() {
				_ = locationService.SyncMerchantLocationsFromDB()
			}()
		}
		return utils.SendResponse(c, fiber.StatusOK, constants.Success, "Sinkronisasi visibilitas KYC berhasil", fiber.Map{
			"account_id":        accountID,
			"synced_businesses": syncedCount,
		})
	})

	// Batch impressions telemetry.
	api.Post("/customer/analytics/impressions", analyticsHandler.TrackImpressions)

	// Customer promotion tracking and details.
	api.Post("/customer/promotions/events", middlewares.OptionalAuth, customerPromotionHandler.RecordEvents)
	api.Get("/customer/promotions/:promotion_id", middlewares.OptionalAuth, customerPromotionHandler.GetPromotionDetail)

	// Live location routes
	locationGroup := api.Group("/location")
	locationGroup.Get("/nearby", middlewares.OptionalAuth, locationHTTPHandler.GetNearbyMerchants)
	locationGroup.Get("/track/:merchant_id", middlewares.OptionalAuth, locationHTTPHandler.TrackMerchant)
	locationGroup.Post("/update", middlewares.RequireAuth, locationHTTPHandler.UpdateLocationHTTP)

	// Customer profile routes
	api.Get("/customer/profile/:account_id/summary", middlewares.OptionalAuth, publicProfileHandler.GetPublicProfileSummary)

	// Customer support live chat routes
	supportGroup := api.Group("/support/chat", middlewares.RequireAuth, middlewares.RequireEmailVerified)
	supportGroup.Post("/initiate", supportChatHandler.InitiateChat)
	supportGroup.Get("/active", supportChatHandler.GetActiveChat)
	supportGroup.Get("/:session_id/messages", supportChatHandler.GetMessages)
	supportGroup.Post("/:session_id/messages", supportChatHandler.SendMessage)
	supportGroup.Post("/:session_id/end", supportChatHandler.EndChat)
	supportGroup.Post("/:session_id/read", supportChatHandler.MarkAsRead)

	// Support ticket routes
	ticketGroup := api.Group("/support/tickets", middlewares.RequireAuth, middlewares.RequireEmailVerified)
	ticketGroup.Post("/", supportTicketHandler.CreateTicket)
	ticketGroup.Get("/", supportTicketHandler.GetMyTickets)
	ticketGroup.Get("/:ticket_id", supportTicketHandler.GetTicketDetail)

	// User feedback routes
	feedbackGroup := api.Group("/support/feedback", middlewares.RequireAuth)
	feedbackGroup.Post("/", feedbackHandler.SendFeedback)

		// Direct business chat routes
	customerBizChat := api.Group("/customer/business-chat", middlewares.RequireAuth, middlewares.RequireEmailVerified)
	customerBizChat.Post("/:business_id/initiate", businessChatHandler.InitiateChat)
	customerBizChat.Get("/conversations", businessChatHandler.GetCustomerConversations)

	merchantBizChat := api.Group("/merchant/business-chat", middlewares.RequireAuth, middlewares.RequireEmailVerified)
	merchantBizChat.Get("/:business_id/conversations", businessChatHandler.GetMerchantConversations)

	sharedBizChat := api.Group("/business-chat", middlewares.RequireAuth, middlewares.RequireEmailVerified)
	sharedBizChat.Get("/:conversation_id/messages", businessChatHandler.GetMessages)
	sharedBizChat.Post("/:conversation_id/messages", businessChatHandler.SendMessage)

		// Store ownership transfer routes
	bizTransferGroup := api.Group("/merchant/businesses/:business_id/transfer", middlewares.RequireAuth, middlewares.AllowRoles("owner"), middlewares.RequireKYC)
	bizTransferGroup.Post("/check-target", businessTransferHandler.CheckTarget)
	bizTransferGroup.Post("/initiate", businessTransferHandler.InitiateTransfer)
	bizTransferGroup.Post("/verify-otp", businessTransferHandler.VerifyOTP)
	bizTransferGroup.Post("/cancel", businessTransferHandler.CancelTransfer)

	incomingTransferGroup := api.Group("/merchant/transfers", middlewares.RequireAuth, middlewares.AllowRoles("owner"))
	incomingTransferGroup.Get("/incoming", businessTransferHandler.GetIncomingTransfers)
	incomingTransferGroup.Post("/:transfer_id/respond", middlewares.RequireKYC, businessTransferHandler.RespondTransfer)

	// Unmatched route 404 fallback.
	app.Use(func(c *fiber.Ctx) error {
		return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, "Rute tidak ditemukan", nil)
	})

		// 4. Start Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Println("Server is running on port", port)
	log.Fatal(app.Listen(":" + port))
}
