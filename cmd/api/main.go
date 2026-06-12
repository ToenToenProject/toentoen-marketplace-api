package main

import (
	"log"
	"os"
	"time"

	"toentoen-backend/internal/config"
	"toentoen-backend/internal/handlers"
	"toentoen-backend/internal/middlewares"
	"toentoen-backend/internal/repositories"
	"toentoen-backend/internal/services"
	"toentoen-backend/pkg/database"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
)

// Pekerja Belakang Layar (Cron)
func startMidnightResetWorker(businessRepo repositories.BusinessRepository) {
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
	// config.InitFirebase()

	// ==========================================
	// 2. INIT REPO, SERVICE, & HANDLER
	// ==========================================
	authRepo := repositories.NewAuthRepository(database.DB)
	authService := services.NewAuthService(authRepo)
	authHandler := handlers.NewAuthHandler(authService)

	profileRepo := repositories.NewProfileRepository(database.DB)
	profileService := services.NewProfileService(profileRepo)
	profileHandler := handlers.NewProfileHandler(profileService)

	businessRepo := repositories.NewBusinessRepository(database.DB)
	startMidnightResetWorker(businessRepo) // Jalankan cron job
	businessService := services.NewBusinessService(businessRepo)
	businessHandler := handlers.NewBusinessHandler(businessService)

	// 🪄 INIT BUSINESS ADMIN (BARU)
	businessAdminRepo := repositories.NewBusinessAdminRepository(database.DB)
	businessAdminService := services.NewBusinessAdminService(businessAdminRepo, businessRepo)
	businessAdminHandler := handlers.NewBusinessAdminHandler(businessAdminService)

	productRepo := repositories.NewProductRepository(database.DB)
	productService := services.NewProductService(productRepo)
	productHandler := handlers.NewProductHandler(productService)

	productGroupRepo := repositories.NewProductGroupRepository(database.DB)
	productGroupService := services.NewProductGroupService(productGroupRepo)
	productGroupHandler := handlers.NewProductGroupHandler(productGroupService)

	productInteractionRepo := repositories.NewProductInteractionRepository(database.DB)
	productInteractionService := services.NewProductInteractionService(productInteractionRepo)
	productInteractionHandler := handlers.NewProductInteractionHandler(productInteractionService)

	postInteractionRepo := repositories.NewPostInteractionRepository(database.DB)
	postInteractionService := services.NewPostInteractionService(postInteractionRepo)
	postInteractionHandler := handlers.NewPostInteractionHandler(postInteractionService)

	businessPostRepo := repositories.NewBusinessPostRepository(database.DB)
	businessPostService := services.NewBusinessPostService(businessPostRepo, productRepo)
	businessPostHandler := handlers.NewBusinessPostHandler(businessPostService)

	demandRepo := repositories.NewDemandRepository(database.DB)
	demandService := services.NewDemandService(demandRepo, businessRepo)
	demandHandler := handlers.NewDemandHandler(demandService)

	analyticsRepo := repositories.NewAnalyticsRepository(database.DB)
	analyticsService := services.NewAnalyticsService(analyticsRepo, productRepo)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService)

	reviewRepo := repositories.NewReviewRepository(database.DB)
	reviewService := services.NewReviewService(reviewRepo, businessRepo)
	reviewHandler := handlers.NewReviewHandler(reviewService)

	// ==========================================
	// 3. FIBER APP SETUP
	// ==========================================
	app := fiber.New()
	app.Use(logger.New())
	app.Use(cors.New())

	api := app.Group("/api/v1")

	// ==========================================
	// ROUTES: AUTHENTICATION
	// ==========================================
	api.Post("/logout", handlers.Logout)

	authGroup := api.Group("/auth")
	authGroup.Post("/register", authHandler.Register)
	authGroup.Post("/verify-register-otp", authHandler.VerifyRegisterOTP)
	authGroup.Post("/login", authHandler.Login)
	authGroup.Post("/verify-login-otp", authHandler.VerifyLoginOTP)
	authGroup.Post("/forgot-password", authHandler.ForgotPassword)
	authGroup.Post("/verify-forgot-otp", authHandler.VerifyForgotOTP)
	authGroup.Post("/reset-password", authHandler.ResetPassword)
	authGroup.Post("/change-password", middlewares.RequireAuth, authHandler.ChangePassword)
	authGroup.Post("/fcm-token", middlewares.RequireAuth, authHandler.RegisterFCMToken)

	// ==========================================
	// ROUTES: USER PROFILE
	// ==========================================
	protected := api.Group("/account", middlewares.RequireAuth)
	protected.Put("/profile/customer", middlewares.AllowRoles("customer"), profileHandler.UpdateProfileCustomer)
	protected.Put("/profile/owner", middlewares.AllowRoles("owner"), profileHandler.UpdateProfileOwner)

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
	publicProductRoutes.Post("/:product_id/like", middlewares.RequireAuth, productInteractionHandler.ToggleLike)
	publicProductRoutes.Post("/:product_id/comments", middlewares.RequireAuth, productInteractionHandler.AddProductComment)
	publicProductRoutes.Get("/:product_id/comments", middlewares.RequireAuth, productInteractionHandler.GetProductComments)

	publicPostRoutes := api.Group("/posts")
	publicPostRoutes.Post("/:post_id/like", middlewares.RequireAuth, postInteractionHandler.ToggleLike)
	publicPostRoutes.Post("/:post_id/comments", middlewares.RequireAuth, postInteractionHandler.AddPostComment)
	publicPostRoutes.Get("/:post_id/comments", middlewares.RequireAuth, postInteractionHandler.GetPostComments)

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
