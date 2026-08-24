package chat

import (
	"time"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

type SupportChatRepository interface {
	FindActiveSessionByAccountID(accountID string) (*models.CSChatSession, error)
	GetSessionByID(sessionID string) (*models.CSChatSession, error)
	CreateSessionWithGreeting(session *models.CSChatSession, greetingMessage *models.CSChatMessage) error
	SaveMessage(message *models.CSChatMessage) error
	UpdateSessionLastMessage(sessionID string, preview string, lastMessageAt time.Time, status string, csAdminID *string) error
	GetMessagesBySessionID(sessionID string, limit int, cursor string) ([]models.CSChatMessage, error)
	MarkMessagesAsRead(sessionID string, readerType string) error
	EndSession(sessionID string, endedBy string, rating *int, feedback *string, endMessage *models.CSChatMessage) error
	GetAdminQueue(status string, limit int, cursor string) ([]models.CSChatSession, error)
	GetAdminHistory(limit int, cursor string) ([]models.CSChatSession, error)
	AssignCSToSession(sessionID string, csAdminID string) error
	GetAccountByID(accountID string) (*models.Account, error)
}

type supportChatRepository struct {
	db *gorm.DB
}

func NewSupportChatRepository(db *gorm.DB) SupportChatRepository {
	return &supportChatRepository{db: db}
}

func (r *supportChatRepository) FindActiveSessionByAccountID(accountID string) (*models.CSChatSession, error) {
	var session models.CSChatSession
	err := r.db.
		Preload("Account").
		Preload("Business").
		Preload("CSAdmin").
		Where("account_id = ? AND status IN ('open', 'in_progress') AND user_hidden = false", accountID).
		Order("created_at DESC").
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *supportChatRepository) GetSessionByID(sessionID string) (*models.CSChatSession, error) {
	var session models.CSChatSession
	err := r.db.
		Preload("Account").
		Preload("Business").
		Preload("CSAdmin").
		Where("session_id = ?", sessionID).
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *supportChatRepository) CreateSessionWithGreeting(session *models.CSChatSession, greetingMessage *models.CSChatMessage) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(session).Error; err != nil {
			return err
		}
		if greetingMessage != nil {
			if err := tx.Create(greetingMessage).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *supportChatRepository) SaveMessage(message *models.CSChatMessage) error {
	return r.db.Create(message).Error
}

func (r *supportChatRepository) UpdateSessionLastMessage(sessionID string, preview string, lastMessageAt time.Time, status string, csAdminID *string) error {
	updates := map[string]interface{}{
		"last_message_preview": preview,
		"last_message_at":      lastMessageAt,
		"updated_at":           time.Now(),
	}
	if status != "" {
		updates["status"] = status
	}
	if csAdminID != nil && *csAdminID != "" {
		updates["cs_admin_id"] = *csAdminID
	}

	return r.db.Model(&models.CSChatSession{}).Where("session_id = ?", sessionID).Updates(updates).Error
}

func (r *supportChatRepository) GetMessagesBySessionID(sessionID string, limit int, cursor string) ([]models.CSChatMessage, error) {
	var messages []models.CSChatMessage
	query := r.db.Where("session_id = ?", sessionID)

	if cursor != "" {
		var cursorMsg models.CSChatMessage
		if err := r.db.Select("created_at").Where("message_id = ?", cursor).First(&cursorMsg).Error; err == nil {
			query = query.Where("created_at < ?", cursorMsg.CreatedAt)
		}
	}

	err := query.Order("created_at ASC").Limit(limit).Find(&messages).Error
	return messages, err
}

func (r *supportChatRepository) MarkMessagesAsRead(sessionID string, readerType string) error {
	targetSenderType := "cs"
	if readerType == "cs" {
		targetSenderType = "user"
	}

	now := time.Now()
	return r.db.Model(&models.CSChatMessage{}).
		Where("session_id = ? AND sender_type = ? AND is_read = false", sessionID, targetSenderType).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": &now,
		}).Error
}

func (r *supportChatRepository) EndSession(sessionID string, endedBy string, rating *int, feedback *string, endMessage *models.CSChatMessage) error {
	now := time.Now()
	return r.db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"status":      "ended",
			"ended_at":    &now,
			"ended_by":    endedBy,
			"user_hidden": true,
			"updated_at":  now,
		}
		if rating != nil {
			updates["rating"] = *rating
		}
		if feedback != nil {
			updates["feedback"] = *feedback
		}

		if err := tx.Model(&models.CSChatSession{}).Where("session_id = ?", sessionID).Updates(updates).Error; err != nil {
			return err
		}

		if endMessage != nil {
			if err := tx.Create(endMessage).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *supportChatRepository) GetAdminQueue(status string, limit int, cursor string) ([]models.CSChatSession, error) {
	var sessions []models.CSChatSession
	query := r.db.
		Preload("Account").
		Preload("Business").
		Preload("CSAdmin")

	if status != "" {
		query = query.Where("status = ?", status)
	} else {
		query = query.Where("status IN ('open', 'in_progress')")
	}

	if cursor != "" {
		var cursorSession models.CSChatSession
		if err := r.db.Select("last_message_at").Where("session_id = ?", cursor).First(&cursorSession).Error; err == nil {
			query = query.Where("last_message_at < ?", cursorSession.LastMessageAt)
		}
	}

	err := query.Order("last_message_at DESC").Limit(limit).Find(&sessions).Error
	return sessions, err
}

func (r *supportChatRepository) GetAdminHistory(limit int, cursor string) ([]models.CSChatSession, error) {
	var sessions []models.CSChatSession
	query := r.db.
		Preload("Account").
		Preload("Business").
		Preload("CSAdmin").
		Where("status = 'ended'")

	if cursor != "" {
		var cursorSession models.CSChatSession
		if err := r.db.Select("ended_at").Where("session_id = ?", cursor).First(&cursorSession).Error; err == nil && cursorSession.EndedAt != nil {
			query = query.Where("ended_at < ?", cursorSession.EndedAt)
		}
	}

	err := query.Order("ended_at DESC").Limit(limit).Find(&sessions).Error
	return sessions, err
}

func (r *supportChatRepository) AssignCSToSession(sessionID string, csAdminID string) error {
	return r.db.Model(&models.CSChatSession{}).
		Where("session_id = ?", sessionID).
		Updates(map[string]interface{}{
			"cs_admin_id": csAdminID,
			"status":      "in_progress",
			"updated_at":  time.Now(),
		}).Error
}

func (r *supportChatRepository) GetAccountByID(accountID string) (*models.Account, error) {
	var account models.Account
	err := r.db.Where("account_id = ?", accountID).First(&account).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}
