package chat

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"toentoen-backend/internal/domain/realtime"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/utils"
)

type SupportChatService interface {
	InitiateChat(accountID string, userRole string, businessID *string) (*models.CSChatRoom, *models.CSChatMessage, bool, error)
	GetActiveChat(accountID string) (*models.CSChatRoom, error)
	GetChatMessages(roomID string, requesterID string, isCS bool, limit int, cursor string) ([]models.CSChatMessage, error)
	SendMessage(roomID string, senderID string, senderType string, senderName string, input models.SendCSMessageInput) (*models.CSChatMessage, error)
	EndChat(roomID string, requesterID string, isCS bool, input models.EndCSChatInput) error
	AdminGetQueue(status string, limit int, cursor string) ([]models.CSChatRoom, error)
	AdminGetHistory(limit int, cursor string) ([]models.CSChatRoom, error)
	AdminAssignRoom(roomID string, csAdminID string) error
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

func (s *supportChatService) InitiateChat(accountID string, userRole string, businessID *string) (*models.CSChatRoom, *models.CSChatMessage, bool, error) {
	// 1. Cek apakah user sudah memiliki room aktif yang belum di-end/hide
	existingRoom, err := s.repo.FindActiveRoomByAccountID(accountID)
	if err == nil && existingRoom != nil {
		// Sudah ada room aktif, kembalikan room tersebut (isNew = false)
		messages, _ := s.repo.GetMessagesByRoomID(existingRoom.RoomID, 1, "")
		var lastMsg *models.CSChatMessage
		if len(messages) > 0 {
			lastMsg = &messages[len(messages)-1]
		}
		return existingRoom, lastMsg, false, nil
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
	roomID := utils.GenerateUUID("CSR")
	greetingContent := fmt.Sprintf("Halo %s! Selamat datang di Layanan Bantuan ToenToen. Ada yang bisa kami bantu hari ini?", displayName)

	greetingMsg := &models.CSChatMessage{
		MessageID:   utils.GenerateUUID("CSM"),
		RoomID:      roomID,
		SenderID:    "SYSTEM",
		SenderType:  "system",
		SenderName:  "ToenToen Support",
		MessageType: "template",
		Content:     greetingContent,
		IsRead:      false,
		CreatedAt:   now,
	}

	newRoom := &models.CSChatRoom{
		RoomID:             roomID,
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

	if err := s.repo.CreateRoomWithGreeting(newRoom, greetingMsg); err != nil {
		return nil, nil, false, errors.New("Gagal membuat sesi live chat")
	}

	// 3. Broadcast event real-time ke CS via WebSocket jika hub tersedia
	if s.hub != nil {
		s.hub.BroadcastToRole("platform_admin", realtime.WSEvent{
			Type: "CS_ROOM_CREATED",
			Data: map[string]interface{}{
				"room_id":      newRoom.RoomID,
				"account_id":   newRoom.AccountID,
				"user_role":    newRoom.UserRole,
				"display_name": displayName,
				"created_at":   newRoom.CreatedAt,
			},
		})
	}

	return newRoom, greetingMsg, true, nil
}

func (s *supportChatService) GetActiveChat(accountID string) (*models.CSChatRoom, error) {
	room, err := s.repo.FindActiveRoomByAccountID(accountID)
	if err != nil {
		return nil, nil
	}
	return room, nil
}

func (s *supportChatService) GetChatMessages(roomID string, requesterID string, isCS bool, limit int, cursor string) ([]models.CSChatMessage, error) {
	room, err := s.repo.GetRoomByID(roomID)
	if err != nil {
		return nil, errors.New("Room chat tidak ditemukan")
	}

	// Validasi akses: hanya pemilik chat atau admin CS yang boleh membaca
	if !isCS && room.AccountID != requesterID {
		return nil, errors.New("Anda tidak memiliki akses ke percakapan ini")
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	readerType := "user"
	if isCS {
		readerType = "cs"
	}
	_ = s.repo.MarkMessagesAsRead(roomID, readerType)

	return s.repo.GetMessagesByRoomID(roomID, limit, cursor)
}

func (s *supportChatService) SendMessage(roomID string, senderID string, senderType string, senderName string, input models.SendCSMessageInput) (*models.CSChatMessage, error) {
	room, err := s.repo.GetRoomByID(roomID)
	if err != nil {
		return nil, errors.New("Room chat tidak ditemukan")
	}

	if room.Status == "ended" {
		return nil, errors.New("Sesi chat ini sudah diakhiri. Silakan mulai live chat baru.")
	}

	// Validasi sender
	if senderType == "user" && room.AccountID != senderID {
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
		MessageID:   utils.GenerateUUID("CSM"),
		RoomID:      roomID,
		SenderID:    senderID,
		SenderType:  senderType,
		SenderName:  senderName,
		MessageType: msgType,
		Content:     strings.TrimSpace(input.Content),
		MediaURL:    input.MediaURL,
		IsRead:      false,
		CreatedAt:   now,
	}

	if err := s.repo.SaveMessage(message); err != nil {
		return nil, errors.New("Gagal mengirim pesan")
	}

	// Update metadata room
	newStatus := room.Status
	var csAdminIDToUpdate *string
	if senderType == "cs" {
		newStatus = "in_progress"
		if room.CSAdminID == nil || *room.CSAdminID == "" {
			csAdminIDToUpdate = &senderID
		}
	}
	_ = s.repo.UpdateRoomLastMessage(roomID, message.Content, now, newStatus, csAdminIDToUpdate)

	// Broadcast via WebSocket
	if s.hub != nil {
		event := realtime.WSEvent{
			Type: "CS_CHAT_NEW_MESSAGE",
			Data: message,
		}

		// Kirim ke pemilik chat
		s.hub.SendToAccount(room.AccountID, event)

		// Jika ada CS assigned, kirim ke CS tersebut, jika belum, broadcast ke pool CS
		if room.CSAdminID != nil && *room.CSAdminID != "" {
			s.hub.SendToAccount(*room.CSAdminID, event)
		} else {
			s.hub.BroadcastToRole("platform_admin", event)
		}
	}

	return message, nil
}

func (s *supportChatService) EndChat(roomID string, requesterID string, isCS bool, input models.EndCSChatInput) error {
	room, err := s.repo.GetRoomByID(roomID)
	if err != nil {
		return errors.New("Room chat tidak ditemukan")
	}

	if !isCS && room.AccountID != requesterID {
		return errors.New("Anda tidak memiliki akses ke percakapan ini")
	}

	if room.Status == "ended" {
		return nil // Sudah berstatus ended
	}

	endedBy := "user"
	if isCS {
		endedBy = "cs"
	}

	now := time.Now()
	endMsg := &models.CSChatMessage{
		MessageID:   utils.GenerateUUID("CSM"),
		RoomID:      roomID,
		SenderID:    "SYSTEM",
		SenderType:  "system",
		SenderName:  "ToenToen Support",
		MessageType: "system_event",
		Content:     "Percakapan telah diakhiri. Terima kasih telah menghubungi Customer Service ToenToen.",
		IsRead:      true,
		CreatedAt:   now,
	}

	if err := s.repo.EndRoom(roomID, endedBy, input.Rating, input.Feedback, endMsg); err != nil {
		return errors.New("Gagal mengakhiri percakapan")
	}

	// Broadcast WS event
	if s.hub != nil {
		event := realtime.WSEvent{
			Type: "CS_CHAT_ENDED",
			Data: map[string]interface{}{
				"room_id":  roomID,
				"ended_by": endedBy,
				"ended_at": now,
				"rating":   input.Rating,
				"feedback": input.Feedback,
			},
		}
		s.hub.SendToAccount(room.AccountID, event)
		if room.CSAdminID != nil && *room.CSAdminID != "" {
			s.hub.SendToAccount(*room.CSAdminID, event)
		} else {
			s.hub.BroadcastToRole("platform_admin", event)
		}
	}

	return nil
}

func (s *supportChatService) AdminGetQueue(status string, limit int, cursor string) ([]models.CSChatRoom, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.GetAdminQueue(status, limit, cursor)
}

func (s *supportChatService) AdminGetHistory(limit int, cursor string) ([]models.CSChatRoom, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.GetAdminHistory(limit, cursor)
}

func (s *supportChatService) AdminAssignRoom(roomID string, csAdminID string) error {
	room, err := s.repo.GetRoomByID(roomID)
	if err != nil {
		return errors.New("Room chat tidak ditemukan")
	}
	if room.Status == "ended" {
		return errors.New("Room chat sudah berakhir")
	}
	return s.repo.AssignCSToRoom(roomID, csAdminID)
}
