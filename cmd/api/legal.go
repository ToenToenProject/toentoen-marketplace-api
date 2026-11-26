package main

import (
	_ "embed"

	"github.com/gofiber/fiber/v2"
)

//go:embed assets/privacy-policy.html
var privacyPolicyHTML []byte

// registerLegalRoutes registers public compliance endpoints (Privacy Policy & Account Deletion)
// required by app stores (Google Play Console).
func registerLegalRoutes(app *fiber.App, api fiber.Router) {
	privacyHandler := func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.Send(privacyPolicyHTML)
	}

	accountDeletionHandler := func(c *fiber.Ctx) error {
		// Redirect to Section 8 anchor for account deletion procedures.
		return c.Redirect("/privacy-policy#account-deletion", fiber.StatusTemporaryRedirect)
	}

	// Root-level endpoints for store listing URLs.
	app.Get("/privacy-policy", privacyHandler)
	app.Get("/privacy", privacyHandler)
	app.Get("/kebijakan-privasi", privacyHandler)
	app.Get("/account-deletion", accountDeletionHandler)
	app.Get("/delete-account", accountDeletionHandler)

	// API prefix aliases for client routing.
	if api != nil {
		api.Get("/privacy-policy", privacyHandler)
		api.Get("/privacy", privacyHandler)
		api.Get("/account-deletion", accountDeletionHandler)
	}
}

