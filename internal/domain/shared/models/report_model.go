package models

import "time"

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

type ProductReviewReport struct {
	ReportID  string    `gorm:"primaryKey;column:report_id" json:"report_id"`
	ReviewID  string    `gorm:"column:review_id;index" json:"review_id"` // 👈 Foreign Key ke Product Review
	AccountID string    `gorm:"column:account_id;index" json:"account_id"`
	Reason    string    `gorm:"column:reason" json:"reason"`
	Status    string    `gorm:"column:status;default:'pending'" json:"status"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
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

type CreatePostReportInput struct {
	Reason      string   `json:"reason" validate:"required"`
	Description *string  `json:"description,omitempty"`
	Evidence    []string `json:"evidence,omitempty"`
}

type BusinessReviewReport struct {
	ReportID    int64      `gorm:"primaryKey;autoIncrement;column:report_id" json:"report_id"`
	AccountID   string     `gorm:"not null;column:account_id" json:"account_id"`
	ReviewID    string     `gorm:"not null;column:review_id" json:"review_id"`
	Reason      string     `gorm:"not null;column:reason" json:"reason"`
	Status      string     `gorm:"not null;column:status;default:'pending'" json:"status"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

type BusinessReviewCommentReport struct {
	ReportID    int64      `gorm:"primaryKey;autoIncrement;column:report_id" json:"report_id"`
	AccountID   string     `gorm:"not null;column:account_id" json:"account_id"`
	CommentID   string     `gorm:"not null;column:comment_id" json:"comment_id"`
	Reason      string     `gorm:"not null;column:reason" json:"reason"`
	Status      string     `gorm:"not null;column:status;default:'pending'" json:"status"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}
