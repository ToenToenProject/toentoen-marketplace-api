package main

import (
	"log"
	"os"

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

func main() {
	// Load env
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	config.InitConfig()

	// Connect to database
	database.ConnectDB()

	config.ConnectRedis()

	// Initialize fiber app
	app := fiber.New()
	app.Use(logger.New())
	app.Use(cors.New())

	// Initialize
	authRepo := repositories.NewAuthRepository(database.DB)
	authService := services.NewAuthService(authRepo)
	authHandler := handlers.NewAuthHandler(authService)
	profileRepo := repositories.NewProfileRepository(database.DB)
	profileService := services.NewProfileService(profileRepo)
	profileHandler := handlers.NewProfileHandler(profileService)
	assignmentRepo := repositories.NewAssignmentRepository(database.DB)
	assignmentService := services.NewAssignmentService(assignmentRepo)
	assignmentHandler := handlers.NewAssignmentHandler(assignmentService)
	businessRepo := repositories.NewBusinessRepository(database.DB)
businessService := services.NewBusinessService(businessRepo)
businessHandler := handlers.NewBusinessHandler(businessService)

	// API routes
	api := app.Group("/api/v1")

	// Logout routes
	api.Post("/logout", handlers.Logout)

	// Auth routes
	authGroup := api.Group("/auth")
	authGroup.Post("/register", authHandler.Register)
	authGroup.Post("/login", authHandler.Login)
	authGroup.Post("/verify-otp", authHandler.VerifyLoginOTP)
	authGroup.Post("/forgot-password", authHandler.ForgotPassword)
	authGroup.Post("/verify-forgot-otp", authHandler.VerifyForgotOTP)
	authGroup.Post("/reset-password", authHandler.ResetPassword)
	authGroup.Post("/change-password",
		middlewares.RequireAuth,
		authHandler.ChangePassword,
	)

	// Create business admin
	ownerGroup := api.Group("/owner", middlewares.RequireAuth, middlewares.AllowRoles("owner"))
	ownerGroup.Post("/create-business-admin", authHandler.CreateBusinessAdmin)
	ownerGroup.Post("/assign-business-admin", assignmentHandler.AssignBusinessAdmin)
	ownerGroup.Get("/business-admins", assignmentHandler.GetMyAdmins)
	ownerGroup.Put("/assign-business-admin/:admin_id/business/:business_id", assignmentHandler.UpdateAssignment)
	ownerGroup.Delete("/business-admins/:admin_id", assignmentHandler.DeleteBusinessAdmin)

	// Edit account profile
	protected := api.Group("/account", middlewares.RequireAuth)
	protected.Put("/profile/user", middlewares.AllowRoles("user"), profileHandler.UpdateProfileUser)
	protected.Put("/profile/owner", middlewares.AllowRoles("owner"), profileHandler.UpdateProfileOwner)

	// Create business
	businessGroup := api.Group("/business", middlewares.RequireAuth, middlewares.AllowRoles("owner"))
	businessGroup.Post("/", businessHandler.CreateBusiness)
	businessGroup.Get("/", businessHandler.GetMyBusinesses)
	businessGroup.Put("/:id", businessHandler.UpdateBusiness)
	businessGroup.Delete("/:id", businessHandler.DeleteBusiness)

	// Product routes
	productGroup := api.Group("/products", middlewares.RequireAuth)

	productGroup.Post("/",
		middlewares.AllowRoles("admin", "owner"),
		handlers.CreateProduct,
	)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Fatal(app.Listen(":" + port))
}
