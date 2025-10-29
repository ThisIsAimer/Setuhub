package models

import (
	"time"
)

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
	Name    string `json:"name,omitempty" db:"name,omitempty"`
	Phone   string `json:"phone,omitempty" db:"phone,omitempty"`
	Gender  string `json:"gender,omitempty" db:"gender,omitempty"`
	Address string `json:"address,omitempty" db:"address,omitempty"`
	DateOfBirth string `json:"date_of_birth,omitempty" db:"date_of_birth,omitempty"`
}

type ResetPassword struct {
	Otp             string `json:"otp,omitempty" db:"otp,omitempty"`
	NewPassword     string `json:"new_password,omitempty" db:"new_password,omitempty"`
	ConfirmPassword string `json:"confirm_password,omitempty" db:"confirm_password,omitempty"`
}

type Coordinates struct {
	Latitude  float64 `json:"latitude,omitempty" db:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty" db:"longitude,omitempty"`
}

type NearbyUser struct {
	Uuid     string  `json:"uuid,omitempty" db:"uuid,omitempty"`
	Name     string  `json:"name,omitempty" db:"name,omitempty"`
	Phone    string  `json:"phone,omitempty" db:"phone,omitempty"`
	Distance float64 `json:"distance,omitempty" db:"distance,omitempty"`
}

//-------------------------------------------------------------------------------------------------------------------

type Post struct {
	PostUUID string `json:"post_uuid" db:"post_uuid"`
	UUID     string `json:"uuid" db:"uuid"` // FK to users(uuid)
	Name    string `json:"name,omitempty" db:"name,omitempty"`

	Type        string `json:"type,omitempty" db:"type,omitempty"`
	Title       string `json:"title,omitempty" db:"title,omitempty"`
	Description string `json:"description,omitempty" db:"description,omitempty"`
	Phone   string `json:"phone,omitempty" db:"phone,omitempty"`



	BloodGroup  string `json:"blood_group,omitempty" db:"blood_group,omitempty"`
	Gender  string `json:"gender,omitempty" db:"gender,omitempty"`
	Age     string `json:"age,omitempty" db:"age,omitempty"`

	Media string `json:"media,omitempty" db:"media,omitempty"` // nullable
	Coordinates
	Location string `json:"location,omitempty" db:"location"` // nullable
	Radius   int    `json:"radius,omitempty" db:"radius,omitempty"`     // nullable (meters)

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	EventAt   time.Time `json:"event_at" db:"event_at"` // nullable

}
