package account

import (
	"fmt"
	"strings"
	"time"
	"toentoen-backend/internal/domain/shared/kychelper"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/crypto"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	CheckUsernameExists(username string, excludeAccountID string) (bool, error)
	CheckEmailExists(email string, excludeAccountID string, roleGroup ...string) (bool, error)
	CheckPhoneNumberExists(phone string, excludeAccountID string, roleGroup ...string) (bool, error)
	CheckKtpNumberExists(ktp string, excludeAccountID string) (bool, error)
	FindReferralCode(code string) (*models.ReferralUser, error)
	UpdateConsent(accountID string, acceptedTermsAt *time.Time, termsVersion *string, acceptedPrivacyAt *time.Time, privacyVersion *string) error
	UpsertDeviceToken(accountID string, token string, platform string) error
	DeleteDeviceToken(accountID string, token string) error
	SyncVerifiedKYC(accountID string, role string) bool
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
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *profileRepository) GetAccountByIdentifier(identifier string) (*models.Account, error) {
	var account models.Account
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return nil, gorm.ErrRecordNotFound
	}

	// 1. Direct match by account_id (e.g. ACC-xxx)
	if strings.HasPrefix(identifier, "ACC-") || strings.HasPrefix(identifier, "ACC_") {
		if err := r.db.Where("account_id = ?", identifier).First(&account).Error; err == nil {
			return &account, nil
		}
	}

	// 2. Email match (case-insensitive)
	if strings.Contains(identifier, "@") {
		err := r.db.Where("LOWER(email) = LOWER(?) OR account_id = ?", identifier, identifier).First(&account).Error
		if err != nil {
			return nil, err
		}
		return &account, nil
	}

	// 3. Phone format variations (08xxx, 628xxx, +628xxx)
	clean := strings.TrimPrefix(identifier, "+")
	var phones []string
	if strings.HasPrefix(clean, "08") {
		phones = []string{clean, "62" + clean[1:], "+62" + clean[1:]}
	} else if strings.HasPrefix(clean, "628") {
		phones = []string{clean, "0" + clean[2:], "+" + clean}
	} else {
		phones = []string{identifier, "+" + identifier, clean}
	}

	// 4. Combined lookup: phone, username, or account_id
	err := r.db.Where("phone_number IN ? OR LOWER(username) = LOWER(?) OR account_id = ?", phones, identifier, identifier).First(&account).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *profileRepository) GetCustomerByAccountID(accountID string) (*models.Customer, error) {
	var customer models.Customer
	err := r.db.Where("account_id = ?", accountID).First(&customer).Error
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

func (r *profileRepository) GetOwnerByAccountID(accountID string) (*models.Owner, error) {
	var owner models.Owner
	err := r.db.Where("account_id = ?", accountID).First(&owner).Error
	if err != nil {
		return nil, err
	}
	return &owner, nil
}

func (r *profileRepository) GetBusinessAdminByAccountID(accountID string) (*models.BusinessAdmin, error) {
	var businessAdmin models.BusinessAdmin
	err := r.db.
		Preload("Assignments").
		Preload("Assignments.Business").
		Preload("Assignments.Shifts").
		Where("account_id = ?", accountID).
		First(&businessAdmin).Error
	if err != nil {
		return nil, err
	}
	return &businessAdmin, nil
}

func (r *profileRepository) SaveCustomerProfile(account *models.Account, customer *models.Customer) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit(clause.Associations).Save(account).Error; err != nil {
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
		if err := tx.Omit(clause.Associations).Save(account).Error; err != nil {
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
		if err := tx.Omit(clause.Associations).Save(account).Error; err != nil {
			return err
		}

		if err := tx.Save(businessAdmin).Error; err != nil {
			return err
		}
		return nil
	})
}

// DeleteAccountTx deletes the account and all associated data, returning lists of deleted business IDs and product IDs for Meilisearch cleanup
func (r *profileRepository) CheckUsernameExists(username string, excludeAccountID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Account{}).Where("LOWER(username) = LOWER(?) AND account_id != ?", strings.TrimSpace(username), excludeAccountID).Count(&count).Error
	return count > 0, err
}

func (r *profileRepository) CheckEmailExists(email string, excludeAccountID string, roleGroup ...string) (bool, error) {
	query := r.db.Model(&models.Account{}).Where("LOWER(email) = LOWER(?) AND account_id != ?", strings.TrimSpace(email), excludeAccountID)
	var group string
	if len(roleGroup) > 0 && roleGroup[0] != "" {
		group = roleGroup[0]
	} else if excludeAccountID != "" {
		// Dapatkan group dari akun yang sedang mengedit
		var acc models.Account
		if err := r.db.Select("role").Where("account_id = ?", excludeAccountID).First(&acc).Error; err == nil {
			group = models.RoleGroupOf(acc.Role)
		}
	}
	if roles := models.RolesInGroup(group); len(roles) > 0 {
		query = query.Where("role IN ?", roles)
	}
	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *profileRepository) CheckPhoneNumberExists(phone string, excludeAccountID string, roleGroup ...string) (bool, error) {
	clean := strings.TrimPrefix(strings.TrimSpace(phone), "+")
	var phones []string
	if strings.HasPrefix(clean, "08") {
		phones = []string{clean, "62" + clean[1:], "+62" + clean[1:]}
	} else if strings.HasPrefix(clean, "628") {
		phones = []string{clean, "0" + clean[2:], "+" + clean}
	} else {
		phones = []string{phone, "+" + phone, clean}
	}

	query := r.db.Model(&models.Account{}).Where("phone_number IN ? AND account_id != ?", phones, excludeAccountID)
	var group string
	if len(roleGroup) > 0 && roleGroup[0] != "" {
		group = roleGroup[0]
	} else if excludeAccountID != "" {
		var acc models.Account
		if err := r.db.Select("role").Where("account_id = ?", excludeAccountID).First(&acc).Error; err == nil {
			group = models.RoleGroupOf(acc.Role)
		}
	}
	if roles := models.RolesInGroup(group); len(roles) > 0 {
		query = query.Where("role IN ?", roles)
	}

	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *profileRepository) SyncVerifiedKYC(accountID string, role string) bool {
	return kychelper.SyncVerifiedKYCIfSiblingVerified(r.db, accountID, role)
}

func (r *profileRepository) CheckKtpNumberExists(ktp string, excludeAccountID string) (bool, error) {
	cleanKtp := strings.TrimSpace(ktp)
	if cleanKtp == "" {
		return false, nil
	}

	excludeIDs := []string{excludeAccountID}
	if siblings := kychelper.FindSiblingAccountIDs(r.db, excludeAccountID); len(siblings) > 0 {
		excludeIDs = append(excludeIDs, siblings...)
	}

	var ownerKtps []string
	if err := r.db.Model(&models.Owner{}).
		Where("ktp_number IS NOT NULL AND ktp_number != '' AND account_id NOT IN ?", excludeIDs).
		Pluck("ktp_number", &ownerKtps).Error; err != nil {
		return false, err
	}

	for _, k := range ownerKtps {
		dec, _ := crypto.Decrypt(k)
		if strings.TrimSpace(dec) == cleanKtp {
			return true, nil
		}
	}

	var adminKtps []string
	if err := r.db.Model(&models.BusinessAdmin{}).
		Where("ktp_number IS NOT NULL AND ktp_number != '' AND account_id NOT IN ?", excludeIDs).
		Pluck("ktp_number", &adminKtps).Error; err != nil {
		return false, err
	}

	for _, k := range adminKtps {
		dec, _ := crypto.Decrypt(k)
		if strings.TrimSpace(dec) == cleanKtp {
			return true, nil
		}
	}

	return false, nil
}

func (r *profileRepository) DeleteAccountTx(accountID string, role string) ([]string, []string, error) {
	var deletedBusinessIDs []string
	var deletedProductIDs []string

	err := r.db.Transaction(func(tx *gorm.DB) error {
		isRelationNotExist := func(err error) bool {
			if err == nil {
				return false
			}
			errMsg := strings.ToLower(err.Error())
			return strings.Contains(errMsg, "42p01") || strings.Contains(errMsg, "does not exist")
		}

		// Helper function to delete with savepoint protection so missing tables never abort transaction
		del := func(condition string, arg interface{}, model interface{}) error {
			sp := fmt.Sprintf("sp_del_%d", time.Now().UnixNano())
			tx.SavePoint(sp)
			if err := tx.Unscoped().Where(condition, arg).Delete(model).Error; err != nil {
				if isRelationNotExist(err) {
					tx.RollbackTo(sp)
					return nil
				}
				return err
			}
			return nil
		}

		delIn := func(condition string, arg []string, model interface{}) error {
			if len(arg) == 0 {
				return nil
			}
			sp := fmt.Sprintf("sp_delin_%d", time.Now().UnixNano())
			tx.SavePoint(sp)
			if err := tx.Unscoped().Where(condition, arg).Delete(model).Error; err != nil {
				if isRelationNotExist(err) {
					tx.RollbackTo(sp)
					return nil
				}
				return err
			}
			return nil
		}

		updateNull := func(model interface{}, condition string, arg interface{}, column string) error {
			sp := fmt.Sprintf("sp_upd_%d", time.Now().UnixNano())
			tx.SavePoint(sp)
			if err := tx.Model(model).Where(condition, arg).Update(column, nil).Error; err != nil {
				if isRelationNotExist(err) {
					tx.RollbackTo(sp)
					return nil
				}
				return err
			}
			return nil
		}

		delRaw := func(query string, arg interface{}) {
			sp := fmt.Sprintf("sp_raw_%d", time.Now().UnixNano())
			tx.SavePoint(sp)
			if err := tx.Exec(query, arg).Error; err != nil {
				tx.RollbackTo(sp)
			}
		}

		// 1. Clean up common interactions, tokens, and comments for any account
		if err := del("account_id = ?", accountID, &models.DeviceToken{}); err != nil { return err }
		if err := del("account_id = ?", accountID, &models.ProductLike{}); err != nil { return err }
		if err := del("account_id = ?", accountID, &models.ProductWishlist{}); err != nil { return err }
		if err := del("account_id = ?", accountID, &models.BusinessBookmark{}); err != nil { return err }
		if err := del("account_id = ?", accountID, &models.UserPreferredCategory{}); err != nil { return err }
		if err := del("account_id = ?", accountID, &models.UserCategoryAffinity{}); err != nil { return err }
		if err := del("account_id = ?", accountID, &models.UserPostInteraction{}); err != nil { return err }
		if err := del("recipient_account_id = ?", accountID, &models.UserNotification{}); err != nil { return err }

		// Product Comments & Discussions
		if err := del("account_id = ?", accountID, &models.ProductCommentLike{}); err != nil { return err }
		if err := updateNull(&models.ProductComment{}, "replied_to_account_id = ?", accountID, "replied_to_account_id"); err != nil { return err }
		if err := del("account_id = ?", accountID, &models.ProductComment{}); err != nil { return err }

		// Product Reviews & Comments
		if err := del("account_id = ?", accountID, &models.ProductReviewLike{}); err != nil { return err }
		if err := del("account_id = ?", accountID, &models.ProductReviewCommentLike{}); err != nil { return err }
		if err := updateNull(&models.ProductReviewComment{}, "replied_to_account_id = ?", accountID, "replied_to_account_id"); err != nil { return err }
		if err := del("account_id = ?", accountID, &models.ProductReviewComment{}); err != nil { return err }
		if err := del("account_id = ?", accountID, &models.ProductReview{}); err != nil { return err }

		// Business Reviews & Comments
		if err := del("account_id = ?", accountID, &models.BusinessReviewLike{}); err != nil { return err }
		if err := del("account_id = ?", accountID, &models.BusinessReviewCommentLike{}); err != nil { return err }
		if err := updateNull(&models.BusinessReviewComment{}, "replied_to_account_id = ?", accountID, "replied_to_account_id"); err != nil { return err }
		if err := del("account_id = ?", accountID, &models.BusinessReviewComment{}); err != nil { return err }
		if err := del("account_id = ?", accountID, &models.Review{}); err != nil { return err }

		// Business Posts Interactions
		if err := del("account_id = ?", accountID, &models.BusinessPostLike{}); err != nil { return err }
		if err := del("account_id = ?", accountID, &models.BusinessPostCommentLike{}); err != nil { return err }
		if err := updateNull(&models.BusinessPostComment{}, "replied_to_account_id = ?", accountID, "replied_to_account_id"); err != nil { return err }
		if err := del("account_id = ?", accountID, &models.BusinessPostComment{}); err != nil { return err }

		// Direct Business Chat (messages first, then conversations)
		if err := del("sender_id = ?", accountID, &models.DirectBusinessMessage{}); err != nil { return err }
		if err := del("customer_id = ?", accountID, &models.DirectBusinessConversation{}); err != nil { return err }

		// CS Live Chat (messages first, then sessions)
		if err := del("sender_id = ?", accountID, &models.CSChatMessage{}); err != nil { return err }
		if err := del("account_id = ?", accountID, &models.CSChatSession{}); err != nil { return err }
		if err := updateNull(&models.CSChatSession{}, "cs_admin_id = ?", accountID, "cs_admin_id"); err != nil { return err }

		// Support Tickets (attachments cascade via FK, but explicit for safety)
		delRaw("DELETE FROM support_ticket_attachments WHERE ticket_id IN (SELECT ticket_id FROM support_tickets WHERE account_id = ?)", accountID)
		if err := del("account_id = ?", accountID, &models.SupportTicket{}); err != nil { return err }
		if err := updateNull(&models.SupportTicket{}, "resolved_by = ?", accountID, "resolved_by"); err != nil { return err }

		// Business Transfer Requests
		if err := del("current_owner_id = ?", accountID, &models.BusinessTransferRequest{}); err != nil { return err }
		if err := del("target_account_id = ?", accountID, &models.BusinessTransferRequest{}); err != nil { return err }

		// Reports & Evidences
		delRaw("DELETE FROM business_report_evidences WHERE report_id IN (SELECT report_id FROM business_reports WHERE account_id = ?)", accountID)
		delRaw("DELETE FROM business_post_report_evidences WHERE report_id IN (SELECT report_id FROM business_post_reports WHERE account_id = ?)", accountID)
		if err := del("account_id = ?", accountID, &models.Report{}); err != nil { return err }
		if err := del("account_id = ?", accountID, &models.BusinessReport{}); err != nil { return err }
		if err := del("account_id = ?", accountID, &models.BusinessPostReport{}); err != nil { return err }
		if err := del("account_id = ?", accountID, &models.BusinessPostCommentReport{}); err != nil { return err }
		if err := del("account_id = ?", accountID, &models.BusinessReviewReport{}); err != nil { return err }
		if err := del("account_id = ?", accountID, &models.BusinessReviewCommentReport{}); err != nil { return err }
		if err := del("account_id = ?", accountID, &models.ProductReviewReport{}); err != nil { return err }
		if err := del("account_id = ?", accountID, &models.ProductReviewCommentReport{}); err != nil { return err }
		if err := del("account_id = ?", accountID, &models.ProductCommentReport{}); err != nil { return err }

		// Legacy interaction and notification tables if they exist
		delRaw("DELETE FROM notifications WHERE recipient_account_id = ?", accountID)
		delRaw("DELETE FROM likes WHERE account_id = ?", accountID)
		delRaw("DELETE FROM bookmarks WHERE account_id = ?", accountID)
		// Legacy demand_signals v1 (account_id ON DELETE CASCADE, but table may not exist)
		delRaw("DELETE FROM demand_signals WHERE account_id = ?", accountID)

		// 2. Delete Customer & Demand Signal Entities (if any)
		if err := del("sender_id = ?", accountID, &models.SignalMessage{}); err != nil {
			return err
		}
		if err := del("merchant_id = ?", accountID, &models.SignalConversation{}); err != nil {
			return err
		}

		var signalIDs []string
		if err := tx.Model(&models.Signal{}).Where("customer_id = ?", accountID).Pluck("signal_id", &signalIDs).Error; err == nil && len(signalIDs) > 0 {
			var convIDs []string
			if err := tx.Model(&models.SignalConversation{}).Where("signal_id IN ?", signalIDs).Pluck("conversation_id", &convIDs).Error; err == nil && len(convIDs) > 0 {
				if err := delIn("conversation_id IN ?", convIDs, &models.SignalMessage{}); err != nil {
					return err
				}
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

		// 3. Delete Business & Owner Entities (if any businesses are owned by this account)
		var businesses []models.Business
		if err := tx.Where("account_id = ?", accountID).Find(&businesses).Error; err == nil && len(businesses) > 0 {
			for _, b := range businesses {
				deletedBusinessIDs = append(deletedBusinessIDs, b.BusinessID)

				// Find products
				var products []models.Product
				if err := tx.Where("business_id = ?", b.BusinessID).Find(&products).Error; err == nil {
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

						// Comments on this product
						var commentIDs []string
						_ = tx.Model(&models.ProductComment{}).Where("product_id = ?", p.ProductID).Pluck("comment_id", &commentIDs).Error
						if len(commentIDs) > 0 {
							if err := delIn("comment_id IN ?", commentIDs, &models.ProductCommentLike{}); err != nil {
								return err
							}
							if err := delIn("comment_id IN ?", commentIDs, &models.ProductCommentReport{}); err != nil {
								return err
							}
						}
						if err := del("product_id = ?", p.ProductID, &models.ProductComment{}); err != nil {
							return err
						}

						// Reviews on this product
						var reviewIDs []string
						_ = tx.Model(&models.ProductReview{}).Where("product_id = ?", p.ProductID).Pluck("review_id", &reviewIDs).Error
						if len(reviewIDs) > 0 {
							var revCommentIDs []string
							_ = tx.Model(&models.ProductReviewComment{}).Where("review_id IN ?", reviewIDs).Pluck("comment_id", &revCommentIDs).Error
							if len(revCommentIDs) > 0 {
								if err := delIn("comment_id IN ?", revCommentIDs, &models.ProductReviewCommentLike{}); err != nil {
									return err
								}
							}
							if err := delIn("review_id IN ?", reviewIDs, &models.ProductReviewComment{}); err != nil {
								return err
							}
							if err := delIn("review_id IN ?", reviewIDs, &models.ProductReviewLike{}); err != nil {
								return err
							}
							if err := delIn("review_id IN ?", reviewIDs, &models.ProductReviewReport{}); err != nil {
								return err
							}
						}
						if err := del("product_id = ?", p.ProductID, &models.ProductReview{}); err != nil {
							return err
						}
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
				if err := tx.Where("business_id = ?", b.BusinessID).Find(&posts).Error; err == nil {
					var postIDs []string
					for _, post := range posts {
						postIDs = append(postIDs, post.PostID)
					}
					if len(postIDs) > 0 {
						if err := delIn("post_id IN ?", postIDs, &models.BusinessPostLike{}); err != nil {
							return err
						}

						var postCommentIDs []string
						_ = tx.Model(&models.BusinessPostComment{}).Where("post_id IN ?", postIDs).Pluck("comment_id", &postCommentIDs).Error
						if len(postCommentIDs) > 0 {
							if err := delIn("comment_id IN ?", postCommentIDs, &models.BusinessPostCommentLike{}); err != nil {
								return err
							}
						}
						if err := delIn("post_id IN ?", postIDs, &models.BusinessPostComment{}); err != nil {
							return err
						}
						delRaw("DELETE FROM business_post_report_evidences WHERE report_id IN (SELECT report_id FROM business_post_reports WHERE post_id IN ?)", postIDs)
						if err := delIn("post_id IN ?", postIDs, &models.BusinessPostReport{}); err != nil {
							return err
						}
						if err := delIn("post_id IN ?", postIDs, &models.UserPostInteraction{}); err != nil {
							return err
						}
						if err := del("business_id = ?", b.BusinessID, &models.BusinessPost{}); err != nil {
							return err
						}
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
				if err := tx.Where("business_id = ?", b.BusinessID).Find(&reviews).Error; err == nil {
					for _, rev := range reviews {
						var bRevCommentIDs []string
						_ = tx.Model(&models.BusinessReviewComment{}).Where("review_id = ?", rev.ReviewID).Pluck("comment_id", &bRevCommentIDs).Error
						if len(bRevCommentIDs) > 0 {
							if err := delIn("comment_id IN ?", bRevCommentIDs, &models.BusinessReviewCommentLike{}); err != nil {
								return err
							}
						}
						if err := del("review_id = ?", rev.ReviewID, &models.BusinessReviewComment{}); err != nil {
							return err
						}
						if err := del("review_id = ?", rev.ReviewID, &models.BusinessReviewLike{}); err != nil {
							return err
						}
					}
				}
				if err := del("business_id = ?", b.BusinessID, &models.Review{}); err != nil {
					return err
				}

				if err := del("business_id = ?", b.BusinessID, &models.BusinessBookmark{}); err != nil {
					return err
				}
				delRaw("DELETE FROM business_report_evidences WHERE report_id IN (SELECT report_id FROM business_reports WHERE business_id = ?)", b.BusinessID)
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
				_ = del("business_id = ?", b.BusinessID, &models.UserPostInteraction{})
				_ = del("business_id = ?", b.BusinessID, &models.DirectBusinessConversation{})
				_ = del("business_id = ?", b.BusinessID, &models.BusinessTransferRequest{})
				_ = updateNull(&models.CSChatSession{}, "business_id = ?", b.BusinessID, "business_id")
				_ = updateNull(&models.SupportTicket{}, "business_id = ?", b.BusinessID, "business_id")

				// Clear legacy tables if present
				delRaw("DELETE FROM notifications WHERE recipient_business_id = ?", b.BusinessID)
				delRaw("DELETE FROM likes WHERE business_id = ?", b.BusinessID)
				delRaw("DELETE FROM transactions WHERE business_id = ?", b.BusinessID)

				// Clear join tables
				if err := tx.Exec("DELETE FROM business_has_categories WHERE business_id = ?", b.BusinessID).Error; err != nil {
					return err
				}
				if err := tx.Exec("DELETE FROM business_has_tags WHERE business_id = ?", b.BusinessID).Error; err != nil {
					return err
				}

				if err := del("business_id = ?", b.BusinessID, &models.Business{}); err != nil {
					return err
				}
			}
		}

		if err := del("account_id = ?", accountID, &models.Owner{}); err != nil {
			return err
		}

		// 4. Delete Business Admin Entities (if any assignments or profile exist)
		if err := del("admin_id = ?", accountID, &models.BusinessAdminAssignment{}); err != nil {
			return err
		}
		if err := updateNull(&models.BusinessAdminAssignment{}, "supervisor_id = ?", accountID, "supervisor_id"); err != nil {
			return err
		}
		if err := updateNull(&models.BusinessAdmin{}, "invited_by = ?", accountID, "invited_by"); err != nil {
			return err
		}
		if err := del("account_id = ?", accountID, &models.BusinessAdmin{}); err != nil {
			return err
		}

		// 5. Update references in accounts table
		if err := updateNull(&models.Account{}, "status_updated_by = ?", accountID, "status_updated_by"); err != nil {
			return err
		}

		// 6. Finally delete the Account row
		if err := del("account_id = ?", accountID, &models.Account{}); err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM accounts WHERE account_id = ?", accountID).Error; err != nil {
			return err
		}

		return nil
	})

	return deletedBusinessIDs, deletedProductIDs, err
}

func (r *profileRepository) FindReferralCode(code string) (*models.ReferralUser, error) {
	var user models.ReferralUser
	cleanCode := strings.TrimSpace(code)
	if cleanCode == "" {
		return nil, gorm.ErrRecordNotFound
	}
	err := r.db.Table("operations.users").
		Where("LOWER(referral_code) = LOWER(?) AND is_active = true", cleanCode).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}


func (r *profileRepository) UpdateConsent(accountID string, acceptedTermsAt *time.Time, termsVersion *string, acceptedPrivacyAt *time.Time, privacyVersion *string) error {
	updates := map[string]interface{}{
		"accepted_terms_at":   acceptedTermsAt,
		"terms_version":      termsVersion,
		"accepted_privacy_at": acceptedPrivacyAt,
		"privacy_version":    privacyVersion,
		"updated_at":         time.Now(),
	}
	return r.db.Model(&models.Account{}).Where("account_id = ?", accountID).Updates(updates).Error
}

func (r *profileRepository) UpsertDeviceToken(accountID string, token string, platform string) error {
	cleanAccountID := strings.TrimSpace(accountID)
	cleanToken := strings.TrimSpace(token)
	cleanPlatform := strings.ToLower(strings.TrimSpace(platform))
	if cleanAccountID == "" || cleanToken == "" {
		return fmt.Errorf("account_id dan token tidak boleh kosong")
	}
	if cleanPlatform == "" {
		cleanPlatform = "android"
	}

	deviceToken := models.DeviceToken{
		AccountID:  cleanAccountID,
		FCMToken:   cleanToken,
		DeviceType: cleanPlatform,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "fcm_token"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"account_id":  cleanAccountID,
			"device_type": cleanPlatform,
			"updated_at":  time.Now(),
		}),
	}).Create(&deviceToken).Error
}

func (r *profileRepository) DeleteDeviceToken(accountID string, token string) error {
	cleanAccountID := strings.TrimSpace(accountID)
	cleanToken := strings.TrimSpace(token)
	if cleanAccountID == "" {
		return fmt.Errorf("account_id tidak boleh kosong")
	}

	query := r.db.Where("account_id = ?", cleanAccountID)
	if cleanToken != "" {
		query = query.Where("fcm_token = ?", cleanToken)
	}
	return query.Delete(&models.DeviceToken{}).Error
}

