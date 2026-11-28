package utils

import (
	"database/sql"
	"errors"
	"strings"
	"toentoen-backend/internal/constants"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// AppError represents a structured domain/application error with an explicit HTTP status and response code.
type AppError struct {
	HTTPStatus   int
	ResponseCode string
	Message      string
	Err          error
}

func (e *AppError) Error() string {
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewBadRequestError(message string) *AppError {
	return &AppError{HTTPStatus: fiber.StatusBadRequest, ResponseCode: constants.FailedValidation, Message: message}
}

func NewConflictError(code, message string) *AppError {
	if code == "" {
		code = constants.DuplicateData
	}
	return &AppError{HTTPStatus: fiber.StatusConflict, ResponseCode: code, Message: message}
}

func NewNotFoundError(message string) *AppError {
	return &AppError{HTTPStatus: fiber.StatusNotFound, ResponseCode: constants.DataNotFound, Message: message}
}

func NewForbiddenError(message string) *AppError {
	return &AppError{HTTPStatus: fiber.StatusForbidden, ResponseCode: constants.ForbiddenAccess, Message: message}
}

func NewUnauthorizedError(message string) *AppError {
	return &AppError{HTTPStatus: fiber.StatusUnauthorized, ResponseCode: constants.FailedAuth, Message: message}
}

// TranslateDBError maps PostgreSQL and GORM errors into user-friendly AppError instances.
func TranslateDBError(err error) *AppError {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, sql.ErrNoRows) {
		return NewNotFoundError("Data tidak ditemukan")
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return translateUniqueViolation(pgErr.ConstraintName + " " + pgErr.Detail + " " + pgErr.Message)
		case "23503":
			return NewBadRequestError("Data referensi tidak ditemukan atau tidak valid")
		case "22P02":
			return &AppError{
				HTTPStatus:   fiber.StatusBadRequest,
				ResponseCode: constants.InvalidParam,
				Message:      "Format parameter atau ID tidak valid",
				Err:          err,
			}
		case "23502":
			return NewBadRequestError("Ada data wajib yang belum diisi")
		case "23514":
			return NewBadRequestError("Nilai data tidak memenuhi aturan yang diizinkan")
		}
	}

	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "23505") || strings.Contains(lower, "duplicate key value") || strings.Contains(lower, "unique constraint") {
		return translateUniqueViolation(lower)
	}
	if strings.Contains(lower, "23503") || strings.Contains(lower, "violates foreign key constraint") {
		return NewBadRequestError("Data referensi tidak ditemukan atau tidak valid")
	}
	if strings.Contains(lower, "22p02") || strings.Contains(lower, "invalid input syntax") {
		return &AppError{
			HTTPStatus:   fiber.StatusBadRequest,
			ResponseCode: constants.InvalidParam,
			Message:      "Format parameter atau ID tidak valid",
			Err:          err,
		}
	}

	return nil
}

func translateUniqueViolation(hint string) *AppError {
	h := strings.ToLower(hint)
	switch {
	case strings.Contains(h, "username"):
		return NewConflictError(constants.DuplicateData, "Username sudah terpakai, gunakan username lain")
	case strings.Contains(h, "email"):
		return NewConflictError(constants.DuplicateData, "Email sudah terdaftar, gunakan email lain")
	case strings.Contains(h, "phone"):
		return NewConflictError(constants.DuplicateData, "Nomor telepon sudah terdaftar, gunakan nomor telepon lain")
	case strings.Contains(h, "handle"):
		return NewConflictError(constants.HandleTaken, "Handle toko sudah digunakan, silakan pilih handle lain")
	case strings.Contains(h, "ktp"):
		return NewConflictError(constants.DuplicateData, "Nomor KTP sudah terdaftar pada akun lain")
	case strings.Contains(h, "referral"):
		return NewConflictError(constants.DuplicateData, "Kode referral sudah terpakai")
	case strings.Contains(h, "wishlist"), strings.Contains(h, "like"), strings.Contains(h, "follow"), strings.Contains(h, "review"), strings.Contains(h, "report"):
		return NewConflictError(constants.DuplicateData, "Data sudah pernah dikirim atau sudah terdaftar")
	default:
		return NewConflictError(constants.DuplicateData, "Data sudah terdaftar, silakan gunakan nilai yang berbeda")
	}
}

// HandleServiceError converts any service/repository error into a consistent JSON response with a clear reason.
func HandleServiceError(c *fiber.Ctx, err error, fallbackMessage string) error {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return SendResponse(c, appErr.HTTPStatus, appErr.ResponseCode, appErr.Message, nil)
	}

	if dbErr := TranslateDBError(err); dbErr != nil {
		return SendResponse(c, dbErr.HTTPStatus, dbErr.ResponseCode, dbErr.Message, nil)
	}

	msg := err.Error()
	lower := strings.ToLower(msg)

	switch {
	case strings.Contains(lower, "handle") && (strings.Contains(lower, "sudah") || strings.Contains(lower, "taken") || strings.Contains(lower, "digunakan")):
		return SendResponse(c, fiber.StatusConflict, constants.HandleTaken, msg, nil)
	case strings.Contains(lower, "sudah digunakan") ||
		strings.Contains(lower, "sudah terpakai") ||
		strings.Contains(lower, "sudah terdaftar") ||
		strings.Contains(lower, "sudah dipakai") ||
		strings.Contains(lower, "sudah terisi") ||
		strings.Contains(lower, "sudah pernah") ||
		strings.Contains(lower, "already in use") ||
		strings.Contains(lower, "already exists") ||
		strings.Contains(lower, "already registered"):
		return SendResponse(c, fiber.StatusConflict, constants.DuplicateData, msg, nil)
	case strings.Contains(lower, "akun tidak ditemukan") || strings.Contains(lower, "account not found"):
		return SendResponse(c, fiber.StatusNotFound, constants.UserNotFound, msg, nil)
	case strings.Contains(lower, "tidak ditemukan") || strings.Contains(lower, "not found"):
		return SendResponse(c, fiber.StatusNotFound, constants.DataNotFound, msg, nil)
	case strings.Contains(lower, "forbidden") ||
		strings.Contains(lower, "akses ditolak") ||
		strings.Contains(lower, "tidak memiliki izin") ||
		strings.Contains(lower, "tidak berhak") ||
		strings.Contains(lower, "bukan pemilik") ||
		strings.Contains(lower, "belum terverifikasi") ||
		strings.Contains(lower, "diblokir"):
		return SendResponse(c, fiber.StatusForbidden, constants.ForbiddenAccess, msg, nil)
	case strings.Contains(lower, "unauthorized") ||
		strings.Contains(lower, "sesi tidak valid") ||
		strings.Contains(lower, "sesi login") ||
		strings.Contains(lower, "token tidak valid"):
		return SendResponse(c, fiber.StatusUnauthorized, constants.FailedAuth, msg, nil)
	case strings.Contains(lower, "wajib") ||
		strings.Contains(lower, "tidak valid") ||
		strings.Contains(lower, "harus") ||
		strings.Contains(lower, "minimal") ||
		strings.Contains(lower, "maksimal") ||
		strings.Contains(lower, "tidak boleh") ||
		strings.Contains(lower, "kadaluarsa") ||
		strings.Contains(lower, "kedaluwarsa") ||
		strings.Contains(lower, "tidak sesuai") ||
		strings.Contains(lower, "tidak cocok") ||
		strings.Contains(lower, "tidak mencukupi") ||
		strings.Contains(lower, "tidak dapat") ||
		strings.Contains(lower, "masih aktif") ||
		strings.Contains(lower, "sudah terverifikasi") ||
		strings.Contains(lower, "salah"):
		return SendResponse(c, fiber.StatusBadRequest, constants.FailedValidation, msg, nil)
	default:
		if fallbackMessage == "" {
			fallbackMessage = "Terjadi kesalahan pada server"
		}
		return SendResponse(c, fiber.StatusInternalServerError, constants.InternalServerErr, fallbackMessage, nil)
	}
}
