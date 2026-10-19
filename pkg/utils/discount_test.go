package utils

import (
	"testing"
	"time"
)

func TestCalculateProductDiscount(t *testing.T) {
	maxCap10k := 10000.0
	maxCap50k := 50000.0

	tests := []struct {
		name           string
		price          float64
		discountType   string
		discountValue  float64
		maxDiscount    *float64
		wantAmount     float64
		wantPct        float64
	}{
		{
			name:          "Zero discount",
			price:         25000,
			discountType:  "nominal",
			discountValue: 0,
			maxDiscount:   nil,
			wantAmount:    0,
			wantPct:       0,
		},
		{
			name:          "Nominal discount normal",
			price:         25000,
			discountType:  "nominal",
			discountValue: 5000,
			maxDiscount:   nil,
			wantAmount:    5000,
			wantPct:       20.0,
		},
		{
			name:          "Percentage discount without max cap (20% of 50000)",
			price:         50000,
			discountType:  "percentage",
			discountValue: 20,
			maxDiscount:   nil,
			wantAmount:    10000,
			wantPct:       20.0,
		},
		{
			name:          "Percentage discount with max cap triggered (50% of 50000 = 25000, capped to 10000)",
			price:         50000,
			discountType:  "percentage",
			discountValue: 50,
			maxDiscount:   &maxCap10k,
			wantAmount:    10000,
			wantPct:       20.0, // Effective 10000 / 50000 = 20%
		},
		{
			name:          "Percentage discount with max cap NOT triggered (10% of 50000 = 5000, cap is 10000)",
			price:         50000,
			discountType:  "percentage",
			discountValue: 10,
			maxDiscount:   &maxCap10k,
			wantAmount:    5000,
			wantPct:       10.0,
		},
		{
			name:          "Nominal discount exceeds price (guard against negative price)",
			price:         25000,
			discountType:  "nominal",
			discountValue: 30000,
			maxDiscount:   nil,
			wantAmount:    25000,
			wantPct:       100.0,
		},
		{
			name:          "Percentage 100% of price",
			price:         50000,
			discountType:  "percentage",
			discountValue: 100,
			maxDiscount:   &maxCap50k,
			wantAmount:    50000,
			wantPct:       100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAmount, gotPct := CalculateProductDiscount(tt.price, tt.discountType, tt.discountValue, tt.maxDiscount)
			if gotAmount != tt.wantAmount {
				t.Errorf("CalculateProductDiscount() gotAmount = %v, want %v", gotAmount, tt.wantAmount)
			}
			if gotPct != tt.wantPct {
				t.Errorf("CalculateProductDiscount() gotPct = %v, want %v", gotPct, tt.wantPct)
			}
		})
	}
}

func TestIsDiscountActive(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	past := now.Add(-2 * time.Hour)
	future := now.Add(2 * time.Hour)
	farFuture := now.Add(24 * time.Hour)

	tests := []struct {
		name     string
		discount float64
		startAt  *time.Time
		endAt    *time.Time
		want     bool
	}{
		{
			name:     "Zero discount is inactive",
			discount: 0,
			startAt:  &past,
			endAt:    &future,
			want:     false,
		},
		{
			name:     "Negative discount is inactive",
			discount: -10,
			startAt:  nil,
			endAt:    nil,
			want:     false,
		},
		{
			name:     "Discount without dates is active",
			discount: 10,
			startAt:  nil,
			endAt:    nil,
			want:     true,
		},
		{
			name:     "Discount scheduled for future is inactive",
			discount: 10,
			startAt:  &future,
			endAt:    &farFuture,
			want:     false,
		},
		{
			name:     "Discount ended in past is inactive",
			discount: 10,
			startAt:  &past,
			endAt:    &past,
			want:     false,
		},
		{
			name:     "Discount in valid range is active",
			discount: 10,
			startAt:  &past,
			endAt:    &future,
			want:     true,
		},
		{
			name:     "Discount with only future endAt is active",
			discount: 10,
			startAt:  nil,
			endAt:    &future,
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDiscountActive(tt.discount, tt.startAt, tt.endAt, now)
			if got != tt.want {
				t.Errorf("IsDiscountActive() = %v, want %v", got, tt.want)
			}
		})
	}
}
