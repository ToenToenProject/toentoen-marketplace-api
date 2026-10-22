package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreatePromotionInput_UnmarshalJSON(t *testing.T) {
	t.Run("Parses RFC3339 with +07:00 offset and auto fills MainProductID", func(t *testing.T) {
		rawJSON := `{
			"start_at": "2026-10-22T20:01:00+07:00",
			"end_at": "2026-10-29T20:01:00+07:00",
			"products": [
				{
					"product_id": "PROD-123",
					"discount_value": 15,
					"discount_type": "percentage"
				}
			]
		}`

		var input CreatePromotionInput
		err := json.Unmarshal([]byte(rawJSON), &input)
		assert.NoError(t, err)
		assert.False(t, input.StartAt.IsZero())
		assert.False(t, input.EndAt.IsZero())
		assert.Equal(t, 2026, input.StartAt.Year())
		assert.Equal(t, time.Month(10), input.StartAt.Month())
		assert.Equal(t, 22, input.StartAt.Day())
		assert.Equal(t, 20, input.StartAt.Hour())
		assert.Equal(t, 1, input.StartAt.Minute())

		// Check auto fallbacks
		assert.Equal(t, "PROD-123", input.MainProductID, "MainProductID should fallback to first product")
		assert.Equal(t, "Promo Spesial Toko", input.Caption, "Caption should have default fallback")
	})

	t.Run("Parses SQL datetime format and preserves explicit MainProductID and Caption", func(t *testing.T) {
		rawJSON := `{
			"caption": "Diskon Akhir Pekan",
			"main_product_id": "PROD-999",
			"start_at": "2026-10-22 20:01:00",
			"end_at": "2026-10-29 23:59:59",
			"products": [
				{
					"product_id": "PROD-123",
					"discount_value": 10000,
					"discount_type": "nominal"
				}
			]
		}`

		var input CreatePromotionInput
		err := json.Unmarshal([]byte(rawJSON), &input)
		assert.NoError(t, err)
		assert.Equal(t, "Diskon Akhir Pekan", input.Caption)
		assert.Equal(t, "PROD-999", input.MainProductID)
		assert.Equal(t, 2026, input.StartAt.Year())
		assert.Equal(t, 20, input.StartAt.Hour())
		assert.Equal(t, 1, input.StartAt.Minute())
		assert.Equal(t, 23, input.EndAt.Hour())
		assert.Equal(t, 59, input.EndAt.Minute())
	})
}

func TestCreateBundlingPromotionInput_UnmarshalJSON(t *testing.T) {
	t.Run("Parses RFC3339 with +07:00 offset and auto fills caption", func(t *testing.T) {
		rawJSON := `{
			"package_name": "Paket Kenyang Berdua",
			"package_price": 45000,
			"start_at": "2026-10-22T20:01:00+07:00",
			"products": [
				{"product_id": "PROD-1"},
				{"product_id": "PROD-2"}
			]
		}`

		var input CreateBundlingPromotionInput
		err := json.Unmarshal([]byte(rawJSON), &input)
		assert.NoError(t, err)
		assert.Equal(t, "Paket Kenyang Berdua", input.PackageName)
		assert.Equal(t, "Promo Paket: Paket Kenyang Berdua", input.Caption)
		assert.False(t, input.StartAt.IsZero())
		assert.Equal(t, 2026, input.StartAt.Year())
		assert.Equal(t, 20, input.StartAt.Hour())
		assert.Equal(t, 1, input.StartAt.Minute())
	})
}
