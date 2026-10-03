package models

import (
	"strings"
	"time"
)

// ==========================================
// 1. DTO INPUT PELAPORAN UNIVERSAL
// ==========================================

type CreateReportInput struct {
	TargetType  string   `json:"target_type" validate:"omitempty,oneof=business product review comment post"`
	TargetID    string   `json:"target_id"`
	Reason      string   `json:"reason" validate:"required"`
	Description *string  `json:"description,omitempty"`
	Evidence    []string `json:"evidence,omitempty"`
}

// Alias untuk kompatibilitas kode lain
type CreatePostReportInput = CreateReportInput

// ==========================================
// 2. MODEL TABEL UNIVERSAL REPORTS
// ==========================================

type Report struct {
	ReportID    string     `gorm:"primaryKey;column:report_id" json:"report_id"`
	AccountID   string     `gorm:"column:account_id;index" json:"account_id"`
	TargetType  string     `gorm:"column:target_type;index" json:"target_type"` // "business" | "product" | "review" | "comment" | "post"
	TargetID    string     `gorm:"column:target_id;index" json:"target_id"`     // BUS-..., PRD-..., REV-..., CMT-..., PST-...
	Reason      string     `gorm:"column:reason" json:"reason"`
	Description *string    `gorm:"column:description" json:"description"`
	Evidence    []string   `gorm:"column:evidence;serializer:json;default:'[]'" json:"evidence"`
	Status      string     `gorm:"column:status;default:'pending'" json:"status"`
	Priority    string     `gorm:"column:priority;default:'medium'" json:"priority"` // "high" | "medium" | "low"
	AdminNote   *string    `gorm:"column:admin_note" json:"admin_note"`
	ResolvedAt  *time.Time `gorm:"column:resolved_at" json:"resolved_at"`
	ResolvedBy  *string    `gorm:"column:resolved_by" json:"resolved_by"`
	CreatedAt   time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at" json:"updated_at"`

	Account *Account `gorm:"foreignKey:AccountID;references:AccountID" json:"account,omitempty"`
}

func (Report) TableName() string {
	return "reports"
}

type ReportResponse struct {
	ReportID       string     `json:"report_id"`
	AccountID      string     `json:"account_id"`
	ReporterName   string     `json:"reporter_name"`
	ReporterAvatar *string    `json:"reporter_avatar,omitempty"`
	TargetType     string     `json:"target_type"`
	TargetID       string     `json:"target_id"`
	Reason         string     `json:"reason"`
	Description    *string    `json:"description"`
	Evidence       []string   `json:"evidence"`
	Status         string     `json:"status"`
	Priority       string     `json:"priority"`
	AdminNote      *string    `json:"admin_note,omitempty"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy     *string    `json:"resolved_by,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type UpdateReportStatusInput struct {
	Status    string  `json:"status" validate:"required,oneof=pending in_review resolved rejected dismissed"`
	AdminNote *string `json:"admin_note,omitempty"`
	Action    *string `json:"action,omitempty"` // "delete_target" | "none"
}

type ReportFilterQuery struct {
	Status     string `query:"status"`
	Priority   string `query:"priority"`
	TargetType string `query:"target_type"`
	TargetID   string `query:"target_id"`
	Page       int    `query:"page"`
	Limit      int    `query:"limit"`
}

// ==========================================
// 3. MODEL PELAPORAN BISNIS LAMA (UNTUK KOMPATIBILITAS)
// ==========================================

type BusinessReport struct {
	ReportID    int64      `gorm:"primaryKey;autoIncrement;column:report_id" json:"report_id"`
	AccountID   string     `gorm:"not null;column:account_id" json:"account_id"`
	BusinessID  string     `gorm:"not null;column:business_id" json:"business_id"`
	Reason      string     `gorm:"not null;column:reason" json:"reason"`
	Description *string    `gorm:"column:description" json:"description"`
	Status      string     `gorm:"not null;column:status;default:'pending'" json:"status"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	Evidence []BusinessReportEvidence `gorm:"foreignKey:ReportID;references:ReportID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"evidence,omitempty"`
	Business *Business                `gorm:"foreignKey:BusinessID;references:BusinessID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"business,omitempty"`
	Account  *Account                 `gorm:"foreignKey:AccountID;references:AccountID" json:"account,omitempty"`
}

type BusinessReportEvidence struct {
	EvidenceID int64  `gorm:"primaryKey;autoIncrement;column:evidence_id" json:"evidence_id"`
	ReportID   int64  `gorm:"not null;column:report_id" json:"report_id"`
	FileURL    string `gorm:"not null;column:file_url" json:"file_url"`

	Report *BusinessReport `gorm:"foreignKey:ReportID;references:ReportID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"report,omitempty"`
}

type BusinessReviewReport struct {
	ReportID    int64      `gorm:"primaryKey;autoIncrement;column:report_id" json:"report_id"`
	AccountID   string     `gorm:"not null;column:account_id" json:"account_id"`
	ReviewID    string     `gorm:"not null;column:review_id" json:"review_id"`
	Reason      string     `gorm:"not null;column:reason" json:"reason"`
	Description *string    `gorm:"column:description" json:"description"`
	Evidence    []string   `gorm:"column:evidence;serializer:json;default:'[]'" json:"evidence"`
	Status      string     `gorm:"not null;column:status;default:'pending'" json:"status"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

type BusinessReviewCommentReport struct {
	ReportID    int64      `gorm:"primaryKey;autoIncrement;column:report_id" json:"report_id"`
	AccountID   string     `gorm:"not null;column:account_id" json:"account_id"`
	CommentID   string     `gorm:"not null;column:comment_id" json:"comment_id"`
	Reason      string     `gorm:"not null;column:reason" json:"reason"`
	Description *string    `gorm:"column:description" json:"description"`
	Evidence    []string   `gorm:"column:evidence;serializer:json;default:'[]'" json:"evidence"`
	Status      string     `gorm:"not null;column:status;default:'pending'" json:"status"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

type ProductReviewReport struct {
	ReportID    string    `gorm:"primaryKey;column:report_id" json:"report_id"`
	ReviewID    string    `gorm:"column:review_id;index" json:"review_id"`
	AccountID   string    `gorm:"column:account_id;index" json:"account_id"`
	Reason      string    `gorm:"column:reason" json:"reason"`
	Description *string   `gorm:"column:description" json:"description"`
	Evidence    []string  `gorm:"column:evidence;serializer:json;default:'[]'" json:"evidence"`
	Status      string    `gorm:"column:status;default:'pending'" json:"status"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"created_at"`
}

type ProductReviewCommentReport struct {
	ReportID    string    `gorm:"primaryKey;column:report_id" json:"report_id"`
	CommentID   string    `gorm:"column:comment_id" json:"comment_id"`
	AccountID   string    `gorm:"column:account_id" json:"account_id"`
	Reason      string    `gorm:"column:reason" json:"reason"`
	Description *string   `gorm:"column:description" json:"description"`
	Evidence    []string  `gorm:"column:evidence;serializer:json;default:'[]'" json:"evidence"`
	Status      string    `gorm:"column:status;default:'pending'" json:"status"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"created_at"`
}

type ProductCommentReport struct {
	ReportID    string    `gorm:"primaryKey;column:report_id" json:"report_id"`
	CommentID   string    `gorm:"column:comment_id" json:"comment_id"`
	AccountID   string    `gorm:"column:account_id" json:"account_id"`
	Reason      string    `gorm:"column:reason" json:"reason"`
	Description *string   `gorm:"column:description" json:"description"`
	Evidence    []string  `gorm:"column:evidence;serializer:json;default:'[]'" json:"evidence"`
	Status      string    `gorm:"column:status;default:'pending'" json:"status"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"created_at"`
}

type BusinessPostReport struct {
	ReportID    int64      `gorm:"primaryKey;autoIncrement;column:report_id" json:"report_id"`
	AccountID   string     `gorm:"not null;column:account_id" json:"account_id"`
	PostID      string     `gorm:"not null;column:post_id" json:"post_id"`
	Reason      string     `gorm:"not null;column:reason" json:"reason"`
	Description *string    `gorm:"column:description" json:"description"`
	Status      string     `gorm:"not null;column:status;default:'pending'" json:"status"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	Evidence []BusinessPostReportEvidence `gorm:"foreignKey:ReportID" json:"evidence,omitempty"`
}

type BusinessPostReportEvidence struct {
	EvidenceID int64  `gorm:"primaryKey;autoIncrement;column:evidence_id" json:"evidence_id"`
	ReportID   int64  `gorm:"not null;column:report_id" json:"report_id"`
	FileURL    string `gorm:"not null;column:file_url" json:"file_url"`
}

type BusinessPostCommentReport struct {
	ReportID    string    `gorm:"primaryKey;column:report_id" json:"report_id"`
	CommentID   string    `gorm:"column:comment_id" json:"comment_id"`
	AccountID   string    `gorm:"column:account_id" json:"account_id"`
	Reason      string    `gorm:"column:reason" json:"reason"`
	Description *string   `gorm:"column:description" json:"description"`
	Evidence    []string  `gorm:"column:evidence;serializer:json;default:'[]'" json:"evidence"`
	Status      string    `gorm:"column:status;default:'pending'" json:"status"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"created_at"`
}

// ==========================================
// 4. MASTER KATEGORI & ALASAN LAPORAN (REPORT CATEGORIES)
// ==========================================

type ReportSubCategory struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ReportCategory struct {
	ID             string              `json:"id"`
	ParentCategory string              `json:"parent_category"`
	TargetType     string              `json:"target_type"`
	SubCategories  []ReportSubCategory `json:"sub_categories"`
}

var MasterReportCategories = []ReportCategory{
	{
		ID:             "cat_biz_01",
		ParentCategory: "Report a Business",
		TargetType:     "business",
		SubCategories: []ReportSubCategory{
			{ID: "sub_biz_1", Name: "Fake or fraudulent business"},
			{ID: "sub_biz_2", Name: "Inaccurate business information"},
			{ID: "sub_biz_3", Name: "Policy violation (Illegal business)"},
			{ID: "sub_biz_4", Name: "Sham or misleading practices"},
			{ID: "sub_biz_5", Name: "Other"},
		},
	},
	{
		ID:             "cat_post_02",
		ParentCategory: "Report a post",
		TargetType:     "post",
		SubCategories: []ReportSubCategory{
			{ID: "sub_post_1", Name: "Spam content"},
			{ID: "sub_post_2", Name: "Inappropriate media"},
			{ID: "sub_post_3", Name: "Copyright violation"},
		},
	},
	{
		ID:             "cat_cmt_03",
		ParentCategory: "Report a comment",
		TargetType:     "comment",
		SubCategories: []ReportSubCategory{
			{ID: "sub_cmt_1", Name: "Hate speech"},
			{ID: "sub_cmt_2", Name: "Harassment or bullying"},
			{ID: "sub_cmt_3", Name: "Promotional spam"},
		},
	},
	{
		ID:             "cat_prod_04",
		ParentCategory: "Report a product",
		TargetType:     "product",
		SubCategories: []ReportSubCategory{
			{ID: "sub_prod_1", Name: "Counterfeit item"},
			{ID: "sub_prod_2", Name: "Misleading description"},
			{ID: "sub_prod_3", Name: "Prohibited item"},
		},
	},
	{
		ID:             "cat_rev_05",
		ParentCategory: "Report a review",
		TargetType:     "review",
		SubCategories: []ReportSubCategory{
			{ID: "sub_rev_1", Name: "Fake review"},
			{ID: "sub_rev_2", Name: "Offensive language"},
			{ID: "sub_rev_3", Name: "Irrelevant to product"},
		},
	},
}

func GetReportCategories(targetType string) []ReportCategory {
	if targetType == "" {
		return MasterReportCategories
	}
	targetType = strings.ToLower(strings.TrimSpace(targetType))
	var filtered []ReportCategory
	for _, cat := range MasterReportCategories {
		if strings.ToLower(cat.TargetType) == targetType || strings.Contains(strings.ToLower(cat.ParentCategory), targetType) {
			filtered = append(filtered, cat)
		}
	}
	if len(filtered) == 0 {
		return MasterReportCategories
	}
	return filtered
}

