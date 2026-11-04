package main

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestLegalRoutes_PrivacyPolicy(t *testing.T) {
	app := fiber.New()
	api := app.Group("/api/v1")
	registerLegalRoutes(app, api)

	testCases := []struct {
		name       string
		path       string
		wantStatus int
		wantSubstr string
	}{
		{
			name:       "Root /privacy-policy",
			path:       "/privacy-policy",
			wantStatus: fiber.StatusOK,
			wantSubstr: "Kebijakan Privasi Toentoen",
		},
		{
			name:       "Root /privacy alias",
			path:       "/privacy",
			wantStatus: fiber.StatusOK,
			wantSubstr: "Kebijakan Privasi Toentoen",
		},
		{
			name:       "Root /kebijakan-privasi alias",
			path:       "/kebijakan-privasi",
			wantStatus: fiber.StatusOK,
			wantSubstr: "Kebijakan Privasi Toentoen",
		},
		{
			name:       "API /api/v1/privacy-policy",
			path:       "/api/v1/privacy-policy",
			wantStatus: fiber.StatusOK,
			wantSubstr: "Kebijakan Privasi Toentoen",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			resp, err := app.Test(req, 2000)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatus {
				t.Errorf("expected status %d, got %d", tc.wantStatus, resp.StatusCode)
			}

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read body: %v", err)
			}
			body := string(bodyBytes)

			if !strings.Contains(body, tc.wantSubstr) {
				t.Errorf("expected body to contain %q, but did not find it", tc.wantSubstr)
			}

			// Pastikan bagian penting ada di dalam HTML
			if !strings.Contains(body, "support@toentoen.com") {
				t.Errorf("expected body to contain contact email 'support@toentoen.com'")
			}
			if !strings.Contains(body, "Penghapusan Akun") {
				t.Errorf("expected body to contain account deletion section")
			}
			if !strings.Contains(body, "id=\"account-deletion\"") {
				t.Errorf("expected body to contain id=\"account-deletion\" anchor")
			}
		})
	}
}

func TestLegalRoutes_AccountDeletionRedirect(t *testing.T) {
	app := fiber.New()
	api := app.Group("/api/v1")
	registerLegalRoutes(app, api)

	req := httptest.NewRequest("GET", "/account-deletion", nil)
	resp, err := app.Test(req, 2000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusTemporaryRedirect {
		t.Errorf("expected status %d, got %d", fiber.StatusTemporaryRedirect, resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != "/privacy-policy#account-deletion" {
		t.Errorf("expected Location '/privacy-policy#account-deletion', got %q", location)
	}
}

