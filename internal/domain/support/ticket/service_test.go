package ticket

import (
	"testing"
	"time"
	"toentoen-backend/internal/domain/shared/models"
	"toentoen-backend/pkg/crypto"

	"github.com/stretchr/testify/assert"
)

func TestFormatDetailResponse_WithReplyMessage(t *testing.T) {
	replyMsg := "Halo Kak, kendala transaksi sudah kami proses dan saldo Anda sudah dikembalikan. Terima kasih."
	ticket := &models.SupportTicket{
		TicketID:                   "TCK-20261122155745-DEI4ZF",
		Category:                   "Kendala Transaksi",
		Description:                "Saldo saya sudah terpotong namun transaksi dinyatakan gagal oleh sistem.",
		SupportTicketStatus:        "in_progress",
		CustomerServiceSuggestions: &replyMsg,
		CreatedAt:                  time.Now(),
	}

	res := ticket.FormatDetailResponse()

	assert.Equal(t, "TCK-20261122155745-DEI4ZF", res.TicketID)
	assert.Equal(t, "in_progress", res.SupportTicketStatus)
	assert.Equal(t, replyMsg, res.CustomerServiceSuggestions, "Balasan dari CS harus tepat sesuai dengan yang tersimpan di DB")
}

func TestFormatDetailResponse_FallbackDinamis(t *testing.T) {
	testCases := []struct {
		status   string
		expected string
	}{
		{"pending", "Tiket masih direview"},
		{"in_progress", "Tiket sedang dalam penanganan oleh tim Customer Service"},
		{"solved", "Tiket telah diselesaikan oleh tim Customer Service"},
		{"closed", "Tiket telah ditutup"},
		{"unknown", "Tiket masih direview"},
	}

	for _, tc := range testCases {
		t.Run(tc.status, func(t *testing.T) {
			ticket := &models.SupportTicket{
				TicketID:                   "TCK-TEST-001",
				Category:                   "Support",
				Description:                "Masalah pada aplikasi",
				SupportTicketStatus:        tc.status,
				CustomerServiceSuggestions: nil,
			}

			res := ticket.FormatDetailResponse()
			assert.Equal(t, tc.expected, res.CustomerServiceSuggestions)
		})
	}
}

func TestDecryptTicket_WithEncryptedAndPlaintextReplies(t *testing.T) {
	// 1. Plaintext reply dari toentoen-internal-operations
	plainReply := "Pesan balasan plaintext dari tim CS Operations"
	ticketPlain := &models.SupportTicket{
		CustomerServiceSuggestions: &plainReply,
	}
	decryptTicket(ticketPlain)
	assert.NotNil(t, ticketPlain.CustomerServiceSuggestions)
	assert.Equal(t, plainReply, *ticketPlain.CustomerServiceSuggestions)

	// 2. Encrypted reply jika ada
	encryptedReply, err := crypto.Encrypt("Pesan balasan rahasia dari CS")
	assert.NoError(t, err)
	ticketEnc := &models.SupportTicket{
		CustomerServiceSuggestions: &encryptedReply,
	}
	decryptTicket(ticketEnc)
	assert.NotNil(t, ticketEnc.CustomerServiceSuggestions)
	assert.Equal(t, "Pesan balasan rahasia dari CS", *ticketEnc.CustomerServiceSuggestions)
}
