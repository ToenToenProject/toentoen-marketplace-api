package explore

import (
	"fmt"
	"strings"
)

// ParsedSearchIntent merepresentasikan hasil ekstraksi intent & filter terstruktur dari query bahasa natural pengguna.
type ParsedSearchIntent struct {
	RawQuery           string   `json:"raw_query"`
	CleanQuery         string   `json:"clean_query"`
	EntityType         string   `json:"entity_type"` // GENERAL, OFFERING, CATEGORY, SERVICE, FACILITY
	DetectedFacilities []string `json:"detected_facilities,omitempty"`
	DetectedOpenNow    bool     `json:"detected_open_now,omitempty"`
	DetectedNearby     bool     `json:"detected_nearby,omitempty"`
	DetectedPromo      bool     `json:"detected_promo,omitempty"`
}

type phraseRule struct {
	phrases   []string
	canonical string
}

var facilityPhraseRules = []phraseRule{
	{
		phrases:   []string{"dengan free wifi", "ada free wifi", "free wifi", "wifi gratis", "dengan wi-fi", "ada wi-fi", "dengan wifi", "ada wifi", "pakai wifi"},
		canonical: "Wi-Fi",
	},
	{
		phrases:   []string{"bebas asap rokok", "tanpa asap rokok", "no smoking", "non smoking", "non-smoking"},
		canonical: "Bebas Asap Rokok",
	},
	{
		phrases:   []string{"ada parkir luas", "parkir luas", "ada tempat parkir", "tempat parkir", "ada parkir", "dengan parkir"},
		canonical: "Parkir",
	},
	{
		phrases:   []string{"ada ruang tunggu", "dengan ruang tunggu", "ruang tunggu"},
		canonical: "Ruang Tunggu",
	},
	{
		phrases:   []string{"ruangan ber-ac", "ruangan ac", "full ac", "ber-ac", "ada ac", "dengan ac"},
		canonical: "Ber-AC",
	},
	{
		phrases:   []string{"ramah hewan", "pet friendly", "pet-friendly"},
		canonical: "Ramah Hewan",
	},
	{
		phrases:   []string{"pembayaran qris", "terima qris", "bayar qris", "ada qris", "pakai qris"},
		canonical: "QRIS",
	},
}

var openNowPhrases = []string{
	"yang buka sekarang", "yang lagi buka", "yang masih buka",
	"buka sekarang", "lagi buka", "masih buka", "yang buka", "open now",
}

var nearbyPhrases = []string{
	"di sekitar sini", "sekitar sini", "dekat sini", "paling dekat",
	"terdekat", "di dekat sini", "near me", "nearby",
}

var promoPhrases = []string{
	"yang ada promo", "yang lagi promo", "ada promo", "lagi promo", "ada diskon", "lagi diskon",
}

var trailingStopWords = map[string]bool{
	"yang":   true,
	"dan":    true,
	"dengan": true,
	"ada":    true,
	"di":     true,
	"yg":     true,
	"untuk":  true,
	"buat":   true,
}

func removePhraseNormalized(text, phrase string) (string, bool) {
	lowerText := strings.ToLower(text)
	lowerPhrase := strings.ToLower(phrase)
	idx := strings.Index(lowerText, lowerPhrase)
	if idx == -1 {
		return text, false
	}

	// Pastikan batas kata (word boundary) agar tidak memotong substring di tengah kata
	beforeOK := idx == 0 || isWordBoundary(lowerText[idx-1])
	endIdx := idx + len(lowerPhrase)
	afterOK := endIdx == len(lowerText) || isWordBoundary(lowerText[endIdx])
	if !beforeOK || !afterOK {
		return text, false
	}

	cleaned := strings.TrimSpace(text[:idx] + " " + text[endIdx:])
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	return cleaned, true
}

func isWordBoundary(b byte) bool {
	return b == ' ' || b == ',' || b == '.' || b == ';' || b == '?' || b == '!' || b == '/' || b == '(' || b == ')'
}

func trimBoundaryStopWords(text string) string {
	cleaned := strings.Trim(text, " ,.;!?/-")
	words := strings.Fields(cleaned)
	for len(words) > 0 && trailingStopWords[strings.ToLower(words[len(words)-1])] {
		words = words[:len(words)-1]
	}
	for len(words) > 0 && trailingStopWords[strings.ToLower(words[0])] {
		words = words[1:]
	}
	return strings.Trim(strings.Join(words, " "), " ,.;!?/-")
}

// ParseSearchQuery menganalisis query pencarian natural Bahasa Indonesia menjadi komponen terstruktur.
func ParseSearchQuery(rawQuery string) ParsedSearchIntent {
	trimmed := strings.TrimSpace(rawQuery)
	intent := ParsedSearchIntent{
		RawQuery:   trimmed,
		CleanQuery: trimmed,
		EntityType: "GENERAL",
	}
	if trimmed == "" {
		return intent
	}

	working := trimmed

	// 1. Deteksi status buka sekarang
	for _, p := range openNowPhrases {
		if next, ok := removePhraseNormalized(working, p); ok {
			intent.DetectedOpenNow = true
			working = next
			break
		}
	}

	// 2. Deteksi lokasi terdekat
	for _, p := range nearbyPhrases {
		if next, ok := removePhraseNormalized(working, p); ok {
			intent.DetectedNearby = true
			working = next
			break
		}
	}

	// 3. Deteksi promo
	for _, p := range promoPhrases {
		if next, ok := removePhraseNormalized(working, p); ok {
			intent.DetectedPromo = true
			working = next
			break
		}
	}

	// 4. Deteksi fasilitas eksplisit (Hard Filter)
	seenFac := make(map[string]bool)
	for _, rule := range facilityPhraseRules {
		for _, p := range rule.phrases {
			if next, ok := removePhraseNormalized(working, p); ok {
				if !seenFac[rule.canonical] {
					seenFac[rule.canonical] = true
					intent.DetectedFacilities = append(intent.DetectedFacilities, rule.canonical)
				}
				working = next
				break
			}
		}
	}

	working = trimBoundaryStopWords(working)
	intent.CleanQuery = working

	// 5. Tentukan estimasi entity_type awal
	if intent.CleanQuery == "" && len(intent.DetectedFacilities) > 0 {
		intent.EntityType = "FACILITY"
	}

	return intent
}

// ComputeMatchExplanation menganalisis mengapa sebuah bisnis cocok dengan pencarian user
// dan mengisi field MatchType, MatchedOffering, MatchReason, serta MatchConfidence.
func ComputeMatchExplanation(doc *MeiliBusinessDocument, cleanQuery string, expandedTerms []string) {
	if doc == nil {
		return
	}
	q := strings.ToLower(strings.TrimSpace(cleanQuery))
	if q == "" {
		return
	}

	terms := make([]string, 0, len(expandedTerms)+1)
	terms = append(terms, q)
	for _, t := range expandedTerms {
		lt := strings.ToLower(strings.TrimSpace(t))
		if lt != "" && lt != q {
			terms = append(terms, lt)
		}
	}

	// 1. Cek Exact atau Related Match pada Produk / Menu (Offerings)
	for _, mp := range doc.MatchedProducts {
		pName := strings.ToLower(strings.TrimSpace(mp.ProductName))
		if pName == "" {
			continue
		}
		if pName == q {
			doc.MatchType = "EXACT_OFFERING"
			doc.MatchedOffering = mp.ProductName
			doc.MatchReason = fmt.Sprintf("Tersedia: %s", mp.ProductName)
			doc.MatchConfidence = 1.0
			return
		}
	}
	for _, mp := range doc.MatchedProducts {
		pName := strings.ToLower(strings.TrimSpace(mp.ProductName))
		if pName == "" {
			continue
		}
		for i, term := range terms {
			if strings.Contains(pName, term) {
				if i == 0 {
					doc.MatchType = "EXACT_OFFERING"
					doc.MatchConfidence = 0.95
				} else {
					doc.MatchType = "RELATED_OFFERING"
					doc.MatchConfidence = 0.85
				}
				doc.MatchedOffering = mp.ProductName
				doc.MatchReason = fmt.Sprintf("Tersedia: %s", mp.ProductName)
				return
			}
		}
	}

	// Fallback cek array string doc.Products jika MatchedProducts kosong
	for _, pStr := range doc.Products {
		pName := strings.ToLower(strings.TrimSpace(pStr))
		for i, term := range terms {
			if pName != "" && strings.Contains(pName, term) {
				if i == 0 {
					doc.MatchType = "EXACT_OFFERING"
					doc.MatchConfidence = 0.92
				} else {
					doc.MatchType = "RELATED_OFFERING"
					doc.MatchConfidence = 0.84
				}
				doc.MatchedOffering = pStr
				doc.MatchReason = fmt.Sprintf("Tersedia: %s", pStr)
				return
			}
		}
	}

	// 2. Cek Nama Bisnis / Brand
	bizNameLower := strings.ToLower(strings.TrimSpace(doc.Name))
	if bizNameLower != "" {
		for i, term := range terms {
			if strings.Contains(bizNameLower, term) {
				doc.MatchType = "NAME_MATCH"
				doc.MatchedOffering = doc.Name
				doc.MatchReason = "Nama toko cocok"
				if i == 0 {
					doc.MatchConfidence = 0.95
				} else {
					doc.MatchConfidence = 0.82
				}
				return
			}
		}
	}

	// 3. Cek Layanan (Services)
	for _, srv := range doc.Services {
		srvLower := strings.ToLower(strings.TrimSpace(srv))
		for _, term := range terms {
			if srvLower != "" && strings.Contains(srvLower, term) {
				doc.MatchType = "SERVICE_MATCH"
				doc.MatchedOffering = srv
				doc.MatchReason = fmt.Sprintf("Layanan cocok: %s", srv)
				doc.MatchConfidence = 0.88
				return
			}
		}
	}

	// 4. Cek Kategori (CategoryNames)
	for _, cat := range doc.CategoryNames {
		catLower := strings.ToLower(strings.TrimSpace(cat))
		for _, term := range terms {
			if catLower != "" && strings.Contains(catLower, term) {
				doc.MatchType = "CATEGORY_MATCH"
				doc.MatchedOffering = cat
				doc.MatchReason = fmt.Sprintf("Kategori: %s", cat)
				doc.MatchConfidence = 0.82
				return
			}
		}
	}

	// 5. Cek Fasilitas & Tags (Facilities / TagNames)
	for _, fac := range doc.Facilities {
		facLower := strings.ToLower(strings.TrimSpace(fac))
		for _, term := range terms {
			if facLower != "" && strings.Contains(facLower, term) {
				doc.MatchType = "FACILITY_MATCH"
				doc.MatchedOffering = fac
				doc.MatchReason = fmt.Sprintf("Fasilitas: %s", fac)
				doc.MatchConfidence = 0.78
				return
			}
		}
	}
	for _, tag := range doc.TagNames {
		tagLower := strings.ToLower(strings.TrimSpace(tag))
		for _, term := range terms {
			if tagLower != "" && strings.Contains(tagLower, term) {
				doc.MatchType = "FACILITY_MATCH"
				doc.MatchedOffering = tag
				doc.MatchReason = fmt.Sprintf("Atribut cocok: %s", tag)
				doc.MatchConfidence = 0.75
				return
			}
		}
	}
}
