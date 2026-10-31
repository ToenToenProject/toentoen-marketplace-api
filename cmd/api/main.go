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
	// ==========================================
	// 1. INIT CONFIG & DATABASE
	// ==========================================
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	config.InitConfig()
	database.ConnectDB()
	config.ConnectRedis()
	meilisearch.ConnectMeilisearch()
	storage.InitS3Client()
	config.InitFirebase()

	// 🔍 Auto-sync PostgreSQL businesses & products to Meilisearch on startup
	go func() {
		meilisearch.InitMeilisearchIndices()
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
	}()

	// WebSocket Hub (Real-Time)
	wsHub := realtime.NewHub(config.RDB)
	go wsHub.Run()

	// 📍 INIT LIVE LOCATION
	locationStore := location.NewRedisLocationStore(config.RDB)
	locationService := location.NewLocationService(locationStore, wsHub, database.DB)
	locationWSHandler := location.NewWSLocationHandler(locationService)
	locationHTTPHandler := location.NewHTTPLocationHandler(locationService)

	// Daftarkan WS handler untuk Live Location
	wsHub.RegisterMessageHandler("location", locationWSHandler)

	// Sinkronisasi lokasi merchant dari PostgreSQL ke Redis setiap 15 menit
	locationService.StartPeriodicSync(15 * time.Minute)

	// ==========================================
	// 2. INIT REPO, SERVICE, & HANDLER
	// ==========================================
	authRepo := auth.NewAuthRepository(database.DB)
	authService := auth.NewAuthService(authRepo)
	authHandler := auth.NewAuthHandler(authService)

	profileRepo := account.NewProfileRepository(database.DB)
	profileService := account.NewProfileService(profileRepo)
	profileHandler := account.NewProfileHandler(profileService)

	// 🔔 INIT NOTIFICATION SERVICE
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

	// 🪄 INIT BUSINESS ADMIN (BARU)
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

	// 🎁 INIT PROMOTIONS
	promotionRepo := merchant_promotion.NewPromotionRepository(database.DB)
	promotionService := merchant_promotion.NewPromotionService(promotionRepo, notifService)
	promotionHandler := merchant_promotion.NewPromotionHandler(promotionService)
	customerPromotionHandler := customer_promotion.NewCustomerPromotionHandler(promotionService, promotionRepo)
	merchant_promotion.StartPromotionWorker(database.DB, notifService)

	businessPostRepo := post.NewBusinessPostRepository(database.DB)
	businessPostService := post.NewBusinessPostService(businessPostRepo, productRepo, promotionRepo, notifService)
	businessPostHandler := post.NewBusinessPostHandler(businessPostService)

	// Signal V2 (menggantikan demand lama) — dengan Real-Time Hub & Notification Service
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

	// ==========================================
	// 3. FIBER APP SETUP
	// ==========================================
	app := fiber.New(fiber.Config{
		ProxyHeader: fiber.HeaderXForwardedFor,
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

	// Fallback jika FRONTEND_URL kosong
	if corsConfig.AllowOrigins == "" {
		corsConfig.AllowOrigins = "*"
		// Fiber v2 akan PANIC jika AllowOrigins="*" dan AllowCredentials=true
		corsConfig.AllowCredentials = false
	}

	app.Use(cors.New(corsConfig))

	// Helper untuk mengekstrak Real Client IP di balik Reverse Proxy / Cloudflare / Load Balancer
	getClientIP := func(c *fiber.Ctx) string {
		if cf := c.Get("CF-Connecting-IP"); cf != "" {
			return cf
		}
		if xff := c.Get("X-Forwarded-For"); xff != "" {
			ips := strings.Split(xff, ",")
			if len(ips) > 0 && strings.TrimSpace(ips[0]) != "" {
				return strings.TrimSpace(ips[0])
			}
		}
		if xri := c.Get("X-Real-IP"); xri != "" {
			return xri
		}
		return c.IP()
	}

	// Rate Limiting Global
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
			// Buka bypass jika disetel di environment
			if os.Getenv("ENABLE_RATE_LIMITER") == "false" {
				return true
			}
			// Lewati preflight OPTIONS agar request CORS browser tidak menghabiskan kuota
			if c.Method() == fiber.MethodOptions {
				return true
			}
			// Lewati request yang BUKAN rute resmi /api/v1 (seperti scanner bot internet untuk .env, phpinfo, wp-config, dll.)
			// agar serangan bot tidak menghabiskan kuota rate limit aplikasi!
			if !strings.HasPrefix(c.Path(), "/api/v1") {
				return true
			}
			return false
		},
		KeyGenerator: getClientIP,
		LimitReached: func(c *fiber.Ctx) error {
			clientIP := getClientIP(c)
			log.Printf("⚠️ [RateLimiter Global 429] IP: %s terblokir di %s %s", clientIP, c.Method(), c.Path())
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"responseCode": "429",
				"message":      "Terlalu banyak request. Silakan coba lagi nanti.",
			})
		},
	}))

	// Static Assets (Logos, public media)
	app.Static("/assets", "./assets")

	// ==========================================
	// ROUTES: WEBHOOKS (Meta WhatsApp Cloud API)
	// ==========================================
	app.Get("/webhook/whatsapp", webhookHandler.VerifyWhatsAppWebhook)
	app.Post("/webhook/whatsapp", webhookHandler.HandleWhatsAppWebhookEvent)

	api := app.Group("/api/v1")
	api.Get("/webhook/whatsapp", webhookHandler.VerifyWhatsAppWebhook)
	api.Post("/webhook/whatsapp", webhookHandler.HandleWhatsAppWebhookEvent)

	// ==========================================
	// ROUTES: PUBLIC METADATA (Categories, Tags)
	// ==========================================
	metaGroup := api.Group("/meta")
	metaGroup.Get("/categories", metaHandler.GetCategories)
	metaGroup.Get("/tags", metaHandler.GetTags)
	metaGroup.Get("/store-types", metaHandler.GetStoreTypes)
	metaGroup.Get("/occupations", metaHandler.GetOccupations)
	metaGroup.Get("/report-categories", reportHandler.GetReportCategories)

	// ==========================================
	// ROUTE: WEBSOCKET (Real-Time)
	// ==========================================
	api.Get("/ws", realtime.UpgradeCheck, websocket.New(wsHandler.Handle))

	// ==========================================
	// ROUTES: UPLOAD (S3 Presigned URL)
	// ==========================================
	uploadGroup := api.Group("/upload", middlewares.RequireAuth)
	uploadGroup.Post("/presign", uploadHandler.GeneratePresignURL)
	uploadGroup.Get("/limits", uploadHandler.GetUploadLimits)

	// ==========================================
	// ROUTES: AUTHENTICATION
	// ==========================================
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
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"responseCode": "429",
				"message":      "Terlalu banyak percobaan autentikasi. Silakan coba lagi setelah 1 menit.",
			})
		},
	})

	authGroup := api.Group("/auth", authLimiter)
	authGroup.Get("/check-referral", authHandler.CheckReferralCode)
	authGroup.Post("/check-referral", authHandler.CheckReferralCode)
	authGroup.Post("/register", authHandler.Register)
	authGroup.Post("/verify-register-otp", authHandler.VerifyRegisterOTP)
	authGroup.Post("/login", authHandler.Login)
	authGroup.Post("/verify-login-otp", authHandler.VerifyLoginOTP)
	authGroup.Post("/forgot-password", authHandler.ForgotPassword)
	authGroup.Post("/verify-forgot-otp", authHandler.VerifyForgotOTP)
	authGroup.Post("/reset-password", authHandler.ResetPassword)
	authGroup.Post("/password/change", middlewares.RequireAuth, authHandler.ChangePassword)
	authGroup.Put("/password/change", middlewares.RequireAuth, authHandler.ChangePassword)
	authGroup.Patch("/password/change", middlewares.RequireAuth, authHandler.ChangePassword)
	authGroup.Post("/change-password", middlewares.RequireAuth, authHandler.ChangePassword)
	authGroup.Put("/change-password", middlewares.RequireAuth, authHandler.ChangePassword)
	authGroup.Patch("/change-password", middlewares.RequireAuth, authHandler.ChangePassword)
	authGroup.Post("/email/change/request", middlewares.RequireAuth, authHandler.RequestEmailChange)
	authGroup.Post("/email/change/verify", middlewares.RequireAuth, authHandler.VerifyEmailChange)
	authGroup.Post("/phone/change/request", middlewares.RequireAuth, authHandler.RequestPhoneChange)
	authGroup.Post("/phone/change/verify", middlewares.RequireAuth, authHandler.VerifyPhoneChange)
	authGroup.Post("/email/send-otp", authHandler.SendEmailOTP)
	authGroup.Post("/email/verify-otp", authHandler.VerifyEmailOTP)
	authGroup.Post("/phone/send-otp", authHandler.SendPhoneOTP)
	authGroup.Post("/phone/verify-otp", authHandler.VerifyPhoneOTP)

	// ==========================================
	// ROUTES: CUSTOMER BUSINESS PROFILE & BOOKMARKS
	// ==========================================
	customerBusinessGroup := api.Group("/customer/businesses")
	customerBusinessGroup.Get("/bookmarks", middlewares.OptionalAuth, businessProfileHandler.GetBookmarks)
	customerBusinessGroup.Get("/:business_id/profile", middlewares.OptionalAuth, businessProfileHandler.GetProfile)
	customerBusinessGroup.Get("/:business_id", middlewares.OptionalAuth, businessProfileHandler.GetProfile)
	customerBusinessGroup.Post("/:business_id/bookmark", middlewares.RequireAuth, middlewares.RequireEmailVerified, businessProfileHandler.ToggleBookmark)
	customerBusinessGroup.Post("/:business_id/like", middlewares.RequireAuth, middlewares.RequireEmailVerified, businessProfileHandler.ToggleLike)
	customerBusinessGroup.Post("/:business_id/report", middlewares.RequireAuth, middlewares.RequireEmailVerified, reportHandler.CreateReport)

	// Products
	customerBusinessGroup.Get("/:business_id/product-groups", businessProfileHandler.GetProductGroups)
	customerBusinessGroup.Get("/:business_id/products", businessProfileHandler.GetProducts)
	customerBusinessGroup.Get("/:business_id/products/", businessProfileHandler.GetProducts)
	customerBusinessGroup.Get("/:business_id/product-reviews/summary", productInteractionHandler.GetBusinessProductReviewsSummary)
	customerBusinessGroup.Get("/:business_id/posts", middlewares.OptionalAuth, businessProfileHandler.GetPosts)

	// Direct aliases for customer business products
	api.Get("/customer/businesses/:business_id/products", businessProfileHandler.GetProducts)
	api.Get("/customer/businesses/:business_id/products/", businessProfileHandler.GetProducts)
	app.Get("/customer/businesses/:business_id/products", businessProfileHandler.GetProducts)
	app.Get("/customer/businesses/:business_id/products/", businessProfileHandler.GetProducts)

	// Business Like endpoints & alias routes
	api.Post("/business/:id/like", middlewares.RequireAuth, middlewares.RequireEmailVerified, businessProfileHandler.ToggleLike)
	api.Post("/business/:business_id/like", middlewares.RequireAuth, middlewares.RequireEmailVerified, businessProfileHandler.ToggleLike)
	api.Post("/customer/business/:id/like", middlewares.RequireAuth, middlewares.RequireEmailVerified, businessProfileHandler.ToggleLike)
	api.Post("/customer/business/:business_id/like", middlewares.RequireAuth, middlewares.RequireEmailVerified, businessProfileHandler.ToggleLike)

	// ==========================================
	// ROUTES: USER PROFILE
	// ==========================================
	protected := api.Group("/account", middlewares.RequireAuth)
	protected.Get("/profile/me", profileHandler.GetMyProfile)
	protected.Get("/kyc/status", profileHandler.GetKYCStatus)
	protected.Post("/kyc/submit", middlewares.RequireEmailVerified, profileHandler.SubmitKYC)
	protected.Put("/kyc/submit", middlewares.RequireEmailVerified, profileHandler.SubmitKYC)
	protected.Post("/kyc/consent", middlewares.RequireEmailVerified, profileHandler.SubmitConsent)
	protected.Put("/kyc/consent", middlewares.RequireEmailVerified, profileHandler.SubmitConsent)
	protected.Post("/consent", middlewares.RequireEmailVerified, profileHandler.SubmitConsent)
	protected.Put("/consent", middlewares.RequireEmailVerified, profileHandler.SubmitConsent)

	// Direct alias for KYC consent
	api.Post("/kyc/consent", middlewares.RequireAuth, middlewares.RequireEmailVerified, profileHandler.SubmitConsent)
	api.Put("/kyc/consent", middlewares.RequireAuth, middlewares.RequireEmailVerified, profileHandler.SubmitConsent)
	protected.Put("/profile/customer", middlewares.AllowRoles("customer"), profileHandler.UpdateProfileCustomer)
	protected.Patch("/profile/customer", middlewares.AllowRoles("customer"), profileHandler.UpdateProfileCustomer)
	protected.Post("/profile/customer", middlewares.AllowRoles("customer"), profileHandler.UpdateProfileCustomer)
	protected.Put("/profile/owner", middlewares.AllowRoles("owner"), profileHandler.UpdateProfileOwner)
	protected.Patch("/profile/owner", middlewares.AllowRoles("owner"), profileHandler.UpdateProfileOwner)
	protected.Post("/profile/owner", middlewares.AllowRoles("owner"), profileHandler.UpdateProfileOwner)
	protected.Put("/profile/admin", middlewares.AllowRoles("business_admin"), profileHandler.UpdateProfileBusinessAdmin)
	protected.Patch("/profile/admin", middlewares.AllowRoles("business_admin"), profileHandler.UpdateProfileBusinessAdmin)
	protected.Post("/profile/admin", middlewares.AllowRoles("business_admin"), profileHandler.UpdateProfileBusinessAdmin)
	protected.Post("/password/change", authHandler.ChangePassword)
	protected.Put("/password/change", authHandler.ChangePassword)
	protected.Patch("/password/change", authHandler.ChangePassword)
	protected.Post("/change-password", authHandler.ChangePassword)
	protected.Put("/change-password", authHandler.ChangePassword)
	protected.Patch("/change-password", authHandler.ChangePassword)
	protected.Put("/password", authHandler.ChangePassword)
	protected.Patch("/password", authHandler.ChangePassword)
	protected.Post("/email/change/request", authHandler.RequestEmailChange)
	protected.Post("/email/change/verify", authHandler.VerifyEmailChange)
	protected.Post("/phone/change/request", authHandler.RequestPhoneChange)
	protected.Post("/phone/change/verify", authHandler.VerifyPhoneChange)
	protected.Post("/delete/request", profileHandler.RequestDeleteOTP)
	protected.Delete("/delete/execute", profileHandler.VerifyDeleteOTPAndExecute)
	protected.Post("/delete/execute", profileHandler.VerifyDeleteOTPAndExecute)
	protected.Post("/delete/verify", profileHandler.VerifyDeleteOTPAndExecute)

	// Device Token (Push Notification FCM Registration & Removal)
	protected.Post("/device-token", profileHandler.RegisterDeviceToken)
	protected.Put("/device-token", profileHandler.RegisterDeviceToken)
	protected.Delete("/device-token", profileHandler.DeleteDeviceToken)
	api.Post("/account/device-token", middlewares.RequireAuth, profileHandler.RegisterDeviceToken)
	api.Delete("/account/device-token", middlewares.RequireAuth, profileHandler.DeleteDeviceToken)

	// Notifications (/account/notifications and alias /notifications)
	notifGroup := protected.Group("/notifications")
	notifGroup.Get("/", notifHandler.GetMyNotifications)
	notifGroup.Get("/unread-count", notifHandler.GetUnreadCount)
	notifGroup.Put("/read-all", notifHandler.MarkAllAsRead)
	notifGroup.Put("/:id/read", notifHandler.MarkAsRead)
	notifGroup.Delete("/clear-all", notifHandler.DeleteAllNotifications)
	notifGroup.Delete("/:id", notifHandler.DeleteNotification)
	notifGroup.Delete("/", notifHandler.DeleteAllNotifications)

	directNotifGroup := api.Group("/notifications", middlewares.RequireAuth)
	directNotifGroup.Get("/", notifHandler.GetMyNotifications)
	directNotifGroup.Get("/unread-count", notifHandler.GetUnreadCount)
	directNotifGroup.Put("/read-all", notifHandler.MarkAllAsRead)
	directNotifGroup.Put("/:id/read", notifHandler.MarkAsRead)
	directNotifGroup.Delete("/clear-all", notifHandler.DeleteAllNotifications)
	directNotifGroup.Delete("/:id", notifHandler.DeleteNotification)
	directNotifGroup.Delete("/", notifHandler.DeleteAllNotifications)

	// Endpoint Internal untuk Service toentoen-internal-operations (Trigger WebSocket & Push Notification)
	api.Post("/internal/notifications", notifHandler.SendInternalNotification)

	// Endpoint Internal / Dev Tools: Force Delete Account (Menghapus akun langsung tanpa OTP untuk kemudahan testing)
	api.Post("/internal/account/force-delete", profileHandler.ForceDeleteAccount)
	api.Delete("/internal/account/force-delete", profileHandler.ForceDeleteAccount)
	api.Delete("/internal/account/force-delete/:identifier", profileHandler.ForceDeleteAccount)

	// My Activity
	myActivityGroup := protected.Group("/my-activity")
	myActivityGroup.Get("/summary", myActivityHandler.GetMyProfileSummary)
	myActivityGroup.Get("/comments", myActivityHandler.GetMyComments)
	myActivityGroup.Get("/reviews", myActivityHandler.GetMyBusinessReviews)
	myActivityGroup.Get("/bookmarks", myActivityHandler.GetMyBookmarks)
	myActivityGroup.Get("/wishlist", myActivityHandler.GetMyWishlists)

	api.Post("/customer/wishlist/:product_id", middlewares.RequireAuth, middlewares.RequireEmailVerified, myActivityHandler.ToggleWishlist)
	api.Post("/customer/products/:product_id/wishlist", middlewares.RequireAuth, middlewares.RequireEmailVerified, myActivityHandler.ToggleWishlist)
	api.Get("/customer/wishlist", middlewares.RequireAuth, middlewares.RequireEmailVerified, myActivityHandler.GetMyWishlists)
	api.Get("/customer/bookmarks", middlewares.OptionalAuth, businessProfileHandler.GetBookmarks)
	app.Get("/customer/bookmarks", middlewares.OptionalAuth, businessProfileHandler.GetBookmarks)

	// ==========================================
	// ROUTES: DEMAND SIGNAL V2 - CUSTOMER
	// ==========================================
	customerSignalGroup := api.Group("/customer/signals", middlewares.RequireAuth, middlewares.RequireEmailVerified)
	customerSignalGroup.Post("/", customerSignalHandler.CreateSignal)
	customerSignalGroup.Get("/me", customerSignalHandler.GetMySignals)
	customerSignalGroup.Get("/:signal_id", customerSignalHandler.GetSignalDetail)
	customerSignalGroup.Put("/:signal_id", customerSignalHandler.UpdateSignal)
	customerSignalGroup.Delete("/:signal_id", customerSignalHandler.DeleteSignal)
	customerSignalGroup.Put("/:signal_id/close", customerSignalHandler.CloseSignal)
	customerSignalGroup.Post("/:signal_id/close", customerSignalHandler.CloseSignal)
	customerSignalGroup.Patch("/:signal_id/close", customerSignalHandler.CloseSignal)
	customerSignalGroup.Put("/:signal_id/complete", customerSignalHandler.CompleteSignal)
	customerSignalGroup.Post("/:signal_id/complete", customerSignalHandler.CompleteSignal)
	customerSignalGroup.Patch("/:signal_id/complete", customerSignalHandler.CompleteSignal)

	customerConvGroup := api.Group("/customer/conversations", middlewares.RequireAuth, middlewares.RequireEmailVerified)
	customerConvGroup.Get("/", customerSignalHandler.GetMyConversations)
	customerConvGroup.Get("/:conversation_id/messages", customerSignalHandler.GetMessages)
	customerConvGroup.Post("/:conversation_id/messages", customerSignalHandler.SendMessage)

	// 4. UNIFIED ROUTES (UNIVERSAL REPORTS, LIKES, COMMENTS, REVIEWS)
	api.Get("/reports/categories", reportHandler.GetReportCategories)
	reportsGroup := api.Group("/reports", middlewares.RequireAuth, middlewares.RequireEmailVerified)
	reportsGroup.Post("/", reportHandler.CreateReport)
	reportsGroup.Get("/", reportHandler.GetAllReports)
	reportsGroup.Get("/my", reportHandler.GetMyReports)
	reportsGroup.Get("/:report_id", reportHandler.GetReportDetail)
	reportsGroup.Delete("/:report_id", reportHandler.DeleteReport)
	reportsGroup.Patch("/:report_id/status", reportHandler.UpdateReportStatus)

	commentsGroup := api.Group("/comments", middlewares.RequireAuth, middlewares.RequireEmailVerified)
	commentsGroup.Post("/:comment_id/like", productInteractionHandler.ToggleUnifiedCommentLike)
	commentsGroup.Delete("/:comment_id", productInteractionHandler.DeleteUnifiedComment)
	commentsGroup.Post("/:comment_id/report", reportHandler.CreateReport)

	reviewsGroup := api.Group("/reviews")
	reviewsGroup.Post("/:review_id/like", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.ToggleUnifiedReviewLike)
	reviewsGroup.Post("/:review_id/replies", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.AddReviewReply)
	reviewsGroup.Post("/:review_id/report", middlewares.RequireAuth, middlewares.RequireEmailVerified, reportHandler.CreateReport)

	businessReviewGroup := api.Group("/business/reviews")
	businessReviewGroup.Post("/:review_id/replies", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.AddReviewReply)
	businessReviewGroup.Post("/:review_id/like", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.ToggleUnifiedReviewLike)
	businessReviewGroup.Delete("/replies/:comment_id", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.DeleteUnifiedComment)
	businessReviewGroup.Post("/:review_id/report", middlewares.RequireAuth, middlewares.RequireEmailVerified, reportHandler.CreateReport)

	publicProductRoutes := api.Group("/products")
	// 1. Detail Produk (dan variasi rute detail)
	publicProductRoutes.Get("/:product_id", middlewares.OptionalAuth, productDetailHandler.GetProductDetail)
	publicProductRoutes.Get("/:product_id/detail", middlewares.OptionalAuth, productDetailHandler.GetProductDetail)
	publicProductRoutes.Get("/:product_id/detail/", middlewares.OptionalAuth, productDetailHandler.GetProductDetail)

	// Aliases untuk customer product detail
	api.Get("/customer/products/:product_id", middlewares.OptionalAuth, productDetailHandler.GetProductDetail)
	api.Get("/customer/products/:product_id/detail", middlewares.OptionalAuth, productDetailHandler.GetProductDetail)
	api.Get("/customer/products/:product_id/", middlewares.OptionalAuth, productDetailHandler.GetProductDetail)
	api.Get("/customer/products/:product_id/detail/", middlewares.OptionalAuth, productDetailHandler.GetProductDetail)
	api.Get("/products/:product_id/detail", middlewares.OptionalAuth, productDetailHandler.GetProductDetail)
	api.Get("/products/:product_id/detail/", middlewares.OptionalAuth, productDetailHandler.GetProductDetail)

	// Direct app aliases (tanpa /api/v1)
	app.Get("/products/:product_id", middlewares.OptionalAuth, productDetailHandler.GetProductDetail)
	app.Get("/products/:product_id/detail", middlewares.OptionalAuth, productDetailHandler.GetProductDetail)
	app.Get("/customer/products/:product_id", middlewares.OptionalAuth, productDetailHandler.GetProductDetail)
	app.Get("/customer/products/:product_id/detail", middlewares.OptionalAuth, productDetailHandler.GetProductDetail)

	// 2. Review Produk (Ulasan Berbintang 1-5 + Min 10 Kata)
	publicProductRoutes.Post("/:product_id/reviews", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.AddProductReview)
	publicProductRoutes.Get("/:product_id/reviews", middlewares.OptionalAuth, productInteractionHandler.GetProductReviews)
	publicProductRoutes.Get("/:product_id/reviews/smart", middlewares.OptionalAuth, productInteractionHandler.GetSmartDisplayProductReviews)
	publicProductRoutes.Get("/reviews/detail/:review_id", middlewares.OptionalAuth, productInteractionHandler.GetProductReviewDetail)
	publicProductRoutes.Put("/reviews/:review_id", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.UpdateProductReview)
	publicProductRoutes.Delete("/reviews/:review_id", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.DeleteProductReview)
	publicProductRoutes.Post("/reviews/:review_id/like", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.ToggleUnifiedReviewLike)
	publicProductRoutes.Post("/reviews/:review_id/replies", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.AddReviewReply)
	publicProductRoutes.Delete("/reviews/replies/:comment_id", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.DeleteReviewReply)
	publicProductRoutes.Post("/reviews/replies/:comment_id/like", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.ToggleUnifiedCommentLike)
	publicProductRoutes.Post("/reviews/:review_id/report", middlewares.RequireAuth, middlewares.RequireEmailVerified, reportHandler.CreateReport)
	publicProductRoutes.Post("/reviews/replies/:comment_id/report", middlewares.RequireAuth, middlewares.RequireEmailVerified, reportHandler.CreateReport)
	publicProductRoutes.Post("/:product_id/comments", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.AddProductComment)
	publicProductRoutes.Put("/:product_id/comments/:comment_id", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.EditProductComment)
	publicProductRoutes.Delete("/:product_id/comments/:comment_id", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.DeleteProductComment)
	publicProductRoutes.Get("/:product_id/comments", middlewares.OptionalAuth, productInteractionHandler.GetProductComments)
	publicProductRoutes.Post("/comments/:comment_id/like", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.ToggleUnifiedCommentLike)
	publicProductRoutes.Post("/comments/:comment_id/report", middlewares.RequireAuth, middlewares.RequireEmailVerified, reportHandler.CreateReport)

	publicPostRoutes := api.Group("/posts")
	publicPostRoutes.Get("/business/:business_id", middlewares.OptionalAuth, businessProfileHandler.GetPosts)
	publicPostRoutes.Get("/business/:id", middlewares.OptionalAuth, businessProfileHandler.GetPosts)
	publicPostRoutes.Get("/business/:businessId", middlewares.OptionalAuth, businessProfileHandler.GetPosts)
	publicPostRoutes.Post("/:post_id/like", middlewares.RequireAuth, middlewares.RequireEmailVerified, postInteractionHandler.ToggleLike)
	publicPostRoutes.Post("/:postId/like", middlewares.RequireAuth, middlewares.RequireEmailVerified, postInteractionHandler.ToggleLike)
	publicPostRoutes.Post("/:post_id/comments", middlewares.RequireAuth, middlewares.RequireEmailVerified, postInteractionHandler.AddPostComment)
	publicPostRoutes.Post("/:postId/comments", middlewares.RequireAuth, middlewares.RequireEmailVerified, postInteractionHandler.AddPostComment)
	publicPostRoutes.Put("/:post_id/comments/:comment_id", middlewares.RequireAuth, middlewares.RequireEmailVerified, postInteractionHandler.EditPostComment)
	publicPostRoutes.Put("/:postId/comments/:comment_id", middlewares.RequireAuth, middlewares.RequireEmailVerified, postInteractionHandler.EditPostComment)
	publicPostRoutes.Delete("/:post_id/comments/:comment_id", middlewares.RequireAuth, middlewares.RequireEmailVerified, postInteractionHandler.DeletePostComment)
	publicPostRoutes.Delete("/:postId/comments/:comment_id", middlewares.RequireAuth, middlewares.RequireEmailVerified, postInteractionHandler.DeletePostComment)
	publicPostRoutes.Get("/:post_id/comments", middlewares.OptionalAuth, postInteractionHandler.GetPostComments)
	publicPostRoutes.Get("/:postId/comments", middlewares.OptionalAuth, postInteractionHandler.GetPostComments)
	publicPostRoutes.Post("/comments/:comment_id/like", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.ToggleUnifiedCommentLike)
	publicPostRoutes.Post("/:post_id/report", middlewares.RequireAuth, middlewares.RequireEmailVerified, reportHandler.CreateReport)
	publicPostRoutes.Post("/:postId/report", middlewares.RequireAuth, middlewares.RequireEmailVerified, reportHandler.CreateReport)

	// ==========================================
	// ROUTES: BUSINESS OWNER / ADMIN ACTIONS
	// ==========================================

	// 🪄 Admin Management (Owner & Supervisor)
	adminGroup := api.Group("/business/:business_id/admins", middlewares.RequireAuth, middlewares.AllowRoles("owner", "business_admin"), middlewares.BusinessAccessGuard())
	adminGroup.Post("/invite", businessAdminHandler.InviteAdmin)
	adminGroup.Post("/assignments/:assignment_id/approve", middlewares.AllowRoles("owner"), businessAdminHandler.RespondOwnerApproval)
	adminGroup.Get("", businessAdminHandler.GetAdminTree)
	adminGroup.Get("/", businessAdminHandler.GetAdminTree)
	adminGroup.Put("/:assignment_id", businessAdminHandler.UpdateAdminAssignment)
	adminGroup.Patch("/:assignment_id", businessAdminHandler.UpdateAdminAssignment)
	adminGroup.Delete("/:assignment_id", businessAdminHandler.TerminateAdmin)
	adminGroup.Post("/:assignment_id/cancel-termination", businessAdminHandler.CancelTermination)
	adminGroup.Get("/:assignment_id/permissions", businessAdminHandler.GetPermissions)
	adminGroup.Put("/:assignment_id/permissions", businessAdminHandler.UpdatePermissions)
	adminGroup.Patch("/:assignment_id/permissions", businessAdminHandler.UpdatePermissions)

	// 🪄 Direct Admin Permissions (Owner & Supervisor by assignment_id)
	adminDirectGroup := api.Group("/admins", middlewares.RequireAuth, middlewares.AllowRoles("owner", "business_admin"))
	adminDirectGroup.Get("/:assignment_id/permissions", businessAdminHandler.GetPermissions)
	adminDirectGroup.Put("/:assignment_id/permissions", businessAdminHandler.UpdatePermissions)
	adminDirectGroup.Patch("/:assignment_id/permissions", businessAdminHandler.UpdatePermissions)

	// 🪄 Rute Personal Calon Admin (Lihat Undangan, Terima/Tolak, Resign)
	workerGroup := api.Group("/my-jobs", middlewares.RequireAuth, middlewares.AllowRoles("business_admin"))
	workerGroup.Get("", businessAdminHandler.GetMyAssignments)
	workerGroup.Get("/", businessAdminHandler.GetMyAssignments)
	workerGroup.Post("/:assignment_id/respond", businessAdminHandler.RespondToInvitation)
	workerGroup.Put("/:assignment_id/respond", businessAdminHandler.RespondToInvitation) // Backward compatibility
	workerGroup.Post("/:assignment_id/resign", businessAdminHandler.ResignFromBusiness)
	workerGroup.Post("/:assignment_id/cancel-resign", businessAdminHandler.CancelTermination)

	// Owner Management (All Businesses)
	ownerGroup := api.Group("/owner", middlewares.RequireAuth, middlewares.AllowRoles("owner"))
	ownerGroup.Get("/admins", businessAdminHandler.GetAdminsByOwner)
	ownerGroup.Get("/admins/:assignment_id/permissions", businessAdminHandler.GetPermissions)
	ownerGroup.Put("/admins/:assignment_id/permissions", businessAdminHandler.UpdatePermissions)
	ownerGroup.Patch("/admins/:assignment_id/permissions", businessAdminHandler.UpdatePermissions)

	// Admin / Owner Shared Access (Products, Posts, Analytics, Business Detail) dengan BusinessAccessGuard
	sharedAccess := middlewares.AllowRoles("owner", "business_admin")

	// Smart Dispatcher: GET /api/v1/business/:business_id (Profil Toko & Promosi Aktif)
	// Memungkinkan customer, guest, dan merchant mengakses data profil/detail toko beserta active_promotions tanpa terhalang 401/403.
	businessProfileDispatcher := func(c *fiber.Ctx) error {
		role, _ := c.Locals("role").(string)
		accountID, _ := c.Locals("account_id").(string)
		businessID := strings.TrimSpace(c.Params("business_id"))
		if businessID == "" {
			businessID = strings.TrimSpace(c.Params("id"))
		}

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

		// Fallback ke profil publik toko (dengan array active_promotions)
		return businessProfileHandler.GetProfile(c)
	}

	api.Get("/business/:business_id", middlewares.OptionalAuth, businessProfileDispatcher)
	api.Get("/business/:business_id/", middlewares.OptionalAuth, businessProfileDispatcher)
	app.Get("/business/:business_id", middlewares.OptionalAuth, businessProfileDispatcher)
	app.Get("/business/:business_id/", middlewares.OptionalAuth, businessProfileDispatcher)

	// Business Root Group (RequireAuth)
	businessGroup := api.Group("/business", middlewares.RequireAuth)

	// Business List (Owner & Business Admin)
	businessGroup.Get("", sharedAccess, businessHandler.GetMyBusinesses)
	businessGroup.Get("/", sharedAccess, businessHandler.GetMyBusinesses)

	// Owner-only Business Creation & Deletion
	businessGroup.Post("", middlewares.RequireEmailVerified, middlewares.AllowRoles("owner"), middlewares.RequireOwnerKYCSubmitted, businessHandler.CreateBusiness)
	businessGroup.Post("/", middlewares.RequireEmailVerified, middlewares.AllowRoles("owner"), middlewares.RequireOwnerKYCSubmitted, businessHandler.CreateBusiness)
	businessGroup.Delete("/:id", middlewares.RequireEmailVerified, middlewares.AllowRoles("owner"), businessHandler.DeleteBusiness)

	// Shared Business Scope (Owner & Business Admin dengan BusinessAccessGuard)
	sharedBusinessScope := businessGroup.Group("/:business_id", sharedAccess, middlewares.BusinessAccessGuard())
	sharedBusinessScope.Get("", businessHandler.GetBusinessDetail)
	sharedBusinessScope.Get("/", businessHandler.GetBusinessDetail)
	sharedBusinessScope.Put("", businessHandler.UpdateBusiness)
	sharedBusinessScope.Put("/", businessHandler.UpdateBusiness)
	sharedBusinessScope.Patch("", businessHandler.UpdateBusiness)
	sharedBusinessScope.Patch("/", businessHandler.UpdateBusiness)

	// Aliases untuk /merchant/businesses dan /merchant/business
	for _, mPrefix := range []string{"/merchant/businesses", "/merchant/business"} {
		mScope := api.Group(mPrefix+"/:business_id", sharedAccess, middlewares.BusinessAccessGuard())
		mScope.Get("", businessHandler.GetBusinessDetail)
		mScope.Get("/", businessHandler.GetBusinessDetail)
		mScope.Put("", businessHandler.UpdateBusiness)
		mScope.Put("/", businessHandler.UpdateBusiness)
		mScope.Patch("", businessHandler.UpdateBusiness)
		mScope.Patch("/", businessHandler.UpdateBusiness)
	}
	sharedBusinessScope.Patch("/toggle-status", middlewares.RequireKYC, businessHandler.ToggleStoreStatus)
	sharedBusinessScope.Post("/schedule-confirmation", middlewares.RequireKYC, businessHandler.ConfirmSchedule)
	sharedBusinessScope.Post("/schedule-confirmation/", middlewares.RequireKYC, businessHandler.ConfirmSchedule)
	sharedBusinessScope.Patch("/schedule-confirmation", middlewares.RequireKYC, businessHandler.ConfirmSchedule)
	sharedBusinessScope.Patch("/schedule-confirmation/", middlewares.RequireKYC, businessHandler.ConfirmSchedule)
	sharedBusinessScope.Patch("/toggle-live", middlewares.RequireKYC, businessHandler.ToggleLiveStatus)
	sharedBusinessScope.Post("/toggle-live", middlewares.RequireKYC, businessHandler.ToggleLiveStatus)

	// -> Admin Permissions (Scoped ke Bisnis)
	sharedBusinessScope.Get("/admins/:admin_id/permissions", businessAdminHandler.GetPermissions)
	sharedBusinessScope.Put("/admins/:admin_id/permissions", businessAdminHandler.UpdatePermissions)
	sharedBusinessScope.Patch("/admins/:admin_id/permissions", businessAdminHandler.UpdatePermissions)
	sharedBusinessScope.Patch("/admins/:admin_id/notification-toggle", businessAdminHandler.ToggleNotification)

	// -> Products
	productRoutes := sharedBusinessScope.Group("/products")
	productRoutes.Post("/", productHandler.CreateProduct)
	productRoutes.Get("/", productHandler.GetFlatProducts)
	productRoutes.Get("/:product_id", productHandler.GetProductDetail)
	productRoutes.Put("/:product_id", productHandler.UpdateProduct)
	productRoutes.Delete("/:product_id", productHandler.DeleteProduct)
	productRoutes.Patch("/bulk-status", productHandler.BulkUpdateStatus)
	productRoutes.Delete("/bulk-delete", productHandler.BulkDelete)

	// -> Packages (Paket Produk / Standalone Package)
	packageRoutes := sharedBusinessScope.Group("/packages")
	packageRoutes.Post("/calculate", packageHandler.CalculatePackagePrice)
	packageRoutes.Post("/", packageHandler.CreatePackage)
	packageRoutes.Get("/", packageHandler.GetPackages)
	packageRoutes.Get("/:package_id", packageHandler.GetPackageDetail)
	packageRoutes.Put("/:package_id", packageHandler.UpdatePackage)
	packageRoutes.Delete("/:package_id", packageHandler.DeletePackage)

	// Alias route /products/packages
	productPackageRoutes := sharedBusinessScope.Group("/products/packages")
	productPackageRoutes.Post("/calculate", packageHandler.CalculatePackagePrice)
	productPackageRoutes.Post("/", packageHandler.CreatePackage)
	productPackageRoutes.Get("/", packageHandler.GetPackages)
	productPackageRoutes.Get("/:package_id", packageHandler.GetPackageDetail)
	productPackageRoutes.Put("/:package_id", packageHandler.UpdatePackage)
	productPackageRoutes.Delete("/:package_id", packageHandler.DeletePackage)

	// -> Product Groups
	productGroupRoutes := sharedBusinessScope.Group("/product-groups")
	productGroupRoutes.Patch("/reorder", productGroupHandler.ReorderGroups)
	productGroupRoutes.Patch("/:product_group_id/contents", productGroupHandler.UpdateContents)
	productGroupRoutes.Post("/", productGroupHandler.CreateGroup)
	productGroupRoutes.Get("/", productGroupHandler.GetGroups)
	productGroupRoutes.Put("/:product_group_id", productGroupHandler.UpdateGroup)
	productGroupRoutes.Delete("/:product_group_id", productGroupHandler.DeleteGroup)

	// -> Posts & Comments
	postRoutes := sharedBusinessScope.Group("/posts")
	postRoutes.Post("/", middlewares.RequireKYC, businessPostHandler.CreatePost)
	postRoutes.Get("/", businessPostHandler.GetPosts)
	postRoutes.Get("/:post_id", businessPostHandler.GetPostDetail)
	postRoutes.Put("/:post_id", businessPostHandler.UpdatePost)
	postRoutes.Patch("/:post_id", businessPostHandler.UpdatePost)
	postRoutes.Delete("/:post_id", businessPostHandler.DeletePost)
	postRoutes.Patch("/:post_id/promotion/stop", middlewares.RequireKYC, businessPostHandler.StopPostPromotion)
	postRoutes.Post("/:post_id/promotion/stop", middlewares.RequireKYC, businessPostHandler.StopPostPromotion)
	postRoutes.Patch("/:post_id/promotion/extend", middlewares.RequireKYC, businessPostHandler.ExtendPostPromotion)
	postRoutes.Post("/:post_id/promotion/extend", middlewares.RequireKYC, businessPostHandler.ExtendPostPromotion)
	postRoutes.Patch("/:post_id/stop-promotion", middlewares.RequireKYC, businessPostHandler.StopPostPromotion)
	postRoutes.Patch("/:post_id/extend-promotion", middlewares.RequireKYC, businessPostHandler.ExtendPostPromotion)

	// -> Promotions (Promosi Merchant)
	promoRoutes := sharedBusinessScope.Group("/promotions")
	promoRoutes.Get("/eligible-products", promotionHandler.GetEligibleProducts)
	promoRoutes.Post("", middlewares.RequireKYC, promotionHandler.CreatePromotion)
	promoRoutes.Post("/", middlewares.RequireKYC, promotionHandler.CreatePromotion)
	promoRoutes.Post("/bundling", middlewares.RequireKYC, promotionHandler.CreateBundlingPromotion)
	promoRoutes.Get("", promotionHandler.GetPromotions)
	promoRoutes.Get("/", promotionHandler.GetPromotions)
	promoRoutes.Get("/:promotion_id", promotionHandler.GetPromotionDetail)
	promoRoutes.Patch("/:promotion_id/extend", middlewares.RequireKYC, promotionHandler.ExtendPromotion)
	promoRoutes.Patch("/:promotion_id/stop", middlewares.RequireKYC, promotionHandler.StopPromotion)

	// ==========================================
	// ROUTES: DEMAND SIGNAL V2 - MERCHANT
	// ==========================================
	merchantSignalGroup := api.Group("/merchant/signals", middlewares.RequireAuth, sharedAccess)
	merchantSignalGroup.Get("/explore", merchantSignalHandler.ExploreSignals)
	merchantSignalGroup.Get("/:signal_id", merchantSignalHandler.GetSignalDetail)
	merchantSignalGroup.Get("/:signal_id/product-suggestions", merchantSignalHandler.GetProductSuggestions)
	merchantSignalGroup.Post("/:signal_id/reply", merchantSignalHandler.ReplyToSignal)

	// Alias route untuk /signals (agar /signals/explore dapat diakses merchant)
	directSignalGroup := api.Group("/signals", middlewares.RequireAuth, sharedAccess)
	directSignalGroup.Get("/explore", merchantSignalHandler.ExploreSignals)
	directSignalGroup.Get("/:signal_id", merchantSignalHandler.GetSignalDetail)
	directSignalGroup.Get("/:signal_id/product-suggestions", merchantSignalHandler.GetProductSuggestions)
	directSignalGroup.Post("/:signal_id/reply", merchantSignalHandler.ReplyToSignal)

	merchantConvGroup := api.Group("/merchant/conversations", middlewares.RequireAuth, sharedAccess)
	merchantConvGroup.Get("/", merchantSignalHandler.GetMyConversations)
	merchantConvGroup.Get("/:conversation_id/messages", merchantSignalHandler.GetMessages)
	merchantConvGroup.Post("/:conversation_id/messages", merchantSignalHandler.SendMessage)

	// -> Analytics
	analyticsRoutes := sharedBusinessScope.Group("/analytics")
	analyticsRoutes.Get("/daily", analyticsHandler.GetDailyAnalytics)
	analyticsRoutes.Get("/history", analyticsHandler.GetHistoryAnalytics)
	analyticsRoutes.Get("/explore", analyticsHandler.GetExploreAnalytics)

	// ==========================================
	// ROUTES: BUSINESS REVIEWS & SMART DISPLAY
	// ==========================================
	reviewGroup := api.Group("/reviews")
	// Review Toko (Customer Buat, Edit, Hapus)
	reviewGroup.Post("/business/:business_id", middlewares.RequireAuth, middlewares.RequireEmailVerified, reviewHandler.CreateReview)
	reviewGroup.Put("/business/reviews/:review_id", middlewares.RequireAuth, middlewares.RequireEmailVerified, reviewHandler.UpdateReview)
	reviewGroup.Delete("/business/reviews/:review_id", middlewares.RequireAuth, middlewares.RequireEmailVerified, reviewHandler.DeleteReview)

	// Get Reviews Toko (Smart display 5 review, Paginated list, Detail)
	reviewGroup.Get("/business/:business_id/smart", middlewares.OptionalAuth, reviewHandler.GetSmartDisplayReviews)
	reviewGroup.Get("/business/:business_id", middlewares.OptionalAuth, reviewHandler.GetAllReviews)
	reviewGroup.Get("/business", middlewares.OptionalAuth, reviewHandler.GetAllReviews)
	reviewGroup.Get("/business/:business_id/product-reviews/summary", productInteractionHandler.GetBusinessProductReviewsSummary)
	reviewGroup.Get("/business/detail/:review_id", middlewares.OptionalAuth, reviewHandler.GetReviewDetail)
	reviewGroup.Get("/business/detail/:id", middlewares.OptionalAuth, reviewHandler.GetReviewDetail)

	// Direct alias
	api.Get("/business/:business_id/product-reviews/summary", productInteractionHandler.GetBusinessProductReviewsSummary)

	// Balasan & Like Review Toko (Legacy Alias)
	reviewGroup.Post("/business/reviews/:review_id/replies", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.AddReviewReply)
	reviewGroup.Delete("/business/replies/:comment_id", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.DeleteUnifiedComment)
	reviewGroup.Post("/business/reviews/:review_id/like", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.ToggleUnifiedReviewLike)
	reviewGroup.Post("/business/replies/:comment_id/like", middlewares.RequireAuth, middlewares.RequireEmailVerified, productInteractionHandler.ToggleUnifiedCommentLike)
	reviewGroup.Post("/business/reviews/:review_id/report", middlewares.RequireAuth, middlewares.RequireEmailVerified, reportHandler.CreateReport)
	reviewGroup.Post("/business/replies/:comment_id/report", middlewares.RequireAuth, middlewares.RequireEmailVerified, reportHandler.CreateReport)

	// ==========================================
	// ROUTES: CUSTOMER EXPLORE & ANALYTICS
	// ==========================================
	customerExploreGroup := api.Group("/customer/explore")
	customerExploreGroup.Get("/bookmarks", middlewares.OptionalAuth, businessProfileHandler.GetBookmarks)
	customerExploreGroup.Get("/business/:id", middlewares.OptionalAuth, businessProfileHandler.GetProfile)
	customerExploreGroup.Get("/business/:business_id", middlewares.OptionalAuth, businessProfileHandler.GetProfile)
	customerExploreGroup.Get("/businesses/:business_id", middlewares.OptionalAuth, businessProfileHandler.GetProfile)
	customerExploreGroup.Get("/businesses/:business_id/profile", middlewares.OptionalAuth, businessProfileHandler.GetProfile)
	customerExploreGroup.Get("/businesses", exploreHandler.SearchBusinesses)
	customerExploreGroup.Get("/search", exploreHandler.SearchBusinesses)

	// Direct business aliases
	api.Get("/customer/business/:id", middlewares.OptionalAuth, businessProfileHandler.GetProfile)
	api.Get("/customer/business/:business_id", middlewares.OptionalAuth, businessProfileHandler.GetProfile)
	api.Get("/business/:id/profile", middlewares.OptionalAuth, businessProfileHandler.GetProfile)
	api.Get("/business/:business_id/profile", middlewares.OptionalAuth, businessProfileHandler.GetProfile)
	customerExploreGroup.Get("/products", func(c *fiber.Ctx) error {
		if c.Query("scope") == "" {
			c.Request().URI().QueryArgs().Set("scope", "product")
		}
		return exploreHandler.SearchBusinesses(c)
	})
	customerExploreGroup.Get("/timeline", middlewares.OptionalAuth, exploreHandler.GetTimeline)
	customerExploreGroup.Get("/feed", middlewares.OptionalAuth, exploreHandler.GetTimeline)
	customerExploreGroup.Post("/feed/events", middlewares.OptionalAuth, exploreHandler.RecordFeedEvents)
	customerExploreGroup.Post("/timeline/events", middlewares.OptionalAuth, exploreHandler.RecordFeedEvents)
	customerExploreGroup.Post("/sync", func(c *fiber.Ctx) error {
		meilisearch.InitMeilisearchIndices()
		count, err := meilisearch.SyncAllBusinesses(database.DB)
		if err != nil {
			return utils.SendResponse(c, 500, "99", fmt.Sprintf("Gagal sinkronisasi Meilisearch: %v", err), nil)
		}
		return utils.SendResponse(c, 200, "00", fmt.Sprintf("Berhasil menyinkronkan %d bisnis ke Meilisearch", count), fiber.Map{
			"total_synced": count,
		})
	})

	// Batch Impressions dari Flutter
	api.Post("/customer/analytics/impressions", analyticsHandler.TrackImpressions)

	// Customer Promotion Tracking & Detail
	api.Post("/customer/promotions/events", middlewares.OptionalAuth, customerPromotionHandler.RecordEvents)
	api.Get("/customer/promotions/:promotion_id", middlewares.OptionalAuth, customerPromotionHandler.GetPromotionDetail)
	api.Get("/customer/promotions/bundling/:promotion_id", middlewares.OptionalAuth, customerPromotionHandler.GetPromotionDetail)

	// ==========================================
	// ROUTES: LIVE LOCATION
	// ==========================================
	locationGroup := api.Group("/location")
	locationGroup.Get("/nearby", middlewares.OptionalAuth, locationHTTPHandler.GetNearbyMerchants)
	locationGroup.Get("/track/:merchant_id", middlewares.OptionalAuth, locationHTTPHandler.TrackMerchant)
	locationGroup.Post("/update", middlewares.RequireAuth, locationHTTPHandler.UpdateLocationHTTP)

	// ==========================================
	// ROUTES: CUSTOMER PROFILE
	// ==========================================
	api.Get("/customer/profile/:account_id/summary", middlewares.OptionalAuth, publicProfileHandler.GetPublicProfileSummary)
	api.Get("/customer/profile/:accountId/summary", middlewares.OptionalAuth, publicProfileHandler.GetPublicProfileSummary)

	// ==========================================
	// ROUTES: SUPPORT LIVE CHAT (Customer Service)
	// ==========================================
	supportGroup := api.Group("/support/chat", middlewares.RequireAuth, middlewares.RequireEmailVerified)
	supportGroup.Post("/initiate", supportChatHandler.InitiateChat)
	supportGroup.Get("/active", supportChatHandler.GetActiveChat)
	supportGroup.Get("/:session_id/messages", supportChatHandler.GetMessages)
	supportGroup.Post("/:session_id/messages", supportChatHandler.SendMessage)
	supportGroup.Post("/:session_id/end", supportChatHandler.EndChat)
	supportGroup.Post("/:session_id/read", supportChatHandler.MarkAsRead)
	supportGroup.Put("/:session_id/read", supportChatHandler.MarkAsRead)
	supportGroup.Post("/:session_id/mark-read", supportChatHandler.MarkAsRead)
	supportGroup.Put("/:session_id/mark-read", supportChatHandler.MarkAsRead)
	// Backward-compatible aliases
	supportGroup.Get("/:room_id/messages", supportChatHandler.GetMessages)
	supportGroup.Post("/:room_id/messages", supportChatHandler.SendMessage)
	supportGroup.Post("/:room_id/end", supportChatHandler.EndChat)
	supportGroup.Post("/:room_id/read", supportChatHandler.MarkAsRead)
	supportGroup.Post("/:room_id/mark-read", supportChatHandler.MarkAsRead)

	// ==========================================
	// ROUTES: SUPPORT TICKETS (Tiket Bantuan)
	// ==========================================
	ticketGroup := api.Group("/support/tickets", middlewares.RequireAuth, middlewares.RequireEmailVerified)
	ticketGroup.Post("/", supportTicketHandler.CreateTicket)
	ticketGroup.Get("/", supportTicketHandler.GetMyTickets)
	ticketGroup.Get("/:ticket_id", supportTicketHandler.GetTicketDetail)

	// ==========================================
	// ROUTES: USER FEEDBACK (Masukan via Email)
	// ==========================================
	feedbackGroup := api.Group("/support/feedback", middlewares.RequireAuth)
	feedbackGroup.Post("/", feedbackHandler.SendFeedback)
	api.Post("/feedback", middlewares.RequireAuth, feedbackHandler.SendFeedback)

	// ==========================================
	// ROUTES: DIRECT BUSINESS CHAT (Chat Toko)
	// ==========================================
	customerBizChat := api.Group("/customer/business-chat", middlewares.RequireAuth, middlewares.RequireEmailVerified)
	customerBizChat.Post("/:business_id/initiate", businessChatHandler.InitiateChat)
	customerBizChat.Get("/conversations", businessChatHandler.GetCustomerConversations)

	merchantBizChat := api.Group("/merchant/business-chat", middlewares.RequireAuth, middlewares.RequireEmailVerified)
	merchantBizChat.Get("/:business_id/conversations", businessChatHandler.GetMerchantConversations)

	sharedBizChat := api.Group("/business-chat", middlewares.RequireAuth, middlewares.RequireEmailVerified)
	sharedBizChat.Get("/:conversation_id/messages", businessChatHandler.GetMessages)
	sharedBizChat.Post("/:conversation_id/messages", businessChatHandler.SendMessage)

	// ==========================================
	// ROUTES: STORE OWNERSHIP TRANSFER (Transfer Bisnis)
	// ==========================================
	bizTransferGroup := api.Group("/merchant/businesses/:business_id/transfer", middlewares.RequireAuth, middlewares.AllowRoles("owner"), middlewares.RequireKYC)
	bizTransferGroup.Post("/check-target", businessTransferHandler.CheckTarget)
	bizTransferGroup.Post("/initiate", businessTransferHandler.InitiateTransfer)
	bizTransferGroup.Post("/verify-otp", businessTransferHandler.VerifyOTP)
	bizTransferGroup.Post("/cancel", businessTransferHandler.CancelTransfer)

	incomingTransferGroup := api.Group("/merchant/transfers", middlewares.RequireAuth, middlewares.AllowRoles("owner"))
	incomingTransferGroup.Get("/incoming", businessTransferHandler.GetIncomingTransfers)
	incomingTransferGroup.Post("/:transfer_id/respond", middlewares.RequireKYC, businessTransferHandler.RespondTransfer)

	// Fallback untuk rute yang tidak terdaftar (404)
	app.Use(func(c *fiber.Ctx) error {
		return utils.SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, "Rute tidak ditemukan", nil)
	})

	// ==========================================
	// 4. START SERVER
	// ==========================================
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Println("Server is running on port", port)
	log.Fatal(app.Listen(":" + port))
}
