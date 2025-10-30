package models

import (
	"time"
)

type User struct {
	Uuid            string `json:"uuid,omitempty" db:"uuid,omitempty"`
	Email           string `json:"email,omitempty" db:"email,omitempty"`
	Password        string `json:"password,omitempty" db:"password,omitempty"`
	ConfirmPassword string `json:"confirmPassword,omitempty" db:"confirmPassword,omitempty"`
	PhoneNumber     string `json:"phoneNumber,omitempty" db:"phoneNumber,omitempty"`
	Role            string `json:"role,omitempty" db:"role,omitempty"`
	Authentication  string `json:"authentication,omitempty" db:"authentication,omitempty"`
	UserInfo
}

type UserInfo struct {
	Aadhar      string `json:"aadhar,omitempty" db:"aadhar,omitempty"`
	Name        string `json:"name,omitempty" db:"name,omitempty"`
	Phone       string `json:"phone,omitempty" db:"phone,omitempty"`
	Gender      string `json:"gender,omitempty" db:"gender,omitempty"`
	Address     string `json:"address,omitempty" db:"address,omitempty"`
	DateOfBirth string `json:"dateOfBirth,omitempty" db:"dateOfBirth,omitempty"`
}

type ResetPassword struct {
	Otp             string `json:"otp,omitempty" db:"otp,omitempty"`
	NewPassword     string `json:"newPassword,omitempty" db:"newPassword,omitempty"`
	ConfirmPassword string `json:"confirmPassword,omitempty" db:"confirmPassword,omitempty"`
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
	PostUUID    string `json:"postUuid" db:"postUuid"`
	UUID        string `json:"uuid" db:"uuid"` // FK to users(uuid)
	Name        string `json:"name,omitempty" db:"name,omitempty"`
	Type        string `json:"type,omitempty" db:"type,omitempty"`
	Title       string `json:"title,omitempty" db:"title,omitempty"`
	Description string `json:"description,omitempty" db:"description,omitempty"`
	Phone       string `json:"phone,omitempty" db:"phone,omitempty"`

	BloodGroup string `json:"bloodGroup,omitempty" db:"bloodGroup,omitempty"`
	Gender     string `json:"gender,omitempty" db:"gender,omitempty"`
	Age        string `json:"age,omitempty" db:"age,omitempty"`

	Media string `json:"media,omitempty" db:"media,omitempty"` // nullable
	Coordinates

	Location []LocationObj `json:"location,omitempty" db:"location,omitempty"` // nullable

	Radius int `json:"radius,omitempty" db:"radius,omitempty"` // nullable (meters)

	CreatedAt time.Time `json:"createdAt" db:"createdAt"`
	EventAt   time.Time `json:"eventAt" db:"eventAt"` // nullable
}

type LocationObj struct {
	City             string `json:"city,omitempty" db:"city,omitempty"`
	Country          string `json:"country,omitempty" db:"country,omitempty"`
	District         string `json:"district,omitempty" db:"district,omitempty"`
	FormattedAddress string `json:"formattedAddress,omitempty" db:"formattedAddress,omitempty"`
	IsoCountryCode   string `json:"isoCountryCode,omitempty" db:"isoCountryCode,omitempty"`
	Name             string `json:"name,omitempty" db:"name,omitempty"`
	PostalCode       string `json:"postalCode,omitempty" db:"postalCode,omitempty"`
	Region           string `json:"region,omitempty" db:"region,omitempty"`
	Street           string `json:"street,omitempty" db:"street,omitempty"`
	StreetNumber     string `json:"streetNumber,omitempty" db:"streetNumber,omitempty"`
	Subregion        string `json:"subregion,omitempty" db:"subregion,omitempty"`
	Timezone         string `json:"timezone,omitempty" db:"timezone,omitempty"`
}
