package utils

import (
	"crypto/rand"
	"math/big"
)

func GenerateNumericOTP(length int) string {
	const digits = "0123456789"
	otp := make([]byte, length)
	max := big.NewInt(int64(len(digits)))
	for i := range otp {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			otp[i] = digits[0]
			continue
		}
		otp[i] = digits[n.Int64()]
	}
	return string(otp)
}