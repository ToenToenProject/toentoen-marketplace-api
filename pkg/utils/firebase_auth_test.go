package utils

import (
	"testing"
)

func TestNormalizePhoneNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"08123456789", "628123456789"},
		{"+628123456789", "628123456789"},
		{"628123456789", "628123456789"},
		{"+62 812-3456-789", "628123456789"},
		{"(0812) 3456 789", "628123456789"},
		{"+1 555-123-4567", "15551234567"},
		{"", ""},
	}

	for _, tt := range tests {
		result := NormalizePhoneNumber(tt.input)
		if result != tt.expected {
			t.Errorf("NormalizePhoneNumber(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestMatchPhoneNumbers(t *testing.T) {
	tests := []struct {
		phone1   string
		phone2   string
		expected bool
	}{
		{"081234567890", "+6281234567890", true},
		{"+62 812-3456-7890", "081234567890", true},
		{"6281234567890", "+6281234567890", true},
		{"081234567890", "089999999999", false},
		{"", "+6281234567890", false},
	}

	for _, tt := range tests {
		result := MatchPhoneNumbers(tt.phone1, tt.phone2)
		if result != tt.expected {
			t.Errorf("MatchPhoneNumbers(%q, %q) = %v, expected %v", tt.phone1, tt.phone2, result, tt.expected)
		}
	}
}
