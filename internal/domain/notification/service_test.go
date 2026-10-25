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

func TestGetMyNotificationsStandardizedPayload(t *testing.T) {
	sigID := "SIG-20261024-TEST1"
	convID := "CONV-20261024-TEST2"
	revID := "REV-20261024-TEST3"
	postID := "PST-20261024-TEST4"
	promoID := "PRM-20261024-TEST5"
	asnID := "ASN-20261024-TEST6"
	bizID := "BUS-001"
	accID := "ACC-20261024-USER99"

	dataSignal, _ := json.Marshal(map[string]interface{}{
		"signal_id": sigID,
		"category":  "Makanan",
	})
	dataReply, _ := json.Marshal(map[string]interface{}{
		"signal_id":       sigID,
		"conversation_id": convID,
		"business_name":   "Toko Sejahtera",
		"business_id":     bizID,
	})
	dataReview, _ := json.Marshal(map[string]interface{}{
		"review_id":   revID,
		"business_id": bizID,
		"rating":      5,
	})
	dataPostLike, _ := json.Marshal(map[string]interface{}{
		"post_id":     postID,
		"business_id": bizID,
	})
	dataPromo, _ := json.Marshal(map[string]interface{}{
		"promotion_id": promoID,
		"business_id":  bizID,
	})
	dataKYC, _ := json.Marshal(map[string]interface{}{
		"account_id": accID,
		"status":     "verified",
	})
	dataInvitation, _ := json.Marshal(map[string]interface{}{
		"assignment_id": asnID,
		"business_id":   bizID,
		"role":          "staff",
	})

	repo := &mockNotifRepo{
		notifs: []models.UserNotification{
			{
				ID:                 "NOTIF-1",
				RecipientAccountID: "ACC-USER1",
				Type:               "new_demand_signal",
				ReferenceID:        &sigID,
				Title:              "Permintaan Kebutuhan Baru",
				Body:               "Ada permintaan baru",
				Data:               datatypes.JSON(dataSignal),
				CreatedAt:          time.Now(),
			},
			{
				ID:                 "NOTIF-2",
				RecipientAccountID: "ACC-USER1",
				Type:               "signal_reply",
				ReferenceID:        &convID,
				Title:              "Tanggapan Baru",
				Body:               "Toko menanggapi sinyal Anda",
				Data:               datatypes.JSON(dataReply),
				CreatedAt:          time.Now(),
			},
			{
				ID:                 "NOTIF-3",
				RecipientAccountID: "ACC-USER1",
				Type:               "business_review",
				ReferenceID:        &revID,
				Title:              "Ulasan Baru",
				Body:               "Pelanggan memberi ulasan",
				Data:               datatypes.JSON(dataReview),
				CreatedAt:          time.Now(),
			},
			{
				ID:                 "NOTIF-4",
				RecipientAccountID: "ACC-USER1",
				Type:               "post_like",
				ReferenceID:        &postID,
				Title:              "Post Disukai",
				Body:               "Seseorang menyukai post Anda",
				Data:               datatypes.JSON(dataPostLike),
				CreatedAt:          time.Now(),
			},
			{
				ID:                 "NOTIF-5",
				RecipientAccountID: "ACC-USER1",
				Type:               "business_promotion",
				ReferenceID:        &promoID,
				Title:              "Promo Toko",
				Body:               "Ada promo menarik",
				Data:               datatypes.JSON(dataPromo),
				CreatedAt:          time.Now(),
			},
			{
				ID:                 "NOTIF-6",
				RecipientAccountID: "ACC-USER1",
				Type:               "kyc_approved",
				ReferenceID:        &accID,
				Title:              "KYC Berhasil",
				Body:               "Akun Anda telah terverifikasi",
				Data:               datatypes.JSON(dataKYC),
				CreatedAt:          time.Now(),
			},
			{
				ID:                 "NOTIF-7",
				RecipientAccountID: "ACC-USER1",
				Type:               "admin_invitation",
				ReferenceID:        &asnID,
				Title:              "Undangan Admin Toko",
				Body:               "Anda diundang menjadi admin",
				Data:               datatypes.JSON(dataInvitation),
				CreatedAt:          time.Now(),
			},
		},
	}

	svc := NewNotificationService(repo, nil)

	responses, total, err := svc.GetMyNotifications("ACC-USER1", 1, 10)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if total != 7 || len(responses) != 7 {
		t.Fatalf("Expected 7 responses, got %d", len(responses))
	}

	// 1. Verifikasi NOTIF-1 (Sinyal Kebutuhan)
	n1 := responses[0]
	if n1.Type != "signal_post" {
		t.Errorf("Expected Type 'signal_post', got %q", n1.Type)
	}
	if n1.Data["target_type"] != "signal" || n1.TargetType != "signal" {
		t.Errorf("Expected target_type 'signal', got data=%v, root=%v", n1.Data["target_type"], n1.TargetType)
	}
	if n1.Data["target_id"] != sigID || n1.TargetID != sigID {
		t.Errorf("Expected target_id %q, got data=%v, root=%v", sigID, n1.Data["target_id"], n1.TargetID)
	}
	if n1.Data["signal_id"] != sigID || n1.Data["post_id"] != sigID {
		t.Errorf("Expected legacy signal_id and post_id %q, got %v", sigID, n1.Data)
	}

	// 2. Verifikasi NOTIF-2 (Tawaran / Tanggapan Chat)
	n2 := responses[1]
	if n2.Type != "chat" {
		t.Errorf("Expected Type 'chat', got %q", n2.Type)
	}
	if n2.Data["target_type"] != "chat" || n2.TargetType != "chat" {
		t.Errorf("Expected target_type 'chat', got data=%v, root=%v", n2.Data["target_type"], n2.TargetType)
	}
	if n2.Data["target_id"] != convID || n2.TargetID != convID {
		t.Errorf("Expected target_id %q, got data=%v, root=%v", convID, n2.Data["target_id"], n2.TargetID)
	}
	if n2.Data["business_id"] != bizID {
		t.Errorf("Expected business_id %q, got %v", bizID, n2.Data["business_id"])
	}

	// 3. Verifikasi NOTIF-3 (Review)
	n3 := responses[2]
	if n3.Type != "new_review" {
		t.Errorf("Expected Type 'new_review', got %q", n3.Type)
	}
	if n3.Data["target_type"] != "review" || n3.TargetType != "review" {
		t.Errorf("Expected target_type 'review', got data=%v, root=%v", n3.Data["target_type"], n3.TargetType)
	}
	if n3.Data["target_id"] != revID || n3.TargetID != revID {
		t.Errorf("Expected target_id %q, got data=%v, root=%v", revID, n3.Data["target_id"], n3.TargetID)
	}
	if n3.Data["business_id"] != bizID {
		t.Errorf("Expected business_id %q, got %v", bizID, n3.Data["business_id"])
	}

	// 4. Verifikasi NOTIF-4 (Post Like)
	n4 := responses[3]
	if n4.Type != "post_like" {
		t.Errorf("Expected Type 'post_like', got %q", n4.Type)
	}
	if n4.Data["target_type"] != "post" || n4.TargetType != "post" {
		t.Errorf("Expected target_type 'post', got data=%v, root=%v", n4.Data["target_type"], n4.TargetType)
	}
	if n4.Data["target_id"] != postID || n4.TargetID != postID {
		t.Errorf("Expected target_id %q, got data=%v, root=%v", postID, n4.Data["target_id"], n4.TargetID)
	}

	// 5. Verifikasi NOTIF-5 (Promotion)
	n5 := responses[4]
	if n5.Type != "promotion" {
		t.Errorf("Expected Type 'promotion', got %q", n5.Type)
	}
	if n5.Data["target_type"] != "promotion" || n5.TargetType != "promotion" {
		t.Errorf("Expected target_type 'promotion', got data=%v, root=%v", n5.Data["target_type"], n5.TargetType)
	}
	if n5.Data["target_id"] != promoID || n5.TargetID != promoID {
		t.Errorf("Expected target_id %q, got data=%v, root=%v", promoID, n5.Data["target_id"], n5.TargetID)
	}

	// 6. Verifikasi NOTIF-6 (KYC)
	n6 := responses[5]
	if n6.Type != "kyc" {
		t.Errorf("Expected Type 'kyc', got %q", n6.Type)
	}
	if n6.Data["target_type"] != "kyc" || n6.TargetType != "kyc" {
		t.Errorf("Expected target_type 'kyc', got data=%v, root=%v", n6.Data["target_type"], n6.TargetType)
	}
	if n6.Data["target_id"] != accID || n6.TargetID != accID {
		t.Errorf("Expected target_id %q, got data=%v, root=%v", accID, n6.Data["target_id"], n6.TargetID)
	}

	// 7. Verifikasi NOTIF-7 (Invitation)
	n7 := responses[6]
	if n7.Type != "invitation" {
		t.Errorf("Expected Type 'invitation', got %q", n7.Type)
	}
	if n7.Data["target_type"] != "invitation" || n7.TargetType != "invitation" {
		t.Errorf("Expected target_type 'invitation', got data=%v, root=%v", n7.Data["target_type"], n7.TargetType)
	}
	if n7.Data["target_id"] != asnID || n7.TargetID != asnID {
		t.Errorf("Expected target_id %q, got data=%v, root=%v", asnID, n7.Data["target_id"], n7.TargetID)
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
	if saved1.Type != "signal_post" {
		t.Errorf("Expected saved Type 'signal_post', got %q", saved1.Type)
	}
	var data1 map[string]interface{}
	_ = json.Unmarshal(saved1.Data, &data1)
	if data1["target_type"] != "signal" || data1["target_id"] != sigID {
		t.Errorf("Expected target_type 'signal' and target_id %q, got %v", sigID, data1)
	}
	if data1["signal_id"] != sigID || data1["post_id"] != sigID {
		t.Errorf("Expected signal_id and post_id to equal %q, got %v", sigID, data1)
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
	if saved2.Type != "chat" {
		t.Errorf("Expected saved Type 'chat', got %q", saved2.Type)
	}
	var data2 map[string]interface{}
	_ = json.Unmarshal(saved2.Data, &data2)
	if data2["target_type"] != "chat" || data2["target_id"] != convID {
		t.Errorf("Expected target_type 'chat' and target_id %q, got %v", convID, data2)
	}
	if data2["conversation_id"] != convID || data2["offer_id"] != convID {
		t.Errorf("Expected conversation_id and offer_id %q, got %v", convID, data2)
	}
}

func TestStandardizeNotificationMetadataCases(t *testing.T) {
	tests := []struct {
		name         string
		rawType      string
		refID        string
		data         map[string]interface{}
		wantType     string
		wantTarget   string
		wantTargetID string
		wantBizID    string
	}{
		{
			name:         "New Demand Signal with SIG prefix",
			rawType:      "new_demand_signal",
			refID:        "SIG-001",
			data:         map[string]interface{}{"signal_id": "SIG-001"},
			wantType:     "signal_post",
			wantTarget:   "signal",
			wantTargetID: "SIG-001",
		},
		{
			name:         "Signal Message with CONV ref",
			rawType:      "signal_message",
			refID:        "CONV-002",
			data:         map[string]interface{}{"conversation_id": "CONV-002", "signal_id": "SIG-001"},
			wantType:     "chat",
			wantTarget:   "chat",
			wantTargetID: "CONV-002",
		},
		{
			name:         "Business Review with REV ref",
			rawType:      "business_review",
			refID:        "REV-003",
			data:         map[string]interface{}{"business_id": "BUS-001"},
			wantType:     "new_review",
			wantTarget:   "review",
			wantTargetID: "REV-003",
			wantBizID:    "BUS-001",
		},
		{
			name:         "Post Comment",
			rawType:      "post_comment",
			refID:        "PST-004",
			data:         map[string]interface{}{"post_id": "PST-004", "comment_id": "CMT-100"},
			wantType:     "post_comment",
			wantTarget:   "post",
			wantTargetID: "PST-004",
		},
		{
			name:         "Promotion with PRM ref",
			rawType:      "business_promotion",
			refID:        "PRM-005",
			data:         map[string]interface{}{"promotion_id": "PRM-005"},
			wantType:     "promotion",
			wantTarget:   "promotion",
			wantTargetID: "PRM-005",
		},
		{
			name:         "KYC Approved with ACC ref",
			rawType:      "kyc_approved",
			refID:        "ACC-006",
			data:         map[string]interface{}{},
			wantType:     "kyc",
			wantTarget:   "kyc",
			wantTargetID: "ACC-006",
		},
		{
			name:         "Admin Invitation with ASN ref",
			rawType:      "admin_invitation",
			refID:        "ASN-007",
			data:         map[string]interface{}{"assignment_id": "ASN-007", "business_id": "BUS-001"},
			wantType:     "invitation",
			wantTarget:   "invitation",
			wantTargetID: "ASN-007",
			wantBizID:    "BUS-001",
		},
		{
			name:         "Merchant Live with BUS ref",
			rawType:      "merchant_live",
			refID:        "BUS-001",
			data:         map[string]interface{}{"business_id": "BUS-001"},
			wantType:     "merchant_live",
			wantTarget:   "business",
			wantTargetID: "BUS-001",
			wantBizID:    "BUS-001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotTarget, gotID, gotBiz := StandardizeNotificationMetadata(tt.rawType, tt.refID, tt.data)
			if gotType != tt.wantType {
				t.Errorf("StandardizeNotificationMetadata() gotType = %v, want %v", gotType, tt.wantType)
			}
			if gotTarget != tt.wantTarget {
				t.Errorf("StandardizeNotificationMetadata() gotTarget = %v, want %v", gotTarget, tt.wantTarget)
			}
			if gotID != tt.wantTargetID {
				t.Errorf("StandardizeNotificationMetadata() gotID = %v, want %v", gotID, tt.wantTargetID)
			}
			if gotBiz != tt.wantBizID {
				t.Errorf("StandardizeNotificationMetadata() gotBiz = %v, want %v", gotBiz, tt.wantBizID)
			}
		})
	}
}
