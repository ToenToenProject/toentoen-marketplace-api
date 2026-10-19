package utils

import (
	"math"
	"time"
)

// CalculateProductDiscount menghitung nilai potongan riil (Rupiah) dan persentase riil diskon.
// Menerima parameter:
// - price: harga dasar produk (sebelum diskon)
// - discountType: tipe diskon ("percentage" atau "nominal")
// - discountValue: nilai diskon yang dimasukkan (angka persen 1-100 atau nominal Rupiah)
// - maxDiscount: batas maksimal potongan dalam Rupiah (opsional, jika nil atau 0 berarti tanpa batas)
//
// Mengembalikan:
// - discountAmount: total potongan harga riil dalam Rupiah setelah memperhitungkan maxDiscount
// - discountPct: persentase potongan riil terhadap harga dasar
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
		// Default: nominal
		discountAmount = discountValue
		if maxDiscount != nil && *maxDiscount > 0 && discountAmount > *maxDiscount {
			discountAmount = *maxDiscount
		}
		discountPct = (discountAmount / price) * 100.0
	}

	// Safety guard: Potongan tidak boleh melebihi harga produk itu sendiri
	if discountAmount > price {
		discountAmount = price
		discountPct = 100.0
	}

	return math.Round(discountAmount*100) / 100, math.Round(discountPct*10) / 10
}

// IsDiscountActive mengecek apakah diskon produk sedang aktif berdasarkan nilai diskon
// dan batas waktu mulai (startAt) serta batas waktu berakhir (endAt).
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
