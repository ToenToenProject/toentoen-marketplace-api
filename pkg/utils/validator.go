package utils

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New()
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})
	return v
}

var fieldLabels = map[string]string{
	"identifier":         "Email atau nomor telepon",
	"email":              "Email",
	"new_email":          "Email baru",
	"phone_number":       "Nomor telepon",
	"new_phone_number":   "Nomor telepon baru",
	"username":           "Username",
	"password":           "Kata sandi",
	"old_password":       "Kata sandi lama",
	"new_password":       "Kata sandi baru",
	"confirm_password":   "Konfirmasi kata sandi",
	"first_name":         "Nama depan",
	"last_name":          "Nama belakang",
	"gender":             "Jenis kelamin",
	"role":               "Role",
	"otp":                "Kode OTP",
	"code":               "Kode",
	"referral_code":      "Kode referral",
	"ktp_number":         "Nomor KTP",
	"ktp_photo_url":      "Foto KTP",
	"selfie_photo_url":   "Foto selfie",
	"ktp_name":           "Nama sesuai KTP",
	"ktp_birth_province": "Provinsi kelahiran",
	"ktp_birth_city":     "Kota/kabupaten kelahiran",
	"ktp_birth_date":     "Tanggal lahir",
	"ktp_gender":         "Jenis kelamin KTP",
	"ktp_address":        "Alamat KTP",
	"name":               "Nama",
	"business_name":      "Nama toko",
	"handle":             "Handle toko",
	"description":        "Deskripsi",
	"address":            "Alamat",
	"category_id":        "Kategori",
	"product_name":       "Nama produk",
	"price":              "Harga",
	"stock":              "Stok",
	"title":              "Judul",
	"content":            "Isi",
	"rating":             "Rating",
	"comment":            "Komentar",
	"reason":             "Alasan",
	"start_at":           "Tanggal mulai",
	"end_at":             "Tanggal berakhir",
}

func fieldLabel(field string) string {
	if label, ok := fieldLabels[field]; ok {
		return label
	}
	return strings.ReplaceAll(field, "_", " ")
}

// ErrorResponse represents a single validation error.
type ErrorResponse struct {
	FailedField string `json:"field"`
	Tag         string `json:"tag"`
	Value       string `json:"value"`
	Message     string `json:"message"`
}

// ValidateStruct validates a struct using go-playground/validator and returns localized validation errors.
func ValidateStruct(structData interface{}) []*ErrorResponse {
	var errors []*ErrorResponse
	err := validate.Struct(structData)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			var element ErrorResponse
			element.FailedField = err.Field()
			element.Tag = err.Tag()
			element.Value = err.Param()

			label := fieldLabel(err.Field())
			isCollection := err.Kind() == reflect.Slice || err.Kind() == reflect.Array

			switch err.Tag() {
			case "required":
				element.Message = fmt.Sprintf("%s wajib diisi", label)
			case "email":
				element.Message = fmt.Sprintf("%s tidak valid (harus berupa alamat email yang benar)", label)
			case "min":
				if isCollection {
					element.Message = fmt.Sprintf("%s minimal %s item", label, err.Param())
				} else if err.Kind() == reflect.Int || err.Kind() == reflect.Int64 || err.Kind() == reflect.Float64 {
					element.Message = fmt.Sprintf("%s minimal %s", label, err.Param())
				} else {
					element.Message = fmt.Sprintf("%s minimal %s karakter", label, err.Param())
				}
			case "max":
				if isCollection {
					element.Message = fmt.Sprintf("%s maksimal %s item", label, err.Param())
				} else if err.Kind() == reflect.Int || err.Kind() == reflect.Int64 || err.Kind() == reflect.Float64 {
					element.Message = fmt.Sprintf("%s maksimal %s", label, err.Param())
				} else {
					element.Message = fmt.Sprintf("%s maksimal %s karakter", label, err.Param())
				}
			case "len":
				element.Message = fmt.Sprintf("%s harus tepat %s karakter", label, err.Param())
			case "oneof":
				choices := strings.ReplaceAll(err.Param(), " ", ", ")
				element.Message = fmt.Sprintf("%s harus salah satu dari: %s", label, choices)
			case "url":
				element.Message = fmt.Sprintf("%s harus berupa URL yang valid", label)
			case "numeric":
				element.Message = fmt.Sprintf("%s hanya boleh berisi angka", label)
			case "alphanum":
				element.Message = fmt.Sprintf("%s hanya boleh berisi huruf dan angka", label)
			case "gt":
				element.Message = fmt.Sprintf("%s harus lebih besar dari %s", label, err.Param())
			case "gte":
				element.Message = fmt.Sprintf("%s minimal %s", label, err.Param())
			case "lt":
				element.Message = fmt.Sprintf("%s harus kurang dari %s", label, err.Param())
			case "lte":
				element.Message = fmt.Sprintf("%s maksimal %s", label, err.Param())
			case "eqfield":
				element.Message = fmt.Sprintf("%s tidak cocok dengan %s", label, fieldLabel(err.Param()))
			case "uuid":
				element.Message = fmt.Sprintf("%s harus berupa UUID yang valid", label)
			default:
				element.Message = fmt.Sprintf("%s tidak valid", label)
			}
			errors = append(errors, &element)
		}
	}
	return errors
}

// FormatValidationErrors converts a slice of validation errors to a comma-separated summary string.
func FormatValidationErrors(errors []*ErrorResponse) string {
	var msgs []string
	for _, e := range errors {
		msgs = append(msgs, e.Message)
	}
	return strings.Join(msgs, ", ")
}
