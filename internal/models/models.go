package models

import (
	"time"
)

type User struct {
	Uuid            string `json:"uuid,omitempty" db:"uuid,omitempty"`
	Email           string `json:"email,omitempty" db:"email,omitempty"`
	Password        string `json:"password,omitempty" db:"password,omitempty"`
	ConfirmPassword string `json:"confirmPassword,omitempty" db:"confirm_password,omitempty"`
	Role            string `json:"role,omitempty" db:"role,omitempty"`
	Authentication  string `json:"authentication,omitempty" db:"authentication,omitempty"`

	UserInfo

	ProfilePhotoURL string `json:"profilePhotoUrl" db:"profile_photo_url"`
}

type UserInfo struct {
	Aadhar      string `json:"aadhar,omitempty" db:"aadhar,omitempty"`
	Name        string `json:"name,omitempty" db:"name,omitempty"`
	Phone       string `json:"phone,omitempty" db:"phone,omitempty"`
	Gender      string `json:"gender,omitempty" db:"gender,omitempty"`
	Address     string `json:"address,omitempty" db:"address,omitempty"`
	DateOfBirth string `json:"dateOfBirth,omitempty" db:"date_of_birth,omitempty"`
}

type ResetPassword struct {
	Otp             string `json:"otp,omitempty" db:"otp,omitempty"`
	NewPassword     string `json:"newPassword,omitempty" db:"new_password,omitempty"`
	ConfirmPassword string `json:"confirmPassword,omitempty" db:"confirm_password,omitempty"`
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
	PostUUID    string `json:"postUuid" db:"post_uuid"`
	UUID        string `json:"uuid" db:"uuid"` // FK to users(uuid)
	Name        string `json:"name,omitempty" db:"name,omitempty"`
	Type        string `json:"type,omitempty" db:"type,omitempty"`
	Title       string `json:"title,omitempty" db:"title,omitempty"`
	Description string `json:"description,omitempty" db:"description,omitempty"`
	Phone       string `json:"phone,omitempty" db:"phone,omitempty"`

	BloodGroup string `json:"bloodGroup,omitempty" db:"blood_group,omitempty"`
	Gender     string `json:"gender,omitempty" db:"gender,omitempty"`
	Age        string `json:"age,omitempty" db:"age,omitempty"`

	Media []string `json:"media,omitempty" db:"media,omitempty"` // nullable
	Coordinates

	Location []LocationObj `json:"location,omitempty" db:"location,omitempty"` // nullable

	Radius int `json:"radius,omitempty" db:"radius,omitempty"` // nullable (meters)

	CreatedAt time.Time `json:"createdAt,omitempty" db:"created_at,omitempty"`
	EventAt   time.Time `json:"eventAt,omitempty" db:"event_at,omitempty"` // nullable

	ProfilePhotoURL string `json:"profilePhotoUrl" db:"profile_photo_url"`

	InterestedCount int  `json:"interestedCount" db:"interested_count"`
	Interested      bool `json:"interested" db:"interested"`

	CommentCount int `json:"commentCount" db:"comment_count"`

	Done bool `json:"done" db:"done"`
}

type LocationObj struct {
	City             string `json:"city,omitempty" db:"city,omitempty"`
	Country          string `json:"country,omitempty" db:"country,omitempty"`
	District         string `json:"district,omitempty" db:"district,omitempty"`
	FormattedAddress string `json:"formattedAddress,omitempty" db:"formatted_address,omitempty"`
	IsoCountryCode   string `json:"isoCountryCode,omitempty" db:"iso_country_code,omitempty"`
	Name             string `json:"name,omitempty" db:"name,omitempty"`
	PostalCode       string `json:"postalCode,omitempty" db:"postal_code,omitempty"`
	Region           string `json:"region,omitempty" db:"region,omitempty"`
	Street           string `json:"street,omitempty" db:"street,omitempty"`
	StreetNumber     string `json:"streetNumber,omitempty" db:"street_number,omitempty"`
	Subregion        string `json:"subregion,omitempty" db:"subregion,omitempty"`
	Timezone         string `json:"timezone,omitempty" db:"timezone,omitempty"`
}

type InterestResult struct {
	Uuid            string
	Changed         bool
	InterestedCount int
	Type            string
}

type Comment struct {
	CommentUUID     string    `json:"commentUuid,omitempty" db:"comment_uuid"`
	PostUUID        string    `json:"postUuid,omitempty" db:"post_uuid"`
	Uuid            string    `json:"uuid,omitempty" db:"uuid"`
	Content         string    `json:"content,omitempty" db:"content"`
	CreatedAt       time.Time `json:"createdAt,omitempty" db:"created_at"`
	Edited          bool      `json:"edited,omitempty" db:"edited"`
	CommentCount    int       `json:"commentCount,omitempty" db:"comment_count"`
	ProfilePhotoURL string    `json:"profilePhotoUrl" db:"profile_photo_url"`
}

// Did----------------------------------------------------------------------------------
type CreateVerificationRequest struct {
	UserID string `json:"user_id"`
}

type DiditSessionResponse struct {
	SessionID string `json:"session_id"`
	URL       string `json:"url"`
}

type DiditWebhook struct {
	SessionID  string `json:"session_id"`
	Status     string `json:"status"`
	VendorData string `json:"vendor_data"`
}
