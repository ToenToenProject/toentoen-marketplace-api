package models

import "time"

type BusinessAdmin struct {
	AccountID      string    `gorm:"primaryKey;column:account_id" json:"account_id"`
	KtpNumber      *string   `gorm:"unique;column:ktp_number" json:"ktp_number"`
	KtpPhotoUrl    *string   `gorm:"column:ktp_photo_url" json:"ktp_photo_url"`
	KtpName        *string   `gorm:"column:ktp_name" json:"ktp_name"`
	KtpBirthPlace  *string   `gorm:"column:ktp_birth_place" json:"ktp_birth_place"`
	KtpBirthDate   *string   `gorm:"type:date;column:ktp_birth_date" json:"ktp_birth_date"`
	KtpGender      *string   `gorm:"column:ktp_gender" json:"ktp_gender"`
	KtpAddress     *string   `gorm:"column:ktp_address" json:"ktp_address"`
	KtpRtRw        *string   `gorm:"column:ktp_rt_rw" json:"ktp_rt_rw"`
	KtpVillage     *string   `gorm:"column:ktp_village" json:"ktp_village"`
	KtpDistrict    *string   `gorm:"column:ktp_district" json:"ktp_district"`
	KtpReligion    *string   `gorm:"column:ktp_religion" json:"ktp_religion"`
	KtpOccupation  *string   `gorm:"column:ktp_occupation" json:"ktp_occupation"`
	KtpCitizenship *string   `gorm:"column:ktp_citizenship" json:"ktp_citizenship"`
	InvitationCode *string   `gorm:"unique;column:invitation_code" json:"invitation_code"`
	CreatedAt      time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at" json:"updated_at"`

	Account     *Account                  `gorm:"foreignKey:AccountID;references:AccountID" json:"account,omitempty"`
	Assignments []BusinessAdminAssignment `gorm:"foreignKey:AdminID;references:AccountID" json:"assignments"`
}

func (BusinessAdmin) TableName() string {
	return "business_admins"
}

type BusinessAdminAssignment struct {
	AssignmentID string `gorm:"primaryKey;column:assignment_id" json:"assignment_id"`

	AdminID      string  `gorm:"column:admin_id;uniqueIndex:idx_admin_business" json:"admin_id"`
	BusinessID   string  `gorm:"column:business_id;uniqueIndex:idx_admin_business" json:"business_id"`
	InvitedBy    string  `gorm:"column:invited_by" json:"invited_by"`
	SupervisorID *string `gorm:"column:supervisor_id" json:"supervisor_id"` // ID Supervisor atasan (nil jika langsung di bawah Owner)

	Role   string `gorm:"default:'staff';not null;column:role" json:"role"`  // "supervisor" atau "staff"
	Status string `gorm:"default:'pending';column:status" json:"status"` // "pending", "pending_owner_approval", "active", "scheduled_for_deletion", "revoked", "rejected"

	// Grace Period (3 Hari)
	DeletionScheduledAt *time.Time `gorm:"column:deletion_scheduled_at" json:"deletion_scheduled_at"`
	DeletionInitiatedBy *string    `gorm:"column:deletion_initiated_by" json:"deletion_initiated_by"` // "owner", "supervisor", "self"

	// Cooldown Penolakan (3x reject -> 7 days cooldown)
	RejectionCount int        `gorm:"default:0;column:rejection_count" json:"rejection_count"`
	LastRejectedAt *time.Time `gorm:"column:last_rejected_at" json:"last_rejected_at"`

	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`

	// Relasi
	Business   *Business            `gorm:"foreignKey:BusinessID;references:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"business,omitempty"`
	Admin      *BusinessAdmin       `gorm:"foreignKey:AdminID;references:AccountID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"admin,omitempty"`
	Supervisor *Account             `gorm:"foreignKey:SupervisorID;references:AccountID" json:"supervisor,omitempty"`
	Inviter    *Account             `gorm:"foreignKey:InvitedBy;references:AccountID" json:"inviter,omitempty"`
	Shifts     []BusinessAdminShift `gorm:"foreignKey:AssignmentID;references:AssignmentID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"shifts,omitempty"`
}

func (BusinessAdminAssignment) TableName() string {
	return "business_admin_assignments"
}

type BusinessAdminShift struct {
	AssignmentID string  `gorm:"primaryKey;column:assignment_id" json:"assignment_id"`
	DayOfWeek    int     `gorm:"primaryKey;column:day_of_week" json:"day_of_week"`
	StartTime    *string `gorm:"column:start_time" json:"start_time"`
	EndTime      *string `gorm:"column:end_time" json:"end_time"`
}

func (BusinessAdminShift) TableName() string {
	return "business_admin_shifts"
}

// DTO Input & Response

type InviteAdminInput struct {
	InvitationCode string                    `json:"invitation_code" validate:"required"` 
	BusinessID     string                    `json:"business_id"`
	Role           string                    `json:"role" validate:"required,oneof=supervisor staff"`
	Shifts         []BusinessAdminShiftInput `json:"shifts,omitempty"`
}

type BusinessAdminShiftInput struct {
	DayOfWeek int    `json:"day_of_week"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type RespondOwnerApprovalInput struct {
	Action string `json:"action" validate:"required,oneof=approve reject"` // "approve" atau "reject"
}

type RespondInvitationInput struct {
	Action string `json:"action" validate:"required,oneof=accept reject"` // "accept" atau "reject"
}

type UpdateAssignmentInput struct {
	Role         *string                   `json:"role" validate:"omitempty,oneof=supervisor staff"`
	SupervisorID *string                   `json:"supervisor_id"`
	Shifts       []BusinessAdminShiftInput `json:"shifts,omitempty"`
}

type UpdateBusinessAdminKYCInput struct {
	FirstName   *string `json:"first_name"`
	LastName    *string `json:"last_name"`
	Username    *string `json:"username"`
	Email       *string `json:"email"`
	PhoneNumber *string `json:"phone_number"`
	Title       *string `json:"title"`

	ProfilePictureUrl *string `json:"profile_picture_url"`
	Country           *string `json:"country"`

	KtpNumber      *string `json:"ktp_number"`
	KtpPhotoUrl    *string `json:"ktp_photo_url"`
	KtpName        *string `json:"ktp_name"`
	KtpBirthPlace  *string `json:"ktp_birth_place"`
	KtpBirthDate   *string `json:"ktp_birth_date"`
	KtpGender      *string `json:"ktp_gender"`
	KtpAddress     *string `json:"ktp_address"`
	KtpRtRw        *string `json:"ktp_rt_rw"`
	KtpVillage     *string `json:"ktp_village"`
	KtpDistrict    *string `json:"ktp_district"`
	KtpReligion    *string `json:"ktp_religion"`
	KtpOccupation  *string `json:"ktp_occupation"`
	KtpCitizenship *string `json:"ktp_citizenship"`
}

type AdminTreeNode struct {
	AccountID    string               `json:"account_id"`
	AssignmentID string               `json:"assignment_id,omitempty"`
	Name         string               `json:"name"`
	Username     string               `json:"username"`
	AvatarURL    *string              `json:"avatar_url"`
	Role         string               `json:"role"`
	Status       string               `json:"status,omitempty"`
	Shifts       []BusinessAdminShift `json:"shifts,omitempty"`
	Subordinates []AdminTreeNode      `json:"subordinates,omitempty"`
}

type AdminTreeResponse struct {
	BusinessID   string          `json:"business_id"`
	BusinessName string          `json:"business_name"`
	Owner        AdminTreeNode   `json:"owner"`
	Supervisors  []AdminTreeNode `json:"supervisors"`
	DirectStaff  []AdminTreeNode `json:"direct_staff"`
}

type AdminAssignmentListItem struct {
	AssignmentID string           `json:"assignment_id"`
	Role         string           `json:"role"`
	Status       string           `json:"status"`
	Business     BusinessListItem `json:"business"`
}
