package utils

import (
	"testing"
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
