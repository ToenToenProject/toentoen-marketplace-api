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
	groups, err := s.repo.GetGroups(businessID)
	if err != nil {
		return nil, err
	}
	if s.repo.HasActivePromotion(businessID) {
		virtualPromo := models.ProductGroup{
			ProductGroupID: "promotion",
			BusinessID:     businessID,
			Name:           "Promosi",
			DisplayOrder:   0,
			IsActive:       true,
		}
		groups = append([]models.ProductGroup{virtualPromo}, groups...)
	}
	return groups, nil
}

func NewProductGroupService(repo ProductGroupRepository, notifService notification.NotificationService) ProductGroupService {
	return &productGroupService{repo, notifService}
}

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

	// Unspecified display order (0) defaults to the end of the list.
	order := input.DisplayOrder
	if order == 0 {
		maxOrder := s.repo.GetMaxDisplayOrder(businessID)
		order = maxOrder + 1
	}

	group := models.ProductGroup{
		ProductGroupID: utils.GenerateUUID("GRP"),
		BusinessID:     businessID,
		Name:           input.Name,
		DisplayOrder:   order,
		IsActive:       true,
	}

	return s.repo.CreateGroup(&group)
}

func (s *productGroupService) UpdateGroup(requesterID, role, businessID, productGroupID string, input models.UpdateGroupInput) error {
	if productGroupID == "promotion" {
		return errors.New("grup 'Promosi' adalah grup sistem otomatis dan tidak dapat diubah secara langsung")
	}

	if role == "business_admin" && !s.repo.HasPermission(requesterID, businessID, models.PermCreateProductGroup) {
		return errors.New("forbidden: Anda tidak memiliki izin untuk mengubah grup produk")
	}

	group, err := s.repo.GetGroupByID(businessID, productGroupID)
	if err != nil {
		return errors.New("grup produk tidak ditemukan")
	}

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

	// Iterate groups with updated display order.
	for _, item := range input.Orders {
		if item.ProductGroupID == "promotion" {
			continue // Lewati grup virtual promosi
		}
		err := s.repo.UpdateDisplayOrder(businessID, item.ProductGroupID, item.DisplayOrder)
		if err != nil {
			// Return error on failure.
			return errors.New("gagal mengurutkan sebagian grup")
		}
	}
	return nil
}

func (s *productGroupService) UpdateGroupContents(requesterID, role, businessID, productGroupID string, input models.UpdateGroupContentsInput) error {
	if productGroupID == "promotion" {
		return errors.New("isi grup 'Promosi' dikelola secara otomatis melalui fitur Promosi")
	}

	if role == "business_admin" && !s.repo.HasPermission(requesterID, businessID, models.PermCreateProductGroup) {
		return errors.New("forbidden: Anda tidak memiliki izin untuk mengubah isi grup produk")
	}

	_, err := s.repo.GetGroupByID(businessID, productGroupID)
	if err != nil {
		return errors.New("grup produk tidak ditemukan")
	}

	if len(input.RemoveProductIDs) > 0 {
		fmt.Printf("DEBUG: Menghapus %d produk. ID: %v\n", len(input.RemoveProductIDs), input.RemoveProductIDs)
		if err := s.repo.RemoveProductsFromGroup(businessID, input.RemoveProductIDs); err != nil {
			return err
		}
	}

	// Add products to this group.
	if len(input.AddProductIDs) > 0 {
		if err := s.repo.AddProductsToGroup(businessID, productGroupID, input.AddProductIDs); err != nil {
			return err
		}
	}

	return nil
}

func (s *productGroupService) DeleteGroup(requesterID, role, businessID, productGroupID string) error {
	if productGroupID == "promotion" {
		return errors.New("grup 'Promosi' adalah grup sistem otomatis dan tidak dapat dihapus")
	}

	if role == "business_admin" && !s.repo.HasPermission(requesterID, businessID, models.PermDeleteProductGroup) {
		return errors.New("forbidden: Anda tidak memiliki izin untuk menghapus grup produk")
	}

	count, err := s.repo.CountProductsInGroup(businessID, productGroupID)
	if err != nil {
		return errors.New("gagal memverifikasi isi grup produk")
	}

	// Prevent deleting product group if it contains active products.
	if count > 0 {
		return errors.New("tidak bisa menghapus grup karena masih ada produk di dalamnya. Silakan pindahkan atau hapus produknya terlebih dahulu")
	}

	if err := s.repo.DeleteGroup(businessID, productGroupID); err != nil {
		return err
	}

	// Alert owner if staff deletes product group.
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
