package meilisearch

import (
	"log"
	"strings"
	"sync"

	"gorm.io/gorm"
	"toentoen-backend/internal/domain/shared/models"
)

var (
	synonymCacheMu sync.RWMutex
	dynamicSynMap  map[string][]string
)

// GetIndonesianSynonyms mengembalikan kamus statis sinonim, bahasa gaul, singkatan kuliner,
// dan toleransi ejaan umum bahasa Indonesia untuk mesin pencari.
func GetIndonesianSynonyms() map[string][]string {
	return map[string][]string{
		// ☕ Kopi & Minuman
		"kopsus":    {"kopi susu", "coffee", "es kopi"},
		"kopi susu": {"kopsus", "es kopi"},
		"kopi":      {"coffee", "espresso", "latte", "kopsus", "ngopi"},
		"ngopi":     {"kopi", "kopi susu", "coffee"},
		"es":        {"ice", "dingin"},
		"ice":       {"es"},
		"eskrim":    {"es krim", "ice cream"},
		"es krim":   {"eskrim", "ice cream"},
		"ice cream": {"eskrim", "es krim"},
		"jus":       {"juice", "sari buah"},
		"juice":     {"jus"},
		"teh":       {"tea", "esteh"},
		"esteh":     {"es teh", "teh"},
		"boba":      {"bubble tea", "pearl", "boba drink"},
		"susu":      {"milk"},
		"milk":      {"susu"},
		"coklat":    {"cokelat", "chocolate", "choco"},
		"cokelat":   {"coklat", "chocolate", "choco"},
		"chocolate": {"cokelat", "coklat"},

		// 🍲 Makanan, Kuliner & Singkatan Populer
		"nasgor":      {"nasi goreng"},
		"nasi goreng": {"nasgor"},
		"mie":         {"bakmi", "mi", "noodle", "noodles", "ramen"},
		"bakmi":       {"mie", "mi", "noodle"},
		"mi":          {"mie", "bakmi", "noodle"},
		"robar":       {"roti bakar", "toast"},
		"roti bakar":  {"robar", "toast"},
		"roti":        {"bread", "bakery", "robar"},
		"ayam":        {"chicken"},
		"chicken":     {"ayam"},
		"bebek":       {"duck"},
		"daging":      {"meat", "beef"},
		"sapi":        {"beef"},
		"ikan":        {"fish", "seafood"},
		"seafood":     {"ikan", "udang", "cumi"},
		"udang":       {"shrimp", "prawn"},
		"cumi":        {"squid"},
		"bakso":       {"baso", "meatball"},
		"baso":        {"bakso", "meatball"},
		"soto":        {"coto", "soup"},
		"sate":        {"satay"},
		"satay":       {"sate"},
		"seblak":      {"seblak pedas", "kerupuk basah"},
		"dimsum":      {"siomay", "dumpling", "hakau"},
		"siomay":      {"dimsum", "somay"},
		"cemilan":     {"snack", "makanan ringan", "jajanan", "camilan"},
		"camilan":     {"cemilan", "snack", "jajanan"},
		"snack":       {"cemilan", "camilan", "jajanan"},
		"pedas":       {"pedes", "spicy", "hot", "mercon", "geprek"},
		"pedes":       {"pedas", "spicy"},

		// 👗 Fashion & Pakaian
		"baju":     {"pakaian", "kaos", "tshirt", "t-shirt", "kemeja", "outfit"},
		"kaos":     {"tshirt", "t-shirt", "baju"},
		"tshirt":   {"kaos", "baju"},
		"kemeja":   {"shirt", "baju"},
		"celana":   {"pants", "trousers", "jeans"},
		"sepatu":   {"sneakers", "shoes", "alas kaki"},
		"sneakers": {"sepatu"},
		"sandal":   {"sandals", "slippers"},
		"tas":      {"bag", "backpack", "totebag"},
		"hijab":    {"jilbab", "kerudung", "pashmina"},
		"jilbab":   {"hijab", "kerudung"},

		// 🏪 Tipe Tempat & Fasilitas
		"warkop":        {"warung kopi", "kedai kopi", "cafe"},
		"warung":        {"kedai", "toko", "resto", "rumah makan"},
		"cafe":          {"kafe", "coffee shop", "warkop"},
		"kafe":          {"cafe", "coffee shop"},
		"restoran":      {"resto", "rumah makan", "eatery"},
		"resto":         {"restoran", "rumah makan"},
		"alun_alun":     {"alun-alun", "alun alun", "lapangan"},
		"pusat_belanja": {"mall", "plaza", "pertokoan"},

		// 🔤 Typo Umum Bahasa Indonesia
		"antri":     {"antre"},
		"antre":     {"antri"},
		"espreso":   {"espresso"},
		"espresso":  {"espreso"},
		"cappucino": {"cappuccino"},
		"apron":     {"celemek"},

		// 🔧 Otomotif, Jasa & Layanan
		"montir":         {"bengkel mobil", "bengkel", "mekanik", "servis mobil"},
		"mekanik":        {"bengkel mobil", "bengkel", "montir", "servis mobil"},
		"bengkel mtr":    {"bengkel motor", "servis motor"},
		"ganti oli":      {"oli mesin", "servis oli", "oil change"},
		"servis ac":      {"service ac", "cuci ac", "perbaikan ac", "ac mobil"},
		"pangkas rambut": {"barbershop", "potong rambut", "cukur", "haircut"},
		"barbershop":     {"pangkas rambut", "potong rambut", "cukur", "haircut"},
		"ayam bakar":     {"ayam panggang", "grilled chicken"},
		"ayam panggang":  {"ayam bakar", "grilled chicken"},
		"tempat ngopi":   {"kafe", "cafe", "coffee shop", "warkop"},
		"wifi":           {"wi-fi", "internet gratis", "free wifi"},
		"wi-fi":          {"wifi", "internet gratis", "free wifi"},
	}
}

// LoadMergedSynonyms menggabungkan kamus statis bawaan dengan kamus dinamis dari tabel search_synonyms di DB.
func LoadMergedSynonyms(db *gorm.DB) map[string][]string {
	merged := GetIndonesianSynonyms()
	if db == nil {
		return merged
	}

	var rows []models.SearchSynonym
	if err := db.Where("is_active = ?", true).Find(&rows).Error; err != nil {
		log.Printf("⚠️ Gagal memuat sinonim dinamis dari DB (fallback statis): %v", err)
		return merged
	}

	for _, r := range rows {
		term := strings.ToLower(strings.TrimSpace(r.Term))
		if term == "" {
			continue
		}
		seen := make(map[string]bool)
		var list []string
		for _, existing := range merged[term] {
			low := strings.ToLower(strings.TrimSpace(existing))
			if low != "" && !seen[low] {
				seen[low] = true
				list = append(list, low)
			}
		}
		if canon := strings.ToLower(strings.TrimSpace(r.CanonicalTerm)); canon != "" && canon != term && !seen[canon] {
			seen[canon] = true
			list = append(list, canon)
		}
		for _, syn := range r.Synonyms {
			low := strings.ToLower(strings.TrimSpace(syn))
			if low != "" && low != term && !seen[low] {
				seen[low] = true
				list = append(list, low)
			}
		}
		if len(list) > 0 {
			merged[term] = list
		}
	}

	synonymCacheMu.Lock()
	dynamicSynMap = merged
	synonymCacheMu.Unlock()

	return merged
}

// SyncSynonymsFromDB memuat sinonim gabungan (statis + DB) lalu memperbarui setting Meilisearch.
func SyncSynonymsFromDB(db *gorm.DB) error {
	merged := LoadMergedSynonyms(db)
	if Client == nil {
		return nil
	}

	if _, err := Client.Index("businesses").UpdateSynonyms(&merged); err != nil {
		return err
	}
	if _, err := Client.Index("products").UpdateSynonyms(&merged); err != nil {
		return err
	}
	log.Printf("🔍 Meilisearch synonyms synced (%d terms)", len(merged))
	return nil
}

// getEffectiveSynonyms mengembalikan kamus sinonim terbaru (cache DB jika tersedia, atau statis).
func getEffectiveSynonyms() map[string][]string {
	synonymCacheMu.RLock()
	cached := dynamicSynMap
	synonymCacheMu.RUnlock()
	if len(cached) > 0 {
		return cached
	}
	return GetIndonesianSynonyms()
}

// ExpandKeywordWithSynonyms memperluas kata kunci pencarian dengan sinonim (statis + dinamis DB)
// untuk digunakan pada fallback query PostgreSQL jika Meilisearch offline/kosong.
func ExpandKeywordWithSynonyms(keyword string) []string {
	kw := strings.ToLower(strings.TrimSpace(keyword))
	if kw == "" {
		return []string{}
	}

	resultMap := make(map[string]bool)
	resultMap[kw] = true

	dict := getEffectiveSynonyms()

	// 1. Cek kecocokan langsung seluruh frasa
	if syns, ok := dict[kw]; ok {
		for _, s := range syns {
			resultMap[s] = true
		}
	}

	// 2. Cek setiap kata individual jika frasa terdiri dari beberapa kata
	words := strings.Fields(kw)
	for _, w := range words {
		if syns, ok := dict[w]; ok {
			for _, s := range syns {
				resultMap[s] = true
			}
		}
	}

	expanded := make([]string, 0, len(resultMap))
	for k := range resultMap {
		expanded = append(expanded, k)
	}
	return expanded
}
