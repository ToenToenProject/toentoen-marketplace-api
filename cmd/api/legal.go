package main

import (
	_ "embed"

	"github.com/gofiber/fiber/v2"
)

//go:embed assets/privacy-policy.html
var privacyPolicyHTML []byte

// registerLegalRoutes mendaftarkan endpoint halaman legal publik (Privacy Policy & Account Deletion)
// yang diwajibkan oleh Google Play Store.
func registerLegalRoutes(app *fiber.App, api fiber.Router) {
	privacyHandler := func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.Send(privacyPolicyHTML)
	}

	accountDeletionHandler := func(c *fiber.Ctx) error {
		// Mengarahkan langsung ke anchor Section 8 (Hak Pengguna & Prosedur Penghapusan Akun)
		return c.Redirect("/privacy-policy#account-deletion", fiber.StatusTemporaryRedirect)
	}

	// 1. Root-level standard URLs (Sesuai format resmi yang biasa dimasukkan ke Google Play Console)
	app.Get("/privacy-policy", privacyHandler)
	app.Get("/privacy", privacyHandler)
	app.Get("/kebijakan-privasi", privacyHandler)
	app.Get("/account-deletion", accountDeletionHandler)
	app.Get("/delete-account", accountDeletionHandler)

	// 2. Prefix /api/v1 (opsional agar bisa diakses di bawah route API juga)
	if api != nil {
		api.Get("/privacy-policy", privacyHandler)
		api.Get("/privacy", privacyHandler)
		api.Get("/account-deletion", accountDeletionHandler)
	}
}

