package account

import (
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type ProfileRepository interface {
	GetAccountByID(accountID string) (*models.Account, error)
	GetAccountByIdentifier(identifier string) (*models.Account, error)
	GetCustomerByAccountID(accountID string) (*models.Customer, error)
	GetOwnerByAccountID(accountID string) (*models.Owner, error)
	GetBusinessAdminByAccountID(accountID string) (*models.BusinessAdmin, error)

	SaveCustomerProfile(account *models.Account, customer *models.Customer) error
	SaveOwnerProfile(account *models.Account, owner *models.Owner) error
	SaveBusinessAdminProfile(account *models.Account, businessAdmin *models.BusinessAdmin) error

	DeleteAccountTx(accountID string, role string) ([]string, []string, error)
}

type profileRepository struct {
	db *gorm.DB
}

func NewProfileRepository(db *gorm.DB) ProfileRepository {
	return &profileRepository{db}
}

func (r *profileRepository) GetAccountByID(accountID string) (*models.Account, error) {
	var account models.Account
	err := r.db.Where("account_id = ?", accountID).First(&account).Error
	return &account, err
}

func (r *profileRepository) GetAccountByIdentifier(identifier string) (*models.Account, error) {
	var account models.Account
	err := r.db.Where("email = ? OR phone_number = ?", identifier, identifier).First(&account).Error
	return &account, err
}

func (r *profileRepository) GetCustomerByAccountID(accountID string) (*models.Customer, error) {
	var customer models.Customer
	err := r.db.Where("account_id = ?", accountID).First(&customer).Error
	return &customer, err
}

func (r *profileRepository) GetOwnerByAccountID(accountID string) (*models.Owner, error) {
	var owner models.Owner
	err := r.db.Where("account_id = ?", accountID).First(&owner).Error
	return &owner, err
}

func (r *profileRepository) GetBusinessAdminByAccountID(accountID string) (*models.BusinessAdmin, error) {
	var businessAdmin models.BusinessAdmin
	err := r.db.Where("account_id = ?", accountID).First(&businessAdmin).Error
	return &businessAdmin, err
}

func (r *profileRepository) SaveCustomerProfile(account *models.Account, customer *models.Customer) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(account).Error; err != nil {
			return err
		}

		if err := tx.Save(customer).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *profileRepository) SaveOwnerProfile(account *models.Account, owner *models.Owner) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(account).Error; err != nil {
			return err
		}

		if err := tx.Save(owner).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *profileRepository) SaveBusinessAdminProfile(account *models.Account, businessAdmin *models.BusinessAdmin) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(account).Error; err != nil {
			return err
		}

		if err := tx.Save(businessAdmin).Error; err != nil {
			return err
		}
		return nil
	})
}

// DeleteAccountTx deletes the account and all associated data, returning lists of deleted business IDs and product IDs for Meilisearch cleanup
func (r *profileRepository) DeleteAccountTx(accountID string, role string) ([]string, []string, error) {
	var deletedBusinessIDs []string
	var deletedProductIDs []string

	err := r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Delete by Role
		if role == "customer" {
			tx.Where("account_id = ?", accountID).Delete(&models.DeviceToken{})
			tx.Where("account_id = ?", accountID).Delete(&models.ProductLike{})
			tx.Where("account_id = ?", accountID).Delete(&models.BusinessPostLike{})
			tx.Where("account_id = ?", accountID).Delete(&models.ProductWishlist{})
			tx.Where("account_id = ?", accountID).Delete(&models.BusinessBookmark{})
			
			tx.Where("account_id = ?", accountID).Delete(&models.ProductComment{})
			tx.Where("account_id = ?", accountID).Delete(&models.BusinessPostComment{})
			tx.Where("account_id = ?", accountID).Delete(&models.Review{})
			tx.Where("account_id = ?", accountID).Delete(&models.BusinessReviewComment{})
			
			tx.Where("reporter_id = ?", accountID).Delete(&models.BusinessReport{})
			tx.Where("reporter_id = ?", accountID).Delete(&models.BusinessPostReport{})

			// Chat / Signals
			tx.Where("sender_id = ?", accountID).Delete(&models.SignalMessage{})
			tx.Where("customer_id = ?", accountID).Delete(&models.SignalConversation{})

			var signalIDs []string
			tx.Model(&models.Signal{}).Where("customer_id = ?", accountID).Pluck("signal_id", &signalIDs)
			if len(signalIDs) > 0 {
				tx.Where("signal_id IN ?", signalIDs).Delete(&models.SignalImage{})
				tx.Where("signal_id IN ?", signalIDs).Delete(&models.DemandResponse{})
				tx.Where("signal_id IN ?", signalIDs).Delete(&models.Signal{})
			}
			
			tx.Where("account_id = ?", accountID).Delete(&models.Customer{})

		} else if role == "owner" {
			// Find all businesses
			var businesses []models.Business
			tx.Where("owner_id = ?", accountID).Find(&businesses)

			for _, b := range businesses {
				deletedBusinessIDs = append(deletedBusinessIDs, b.BusinessID)

				// Find products
				var products []models.Product
				tx.Where("business_id = ?", b.BusinessID).Find(&products)
				for _, p := range products {
					deletedProductIDs = append(deletedProductIDs, p.ProductID)
					tx.Where("product_id = ?", p.ProductID).Delete(&models.ProductVariant{})
					tx.Where("product_id = ?", p.ProductID).Delete(&models.ProductLike{})
					tx.Where("product_id = ?", p.ProductID).Delete(&models.ProductWishlist{})
					tx.Where("product_id = ?", p.ProductID).Delete(&models.ProductComment{})
				}
				tx.Where("business_id = ?", b.BusinessID).Delete(&models.ProductGroup{})
				tx.Where("business_id = ?", b.BusinessID).Delete(&models.Product{})

				// Posts
				var posts []models.BusinessPost
				tx.Where("business_id = ?", b.BusinessID).Find(&posts)
				var postIDs []string
				for _, post := range posts {
					postIDs = append(postIDs, post.PostID)
				}
				if len(postIDs) > 0 {
					tx.Where("post_id IN ?", postIDs).Delete(&models.BusinessPostLike{})
					tx.Where("post_id IN ?", postIDs).Delete(&models.BusinessPostComment{})
					tx.Where("post_id IN ?", postIDs).Delete(&models.BusinessPostReport{})
					tx.Where("business_id = ?", b.BusinessID).Delete(&models.BusinessPost{})
				}

				// Others
				tx.Where("business_id = ?", b.BusinessID).Delete(&models.BusinessHour{})
				tx.Where("business_id = ?", b.BusinessID).Delete(&models.BusinessAdminAssignment{})
				
				var reviews []models.Review
				tx.Where("business_id = ?", b.BusinessID).Find(&reviews)
				for _, rev := range reviews {
					tx.Where("review_id = ?", rev.ReviewID).Delete(&models.BusinessReviewComment{})
				}
				tx.Where("business_id = ?", b.BusinessID).Delete(&models.Review{})

				tx.Where("business_id = ?", b.BusinessID).Delete(&models.BusinessBookmark{})
				tx.Where("business_id = ?", b.BusinessID).Delete(&models.BusinessReport{})
				tx.Where("business_id = ?", b.BusinessID).Delete(&models.SignalConversation{})
				tx.Where("business_id = ?", b.BusinessID).Delete(&models.DemandResponse{})
				tx.Where("business_id = ?", b.BusinessID).Delete(&models.MerchantAnalytics{})

				// Clear join tables
				tx.Exec("DELETE FROM business_categories WHERE business_id = ?", b.BusinessID)
				tx.Exec("DELETE FROM business_tags WHERE business_id = ?", b.BusinessID)

				tx.Where("business_id = ?", b.BusinessID).Delete(&models.Business{})
			}

			tx.Where("account_id = ?", accountID).Delete(&models.Owner{})

		} else if role == "business_admin" {
			tx.Where("admin_id = ?", accountID).Delete(&models.BusinessAdminAssignment{})
			tx.Where("account_id = ?", accountID).Delete(&models.BusinessAdmin{})
		}

		// 2. Finally delete the Account
		if err := tx.Where("account_id = ?", accountID).Delete(&models.Account{}).Error; err != nil {
			return err
		}

		return nil
	})

	return deletedBusinessIDs, deletedProductIDs, err
}
