package notification

import (
	"strings"
	"toentoen-backend/internal/domain/shared/models"

	"gorm.io/gorm"
)

// NotifyOwnerStaffActivity mengirimkan notifikasi ke Owner (dan Supervisor jika ada)
// saat staf melakukan tindakan berisiko tinggi (High-Risk Activity).
//
// Aturan:
// 1. Jika aktor adalah Owner sendiri, tidak perlu dikirim.
// 2. Cek apakah saklar notifikasi staf bersangkutan aktif (is_notify_owner_enabled == true).
//    Jika dinonaktifkan oleh Owner (false), notifikasi dilewati (anti-spam).
// 3. Notifikasi dikirimkan via FCM, In-App, dan WebSocket.
func NotifyOwnerStaffActivity(
	db *gorm.DB,
	notifService NotificationService,
	businessID string,
	actorAccountID string,
	actionType string,
	title string,
	body string,
	data map[string]interface{},
) {
	if db == nil || notifService == nil || businessID == "" || actorAccountID == "" {
		return
	}

	go func() {
		// 1. Ambil data bisnis untuk mendapatkan Owner ID
		var biz models.Business
		if err := db.Select("account_id, business_name").Where("business_id = ?", businessID).First(&biz).Error; err != nil {
			return
		}

		// Jika aksi dilakukan langsung oleh Owner, jangan kirim notifikasi ke diri sendiri
		if biz.AccountID == actorAccountID {
			return
		}

		// 2. Cek assignment staf untuk memeriksa saklar notifikasi (is_notify_owner_enabled)
		var assignment models.BusinessAdminAssignment
		err := db.Select("assignment_id, role, is_notify_owner_enabled, supervisor_id").
			Where("business_id = ? AND admin_id = ? AND (status = 'active' OR status = 'scheduled_for_deletion')", businessID, actorAccountID).
			First(&assignment).Error
		if err != nil {
			return
		}

		// Jika saklar notifikasi dimatikan oleh Owner untuk staf ini, jangan kirim
		if !assignment.IsNotifyOwnerEnabled {
			return
		}

		// 3. Ambil nama staf untuk personifikasi notifikasi
		var staffAccount models.Account
		staffName := "Staf"
		if err := db.Select("first_name, last_name").Where("account_id = ?", actorAccountID).First(&staffAccount).Error; err == nil {
			fullName := strings.TrimSpace(staffAccount.FirstName + " " + staffAccount.LastName)
			if fullName != "" {
				staffName = fullName
			}
		}

		// Siapkan daftar penerima: Owner utama
		recipients := []string{biz.AccountID}

		// Jika staf memiliki Supervisor langsung (dan bukan aktor), sertakan juga Supervisor
		if assignment.SupervisorID != nil && *assignment.SupervisorID != "" && *assignment.SupervisorID != actorAccountID {
			recipients = append(recipients, *assignment.SupervisorID)
		}

		if data == nil {
			data = make(map[string]interface{})
		}
		data["business_id"] = businessID
		data["actor_id"] = actorAccountID
		data["actor_name"] = staffName
		data["action_type"] = actionType

		notifService.SendNotification(
			recipients,
			actorAccountID,
			"staff_activity_"+actionType,
			businessID,
			title,
			body,
			data,
			nil,
		)
	}()
}
