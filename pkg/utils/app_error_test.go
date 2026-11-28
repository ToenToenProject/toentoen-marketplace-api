package utils

import (
	"errors"
	"net/http"
	"testing"
	"toentoen-backend/internal/constants"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestTranslateDBError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode string
		wantHTTP int
		wantMsg  string
	}{
		{
			name:     "duplicate username pgError",
			err:      &pgconn.PgError{Code: "23505", ConstraintName: "idx_accounts_username_role"},
			wantCode: constants.DuplicateData,
			wantHTTP: http.StatusConflict,
			wantMsg:  "Username sudah terpakai, gunakan username lain",
		},
		{
			name:     "duplicate email string",
			err:      errors.New(`ERROR: duplicate key value violates unique constraint "idx_accounts_email_role" (SQLSTATE 23505)`),
			wantCode: constants.DuplicateData,
			wantHTTP: http.StatusConflict,
			wantMsg:  "Email sudah terdaftar, gunakan email lain",
		},
		{
			name:     "duplicate handle pgError",
			err:      &pgconn.PgError{Code: "23505", ConstraintName: "idx_businesses_handle"},
			wantCode: constants.HandleTaken,
			wantHTTP: http.StatusConflict,
			wantMsg:  "Handle toko sudah digunakan, silakan pilih handle lain",
		},
		{
			name:     "foreign key violation",
			err:      &pgconn.PgError{Code: "23503", ConstraintName: "fk_products_category"},
			wantCode: constants.FailedValidation,
			wantHTTP: http.StatusBadRequest,
			wantMsg:  "Data referensi tidak ditemukan atau tidak valid",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := TranslateDBError(tc.err)
			if got == nil {
				t.Fatalf("expected non-nil AppError")
			}
			if got.HTTPStatus != tc.wantHTTP {
				t.Errorf("HTTPStatus = %d, want %d", got.HTTPStatus, tc.wantHTTP)
			}
			if got.ResponseCode != tc.wantCode {
				t.Errorf("ResponseCode = %s, want %s", got.ResponseCode, tc.wantCode)
			}
			if got.Message != tc.wantMsg {
				t.Errorf("Message = %q, want %q", got.Message, tc.wantMsg)
			}
		})
	}
}

func TestValidateStructIndonesianMessages(t *testing.T) {
	type sampleInput struct {
		Username string `json:"username" validate:"required,min=3"`
		Email    string `json:"email" validate:"required,email"`
		Gender   string `json:"gender" validate:"oneof=male female other"`
	}

	errs := ValidateStruct(&sampleInput{
		Username: "ab",
		Email:    "not-an-email",
		Gender:   "unknown",
	})
	if len(errs) != 3 {
		t.Fatalf("expected 3 errors, got %d", len(errs))
	}
	if errs[0].FailedField != "username" || errs[0].Message != "Username minimal 3 karakter" {
		t.Errorf("unexpected username error: %+v", errs[0])
	}
	if errs[1].FailedField != "email" {
		t.Errorf("expected json field name 'email', got %q", errs[1].FailedField)
	}
	if errs[2].FailedField != "gender" || errs[2].Message != "Jenis kelamin harus salah satu dari: male, female, other" {
		t.Errorf("unexpected gender error: %+v", errs[2])
	}
	_ = fiber.StatusBadRequest
}
