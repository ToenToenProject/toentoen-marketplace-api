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
		// Helper function to delete and catch errors immediately
		del := func(condition string, arg interface{}, model interface{}) error {
			if err := tx.Where(condition, arg).Delete(model).Error; err != nil {
				return err
			}
			return nil
		}

		delIn := func(condition string, arg []string, model interface{}) error {
			if len(arg) == 0 {
				return nil
			}
			if err := tx.Where(condition, arg).Delete(model).Error; err != nil {
				return err
			}
			return nil
		}

		// 1. Delete by Role
		if role == "customer" {
			if err := del("account_id = ?", accountID, &models.DeviceToken{}); err != nil {
				return err
			}
			if err := del("account_id = ?", accountID, &models.ProductLike{}); err != nil {
				return err
			}
			if err := del("account_id = ?", accountID, &models.BusinessPostLike{}); err != nil {
				return err
			}
			if err := del("account_id = ?", accountID, &models.ProductWishlist{}); err != nil {
				return err
			}
			if err := del("account_id = ?", accountID, &models.BusinessBookmark{}); err != nil {
				return err
			}

			if err := del("account_id = ?", accountID, &models.ProductReview{}); err != nil {
				return err
			}
			if err := del("account_id = ?", accountID, &models.BusinessPostComment{}); err != nil {
				return err
			}
			if err := del("account_id = ?", accountID, &models.Review{}); err != nil {
				return err
			}
			if err := del("account_id = ?", accountID, &models.BusinessReviewComment{}); err != nil {
				return err
			}

			if err := del("account_id = ?", accountID, &models.BusinessReport{}); err != nil {
				return err
			}
			if err := del("account_id = ?", accountID, &models.BusinessPostReport{}); err != nil {
				return err
			}

			// Chat / Signals
			if err := del("sender_id = ?", accountID, &models.SignalMessage{}); err != nil {
				return err
			}

			var signalIDs []string
			if err := tx.Model(&models.Signal{}).Where("customer_id = ?", accountID).Pluck("signal_id", &signalIDs).Error; err != nil {
				return err
			}

			if len(signalIDs) > 0 {
				var convIDs []string
				if err := tx.Model(&models.SignalConversation{}).Where("signal_id IN ?", signalIDs).Pluck("conversation_id", &convIDs).Error; err != nil {
					return err
				}

				if err := delIn("conversation_id IN ?", convIDs, &models.SignalMessage{}); err != nil {
					return err
				}
				if err := delIn("signal_id IN ?", signalIDs, &models.SignalConversation{}); err != nil {
					return err
				}
				if err := delIn("signal_id IN ?", signalIDs, &models.SignalImage{}); err != nil {
					return err
				}
				if err := delIn("signal_id IN ?", signalIDs, &models.DemandResponse{}); err != nil {
					return err
				}
				if err := delIn("signal_id IN ?", signalIDs, &models.Signal{}); err != nil {
					return err
				}
			}

			if err := del("account_id = ?", accountID, &models.Customer{}); err != nil {
				return err
			}

		} else if role == "owner" {
			// Find all businesses
			var businesses []models.Business
			if err := tx.Where("account_id = ?", accountID).Find(&businesses).Error; err != nil {
				return err
			}

			for _, b := range businesses {
				deletedBusinessIDs = append(deletedBusinessIDs, b.BusinessID)

				// Find products
				var products []models.Product
				if err := tx.Where("business_id = ?", b.BusinessID).Find(&products).Error; err != nil {
					return err
				}
				for _, p := range products {
					deletedProductIDs = append(deletedProductIDs, p.ProductID)
					if err := del("product_id = ?", p.ProductID, &models.ProductVariant{}); err != nil {
						return err
					}
					if err := del("product_id = ?", p.ProductID, &models.ProductLike{}); err != nil {
						return err
					}
					if err := del("product_id = ?", p.ProductID, &models.ProductWishlist{}); err != nil {
						return err
					}
					if err := del("product_id = ?", p.ProductID, &models.ProductReview{}); err != nil {
						return err
					}
				}
				if err := del("business_id = ?", b.BusinessID, &models.ProductGroup{}); err != nil {
					return err
				}
				if err := del("business_id = ?", b.BusinessID, &models.Product{}); err != nil {
					return err
				}

				// Posts
				var posts []models.BusinessPost
				if err := tx.Where("business_id = ?", b.BusinessID).Find(&posts).Error; err != nil {
					return err
				}
				var postIDs []string
				for _, post := range posts {
					postIDs = append(postIDs, post.PostID)
				}
				if len(postIDs) > 0 {
					if err := delIn("post_id IN ?", postIDs, &models.BusinessPostLike{}); err != nil {
						return err
					}
					if err := delIn("post_id IN ?", postIDs, &models.BusinessPostComment{}); err != nil {
						return err
					}
					if err := delIn("post_id IN ?", postIDs, &models.BusinessPostReport{}); err != nil {
						return err
					}
					if err := del("business_id = ?", b.BusinessID, &models.BusinessPost{}); err != nil {
						return err
					}
				}

				// Others
				if err := del("business_id = ?", b.BusinessID, &models.BusinessHour{}); err != nil {
					return err
				}
				if err := del("business_id = ?", b.BusinessID, &models.BusinessAdminAssignment{}); err != nil {
					return err
				}

				var reviews []models.Review
				if err := tx.Where("business_id = ?", b.BusinessID).Find(&reviews).Error; err != nil {
					return err
				}
				for _, rev := range reviews {
					if err := del("review_id = ?", rev.ReviewID, &models.BusinessReviewComment{}); err != nil {
						return err
					}
				}
				if err := del("business_id = ?", b.BusinessID, &models.Review{}); err != nil {
					return err
				}

				if err := del("business_id = ?", b.BusinessID, &models.BusinessBookmark{}); err != nil {
					return err
				}
				if err := del("business_id = ?", b.BusinessID, &models.BusinessReport{}); err != nil {
					return err
				}
				if err := del("business_id = ?", b.BusinessID, &models.SignalConversation{}); err != nil {
					return err
				}
				if err := del("business_id = ?", b.BusinessID, &models.DemandResponse{}); err != nil {
					return err
				}
				if err := del("business_id = ?", b.BusinessID, &models.MerchantAnalytics{}); err != nil {
					return err
				}

				// Clear join tables
				if err := tx.Exec("DELETE FROM business_categories WHERE business_id = ?", b.BusinessID).Error; err != nil {
					return err
				}
				if err := tx.Exec("DELETE FROM business_tags WHERE business_id = ?", b.BusinessID).Error; err != nil {
					return err
				}

				if err := del("business_id = ?", b.BusinessID, &models.Business{}); err != nil {
					return err
				}
			}

			if err := del("account_id = ?", accountID, &models.Owner{}); err != nil {
				return err
			}

		} else if role == "business_admin" {
			if err := del("admin_id = ?", accountID, &models.BusinessAdminAssignment{}); err != nil {
				return err
			}
			if err := del("account_id = ?", accountID, &models.BusinessAdmin{}); err != nil {
				return err
			}
		}

		// 2. Finally delete the Account
		if err := del("account_id = ?", accountID, &models.Account{}); err != nil {
			return err
		}

		return nil
	})

	return deletedBusinessIDs, deletedProductIDs, err
}
