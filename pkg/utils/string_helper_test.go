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

func TestValidateHashtags(t *testing.T) {
	// Valid count and length
	valid := []string{"kopi", "teh", "jus"}
	if err := ValidateHashtags(valid, 5, 10); err != nil {
		t.Errorf("expected nil error for valid tags, got %v", err)
	}

	// Exceeded count
	if err := ValidateHashtags(valid, 2, 10); err == nil {
		t.Errorf("expected error for exceeded count, got nil")
	}

	// Exceeded length
	longTag := []string{"kopisusupalingenaksekali"}
	if err := ValidateHashtags(longTag, 5, 10); err == nil {
		t.Errorf("expected error for exceeded length, got nil")
	}
}

func TestCleanCustomTags(t *testing.T) {
	input := []string{"  Pet Friendly  ", "Live Music", "pet friendly", "   ", "Musholla"}
	expected := []string{"Pet Friendly", "Live Music", "Musholla"}
	got := CleanCustomTags(input)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("CleanCustomTags(%v) = %v, expected %v", input, got, expected)
	}

	empty := CleanCustomTags([]string{})
	if len(empty) != 0 {
		t.Errorf("expected empty slice, got %v", empty)
	}
}

func TestValidateCustomTags(t *testing.T) {
	valid := []string{"Pet Friendly", "Live Music"}
	if err := ValidateCustomTags(valid, 5, 20); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	// Exceed count
	if err := ValidateCustomTags(valid, 1, 20); err == nil {
		t.Errorf("expected error for exceed count, got nil")
	}

	// Exceed length
	long := []string{"Nama Tag Sangat Amat Panjang Melebihi Batas Karakter"}
	if err := ValidateCustomTags(long, 5, 15); err == nil {
		t.Errorf("expected error for exceed length, got nil")
	}
}


