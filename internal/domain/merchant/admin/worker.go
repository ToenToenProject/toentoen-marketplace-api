package admin

import (
	"log"
	"time"
)

// StartGracePeriodWorker menjalankan background job secara berkala untuk
// menonaktifkan akun admin yang masa tenggang 3 harinya telah habis.
func StartGracePeriodWorker(repo BusinessAdminRepository, interval time.Duration) {
	go func() {
		log.Println("🕒 Grace Period Expiry Worker started...")
		// Jalankan sekali saat startup
		processExpiredAssignments(repo)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			processExpiredAssignments(repo)
		}
	}()
}

func processExpiredAssignments(repo BusinessAdminRepository) {
	expired, err := repo.GetExpiredScheduledDeletions()
	if err != nil {
		log.Printf("⚠️ Error querying expired admin assignments: %v\n", err)
		return
	}

	if len(expired) == 0 {
		return
	}

	var ids []string
	for _, a := range expired {
		ids = append(ids, a.AssignmentID)
	}

	if err := repo.BatchRevokeAssignments(ids); err != nil {
		log.Printf("⚠️ Error batch revoking expired admin assignments: %v\n", err)
		return
	}

	log.Printf("✅ Grace Period Worker: Successfully revoked %d expired admin assignment(s)\n", len(ids))
}
