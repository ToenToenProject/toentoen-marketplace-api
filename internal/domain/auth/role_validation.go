package auth

import (
	"fmt"
	"strings"
)

// RoleMismatchError represents an error when a user attempts to log in via a portal meant for a different role.
type RoleMismatchError struct {
	ActualRole     string `json:"actual_role"`
	ExpectedRole   string `json:"expected_role"`
	RedirectPortal string `json:"redirect_portal"`
	Message        string `json:"message"`
}

func (e *RoleMismatchError) Error() string {
	return e.Message
}

// NormalizeExpectedRole normalizes various frontend role aliases to canonical backend roles.
func NormalizeExpectedRole(role string) string {
	r := strings.ToLower(strings.TrimSpace(role))
	switch r {
	case "customer", "buyer", "user", "pelanggan":
		return "customer"
	case "owner", "merchant", "seller", "pemilik", "toko":
		return "owner"
	case "admin", "business_admin", "staff", "karyawan":
		return "business_admin"
	default:
		return r
	}
}

func getPortalPath(role string) string {
	switch role {
	case "owner":
		return "/owner/login"
	case "business_admin":
		return "/admin/login"
	case "customer":
		return "/customer/login"
	default:
		return "/login"
	}
}

// NewRoleMismatchError creates a new RoleMismatchError with clear localized guidance and redirect recommendations.
func NewRoleMismatchError(actualRole, expectedRole string) *RoleMismatchError {
	var msg string
	switch actualRole {
	case "owner":
		msg = "Akun Anda terdaftar sebagai Pemilik Toko (Owner). Silakan masuk melalui halaman login Pemilik Toko."
	case "business_admin":
		msg = "Akun Anda terdaftar sebagai Admin/Staff Toko. Silakan masuk melalui halaman login Admin Toko."
	case "customer":
		msg = "Akun Anda terdaftar sebagai Pelanggan (Customer). Silakan masuk melalui aplikasi atau portal Pelanggan."
	default:
		msg = fmt.Sprintf("Peran akun Anda (%s) tidak memiliki akses ke halaman login %s.", actualRole, expectedRole)
	}

	return &RoleMismatchError{
		ActualRole:     actualRole,
		ExpectedRole:   expectedRole,
		RedirectPortal: getPortalPath(actualRole),
		Message:        msg,
	}
}
