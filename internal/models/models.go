package models

import "database/sql"

type User struct {
	Uuid            string `json:"uuid,omitempty" db:"uuid,omitempty"`
	Email           string `json:"email,omitempty" db:"email,omitempty"`
	Password        string `json:"password,omitempty" db:"password,omitempty"`
	ConfirmPassword string `json:"confirm_password,omitempty" db:"confirm_password,omitempty"`
	PhoneNumber     string `json:"phone_number,omitempty" db:"phone_number,omitempty"`
	Role            string `json:"role,omitempty" db:"role,omitempty"`
	Authentication  string `json:"authentication,omitempty" db:"authentication,omitempty"`
}

type OTP struct{
	Otp string `json:"otp,omitempty" db:"otp,omitempty"`
}

type ResetPassword struct {
	PasswordResetCode sql.NullString `json:"password_reset_code" db:"password_reset_code,omitempty"`
	NewPassword       string         `json:"new_password,omitempty" db:"new_password,omitempty"`
	ConfirmPassword   string         `json:"confirm_password,omitempty" db:"confirm_password,omitempty"`
}
