package utils

import (
	"fmt"
	"math/rand"
	"strings"
)

func GenerateRandomUsername(name string) string {
	cleanName := strings.ToLower(strings.ReplaceAll(name, " ", ""))
	return fmt.Sprintf("%s%d", cleanName, rand.Intn(9999))
}