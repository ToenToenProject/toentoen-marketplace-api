package models

import "time"

// ==========================================
// 1. ENTITAS DATABASE (GORM MODELS)
// ==========================================

type BusinessAdmin struct {
	AccountID string    `gorm:"primaryKey;column:account_id" json:"account_id"`
	// 🚨 OwnerID DIHAPUS: Admin bisa bekerja untuk banyak toko/owner yang berbeda
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`

	// Relasi
	Account     Account                   `gorm:"foreignKey:AccountID;references:AccountID" json:"account"`
	Assignments []BusinessAdminAssignment `gorm:"foreignKey:AdminID;references:AccountID" json:"assignments"`
}

type BusinessAdminAssignment struct {
	// 🪄 Gunakan AssignmentID sebagai Primary Key tunggal agar lebih mudah saat update status (terima/tolak undangan)
	AssignmentID string `gorm:"primaryKey;column:assignment_id" json:"assignment_id"`
	
	AdminID    string `gorm:"column:admin_id;uniqueIndex:idx_admin_business" json:"admin_id"`
	BusinessID string `gorm:"column:business_id;uniqueIndex:idx_admin_business" json:"business_id"`
	InvitedBy  string `gorm:"column:invited_by" json:"invited_by"` // Menyimpan AccountID owner yang mengundang

	Role       string  `gorm:"default:'admin';not null;column:role" json:"role"`
	
	// 🪄 IsActive diganti menjadi Status untuk mengakomodasi flow undangan
	Status     string  `gorm:"default:'pending';column:status" json:"status"` // Enum: pending, active, rejected, revoked
	
	ShiftStart *string `gorm:"column:shift_start" json:"shift_start"`
	ShiftEnd   *string `gorm:"column:shift_end" json:"shift_end"`
	
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`

	// Relasi
	Business *Business      `gorm:"foreignKey:BusinessID;references:BusinessID" json:"business,omitempty"`
	Admin    *BusinessAdmin `gorm:"foreignKey:AdminID;references:AccountID" json:"admin,omitempty"`
}

// ==========================================
// 2. DTO UNTUK INPUT (REQUEST)
// ==========================================

// DTO untuk Owner saat mengundang seseorang menjadi Admin
type InviteAdminInput struct {
	Email      string  `json:"email" validate:"required,email"` // 🪄 Diganti jadi Email
	BusinessID string  `json:"business_id" validate:"required"`
	Role       string  `json:"role"`
	ShiftStart *string `json:"shift_start"`
	ShiftEnd   *string `json:"shift_end"`
}

// DTO untuk Calon Admin saat menerima atau menolak undangan
type RespondInvitationInput struct {
	Status string `json:"status" validate:"required"` // Isinya hanya boleh "active" (terima) atau "rejected" (tolak)
}

// DTO untuk Owner saat mengubah jabatan/jadwal Admin yang sudah ada
type UpdateAssignmentInput struct {
	// 🚨 Data pribadi seperti Fullname, Email, Password DIHAPUS. Owner tidak berhak mengubahnya.
	Role       *string `json:"role"`
	ShiftStart *string `json:"shift_start"`
	ShiftEnd   *string `json:"shift_end"`
	Status     *string `json:"status"` // Digunakan owner jika ingin memberhentikan admin ("revoked")
}