package utils

import (
	"math"
	"time"
)

// CalculateProductDiscount computes the net discount amount and discount percentage against the base price,
// taking into account percentage or nominal discount types and max discount caps.
func CalculateProductDiscount(price float64, discountType string, discountValue float64, maxDiscount *float64) (discountAmount float64, discountPct float64) {
	if discountValue <= 0 || price <= 0 {
		return 0, 0
	}

	if discountType == "percentage" {
		discountAmount = price * (discountValue / 100.0)
		if maxDiscount != nil && *maxDiscount > 0 && discountAmount > *maxDiscount {
			discountAmount = *maxDiscount
		}
		discountPct = (discountAmount / price) * 100.0
	} else {
		discountAmount = discountValue
		if maxDiscount != nil && *maxDiscount > 0 && discountAmount > *maxDiscount {
			discountAmount = *maxDiscount
		}
		discountPct = (discountAmount / price) * 100.0
	}

	// Safety guard: discount cannot exceed base price.
	if discountAmount > price {
		discountAmount = price
		discountPct = 100.0
	}

	return math.Round(discountAmount*100) / 100, math.Round(discountPct*10) / 10
}

// IsDiscountActive checks whether a discount is within its valid date range and has a positive value.
func IsDiscountActive(discount float64, startAt, endAt *time.Time, now time.Time) bool {
	if discount <= 0 {
		return false
	}
	if startAt != nil && now.Before(*startAt) {
		return false
	}
	if endAt != nil && now.After(*endAt) {
		return false
	}
	return true
}
