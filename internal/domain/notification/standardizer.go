package notification

import (
	"strings"
)

// StandardizeNotificationMetadata menstandarisasi type, target_type, target_id, dan business_id
// agar konsisten dengan kebutuhan routing frontend:
// Standar Type: "signal_post", "chat", "new_review", "post_like", "kyc", "invitation", "promotion", dll.
// Target Type: "signal" | "chat" | "review" | "post" | "kyc" | "invitation" | "promotion" | "business"
// Target ID: ID unik entitas terkait (signal_id / conversation_id / review_id / post_id / promotion_id / assignment_id)
func StandardizeNotificationMetadata(rawType, refID string, data map[string]interface{}) (standardType, targetType, targetID, businessID string) {
	if data == nil {
		data = make(map[string]interface{})
	}

	cleanRefID := strings.TrimSpace(refID)
	lowerType := strings.ToLower(strings.TrimSpace(rawType))

	// 1. Ambil target_id dan target_type yang sudah ada (jika sudah di-set secara eksplisit)
	if tid, ok := data["target_id"].(string); ok && strings.TrimSpace(tid) != "" {
		targetID = strings.TrimSpace(tid)
	}
	if tt, ok := data["target_type"].(string); ok && strings.TrimSpace(tt) != "" {
		targetType = strings.TrimSpace(tt)
	}

	// 2. Ambil business_id
	if bid, ok := data["business_id"].(string); ok && strings.TrimSpace(bid) != "" {
		businessID = strings.TrimSpace(bid)
	} else if strings.HasPrefix(cleanRefID, "BUS-") {
		businessID = cleanRefID
	}

	// 3. Mapping berdasarkan lowerType dan cleanRefID
	switch {
	// A. Sinyal Permintaan Kebutuhan
	case lowerType == "new_demand_signal" || lowerType == "signal" || lowerType == "signal_post" || lowerType == "signal_completed":
		standardType = "signal_post"
		targetType = "signal"
		if targetID == "" {
			if sid, ok := data["signal_id"].(string); ok && sid != "" {
				targetID = sid
			} else if pid, ok := data["post_id"].(string); ok && pid != "" {
				targetID = pid
			} else if strings.HasPrefix(cleanRefID, "SIG-") {
				targetID = cleanRefID
			}
		}

	// B. Obrolan / Chat (Signal Reply, Signal Offer, Direct Business Chat, Support Chat, Signal Message)
	case lowerType == "chat" || lowerType == "business_chat" || lowerType == "support_chat" ||
		lowerType == "signal_reply" || lowerType == "signal_offer" || lowerType == "respond_demand_signal" || lowerType == "signal_message":
		standardType = "chat"
		targetType = "chat"
		if targetID == "" {
			if cid, ok := data["conversation_id"].(string); ok && cid != "" {
				targetID = cid
			} else if oid, ok := data["offer_id"].(string); ok && oid != "" {
				targetID = oid
			} else if sid, ok := data["session_id"].(string); ok && sid != "" {
				targetID = sid
			} else if strings.HasPrefix(cleanRefID, "CONV-") {
				targetID = cleanRefID
			}
		}

	// C. Ulasan / Review (Toko & Produk)
	case strings.Contains(lowerType, "review"):
		standardType = "new_review"
		targetType = "review"
		if targetID == "" {
			if rid, ok := data["review_id"].(string); ok && rid != "" {
				targetID = rid
			} else if strings.HasPrefix(cleanRefID, "REV-") {
				targetID = cleanRefID
			} else if cid, ok := data["comment_id"].(string); ok && cid != "" {
				targetID = cid
			}
		}

	// D. Postingan & Interaksi Produk (Like, Comment, Reply)
	case lowerType == "post_like" || lowerType == "product_like":
		standardType = "post_like"
		targetType = "post"
		if targetID == "" {
			if pid, ok := data["post_id"].(string); ok && pid != "" {
				targetID = pid
			} else if prid, ok := data["product_id"].(string); ok && prid != "" {
				targetID = prid
			} else if strings.HasPrefix(cleanRefID, "PST-") || strings.HasPrefix(cleanRefID, "PRD-") {
				targetID = cleanRefID
			}
		}

	case lowerType == "business_new_post" || lowerType == "post":
		standardType = "business_new_post"
		targetType = "post"
		if targetID == "" {
			if pid, ok := data["post_id"].(string); ok && pid != "" {
				targetID = pid
			} else if strings.HasPrefix(cleanRefID, "PST-") {
				targetID = cleanRefID
			}
		}

	case strings.Contains(lowerType, "comment") || strings.Contains(lowerType, "post_reply"):
		standardType = "post_comment"
		targetType = "post"
		if targetID == "" {
			if pid, ok := data["post_id"].(string); ok && pid != "" {
				targetID = pid
			} else if prid, ok := data["product_id"].(string); ok && prid != "" {
				targetID = prid
			} else if strings.HasPrefix(cleanRefID, "PST-") || strings.HasPrefix(cleanRefID, "PRD-") {
				targetID = cleanRefID
			}
		}

	// E. Promosi (Per Produk & Bundling)
	case strings.Contains(lowerType, "promo") || strings.Contains(lowerType, "bundling"):
		standardType = "promotion"
		targetType = "promotion"
		if targetID == "" {
			if prmid, ok := data["promotion_id"].(string); ok && prmid != "" {
				targetID = prmid
			} else if strings.HasPrefix(cleanRefID, "PRM-") {
				targetID = cleanRefID
			}
		}

	// F. Verifikasi Identitas (KYC)
	case strings.Contains(lowerType, "kyc"):
		standardType = "kyc"
		targetType = "kyc"
		if targetID == "" {
			if accID, ok := data["account_id"].(string); ok && accID != "" {
				targetID = accID
			} else if cleanRefID != "" {
				targetID = cleanRefID
			}
		}

	// G. Undangan & Manajemen Tim Toko / Transfer
	case strings.Contains(lowerType, "invit") || strings.Contains(lowerType, "admin_") ||
		strings.Contains(lowerType, "spv_") || strings.Contains(lowerType, "transfer"):
		standardType = "invitation"
		targetType = "invitation"
		if targetID == "" {
			if asnid, ok := data["assignment_id"].(string); ok && asnid != "" {
				targetID = asnid
			} else if trfid, ok := data["transfer_id"].(string); ok && trfid != "" {
				targetID = trfid
			} else if strings.HasPrefix(cleanRefID, "ASN-") || strings.HasPrefix(cleanRefID, "TRF-") {
				targetID = cleanRefID
			}
		}

	// H. Live Streaming & Jadwal Buka/Tutup Toko
	case strings.Contains(lowerType, "live") || strings.Contains(lowerType, "schedule") || strings.Contains(lowerType, "store_status"):
		standardType = rawType
		targetType = "business"
		if targetID == "" {
			if businessID != "" {
				targetID = businessID
			} else if cleanRefID != "" {
				targetID = cleanRefID
			}
		}
		if businessID == "" {
			businessID = targetID
		}

	// I. Welcome
	case strings.Contains(lowerType, "welcome"):
		standardType = "welcome_merchant"
		targetType = "welcome"
		if targetID == "" && cleanRefID != "" {
			targetID = cleanRefID
		}

	// J. Fallback berdasarkan prefix ReferenceID jika type tidak dikenal
	default:
		standardType = rawType
		if strings.HasPrefix(cleanRefID, "SIG-") {
			targetType = "signal"
			targetID = cleanRefID
		} else if strings.HasPrefix(cleanRefID, "CONV-") {
			targetType = "chat"
			targetID = cleanRefID
		} else if strings.HasPrefix(cleanRefID, "REV-") {
			targetType = "review"
			targetID = cleanRefID
		} else if strings.HasPrefix(cleanRefID, "PST-") || strings.HasPrefix(cleanRefID, "PRD-") {
			targetType = "post"
			targetID = cleanRefID
		} else if strings.HasPrefix(cleanRefID, "PRM-") {
			targetType = "promotion"
			targetID = cleanRefID
		} else if strings.HasPrefix(cleanRefID, "ASN-") || strings.HasPrefix(cleanRefID, "TRF-") {
			targetType = "invitation"
			targetID = cleanRefID
		} else if strings.HasPrefix(cleanRefID, "BUS-") {
			targetType = "business"
			targetID = cleanRefID
		} else {
			targetType = "general"
			if cleanRefID != "" {
				targetID = cleanRefID
			}
		}
	}

	// Pastikan targetID tidak kosong jika refID ada
	if targetID == "" && cleanRefID != "" {
		targetID = cleanRefID
	}

	return standardType, targetType, targetID, businessID
}
