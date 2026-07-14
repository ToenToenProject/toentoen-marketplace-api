package main

import (
	"log"
	"os"
	"time"
	"toentoen-backend/internal/domain/account"
	"toentoen-backend/internal/domain/auth"
	"toentoen-backend/internal/domain/customer/business_profile"
	"toentoen-backend/internal/domain/customer/demand"
	"toentoen-backend/internal/domain/customer/explore"
	"toentoen-backend/internal/domain/customer/interaction"
	"toentoen-backend/internal/domain/customer/my_activity"

	"toentoen-backend/internal/domain/customer/product"
	"toentoen-backend/internal/domain/customer/profile"
	"toentoen-backend/internal/domain/customer/report"
	"toentoen-backend/internal/domain/customer/review"
	"toentoen-backend/internal/domain/merchant/admin"
	"toentoen-backend/internal/domain/merchant/analytics"
	"toentoen-backend/internal/domain/merchant/business"
	"toentoen-backend/internal/domain/merchant/catalog"
	"toentoen-backend/internal/domain/merchant/post"

	"toentoen-backend/internal/config"
	"toentoen-backend/internal/domain/shared/middlewares"
	"toentoen-backend/internal/domain/upload"
	"toentoen-backend/pkg/database"
	"toentoen-backend/pkg/meilisearch"
	"toentoen-backend/pkg/storage"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
)

// Pekerja Belakang Layar (Cron)
func startMidnightResetWorker(businessRepo business.BusinessRepository) {
	go func() {
		for {
			loc, _ := time.LoadLocation("Asia/Jakarta")
			now := time.Now().In(loc)

			// Hitung berapa lama lagi menuju jam 00:00:00 (Tengah Malam Berikutnya)
			nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, loc)
			durationUntilMidnight := nextMidnight.Sub(now)

			// Tidurkan worker ini sampai tengah malam
			time.Sleep(durationUntilMidnight)

			// BANGUN! Eksekusi reset database
			log.Println("[CRON] Menjalankan reset is_manually_closed ke false...")
			businessRepo.ResetManualCloseStatus()
		}
	}()
}

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
	// config.InitFirebase()

	// ==========================================
	// 2. INIT REPO, SERVICE, & HANDLER
	// ==========================================
	authRepo := auth.NewAuthRepository(database.DB)
	authService := auth.NewAuthService(authRepo)
	authHandler := auth.NewAuthHandler(authService)

	profileRepo := account.NewProfileRepository(database.DB)
	profileService := account.NewProfileService(profileRepo)
	profileHandler := account.NewProfileHandler(profileService)

	businessRepo := business.NewBusinessRepository(database.DB)
	startMidnightResetWorker(businessRepo) // Jalankan cron job
	businessService := business.NewBusinessService(businessRepo)
	businessHandler := business.NewBusinessHandler(businessService)

	// 🪄 INIT BUSINESS ADMIN (BARU)
	businessAdminRepo := admin.NewBusinessAdminRepository(database.DB)
	businessAdminService := admin.NewBusinessAdminService(businessAdminRepo, businessRepo)
	businessAdminHandler := admin.NewBusinessAdminHandler(businessAdminService)

	productRepo := catalog.NewProductRepository(database.DB)
	productService := catalog.NewProductService(productRepo)
	productHandler := catalog.NewProductHandler(productService)

	productGroupRepo := catalog.NewProductGroupRepository(database.DB)
	productGroupService := catalog.NewProductGroupService(productGroupRepo)
	productGroupHandler := catalog.NewProductGroupHandler(productGroupService)

	productInteractionRepo := interaction.NewProductInteractionRepository(database.DB)
	productInteractionService := interaction.NewProductInteractionService(productInteractionRepo)
	productInteractionHandler := interaction.NewProductInteractionHandler(productInteractionService)

	postInteractionRepo := interaction.NewPostInteractionRepository(database.DB)
	postInteractionService := interaction.NewPostInteractionService(postInteractionRepo)
	postInteractionHandler := interaction.NewPostInteractionHandler(postInteractionService)

	businessPostRepo := post.NewBusinessPostRepository(database.DB)
	businessPostService := post.NewBusinessPostService(businessPostRepo, productRepo)
	businessPostHandler := post.NewBusinessPostHandler(businessPostService)

	demandRepo := demand.NewDemandRepository(database.DB)
	demandService := demand.NewDemandService(demandRepo, businessRepo)
	demandHandler := demand.NewDemandHandler(demandService)

	analyticsRepo := analytics.NewAnalyticsRepository(database.DB)
	analyticsService := analytics.NewAnalyticsService(analyticsRepo, productRepo)
	analyticsHandler := analytics.NewAnalyticsHandler(analyticsService)

	reviewRepo := review.NewReviewRepository(database.DB)
	reviewService := review.NewReviewService(reviewRepo, businessRepo)
	reviewHandler := review.NewReviewHandler(reviewService, productInteractionService)

	exploreRepo := explore.NewExploreRepository(database.DB)
	exploreService := explore.NewExploreService(exploreRepo)
	exploreHandler := explore.NewExploreHandler(exploreService)

	businessProfileRepo := business_profile.NewRepository()
	businessProfileService := business_profile.NewService(businessProfileRepo)
	businessProfileHandler := business_profile.NewHandler(businessProfileService)

	reportRepo := report.NewRepository()
	reportService := report.NewService(reportRepo)
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

	// ==========================================
	// 3. FIBER APP SETUP
	// ==========================================
	app := fiber.New()
	app.Use(logger.New())
	app.Use(cors.New())

	api := app.Group("/api/v1")

	// ==========================================
	// ROUTES: UPLOAD (S3 Presigned URL)
	// ==========================================
	uploadGroup := api.Group("/upload", middlewares.RequireAuth)
	uploadGroup.Post("/presign", uploadHandler.GeneratePresignURL)

	// ==========================================
	// ROUTES: AUTHENTICATION
	// ==========================================
	api.Post("/logout", auth.Logout)

	authGroup := api.Group("/auth")
	authGroup.Post("/register", authHandler.Register)
	authGroup.Post("/verify-register-otp", authHandler.VerifyRegisterOTP)
	authGroup.Post("/login", authHandler.Login)
	authGroup.Post("/verify-login-otp", authHandler.VerifyLoginOTP)
	authGroup.Post("/forgot-password", authHandler.ForgotPassword)
	authGroup.Post("/verify-forgot-otp", authHandler.VerifyForgotOTP)
	authGroup.Post("/reset-password", authHandler.ResetPassword)
	authGroup.Post("/password/change", middlewares.RequireAuth, authHandler.ChangePassword)
	authGroup.Post("/email/change/request", middlewares.RequireAuth, authHandler.RequestEmailChange)
	authGroup.Post("/email/change/verify", middlewares.RequireAuth, authHandler.VerifyEmailChange)

	// ==========================================
	// ROUTES: CUSTOMER BUSINESS PROFILE
	// ==========================================
	customerBusinessGroup := api.Group("/customer/businesses")
	customerBusinessGroup.Get("/bookmarks", middlewares.RequireAuth, businessProfileHandler.GetBookmarks)
	customerBusinessGroup.Get("/:business_id/profile", middlewares.OptionalAuth, businessProfileHandler.GetProfile)
	customerBusinessGroup.Post("/:business_id/bookmark", middlewares.RequireAuth, businessProfileHandler.ToggleBookmark)
	customerBusinessGroup.Post("/:business_id/report", middlewares.RequireAuth, reportHandler.SubmitReport)

	// Products
	customerBusinessGroup.Get("/:business_id/product-groups", businessProfileHandler.GetProductGroups)
	customerBusinessGroup.Get("/:business_id/products", businessProfileHandler.GetProducts)
	customerBusinessGroup.Get("/:business_id/posts", middlewares.OptionalAuth, businessProfileHandler.GetPosts)

	// ==========================================
	// ROUTES: USER PROFILE
	// ==========================================
	protected := api.Group("/account", middlewares.RequireAuth)
	protected.Put("/profile/customer", middlewares.AllowRoles("customer"), profileHandler.UpdateProfileCustomer)
	protected.Put("/profile/owner", middlewares.AllowRoles("owner"), profileHandler.UpdateProfileOwner)

	// My Activity
	myActivityGroup := protected.Group("/my-activity")
	myActivityGroup.Get("/summary", myActivityHandler.GetMyProfileSummary)
	myActivityGroup.Get("/comments", myActivityHandler.GetMyComments)
	myActivityGroup.Get("/reviews", myActivityHandler.GetMyBusinessReviews)
	myActivityGroup.Get("/bookmarks", myActivityHandler.GetMyBookmarks)
	myActivityGroup.Get("/wishlist", myActivityHandler.GetMyWishlists)

	api.Post("/customer/wishlist/:product_id", middlewares.RequireAuth, myActivityHandler.ToggleWishlist)

	// ==========================================
	// ROUTES: CUSTOMER ACTIONS & INTERACTIONS
	// ==========================================
	customerReviewGroup := api.Group("/reviews/customer", middlewares.RequireAuth, middlewares.AllowRoles("customer"))
	customerReviewGroup.Post("/", reviewHandler.CreateReview)

	// Demand Signals
	demandGroup := api.Group("/demands", middlewares.RequireAuth)
	demandGroup.Post("/signals", demandHandler.CreateSignal)
	demandGroup.Put("/signals/:signal_id/responses/:response_id/accept", demandHandler.AcceptOffer)

	publicProductRoutes := api.Group("/products")
	publicProductRoutes.Get("/:product_id/detail", middlewares.OptionalAuth, productDetailHandler.GetProductDetail)
	publicProductRoutes.Post("/:product_id/like", middlewares.RequireAuth, productInteractionHandler.ToggleLike)
	publicProductRoutes.Post("/:product_id/reviews", middlewares.RequireAuth, productInteractionHandler.AddProductComment)
	publicProductRoutes.Get("/:product_id/reviews", middlewares.OptionalAuth, productInteractionHandler.GetProductComments)
	publicProductRoutes.Post("/reviews/:review_id/like", middlewares.RequireAuth, productInteractionHandler.ToggleCommentLike)
	publicProductRoutes.Post("/reviews/:review_id/report", middlewares.RequireAuth, productInteractionHandler.ReportComment)

	publicPostRoutes := api.Group("/posts")
	publicPostRoutes.Post("/:post_id/like", middlewares.RequireAuth, postInteractionHandler.ToggleLike)
	publicPostRoutes.Post("/:post_id/comments", middlewares.RequireAuth, postInteractionHandler.AddPostComment)
	publicPostRoutes.Get("/:post_id/comments", middlewares.RequireAuth, postInteractionHandler.GetPostComments)
	publicPostRoutes.Post("/:post_id/report", middlewares.RequireAuth, postInteractionHandler.ReportPost)

	// ==========================================
	// ROUTES: BUSINESS OWNER / ADMIN ACTIONS
	// ==========================================

	// 🪄 Admin Management (Hanya Owner)
	ownerAdminGroup := api.Group("/business/:business_id/admins", middlewares.RequireAuth, middlewares.AllowRoles("owner"))
	ownerAdminGroup.Post("/invite", businessAdminHandler.InviteAdmin)
	ownerAdminGroup.Get("/", businessAdminHandler.GetAdminsByBusiness)
	ownerAdminGroup.Put("/:assignment_id", businessAdminHandler.UpdateAdminAssignment)

	// 🪄 Rute Personal Calon Admin (Lihat Undangan & Terima/Tolak)
	workerGroup := api.Group("/my-jobs", middlewares.RequireAuth)
	workerGroup.Get("/", businessAdminHandler.GetMyAssignments)
	workerGroup.Put("/:assignment_id/respond", businessAdminHandler.RespondToInvitation)

	// Business Management
	businessGroup := api.Group("/business", middlewares.RequireAuth, middlewares.AllowRoles("owner"))
	businessGroup.Post("/", businessHandler.CreateBusiness)
	businessGroup.Get("/", businessHandler.GetMyBusinesses)
	businessGroup.Get("/:business_id", businessHandler.GetBusinessDetail)
	businessGroup.Put("/:id", businessHandler.UpdateBusiness)
	businessGroup.Delete("/:id", businessHandler.DeleteBusiness)
	businessGroup.Patch("/:business_id/toggle-status", businessHandler.ToggleStoreStatus)

	// Admin / Owner Shared Access (Products, Posts, Analytics)
	sharedAccess := middlewares.AllowRoles("owner", "business_admin")

	// -> Products
	productRoutes := api.Group("/business/:business_id/products", middlewares.RequireAuth, sharedAccess)
	productRoutes.Post("/", productHandler.CreateProduct)
	productRoutes.Post("/:product_id/variants", middlewares.RequireAuth, productHandler.AddVariant)
	productRoutes.Patch("/:product_id/variants/:variant_id", middlewares.RequireAuth, productHandler.UpdateVariant)
	productRoutes.Delete("/:product_id/variants/:variant_id", middlewares.RequireAuth, productHandler.DeleteVariant)
	productRoutes.Get("/catalog", middlewares.RequireAuth, productHandler.GetCatalog)
	productRoutes.Get("/:product_id", productHandler.GetProductDetail)
	productRoutes.Put("/:product_id", productHandler.UpdateProduct)
	productRoutes.Delete("/:product_id", productHandler.DeleteProduct)
	productRoutes.Patch("/bulk-status", middlewares.RequireAuth, productHandler.BulkUpdateStatus)
	productRoutes.Delete("/bulk-delete", middlewares.RequireAuth, productHandler.BulkDelete)

	// -> Product Groups
	productGroupRoutes := api.Group("/business/:business_id/product-groups", middlewares.RequireAuth, sharedAccess)
	productGroupRoutes.Patch("/reorder", productGroupHandler.ReorderGroups)
	productGroupRoutes.Patch("/:product_group_id/contents", productGroupHandler.UpdateContents)
	productGroupRoutes.Post("/", productGroupHandler.CreateGroup)
	productGroupRoutes.Get("/", productGroupHandler.GetGroups)
	productGroupRoutes.Put("/:product_group_id", productGroupHandler.UpdateGroup)
	productGroupRoutes.Delete("/:product_group_id", productGroupHandler.DeleteGroup)

	// -> Posts & Comments
	postRoutes := api.Group("/business/:business_id/posts", middlewares.RequireAuth, sharedAccess)
	postRoutes.Post("/", businessPostHandler.CreatePost)
	postRoutes.Get("/", businessPostHandler.GetPosts)
	postRoutes.Delete("/:post_id", businessPostHandler.DeletePost)

	// -> Demands (Explore & Respond)
	demandGroup.Get("/signals/explore", demandHandler.ExploreSignals)
	demandGroup.Post("/signals/:signal_id/responses", demandHandler.SubmitResponse)
	demandGroup.Get("/signals/active", middlewares.AllowRoles("owner"), demandHandler.GetActiveSignals)
	demandGroup.Get("/signals/:signal_id", middlewares.AllowRoles("owner"), demandHandler.GetSignalDetail)

	// -> Analytics
	analyticsRoutes := api.Group("/business/:business_id/analytics", middlewares.RequireAuth, sharedAccess)
	analyticsRoutes.Get("/daily", analyticsHandler.GetDailyAnalytics)
	analyticsRoutes.Get("/history", analyticsHandler.GetHistoryAnalytics)
	analyticsRoutes.Get("/explore", analyticsHandler.GetExploreAnalytics)

	// -> Merchant Review Reply
	merchantGroup := api.Group("/reviews/merchant", middlewares.RequireAuth, middlewares.AllowRoles("owner"))
	merchantGroup.Patch("/:review_id/reply", reviewHandler.ReplyReview)

	reviewGroup := api.Group("/reviews", middlewares.RequireAuth, middlewares.AllowRoles("customer", "owner"))
	reviewGroup.Get("/:review_id", reviewHandler.GetReviewDetail)
	reviewGroup.Get("/business/:business_id", reviewHandler.GetAllReviews)

	// Review Interactions
	reviewGroup.Post("/:review_id/like", reviewHandler.ToggleReviewLike)
	reviewGroup.Post("/:review_id/comments", reviewHandler.AddReviewComment)
	reviewGroup.Get("/:review_id/comments", reviewHandler.GetReviewComments)
	reviewGroup.Post("/:review_id/report", reviewHandler.ReportReview)

	// Review Comment Interactions
	reviewGroup.Post("/comments/:comment_id/like", reviewHandler.ToggleReviewCommentLike)
	reviewGroup.Post("/comments/:comment_id/report", reviewHandler.ReportReviewComment)

	// ==========================================
	// ROUTES: CUSTOMER EXPLORE
	// ==========================================
	customerExploreGroup := api.Group("/customer/explore")
	customerExploreGroup.Get("/businesses", exploreHandler.SearchBusinesses)
	customerExploreGroup.Get("/timeline", middlewares.OptionalAuth, exploreHandler.GetTimeline)

	// ==========================================
	// ROUTES: CUSTOMER PROFILE
	// ==========================================
	api.Get("/customer/profile/:account_id/summary", middlewares.RequireAuth, publicProfileHandler.GetPublicProfileSummary)

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
