package models

import "time"

type Admin struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` 
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}


func (Admin) TableName() string {
	return "platform_admins"
}

type RegisterAdminInput struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"` 
}

type LoginAdminInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}