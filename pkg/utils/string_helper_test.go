package utils

import (
	"reflect"
	"testing"
)

func TestNormalizeHashtags(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "clean normal hashtags with and without hash",
			input:    []string{"#kopisusu", "tempat nongkrong", "Kopi Enak!"},
			expected: []string{"kopisusu", "tempatnongkrong", "kopienak"},
		},
		{
			name:     "handles double hashes and spaces and symbols",
			input:    []string{"##kopi", "  #kopi susu  ", "jakarta_selatan", "jaksel-hits"},
			expected: []string{"kopi", "kopisusu", "jakarta_selatan", "jakselhits"},
		},
		{
			name:     "deduplicates equivalent hashtags",
			input:    []string{"#kopisusu", "kopi susu", "KOPISUSU", "  #KopiSusu  "},
			expected: []string{"kopisusu"},
		},
		{
			name:     "filters out empty strings or hash-only strings",
			input:    []string{"#", "##", "", "   ", "valid"},
			expected: []string{"valid"},
		},
		{
			name:     "empty input returns empty slice",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeHashtags(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("NormalizeHashtags(%v) = %v, expected %v", tt.input, got, tt.expected)
			}
		})
	}
}
