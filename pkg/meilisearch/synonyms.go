package meilisearch

import "strings"

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
	}
}

// ExpandKeywordWithSynonyms memperluas kata kunci pencarian dengan sinonim statis
// untuk digunakan pada fallback query PostgreSQL jika Meilisearch offline/kosong.
func ExpandKeywordWithSynonyms(keyword string) []string {
	kw := strings.ToLower(strings.TrimSpace(keyword))
	if kw == "" {
		return []string{}
	}

	resultMap := make(map[string]bool)
	resultMap[kw] = true

	dict := GetIndonesianSynonyms()

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
