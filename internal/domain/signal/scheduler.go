package signal

import (
	"log"
	"time"
)

// StartSignalExpiryWorker runs a 5-minute background scheduler to transition overdue
// active signals to 'expired' status and close their associated chat rooms.
func StartSignalExpiryWorker(repo SignalRepository) {
	if repo == nil {
		return
	}

	go func() {
		// Run initial evaluation on startup.
		if count, err := repo.ExpireOverdueSignals(); err != nil {
			log.Printf("[SignalExpiryWorker] Gagal memeriksa sinyal kedaluwarsa saat inisialisasi: %v", err)
		} else if count > 0 {
			log.Printf("[SignalExpiryWorker] Inisialisasi awal berhasil menandai %d sinyal kedaluwarsa sebagai expired", count)
		}

		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			if count, err := repo.ExpireOverdueSignals(); err != nil {
				log.Printf("[SignalExpiryWorker] Gagal memeriksa sinyal kedaluwarsa berkala: %v", err)
			} else if count > 0 {
				log.Printf("[SignalExpiryWorker] Berhasil menandai %d sinyal kedaluwarsa sebagai expired", count)
			}
		}
	}()
}
