package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestRequireKYC_Unauthenticated(t *testing.T) {
	app := fiber.New()
	app.Get("/test-kyc", RequireKYC, func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test-kyc", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", fiber.StatusUnauthorized, resp.StatusCode)
	}
}

func TestRequireKYC_CustomerAllowed(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "customer")
		c.Locals("account_id", "ACC_CUST_123")
		return c.Next()
	})
	app.Get("/test-kyc", RequireKYC, func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test-kyc", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("expected customer to pass through, got status %d", resp.StatusCode)
	}
}

func TestRequireOwnerKYCSubmitted_Unauthenticated(t *testing.T) {
	app := fiber.New()
	app.Post("/test-owner-kyc", RequireOwnerKYCSubmitted, func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/test-owner-kyc", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", fiber.StatusUnauthorized, resp.StatusCode)
	}
}

func TestRequireOwnerKYCSubmitted_NonOwnerRejected(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "customer")
		c.Locals("account_id", "ACC_CUST_123")
		return c.Next()
	})
	app.Post("/test-owner-kyc", RequireOwnerKYCSubmitted, func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/test-owner-kyc", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("expected status %d, got %d", fiber.StatusForbidden, resp.StatusCode)
	}
}
