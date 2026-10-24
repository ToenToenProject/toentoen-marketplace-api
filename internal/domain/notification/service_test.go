package notification

import (
	"encoding/json"
	"testing"
	"time"

	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/datatypes"
)

type mockNotifRepo struct {
	NotificationRepository
	notifs []models.UserNotification
	saved  []*models.UserNotification
}

func (m *mockNotifRepo) GetMyNotifications(accountID string, page, limit int) ([]models.UserNotification, int64, error) {
	return m.notifs, int64(len(m.notifs)), nil
}

func (m *mockNotifRepo) GetUnreadNotificationByRef(recipientID, notifType, refID string) (*models.UserNotification, error) {
	return nil, nil
}

func (m *mockNotifRepo) CreateNotification(n *models.UserNotification) error {
	m.saved = append(m.saved, n)
	return nil
}

func (m *mockNotifRepo) GetFCMTokensByAccountID(accountID string) ([]string, error) {
	return nil, nil
}

func TestGetMyNotificationsSignalPayload(t *testing.T) {
	sigID := "SIG-20261024-TEST1"
	convID := "CONV-20261024-TEST2"

	dataSignal, _ := json.Marshal(map[string]interface{}{
		"signal_id": sigID,
		"category":  "Makanan",
	})
	dataReply, _ := json.Marshal(map[string]interface{}{
		"signal_id":       sigID,
		"conversation_id": convID,
		"business_name":   "Toko Sejahtera",
	})

	repo := &mockNotifRepo{
		notifs: []models.UserNotification{
			{
				ID:                 "NOTIF-1",
				RecipientAccountID: "ACC-USER1",
				Type:               "new_demand_signal", // Legacy type
				ReferenceID:        &sigID,
				Title:              "Permintaan Kebutuhan Baru",
				Body:               "Ada permintaan baru",
				Data:               datatypes.JSON(dataSignal),
				CreatedAt:          time.Now(),
			},
			{
				ID:                 "NOTIF-2",
				RecipientAccountID: "ACC-USER1",
				Type:               "signal_reply", // Legacy type
				ReferenceID:        &convID,
				Title:              "Tanggapan Baru",
				Body:               "Toko menanggapi sinyal Anda",
				Data:               datatypes.JSON(dataReply),
				CreatedAt:          time.Now(),
			},
		},
	}

	svc := NewNotificationService(repo, nil)

	responses, total, err := svc.GetMyNotifications("ACC-USER1", 1, 10)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if total != 2 || len(responses) != 2 {
		t.Fatalf("Expected 2 responses, got %d", len(responses))
	}

	// 1. Verifikasi NOTIF-1 (Sinyal Kebutuhan)
	n1 := responses[0]
	if n1.Type != "signal" {
		t.Errorf("Expected Type 'signal', got %q", n1.Type)
	}
	if n1.Data["signal_id"] != sigID {
		t.Errorf("Expected signal_id %q, got %v", sigID, n1.Data["signal_id"])
	}
	if n1.Data["post_id"] != sigID {
		t.Errorf("Expected post_id %q, got %v", sigID, n1.Data["post_id"])
	}
	if n1.Data["type"] != "signal" {
		t.Errorf("Expected data.type 'signal', got %v", n1.Data["type"])
	}

	// 2. Verifikasi NOTIF-2 (Tawaran / Tanggapan Sinyal)
	n2 := responses[1]
	if n2.Type != "signal_offer" {
		t.Errorf("Expected Type 'signal_offer', got %q", n2.Type)
	}
	if n2.Data["signal_id"] != sigID {
		t.Errorf("Expected signal_id %q, got %v", sigID, n2.Data["signal_id"])
	}
	if n2.Data["post_id"] != sigID {
		t.Errorf("Expected post_id %q, got %v", sigID, n2.Data["post_id"])
	}
	if n2.Data["offer_id"] != convID {
		t.Errorf("Expected offer_id %q, got %v", convID, n2.Data["offer_id"])
	}
	if n2.Data["type"] != "signal_offer" {
		t.Errorf("Expected data.type 'signal_offer', got %v", n2.Data["type"])
	}
}

func TestSendNotificationSignalPayloadNormalization(t *testing.T) {
	repo := &mockNotifRepo{}
	svc := NewNotificationService(repo, nil)

	sigID := "SIG-SEND-123"
	convID := "CONV-SEND-456"

	// 1. Test SendNotification for new signal
	svc.SendNotification(
		[]string{"ACC-MERCHANT1"},
		"SYSTEM",
		"new_demand_signal",
		sigID,
		"Peluang Baru",
		"Ada permintaan",
		map[string]interface{}{
			"signal_id": sigID,
		},
		nil,
	)

	if len(repo.saved) != 1 {
		t.Fatalf("Expected 1 saved notification, got %d", len(repo.saved))
	}
	saved1 := repo.saved[0]
	if saved1.Type != "signal" {
		t.Errorf("Expected saved Type 'signal', got %q", saved1.Type)
	}
	var data1 map[string]interface{}
	_ = json.Unmarshal(saved1.Data, &data1)
	if data1["signal_id"] != sigID || data1["post_id"] != sigID || data1["type"] != "signal" {
		t.Errorf("Expected signal_id and post_id to equal %q with type 'signal', got %v", sigID, data1)
	}

	// 2. Test SendNotification for signal offer
	svc.SendNotification(
		[]string{"ACC-CUSTOMER1"},
		"ACC-MERCHANT1",
		"signal_reply",
		convID,
		"Tawaran Baru",
		"Merchant memberi penawaran",
		map[string]interface{}{
			"signal_id":       sigID,
			"conversation_id": convID,
		},
		nil,
	)

	if len(repo.saved) != 2 {
		t.Fatalf("Expected 2 saved notifications, got %d", len(repo.saved))
	}
	saved2 := repo.saved[1]
	if saved2.Type != "signal_offer" {
		t.Errorf("Expected saved Type 'signal_offer', got %q", saved2.Type)
	}
	var data2 map[string]interface{}
	_ = json.Unmarshal(saved2.Data, &data2)
	if data2["signal_id"] != sigID || data2["post_id"] != sigID || data2["offer_id"] != convID || data2["type"] != "signal_offer" {
		t.Errorf("Expected signal_id, post_id, offer_id with type 'signal_offer', got %v", data2)
	}
}
