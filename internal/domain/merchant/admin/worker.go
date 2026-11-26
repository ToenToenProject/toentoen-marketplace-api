package admin

import (
	"log"
	"time"
)

// StartGracePeriodWorker runs a background job periodically to revoke admin assignments
// whose 3-day deletion grace period has expired.
func StartGracePeriodWorker(repo BusinessAdminRepository, interval time.Duration) {
	go func() {
		log.Println("🕒 Grace Period Expiry Worker started...")
		// Run once on worker startup.
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
