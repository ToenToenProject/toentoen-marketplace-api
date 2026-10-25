package signal

import (
	"log"
	"time"
)

// StartSignalExpiryWorker menjalankan background scheduler secara berkala (setiap 30 detik)
// untuk mengupdate kolom status menjadi 'expired' dan mengunci roomchat pada sinyal-sinyal
// yang masa aktifnya telah terlampaui (status = 'active' AND expires_at <= NOW()).
func StartSignalExpiryWorker(repo SignalRepository) {
	if repo == nil {
		return
	}

	go func() {
		// Jalankan evaluasi langsung saat worker pertama kali dihidupkan
		if count, err := repo.ExpireOverdueSignals(); err != nil {
			log.Printf("[SignalExpiryWorker] Gagal memeriksa sinyal kedaluwarsa saat inisialisasi: %v", err)
		} else if count > 0 {
			log.Printf("[SignalExpiryWorker] Inisialisasi awal berhasil menandai %d sinyal kedaluwarsa sebagai expired", count)
		}

		ticker := time.NewTicker(30 * time.Second)
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
