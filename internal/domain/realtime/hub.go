package realtime

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/redis/go-redis/v9"
)

const redisChannel = "signal_events"

// WSEvent adalah format standar event yang dikirim ke client via WebSocket
type WSEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Hub mengelola semua koneksi WebSocket aktif
type Hub struct {
	// clients: map[accountID] → set of *Client
	clients map[string]map[*Client]bool
	mu      sync.RWMutex

	register   chan *Client
	unregister chan *Client

	rdb *redis.Client
}

// NewHub membuat Hub baru dengan koneksi Redis untuk Pub/Sub
func NewHub(rdb *redis.Client) *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		rdb:        rdb,
	}
}

// Run menjalankan event loop Hub. Harus dipanggil di goroutine terpisah.
func (h *Hub) Run() {
	// Jalankan Redis subscriber di goroutine terpisah
	go h.subscribeRedis()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.AccountID] == nil {
				h.clients[client.AccountID] = make(map[*Client]bool)
			}
			h.clients[client.AccountID][client] = true
			h.mu.Unlock()
			log.Printf("🟢 WS connected: %s (total connections for user: %d)", client.AccountID, len(h.clients[client.AccountID]))

		case client := <-h.unregister:
			h.mu.Lock()
			if conns, ok := h.clients[client.AccountID]; ok {
				if _, exists := conns[client]; exists {
					delete(conns, client)
					close(client.Send)
					if len(conns) == 0 {
						delete(h.clients, client.AccountID)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("🔴 WS disconnected: %s", client.AccountID)
		}
	}
}

// redisMessage adalah wrapper untuk pesan yang dikirim via Redis Pub/Sub
type redisMessage struct {
	TargetIDs []string `json:"target_ids"` // Daftar account_id tujuan
	Event     WSEvent  `json:"event"`
}

// subscribeRedis mendengarkan event dari Redis channel
func (h *Hub) subscribeRedis() {
	if h.rdb == nil {
		log.Println("⚠️ Redis tidak tersedia, Pub/Sub dinonaktifkan")
		return
	}

	ctx := context.Background()
	sub := h.rdb.Subscribe(ctx, redisChannel)
	defer sub.Close()

	log.Println("📡 Redis Pub/Sub subscriber aktif pada channel:", redisChannel)

	ch := sub.Channel()
	for msg := range ch {
		var rm redisMessage
		if err := json.Unmarshal([]byte(msg.Payload), &rm); err != nil {
			log.Printf("⚠️ Gagal decode Redis message: %v", err)
			continue
		}

		// Kirim event ke semua target yang terkoneksi di server ini
		h.deliverToLocal(rm.TargetIDs, rm.Event)
	}
}

// publishToRedis mengirimkan event ke Redis channel agar bisa diterima oleh semua server
func (h *Hub) publishToRedis(targetIDs []string, event WSEvent) {
	if h.rdb == nil {
		// Fallback: kirim langsung secara lokal jika Redis tidak tersedia
		h.deliverToLocal(targetIDs, event)
		return
	}

	rm := redisMessage{
		TargetIDs: targetIDs,
		Event:     event,
	}

	data, err := json.Marshal(rm)
	if err != nil {
		log.Printf("⚠️ Gagal marshal Redis message: %v", err)
		return
	}

	if err := h.rdb.Publish(context.Background(), redisChannel, data).Err(); err != nil {
		log.Printf("⚠️ Gagal publish ke Redis: %v, fallback ke lokal", err)
		h.deliverToLocal(targetIDs, event)
	}
}

// deliverToLocal mengirimkan event ke client yang terkoneksi di server ini saja
func (h *Hub) deliverToLocal(targetIDs []string, event WSEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, id := range targetIDs {
		if conns, ok := h.clients[id]; ok {
			for client := range conns {
				select {
				case client.Send <- data:
				default:
					// Buffer penuh, skip agar tidak blocking
				}
			}
		}
	}
}

// ================================================================
// PUBLIC API — dipanggil dari service layer
// ================================================================

// SendToUser mengirimkan event ke satu user tertentu
func (h *Hub) SendToUser(accountID string, event WSEvent) {
	h.publishToRedis([]string{accountID}, event)
}

// BroadcastToUsers mengirimkan event ke banyak user sekaligus
func (h *Hub) BroadcastToUsers(accountIDs []string, event WSEvent) {
	if len(accountIDs) == 0 {
		return
	}
	h.publishToRedis(accountIDs, event)
}

// IsOnline mengecek apakah user sedang terkoneksi via WebSocket
func (h *Hub) IsOnline(accountID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	conns, ok := h.clients[accountID]
	return ok && len(conns) > 0
}
