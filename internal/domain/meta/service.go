package meta

import (
	"toentoen-backend/internal/domain/shared/models"
)

type MetaService interface {
	GetCategories() ([]models.Category, error)
	GetTags(categoryID *int64, scope string) ([]models.Tag, error)
	GetStoreTypes() []map[string]string
	GetOccupations() ([]models.Occupation, error)
	GetCountries() []CountryItem
}

type CountryItem struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	NameEn string `json:"name_en"`
	Flag   string `json:"flag"`
}

type metaService struct {
	metaRepo MetaRepository
}

func NewMetaService(metaRepo MetaRepository) MetaService {
	return &metaService{metaRepo}
}

func (s *metaService) GetCategories() ([]models.Category, error) {
	return s.metaRepo.GetAllCategories()
}

func (s *metaService) GetTags(categoryID *int64, scope string) ([]models.Tag, error) {
	return s.metaRepo.GetAllTags(categoryID, scope)
}

func (s *metaService) GetStoreTypes() []map[string]string {
	return []map[string]string{
		{"label": "Rumah", "label_en": "Home", "value": "home"},
		{"label": "Keliling", "label_en": "Mobile", "value": "mobile"},
		{"label": "Gerobak", "label_en": "Stand", "value": "stand"},
		{"label": "Toko", "label_en": "Store", "value": "store"},
		{"label": "Ruko", "label_en": "Shophouse", "value": "shophouse"},
		{"label": "Pasar", "label_en": "Market", "value": "market"},
		{"label": "Alun - alun", "label_en": "Plaza", "value": "plaza"},
		{"label": "Pusat Belanja", "label_en": "Mall", "value": "mall"},
		{"label": "Acara/Musiman", "label_en": "Event/Seasonal", "value": "event"},
	}
}

func (s *metaService) GetOccupations() ([]models.Occupation, error) {
	return s.metaRepo.GetAllOccupations()
}

func (s *metaService) GetCountries() []CountryItem {
	return []CountryItem{
		{Code: "ID", Name: "Indonesia", NameEn: "Indonesia", Flag: "🇮🇩"},
		{Code: "CN", Name: "Tiongkok / China", NameEn: "China", Flag: "🇨🇳"},
		{Code: "JP", Name: "Jepang", NameEn: "Japan", Flag: "🇯🇵"},
		{Code: "KR", Name: "Korea Selatan", NameEn: "South Korea", Flag: "🇰🇷"},
		{Code: "US", Name: "Amerika Serikat", NameEn: "United States", Flag: "🇺🇸"},
		{Code: "SG", Name: "Singapura", NameEn: "Singapore", Flag: "🇸🇬"},
		{Code: "MY", Name: "Malaysia", NameEn: "Malaysia", Flag: "🇲🇾"},
		{Code: "TH", Name: "Thailand", NameEn: "Thailand", Flag: "🇹🇭"},
		{Code: "VN", Name: "Vietnam", NameEn: "Vietnam", Flag: "🇻🇳"},
		{Code: "TW", Name: "Taiwan", NameEn: "Taiwan", Flag: "🇹🇼"},
		{Code: "DE", Name: "Jerman", NameEn: "Germany", Flag: "🇩🇪"},
		{Code: "GB", Name: "Inggris / Britania Raya", NameEn: "United Kingdom", Flag: "🇬🇧"},
		{Code: "FR", Name: "Prancis", NameEn: "France", Flag: "🇫🇷"},
		{Code: "IT", Name: "Italia", NameEn: "Italy", Flag: "🇮🇹"},
		{Code: "AU", Name: "Australia", NameEn: "Australia", Flag: "🇦🇺"},
		{Code: "IN", Name: "India", NameEn: "India", Flag: "🇮🇳"},
		{Code: "TR", Name: "Turki", NameEn: "Turkey", Flag: "🇹🇷"},
		{Code: "SA", Name: "Arab Saudi", NameEn: "Saudi Arabia", Flag: "🇸🇦"},
		{Code: "AE", Name: "Uni Emirat Arab", NameEn: "United Arab Emirates", Flag: "🇦🇪"},
		{Code: "NL", Name: "Belanda", NameEn: "Netherlands", Flag: "🇳🇱"},
		{Code: "CH", Name: "Swiss", NameEn: "Switzerland", Flag: "🇨🇭"},
		{Code: "ES", Name: "Spanyol", NameEn: "Spain", Flag: "🇪🇸"},
		{Code: "RU", Name: "Rusia", NameEn: "Russia", Flag: "🇷🇺"},
		{Code: "CA", Name: "Kanada", NameEn: "Canada", Flag: "🇨🇦"},
		{Code: "PH", Name: "Filipina", NameEn: "Philippines", Flag: "🇵🇭"},
		{Code: "BR", Name: "Brasil", NameEn: "Brazil", Flag: "🇧🇷"},
		{Code: "NZ", Name: "Selandia Baru", NameEn: "New Zealand", Flag: "🇳🇿"},
		{Code: "OTHER", Name: "Lainnya", NameEn: "Other", Flag: "🌐"},
	}
}
