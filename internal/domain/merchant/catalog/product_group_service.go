package catalog

import (
	"errors"
	"fmt"
	"strings"
	"toentoen-backend/internal/domain/notification"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/database"
	"toentoen-backend/pkg/utils"
)

type ProductGroupService interface {
	GetGroups(businessID string) ([]models.ProductGroup, error)
	CreateGroup(requesterID, role, businessID string, input models.CreateGroupInput) error
	UpdateGroup(requesterID, role, businessID, productGroupID string, input models.UpdateGroupInput) error
	ReorderGroups(requesterID, role, businessID string, input models.ReorderGroupsInput) error
	DeleteGroup(requesterID, role, businessID, productGroupID string) error
	UpdateGroupContents(requesterID, role, businessID, productGroupID string, input models.UpdateGroupContentsInput) error
}

type productGroupService struct {
	repo         ProductGroupRepository
	notifService notification.NotificationService
}

func (s *productGroupService) GetGroups(businessID string) ([]models.ProductGroup, error) {
	return s.repo.GetGroups(businessID)
}

func NewProductGroupService(repo ProductGroupRepository, notifService notification.NotificationService) ProductGroupService {
	return &productGroupService{repo, notifService}
}

// 1. CREATE GROUP (Cegah Duplikat)
func (s *productGroupService) CreateGroup(requesterID, role, businessID string, input models.CreateGroupInput) error {
	if role == "business_admin" && !s.repo.HasPermission(requesterID, businessID, models.PermCreateProductGroup) {
		return errors.New("forbidden: Anda tidak memiliki izin untuk membuat grup produk baru")
	}

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
func (s *productGroupService) UpdateGroup(requesterID, role, businessID, productGroupID string, input models.UpdateGroupInput) error {
	if role == "business_admin" && !s.repo.HasPermission(requesterID, businessID, models.PermCreateProductGroup) {
		return errors.New("forbidden: Anda tidak memiliki izin untuk mengubah grup produk")
	}

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

func (s *productGroupService) ReorderGroups(requesterID, role, businessID string, input models.ReorderGroupsInput) error {
	if role == "business_admin" && !s.repo.HasPermission(requesterID, businessID, models.PermReorderProductGroups) {
		return errors.New("forbidden: Anda tidak memiliki izin untuk mengatur urutan grup produk")
	}

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
func (s *productGroupService) UpdateGroupContents(requesterID, role, businessID, productGroupID string, input models.UpdateGroupContentsInput) error {
	if role == "business_admin" && !s.repo.HasPermission(requesterID, businessID, models.PermCreateProductGroup) {
		return errors.New("forbidden: Anda tidak memiliki izin untuk mengubah isi grup produk")
	}

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
func (s *productGroupService) DeleteGroup(requesterID, role, businessID, productGroupID string) error {
	if role == "business_admin" && !s.repo.HasPermission(requesterID, businessID, models.PermDeleteProductGroup) {
		return errors.New("forbidden: Anda tidak memiliki izin untuk menghapus grup produk")
	}

	count, err := s.repo.CountProductsInGroup(businessID, productGroupID)
	if err != nil {
		return errors.New("gagal memverifikasi isi grup produk")
	}

	// LOGIKA PENTING: Cegah hapus jika ada isinya
	if count > 0 {
		return errors.New("tidak bisa menghapus grup karena masih ada produk di dalamnya. Silakan pindahkan atau hapus produknya terlebih dahulu")
	}

	if err := s.repo.DeleteGroup(businessID, productGroupID); err != nil {
		return err
	}

	// 🔔 Notifikasi ke Owner jika staf menghapus grup produk
	if role == "business_admin" {
		notification.NotifyOwnerStaffActivity(
			database.DB,
			s.notifService,
			businessID,
			requesterID,
			"delete_product_group",
			"⚠️ Staf Menghapus Grup Produk",
			"Staf toko Anda baru saja menghapus sebuah etalase grup produk.",
			map[string]interface{}{
				"product_group_id": productGroupID,
			},
		)
	}

	return nil
}
