package chat

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"toentoen-backend/internal/domain/realtime"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/crypto"
	"toentoen-backend/pkg/utils"
)

type SupportChatService interface {
	InitiateChat(accountID string, userRole string, businessID *string) (*models.CSChatSession, *models.CSChatMessage, bool, error)
	GetActiveChat(accountID string) (*models.CSChatSession, error)
	GetChatMessages(sessionID string, requesterID string, isCS bool, limit int, cursor string) ([]models.CSChatMessage, error)
	SendMessage(sessionID string, senderID string, senderType string, senderName string, input models.SendCSMessageInput) (*models.CSChatMessage, error)
	EndChat(sessionID string, requesterID string, isCS bool, input models.EndCSChatInput) error
	AdminGetQueue(status string, limit int, cursor string) ([]models.CSChatSession, error)
	AdminGetHistory(limit int, cursor string) ([]models.CSChatSession, error)
	AdminAssignSession(sessionID string, csAdminID string) error
}

type supportChatService struct {
	repo SupportChatRepository
	hub  *realtime.Hub
}

func NewSupportChatService(repo SupportChatRepository, hub *realtime.Hub) SupportChatService {
	return &supportChatService{
		repo: repo,
		hub:  hub,
	}
}

func (s *supportChatService) InitiateChat(accountID string, userRole string, businessID *string) (*models.CSChatSession, *models.CSChatMessage, bool, error) {
	// 1. Cek apakah user sudah memiliki sesi aktif yang belum di-end/hide
	existingSession, err := s.repo.FindActiveSessionByAccountID(accountID)
	if err == nil && existingSession != nil {
		// Sudah ada sesi aktif, kembalikan sesi tersebut (isNew = false)
		existingSession.LastMessagePreview, _ = crypto.Decrypt(existingSession.LastMessagePreview)
		messages, _ := s.repo.GetMessagesBySessionID(existingSession.SessionID, 1, "")
		var lastMsg *models.CSChatMessage
		if len(messages) > 0 {
			msg := messages[len(messages)-1]
			msg.Content, _ = crypto.Decrypt(msg.Content)
			msg.MediaURL, _ = crypto.DecryptPointer(msg.MediaURL)
			lastMsg = &msg
		}
		return existingSession, lastMsg, false, nil
	}

	// 2. Ambil data akun user untuk mendapatkan nama tampilan
	account, err := s.repo.GetAccountByID(accountID)
	displayName := "Kak"
	if err == nil && account != nil {
		if account.FirstName != "" {
			displayName = account.FirstName
		} else if account.Username != "" {
			displayName = account.Username
		}
	}

	now := time.Now()
	// Prefix LCS: Live Chat Session
	sessionID := utils.GenerateUUID("LCS")
	greetingContent := fmt.Sprintf("Halo %s! Selamat datang di Layanan Bantuan ToenToen. Ada yang bisa kami bantu hari ini?", displayName)

	// Prefix LCM: Live Chat Message
	greetingMsg := &models.CSChatMessage{
		MessageID:   utils.GenerateUUID("LCM"),
		SessionID:   sessionID,
		SenderID:    "SYSTEM",
		SenderType:  "system",
		SenderName:  "ToenToen Support",
		MessageType: "template",
		Content:     greetingContent,
		IsRead:      false,
		CreatedAt:   now,
	}

	if businessID != nil && strings.TrimSpace(*businessID) == "" {
		businessID = nil
	}

	newSession := &models.CSChatSession{
		SessionID:          sessionID,
		AccountID:          accountID,
		UserRole:           userRole,
		BusinessID:         businessID,
		Status:             "open",
		UserHidden:         false,
		LastMessagePreview: greetingContent,
		LastMessageAt:      now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// Enkripsi pesan awal dan preview sebelum disimpan
	encGreeting, _ := crypto.Encrypt(greetingContent)
	greetingMsgToSave := *greetingMsg
	greetingMsgToSave.Content = encGreeting

	newSessionToSave := *newSession
	newSessionToSave.LastMessagePreview = encGreeting

	if err := s.repo.CreateSessionWithGreeting(&newSessionToSave, &greetingMsgToSave); err != nil {
		fmt.Printf("⚠️ Error CreateSessionWithGreeting: %v\n", err)
		return nil, nil, false, errors.New("Gagal membuat sesi live chat")
	}

	// 3. Broadcast event real-time ke CS via WebSocket jika hub tersedia
	if s.hub != nil {
		event := realtime.WSEvent{
			Type: "CS_CHAT_NEW_SESSION",
			Data: newSession,
		}
		s.hub.BroadcastToRole("platform_admin", event)
	}

	return newSession, greetingMsg, true, nil
}

func (s *supportChatService) GetActiveChat(accountID string) (*models.CSChatSession, error) {
	session, err := s.repo.FindActiveSessionByAccountID(accountID)
	if err != nil {
		return nil, nil
	}
	if session != nil {
		session.LastMessagePreview, _ = crypto.Decrypt(session.LastMessagePreview)
	}
	return session, nil
}

func (s *supportChatService) GetChatMessages(sessionID string, requesterID string, isCS bool, limit int, cursor string) ([]models.CSChatMessage, error) {
	session, err := s.repo.GetSessionByID(sessionID)
	if err != nil {
		return nil, errors.New("Sesi live chat tidak ditemukan")
	}

	// Validasi akses: hanya pemilik sesi chat atau admin CS yang boleh membaca
	if !isCS && session.AccountID != requesterID {
		return nil, errors.New("Anda tidak memiliki akses ke percakapan ini")
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	readerType := "user"
	if isCS {
		readerType = "cs"
	}
	_ = s.repo.MarkMessagesAsRead(sessionID, readerType)

	messages, err := s.repo.GetMessagesBySessionID(sessionID, limit, cursor)
	if err != nil {
		return nil, err
	}

	for i := range messages {
		messages[i].Content, _ = crypto.Decrypt(messages[i].Content)
		messages[i].MediaURL, _ = crypto.DecryptPointer(messages[i].MediaURL)
	}

	return messages, nil
}

func (s *supportChatService) SendMessage(sessionID string, senderID string, senderType string, senderName string, input models.SendCSMessageInput) (*models.CSChatMessage, error) {
	session, err := s.repo.GetSessionByID(sessionID)
	if err != nil {
		return nil, errors.New("Sesi live chat tidak ditemukan")
	}

	if session.Status == "ended" {
		return nil, errors.New("Sesi live chat ini sudah diakhiri. Silakan mulai sesi live chat baru.")
	}

	// Validasi sender
	if senderType == "user" && session.AccountID != senderID {
		return nil, errors.New("Anda tidak memiliki akses ke percakapan ini")
	}

	msgType := "text"
	if input.MessageType != "" {
		msgType = input.MessageType
	}
	if input.MediaURL != nil && *input.MediaURL != "" && msgType == "text" {
		msgType = "image"
	}

	now := time.Now()
	message := &models.CSChatMessage{
		MessageID:   utils.GenerateUUID("LCM"),
		SessionID:   sessionID,
		SenderID:    senderID,
		SenderType:  senderType,
		SenderName:  senderName,
		MessageType: msgType,
		Content:     strings.TrimSpace(input.Content),
		MediaURL:    input.MediaURL,
		IsRead:      false,
		CreatedAt:   now,
	}

	// Enkripsi sebelum disimpan
	msgToSave := *message
	msgToSave.Content, _ = crypto.Encrypt(message.Content)
	msgToSave.MediaURL, _ = crypto.EncryptPointer(message.MediaURL)

	if err := s.repo.SaveMessage(&msgToSave); err != nil {
		return nil, errors.New("Gagal mengirim pesan")
	}

	// Update metadata session
	newStatus := session.Status
	var csAdminIDToUpdate *string
	if senderType == "cs" {
		newStatus = "in_progress"
		if session.CSAdminID == nil || *session.CSAdminID == "" {
			csAdminIDToUpdate = &senderID
		}
	}

	encPreview, _ := crypto.Encrypt(message.Content)
	_ = s.repo.UpdateSessionLastMessage(sessionID, encPreview, now, newStatus, csAdminIDToUpdate)

	// Broadcast via WebSocket (kirim plaintext ke websocket client)
	if s.hub != nil {
		event := realtime.WSEvent{
			Type: "CS_CHAT_NEW_MESSAGE",
			Data: message,
		}

		// Kirim ke pemilik chat
		s.hub.SendToAccount(session.AccountID, event)

		// Jika ada CS assigned, kirim ke CS tersebut, jika belum, broadcast ke pool CS
		if session.CSAdminID != nil && *session.CSAdminID != "" {
			s.hub.SendToAccount(*session.CSAdminID, event)
		} else {
			s.hub.BroadcastToRole("platform_admin", event)
		}
	}

	return message, nil
}

func (s *supportChatService) EndChat(sessionID string, requesterID string, isCS bool, input models.EndCSChatInput) error {
	session, err := s.repo.GetSessionByID(sessionID)
	if err != nil {
		return errors.New("Sesi live chat tidak ditemukan")
	}

	if !isCS && session.AccountID != requesterID {
		return errors.New("Anda tidak memiliki akses ke percakapan ini")
	}

	if session.Status == "ended" {
		return nil // Sudah berstatus ended
	}

	endedBy := "user"
	if isCS {
		endedBy = "cs"
	}

	now := time.Now()
	endMsg := &models.CSChatMessage{
		MessageID:   utils.GenerateUUID("LCM"),
		SessionID:   sessionID,
		SenderID:    "SYSTEM",
		SenderType:  "system",
		SenderName:  "ToenToen Support",
		MessageType: "system_event",
		Content:     "Percakapan telah diakhiri. Terima kasih telah menghubungi Customer Service ToenToen.",
		IsRead:      true,
		CreatedAt:   now,
	}

	endMsgToSave := *endMsg
	endMsgToSave.Content, _ = crypto.Encrypt(endMsg.Content)

	if err := s.repo.EndSession(sessionID, endedBy, input.Rating, input.Feedback, &endMsgToSave); err != nil {
		return errors.New("Gagal mengakhiri percakapan")
	}

	// Broadcast WS event
	if s.hub != nil {
		event := realtime.WSEvent{
			Type: "CS_CHAT_ENDED",
			Data: map[string]interface{}{
				"session_id": sessionID,
				"ended_by":   endedBy,
				"ended_at":   now,
				"rating":     input.Rating,
				"feedback":   input.Feedback,
			},
		}
		s.hub.SendToAccount(session.AccountID, event)
		if session.CSAdminID != nil && *session.CSAdminID != "" {
			s.hub.SendToAccount(*session.CSAdminID, event)
		} else {
			s.hub.BroadcastToRole("platform_admin", event)
		}
	}

	return nil
}

func (s *supportChatService) AdminGetQueue(status string, limit int, cursor string) ([]models.CSChatSession, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	sessions, err := s.repo.GetAdminQueue(status, limit, cursor)
	if err != nil {
		return nil, err
	}
	for i := range sessions {
		sessions[i].LastMessagePreview, _ = crypto.Decrypt(sessions[i].LastMessagePreview)
	}
	return sessions, nil
}

func (s *supportChatService) AdminGetHistory(limit int, cursor string) ([]models.CSChatSession, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	sessions, err := s.repo.GetAdminHistory(limit, cursor)
	if err != nil {
		return nil, err
	}
	for i := range sessions {
		sessions[i].LastMessagePreview, _ = crypto.Decrypt(sessions[i].LastMessagePreview)
	}
	return sessions, nil
}

func (s *supportChatService) AdminAssignSession(sessionID string, csAdminID string) error {
	session, err := s.repo.GetSessionByID(sessionID)
	if err != nil {
		return errors.New("Sesi live chat tidak ditemukan")
	}
	if session.Status == "ended" {
		return errors.New("Sesi live chat sudah berakhir")
	}
	return s.repo.AssignCSToSession(sessionID, csAdminID)
}
