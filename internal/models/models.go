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

	UserInfo
}

type UserInfo struct {
	Aadhar  string `json:"aadhar,omitempty" db:"aadhar,omitempty"`
	Phone   string `json:"phone,omitempty" db:"phone,omitempty"`
	Gender  string `json:"gender,omitempty" db:"gender,omitempty"`
	Address string `json:"address,omitempty" db:"address,omitempty"`
	Age     string `json:"age,omitempty" db:"age,omitempty"`
}

type OTP struct {
	Otp sql.NullString `json:"otp" db:"otp"`
}

type ResetPassword struct {
	Otp             string `json:"otp,omitempty" db:"otp,omitempty"`
	NewPassword     string `json:"new_password,omitempty" db:"new_password,omitempty"`
	ConfirmPassword string `json:"confirm_password,omitempty" db:"confirm_password,omitempty"`
}
