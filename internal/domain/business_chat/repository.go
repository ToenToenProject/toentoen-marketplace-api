package business_chat

import (
	"sync"
	"time"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type BusinessChatRepository interface {
	FindConversation(businessID, customerID string) (*models.DirectBusinessConversation, error)
	GetConversationByID(conversationID string) (*models.DirectBusinessConversation, error)
	CreateConversation(conv *models.DirectBusinessConversation, firstMsg *models.DirectBusinessMessage) error
	GetCustomerConversations(customerID string, limit int, cursor string) ([]models.CustomerBusinessConversationListItem, error)
	GetMerchantConversations(businessID string, limit int, cursor string) ([]models.MerchantBusinessConversationListItem, error)
	SaveMessage(message *models.DirectBusinessMessage) error
	UpdateConversationLastMessage(conversationID, preview string, lastMessageAt time.Time) error
	GetMessagesByConversationID(conversationID string, limit int, cursor string) ([]models.DirectBusinessMessage, error)
	MarkMessagesAsRead(conversationID string, readerType string) error
	VerifyMerchantAccess(businessID string, accountID string) (bool, error)
	GetMerchantStaffAccountIDs(businessID string) ([]string, error)
	GetAccountByID(accountID string) (*models.Account, error)
	GetBusinessByID(businessID string) (*models.Business, error)
	GetProductByID(productID string) (*models.Product, error)
	FindSignalForConversation(businessID, customerID string) *models.SignalPostReference
	GetSignalPostReferenceByID(signalID string) *models.SignalPostReference
	UpdateConversationSignalID(conversationID, signalID string) error
}

type businessChatRepository struct {
	db *gorm.DB
}

func NewBusinessChatRepository(db *gorm.DB) BusinessChatRepository {
	return &businessChatRepository{db: db}
}

var (
	hasSignalIDColTable     bool
	hasSignalIDColLastCheck time.Time
	hasSignalIDColMu        sync.RWMutex
)

func (r *businessChatRepository) hasSignalIDColumn() bool {
	hasSignalIDColMu.RLock()
	if time.Since(hasSignalIDColLastCheck) < 30*time.Second {
		defer hasSignalIDColMu.RUnlock()
		return hasSignalIDColTable
	}
	hasSignalIDColMu.RUnlock()

	hasSignalIDColMu.Lock()
	defer hasSignalIDColMu.Unlock()
	if time.Since(hasSignalIDColLastCheck) < 30*time.Second {
		return hasSignalIDColTable
	}

	var exists bool
	err := r.db.Raw("SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'direct_business_conversations' AND column_name = 'signal_id')").Scan(&exists).Error
	hasSignalIDColTable = (err == nil && exists)
	hasSignalIDColLastCheck = time.Now()
	return hasSignalIDColTable
}

func (r *businessChatRepository) FindSignalForConversation(businessID, customerID string) *models.SignalPostReference {
	// 1. Cek apakah ada record di signal_conversations untuk business & customer ini
	var sig models.Signal
	err := r.db.Preload("Images").
		Joins("JOIN signal_conversations sc ON sc.signal_id = signals.signal_id").
		Where("sc.business_id = ? AND signals.customer_id = ?", businessID, customerID).
		Order("sc.created_at DESC").
		First(&sig).Error
	if err == nil && sig.SignalID != "" {
		return sig.ToPostReference()
	}

	// 2. Fallback: ambil sinyal aktif atau terbaru milik customer ini
	err = r.db.Preload("Images").
		Where("customer_id = ?", customerID).
		Order("created_at DESC").
		First(&sig).Error
	if err == nil && sig.SignalID != "" {
		return sig.ToPostReference()
	}

	return nil
}

func (r *businessChatRepository) GetSignalPostReferenceByID(signalID string) *models.SignalPostReference {
	if signalID == "" {
		return nil
	}
	var sig models.Signal
	if err := r.db.Preload("Images").Where("signal_id = ?", signalID).First(&sig).Error; err == nil && sig.SignalID != "" {
		return sig.ToPostReference()
	}
	return nil
}

func (r *businessChatRepository) UpdateConversationSignalID(conversationID, signalID string) error {
	if !r.hasSignalIDColumn() {
		return nil
	}
	return r.db.Model(&models.DirectBusinessConversation{}).
		Where("conversation_id = ?", conversationID).
		Update("signal_id", signalID).Error
}

func (r *businessChatRepository) FindConversation(businessID, customerID string) (*models.DirectBusinessConversation, error) {
	var conv models.DirectBusinessConversation
	q := r.db.Preload("Business").Preload("Customer")
	if !r.hasSignalIDColumn() {
		q = q.Omit("signal_id", "SignalID")
	}
	err := q.Where("business_id = ? AND customer_id = ?", businessID, customerID).First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *businessChatRepository) GetConversationByID(conversationID string) (*models.DirectBusinessConversation, error) {
	var conv models.DirectBusinessConversation
	q := r.db.Preload("Business").Preload("Customer")
	if !r.hasSignalIDColumn() {
		q = q.Omit("signal_id", "SignalID")
	}
	err := q.Where("conversation_id = ?", conversationID).First(&conv).Error
	if err == nil {
		return &conv, nil
	}

	// Fallback jika conversationID berasal dari signal_conversations (e.g. CONV-xxx)
	var sigConv models.SignalConversation
	if errSig := r.db.Preload("Business").Where("conversation_id = ?", conversationID).First(&sigConv).Error; errSig == nil {
		var sig models.Signal
		custID := ""
		if errS := r.db.Select("customer_id").Where("signal_id = ?", sigConv.SignalID).First(&sig).Error; errS == nil {
			custID = sig.CustomerID
		}
		var cust models.Account
		if custID != "" {
			_ = r.db.Where("account_id = ?", custID).First(&cust).Error
		}
		return &models.DirectBusinessConversation{
			ConversationID: sigConv.ConversationID,
			BusinessID:     sigConv.BusinessID,
			CustomerID:     custID,
			SignalID:       &sigConv.SignalID,
			Business:       sigConv.Business,
			Customer:       &cust,
			CreatedAt:      sigConv.CreatedAt,
			UpdatedAt:      sigConv.CreatedAt,
		}, nil
	}

	return nil, err
}

func (r *businessChatRepository) CreateConversation(conv *models.DirectBusinessConversation, firstMsg *models.DirectBusinessMessage) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txConv := tx
		if !r.hasSignalIDColumn() {
			txConv = txConv.Omit("signal_id", "SignalID")
		}
		if err := txConv.Create(conv).Error; err != nil {
			return err
		}
		if firstMsg != nil {
			if err := tx.Create(firstMsg).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *businessChatRepository) GetCustomerConversations(customerID string, limit int, cursor string) ([]models.CustomerBusinessConversationListItem, error) {
	var convs []models.DirectBusinessConversation
	query := r.db.
		Preload("Business").
		Where("customer_id = ?", customerID)

	if !r.hasSignalIDColumn() {
		query = query.Omit("signal_id", "SignalID")
	}

	if cursor != "" {
		var cursorConv models.DirectBusinessConversation
		if err := r.db.Select("last_message_at").Where("conversation_id = ?", cursor).First(&cursorConv).Error; err == nil {
			query = query.Where("last_message_at < ?", cursorConv.LastMessageAt)
		}
	}

	err := query.Order("last_message_at DESC").Limit(limit).Find(&convs).Error
	if err != nil {
		return nil, err
	}

	var results []models.CustomerBusinessConversationListItem
	var bizIDs []string
	for _, c := range convs {
		if c.BusinessID != "" {
			bizIDs = append(bizIDs, c.BusinessID)
		}
	}
	verifMap := r.getBusinessVerificationInfo(bizIDs)

	for _, c := range convs {
		var unread int64
		r.db.Model(&models.DirectBusinessMessage{}).
			Where("conversation_id = ? AND sender_type = 'merchant' AND is_read = false", c.ConversationID).
			Count(&unread)

		var bizName string
		var bizLogo *string
		if c.Business != nil {
			if c.Business.BusinessName != nil {
				bizName = *c.Business.BusinessName
			}
			bizLogo = c.Business.LogoURL
		}

		var isVerif, hasNIB bool
		if v, ok := verifMap[c.BusinessID]; ok {
			isVerif = v.IsVerified
			hasNIB = v.HasNIB
		}

		var signalRef *models.SignalPostReference
		if c.SignalID != nil && *c.SignalID != "" {
			signalRef = r.GetSignalPostReferenceByID(*c.SignalID)
		}
		if signalRef == nil {
			signalRef = r.FindSignalForConversation(c.BusinessID, c.CustomerID)
		}

		var sigID, sigTitle, sigDesc string
		if signalRef != nil {
			sigID = signalRef.SignalID
			sigTitle = signalRef.Title
			sigDesc = signalRef.Description
		}

		results = append(results, models.CustomerBusinessConversationListItem{
			ConversationID:      c.ConversationID,
			BusinessID:          c.BusinessID,
			BusinessName:        bizName,
			BusinessLogo:        bizLogo,
			IsVerified:          isVerif,
			HasNIB:              hasNIB,
			LastMessagePreview:  c.LastMessagePreview,
			LastMessageAt:       c.LastMessageAt,
			UnreadCount:         unread,
			SignalID:            sigID,
			SignalTitle:         sigTitle,
			SignalDescription:   sigDesc,
			SignalPostReference: signalRef,
		})
	}

	return results, nil
}

func (r *businessChatRepository) GetMerchantConversations(businessID string, limit int, cursor string) ([]models.MerchantBusinessConversationListItem, error) {
	var convs []models.DirectBusinessConversation
	query := r.db.
		Preload("Customer").
		Where("business_id = ?", businessID)

	if !r.hasSignalIDColumn() {
		query = query.Omit("signal_id", "SignalID")
	}

	if cursor != "" {
		var cursorConv models.DirectBusinessConversation
		if err := r.db.Select("last_message_at").Where("conversation_id = ?", cursor).First(&cursorConv).Error; err == nil {
			query = query.Where("last_message_at < ?", cursorConv.LastMessageAt)
		}
	}

	err := query.Order("last_message_at DESC").Limit(limit).Find(&convs).Error
	if err != nil {
		return nil, err
	}

	verifMap := r.getBusinessVerificationInfo([]string{businessID})
	isVerif := verifMap[businessID].IsVerified
	hasNIB := verifMap[businessID].HasNIB

	var results []models.MerchantBusinessConversationListItem
	for _, c := range convs {
		var unread int64
		r.db.Model(&models.DirectBusinessMessage{}).
			Where("conversation_id = ? AND sender_type = 'customer' AND is_read = false", c.ConversationID).
			Count(&unread)

		var custName string
		var custAvatar *string
		if c.Customer != nil {
			if c.Customer.FirstName != "" {
				custName = c.Customer.FirstName
				if c.Customer.LastName != "" {
					custName += " " + c.Customer.LastName
				}
			} else {
				custName = c.Customer.Username
			}
			custAvatar = c.Customer.ProfilePictureUrl
		}

		var signalRef *models.SignalPostReference
		if c.SignalID != nil && *c.SignalID != "" {
			signalRef = r.GetSignalPostReferenceByID(*c.SignalID)
		}
		if signalRef == nil {
			signalRef = r.FindSignalForConversation(c.BusinessID, c.CustomerID)
		}

		var sigID, sigTitle, sigDesc string
		if signalRef != nil {
			sigID = signalRef.SignalID
			sigTitle = signalRef.Title
			sigDesc = signalRef.Description
		}

		results = append(results, models.MerchantBusinessConversationListItem{
			ConversationID:      c.ConversationID,
			BusinessID:          c.BusinessID,
			CustomerID:          c.CustomerID,
			CustomerName:        custName,
			CustomerAvatar:      custAvatar,
			IsVerified:          isVerif,
			HasNIB:              hasNIB,
			LastMessagePreview:  c.LastMessagePreview,
			LastMessageAt:       c.LastMessageAt,
			UnreadCount:         unread,
			SignalID:            sigID,
			SignalTitle:         sigTitle,
			SignalDescription:   sigDesc,
			SignalPostReference: signalRef,
		})
	}

	return results, nil
}

func (r *businessChatRepository) getBusinessVerificationInfo(businessIDs []string) map[string]struct{ IsVerified, HasNIB bool } {
	resMap := make(map[string]struct{ IsVerified, HasNIB bool })
	if len(businessIDs) == 0 {
		return resMap
	}
	type Row struct {
		BusinessID string `gorm:"column:business_id"`
		IsVerified bool   `gorm:"column:is_verified"`
		HasNIB     bool   `gorm:"column:has_nib"`
	}
	var rows []Row
	query := `
		SELECT 
			b.business_id,
			COALESCE(
				o.kyc_status = 'verified' OR EXISTS (
					SELECT 1 
					FROM business_has_tags bht 
					JOIN business_tags bt ON bt.tag_id = bht.tag_id 
					WHERE bht.business_id = b.business_id 
					  AND (LOWER(bt.name) LIKE '%verif%' OR LOWER(bt.name) LIKE '%(nib)%')
				),
				false
			) AS is_verified,
			COALESCE(b.nib IS NOT NULL AND TRIM(b.nib) != '', false) AS has_nib
		FROM businesses b
		LEFT JOIN owners o ON o.account_id = b.account_id
		WHERE b.business_id IN (?)
	`
	if err := r.db.Raw(query, businessIDs).Scan(&rows).Error; err == nil {
		for _, row := range rows {
			resMap[row.BusinessID] = struct{ IsVerified, HasNIB bool }{
				IsVerified: row.IsVerified,
				HasNIB:     row.HasNIB,
			}
		}
	}
	return resMap
}

func (r *businessChatRepository) SaveMessage(message *models.DirectBusinessMessage) error {
	return r.db.Create(message).Error
}

func (r *businessChatRepository) UpdateConversationLastMessage(conversationID, preview string, lastMessageAt time.Time) error {
	return r.db.Model(&models.DirectBusinessConversation{}).
		Where("conversation_id = ?", conversationID).
		Updates(map[string]interface{}{
			"last_message_preview": preview,
			"last_message_at":      lastMessageAt,
			"updated_at":           time.Now(),
		}).Error
}

func (r *businessChatRepository) GetMessagesByConversationID(conversationID string, limit int, cursor string) ([]models.DirectBusinessMessage, error) {
	var messages []models.DirectBusinessMessage

	if cursor != "" {
		var cursorMsg models.DirectBusinessMessage
		if err := r.db.Select("created_at").Where("message_id = ?", cursor).First(&cursorMsg).Error; err == nil {
			// Mengambil N pesan tepat sebelum kursor (scroll ke atas)
			var reverseMessages []models.DirectBusinessMessage
			err := r.db.Preload("ProductDetail").
				Where("conversation_id = ? AND created_at < ?", conversationID, cursorMsg.CreatedAt).
				Order("created_at DESC").
				Limit(limit).
				Find(&reverseMessages).Error
			if err != nil {
				return nil, err
			}
			for i := len(reverseMessages) - 1; i >= 0; i-- {
				messages = append(messages, reverseMessages[i])
			}
			return messages, nil
		}
	}

	// KETIKA CURSOR KOSONG (Initial Load saat buka chat room):
	// Ambil N pesan TERAKHIR (terbaru) dengan ORDER BY created_at DESC LIMIT limit, lalu dibalik ke ASC!
	var reverseMessages []models.DirectBusinessMessage
	err := r.db.Preload("ProductDetail").
		Where("conversation_id = ?", conversationID).
		Order("created_at DESC").
		Limit(limit).
		Find(&reverseMessages).Error
	if err != nil {
		return nil, err
	}
	for i := len(reverseMessages) - 1; i >= 0; i-- {
		messages = append(messages, reverseMessages[i])
	}

	// Fallback jika tidak ada pesan di direct_business_messages tapi percakapan berasal dari signal_conversations
	if len(messages) == 0 {
		var sigMsgs []models.SignalMessage
		q := r.db.Where("conversation_id = ?", conversationID).Order("created_at ASC")
		if limit > 0 {
			q = q.Limit(limit)
		}
		if err := q.Find(&sigMsgs).Error; err == nil && len(sigMsgs) > 0 {
			for _, sm := range sigMsgs {
				senderName := "Pelanggan"
				if sm.SenderType == "merchant" {
					senderName = "Penjual"
				}
				mediaURL := sm.AttachmentURL
				messages = append(messages, models.DirectBusinessMessage{
					MessageID:      sm.MessageID,
					ConversationID: sm.ConversationID,
					SenderID:       sm.SenderID,
					SenderType:     sm.SenderType,
					SenderName:     senderName,
					MessageType:    "text",
					Content:        sm.MessageText,
					MediaURL:       mediaURL,
					IsRead:         sm.IsRead,
					CreatedAt:      sm.CreatedAt,
				})
			}
		}
	}

	return messages, nil
}

func (r *businessChatRepository) MarkMessagesAsRead(conversationID string, readerType string) error {
	targetSenderType := "merchant"
	if readerType == "merchant" {
		targetSenderType = "customer"
	}

	now := time.Now()
	return r.db.Model(&models.DirectBusinessMessage{}).
		Where("conversation_id = ? AND sender_type = ? AND is_read = false", conversationID, targetSenderType).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": &now,
		}).Error
}

func (r *businessChatRepository) VerifyMerchantAccess(businessID string, accountID string) (bool, error) {
	var count int64
	// 1. Cek apakah owner
	if err := r.db.Model(&models.Business{}).Where("business_id = ? AND account_id = ?", businessID, accountID).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}

	// 2. Cek apakah assigned business admin yang aktif
	if err := r.db.Model(&models.BusinessAdminAssignment{}).
		Where("business_id = ? AND admin_id = ? AND status IN ('accepted', 'active', 'scheduled_for_deletion')", businessID, accountID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *businessChatRepository) GetMerchantStaffAccountIDs(businessID string) ([]string, error) {
	var ids []string

	// 1. Owner account ID
	var business models.Business
	if err := r.db.Select("account_id").Where("business_id = ?", businessID).First(&business).Error; err == nil && business.AccountID != "" {
		ids = append(ids, business.AccountID)
	}

	// 2. Active assigned business admins
	var assignments []models.BusinessAdminAssignment
	if err := r.db.Select("admin_id").
		Where("business_id = ? AND status IN ('accepted', 'active', 'scheduled_for_deletion')", businessID).
		Find(&assignments).Error; err == nil {
		for _, a := range assignments {
			ids = append(ids, a.AdminID)
		}
	}

	return ids, nil
}

func (r *businessChatRepository) GetAccountByID(accountID string) (*models.Account, error) {
	var account models.Account
	err := r.db.Where("account_id = ?", accountID).First(&account).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (r *businessChatRepository) GetBusinessByID(businessID string) (*models.Business, error) {
	var business models.Business
	err := r.db.Where("business_id = ?", businessID).First(&business).Error
	if err != nil {
		return nil, err
	}
	return &business, nil
}

func (r *businessChatRepository) GetProductByID(productID string) (*models.Product, error) {
	var product models.Product
	err := r.db.Where("product_id = ?", productID).First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}
