package catalog

import (
	"errors"
	"fmt"
	"strings"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

type ProductGroupService interface {
	GetGroups(businessID string) ([]models.ProductGroup, error)
	CreateGroup(businessID string, input models.CreateGroupInput) error
	UpdateGroup(businessID, productGroupID string, input models.UpdateGroupInput) error
	ReorderGroups(businessID string, input models.ReorderGroupsInput) error
	DeleteGroup(businessID, productGroupID string) error
	UpdateGroupContents(businessID, productGroupID string, input models.UpdateGroupContentsInput) error
}

type productGroupService struct {
	repo ProductGroupRepository
}

func (s *productGroupService) GetGroups(businessID string) ([]models.ProductGroup, error) {
	return s.repo.GetGroups(businessID)
}

func NewProductGroupService(repo ProductGroupRepository) ProductGroupService {
	return &productGroupService{repo}
}

// 1. CREATE GROUP (Cegah Duplikat)
func (s *productGroupService) CreateGroup(businessID string, input models.CreateGroupInput) error {
	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return errors.New("business_id tidak boleh kosong")
	}

	if !s.repo.CheckBusinessExists(businessID) {
		return fmt.Errorf("bisnis dengan ID '%s' tidak ditemukan", businessID)
	}

	cleanName := strings.TrimSpace(input.Name)
	if cleanName == "" {
		return errors.New("nama grup tidak boleh kosong")
	}
	input.Name = cleanName

	if s.repo.CheckNameExists(businessID, input.Name) {
		return errors.New("grup dengan nama ini sudah ada di toko Anda")
	}

	// 🪄 LOGIKA PINTAR: Jika tidak diisi (0), otomatis taruh di urutan paling bawah
	order := input.DisplayOrder
	if order == 0 {
		maxOrder := s.repo.GetMaxDisplayOrder(businessID)
		order = maxOrder + 1
	}

	group := models.ProductGroup{
		ProductGroupID: utils.GenerateUUID("GRP"),
		BusinessID:     businessID,
		Name:           input.Name,
		DisplayOrder:   order, // 👈 Gunakan variabel order yang sudah dihitung
		IsActive:       true,
	}

	return s.repo.CreateGroup(&group)
}

// 2. UPDATE GROUP METADATA
func (s *productGroupService) UpdateGroup(businessID, productGroupID string, input models.UpdateGroupInput) error {
	group, err := s.repo.GetGroupByID(businessID, productGroupID)
	if err != nil {
		return errors.New("grup produk tidak ditemukan")
	}

	// Cek duplikat jika user mengubah nama
	if input.Name != "" && input.Name != group.Name {
		if s.repo.CheckNameExists(businessID, input.Name) {
			return errors.New("grup dengan nama ini sudah ada")
		}
		group.Name = input.Name
	}

	if input.DisplayOrder != nil {
		group.DisplayOrder = *input.DisplayOrder
	}
	if input.IsActive != nil {
		group.IsActive = *input.IsActive
	}

	return s.repo.UpdateGroup(group)
}

func (s *productGroupService) ReorderGroups(businessID string, input models.ReorderGroupsInput) error {
	// Lakukan perulangan untuk setiap grup yang urutannya berubah
	for _, item := range input.Orders {
		err := s.repo.UpdateDisplayOrder(businessID, item.ProductGroupID, item.DisplayOrder)
		if err != nil {
			// Jika satu gagal, kembalikan error (Ideally pakai DB Transaction, tapi ini cukup aman)
			return errors.New("gagal mengurutkan sebagian grup")
		}
	}
	return nil
}

// 3. MENGUBAH ISI GRUP (Add / Remove Product)
func (s *productGroupService) UpdateGroupContents(businessID, productGroupID string, input models.UpdateGroupContentsInput) error {
	// Pastikan grupnya valid
	_, err := s.repo.GetGroupByID(businessID, productGroupID)
	if err != nil {
		return errors.New("grup produk tidak ditemukan")
	}

	// 🪄 PERBAIKAN: Panggil kembali fungsi repository untuk menghapus dari grup
	if len(input.RemoveProductIDs) > 0 {
		fmt.Printf("DEBUG: Menghapus %d produk. ID: %v\n", len(input.RemoveProductIDs), input.RemoveProductIDs)
		if err := s.repo.RemoveProductsFromGroup(businessID, input.RemoveProductIDs); err != nil {
			return err
		}
	}

	// Masukkan produk baru ke dalam grup ini
	if len(input.AddProductIDs) > 0 {
		if err := s.repo.AddProductsToGroup(businessID, productGroupID, input.AddProductIDs); err != nil {
			return err
		}
	}

	return nil
}

// 4. DELETE GRUP (Validasi harus kosong)
func (s *productGroupService) DeleteGroup(businessID, productGroupID string) error {
	count, err := s.repo.CountProductsInGroup(businessID, productGroupID)
	if err != nil {
		return errors.New("gagal memverifikasi isi grup produk")
	}

	// LOGIKA PENTING: Cegah hapus jika ada isinya
	if count > 0 {
		return errors.New("tidak bisa menghapus grup karena masih ada produk di dalamnya. Silakan pindahkan atau hapus produknya terlebih dahulu")
	}

	return s.repo.DeleteGroup(businessID, productGroupID)
}
