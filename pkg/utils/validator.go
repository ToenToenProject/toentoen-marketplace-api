package utils

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// ErrorResponse represents a single validation error
type ErrorResponse struct {
	FailedField string `json:"field"`
	Tag         string `json:"tag"`
	Value       string `json:"value"`
	Message     string `json:"message"`
}

// ValidateStruct validates a struct using go-playground/validator and returns user-friendly errors
func ValidateStruct(structData interface{}) []*ErrorResponse {
	var errors []*ErrorResponse
	err := validate.Struct(structData)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			var element ErrorResponse
			element.FailedField = err.Field()
			element.Tag = err.Tag()
			element.Value = err.Param()

			isCollection := err.Kind() == reflect.Slice || err.Kind() == reflect.Array

			// Create custom messages based on the tag
			switch err.Tag() {
			case "required":
				element.Message = fmt.Sprintf("%s wajib diisi", err.Field())
			case "email":
				element.Message = fmt.Sprintf("%s tidak valid (harus berupa email)", err.Field())
			case "min":
				if isCollection {
					element.Message = fmt.Sprintf("%s minimal %s item", err.Field(), err.Param())
				} else {
					element.Message = fmt.Sprintf("%s minimal %s karakter", err.Field(), err.Param())
				}
			case "max":
				if isCollection {
					element.Message = fmt.Sprintf("%s maksimal %s item/foto", err.Field(), err.Param())
				} else {
					element.Message = fmt.Sprintf("%s maksimal %s karakter", err.Field(), err.Param())
				}
			default:
				element.Message = fmt.Sprintf("%s tidak valid", err.Field())
			}
			errors = append(errors, &element)
		}
	}
	return errors
}

// FormatValidationErrors converts the array of errors to a single readable string
func FormatValidationErrors(errors []*ErrorResponse) string {
	var msgs []string
	for _, e := range errors {
		msgs = append(msgs, e.Message)
	}
	return strings.Join(msgs, ", ")
}
