package location

import (
	"encoding/json"
	"log"

	"toentoen-backend/internal/domain/realtime"
)

// WSLocationHandler menangani pesan WebSocket yang berhubungan dengan Live Location
type WSLocationHandler struct {
	service *LocationService
	// throttleStates menyimpan state throttle per client connection
	// Key: pointer ke Client (unik per koneksi)
	// Dikelola di Hub level, bukan di sini
}

// NewWSLocationHandler membuat instance baru
func NewWSLocationHandler(service *LocationService) *WSLocationHandler {
	return &WSLocationHandler{
		service: service,
	}
}

// HandleMessage memproses pesan masuk dari WebSocket client yang berhubungan dengan lokasi
func (h *WSLocationHandler) HandleMessage(client *realtime.Client, msgType string, rawData json.RawMessage) {
	switch msgType {
	case "LOCATION_UPDATE":
		h.handleLocationUpdate(client, rawData)
	case "SUBSCRIBE_AREA":
		h.handleSubscribeArea(client, rawData)
	case "UNSUBSCRIBE":
		h.handleUnsubscribe(client)
	case "GET_NEARBY":
		h.handleGetNearby(client, rawData)
	}
}

// handleLocationUpdate menerima update lokasi dari customer
// dan mengembalikan daftar merchant terdekat yang terbaru
func (h *WSLocationHandler) handleLocationUpdate(client *realtime.Client, rawData json.RawMessage) {
	var update LocationUpdate
	if err := json.Unmarshal(rawData, &update); err != nil {
		log.Printf("⚠️ WS Location: invalid LOCATION_UPDATE data from %s: %v", client.AccountID, err)
		return
	}

	// Validasi dasar
	if update.Lat < -90 || update.Lat > 90 || update.Lng < -180 || update.Lng > 180 {
		return
	}

	// Ambil atau buat throttle state untuk client ini
	state := client.GetLocationState()

	// Proses lokasi melalui service (throttle, jitter filter, nearby query)
	merchants, shouldSend := h.service.ProcessCustomerLocation(state, update)
	if !shouldSend {
		return
	}

	// Kirim update nearby merchants ke customer
	response := realtime.WSEvent{
		Type: "NEARBY_UPDATE",
		Data: map[string]interface{}{
			"merchants":   merchants,
			"total":       len(merchants),
			"customer_lat": update.Lat,
			"customer_lng": update.Lng,
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		return
	}

	select {
	case client.Send <- data:
	default:
		// Buffer penuh, skip
	}
}

// handleSubscribeArea menangani customer yang ingin subscribe ke area tertentu
// untuk menerima update real-time jika ada perubahan merchant di area tersebut
func (h *WSLocationHandler) handleSubscribeArea(client *realtime.Client, rawData json.RawMessage) {
	var req SubscribeAreaRequest
	if err := json.Unmarshal(rawData, &req); err != nil {
		log.Printf("⚠️ WS Location: invalid SUBSCRIBE_AREA from %s: %v", client.AccountID, err)
		return
	}

	if req.RadiusKm <= 0 {
		req.RadiusKm = 5
	}

	// Set subscription area di client
	client.SetSubscriptionArea(req.Lat, req.Lng, req.RadiusKm)

	// Kirim snapshot awal merchant terdekat
	merchants, err := h.service.GetNearbyMerchants(req.Lat, req.Lng, req.RadiusKm, 50)
	if err != nil {
		log.Printf("⚠️ WS Location: error getting nearby for %s: %v", client.AccountID, err)
		return
	}

	response := realtime.WSEvent{
		Type: "NEARBY_UPDATE",
		Data: map[string]interface{}{
			"merchants":   merchants,
			"total":       len(merchants),
			"customer_lat": req.Lat,
			"customer_lng": req.Lng,
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		return
	}

	select {
	case client.Send <- data:
	default:
	}

	log.Printf("📍 Customer %s subscribed to area (%.4f, %.4f) radius %.1fkm", client.AccountID, req.Lat, req.Lng, req.RadiusKm)
}

// handleUnsubscribe menghapus subscription area customer
func (h *WSLocationHandler) handleUnsubscribe(client *realtime.Client) {
	client.ClearSubscriptionArea()
	log.Printf("📍 Customer %s unsubscribed from live location", client.AccountID)
}

// handleGetNearby mengembalikan snapshot merchant terdekat tanpa subscribe
func (h *WSLocationHandler) handleGetNearby(client *realtime.Client, rawData json.RawMessage) {
	var req SubscribeAreaRequest
	if err := json.Unmarshal(rawData, &req); err != nil {
		return
	}

	if req.RadiusKm <= 0 {
		req.RadiusKm = 5
	}

	merchants, err := h.service.GetNearbyMerchants(req.Lat, req.Lng, req.RadiusKm, 50)
	if err != nil {
		return
	}

	response := realtime.WSEvent{
		Type: "NEARBY_SNAPSHOT",
		Data: map[string]interface{}{
			"merchants": merchants,
			"total":     len(merchants),
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		return
	}

	select {
	case client.Send <- data:
	default:
	}
}
