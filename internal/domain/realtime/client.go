package realtime

import (
	"log"
	"time"

	"github.com/gofiber/websocket/v2"
)

const (
	// Waktu tunggu untuk menulis pesan ke client
	writeWait = 10 * time.Second
	// Waktu tunggu untuk menerima pong dari client
	pongWait = 60 * time.Second
	// Interval pengiriman ping ke client (harus lebih kecil dari pongWait)
	pingPeriod = (pongWait * 9) / 10
	// Ukuran maksimum buffer pesan keluar
	maxMessageSize = 512
)

// Client merepresentasikan satu koneksi WebSocket
type Client struct {
	Hub       *Hub
	Conn      *websocket.Conn
	Send      chan []byte
	AccountID string
	Role      string // "customer" atau "owner"
}

// ReadPump membaca pesan dari WebSocket client (ping/pong, close)
// Dijalankan di goroutine terpisah per client
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("⚠️ WS read error [%s]: %v", c.AccountID, err)
			}
			break
		}
		// Saat ini client tidak perlu mengirim pesan via WS
		// Chat tetap melalui REST API, WS hanya untuk menerima event
	}
}

// WritePump menulis pesan dari server ke WebSocket client
// Dijalankan di goroutine terpisah per client
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub menutup channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
