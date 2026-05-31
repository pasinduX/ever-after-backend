package dto

import "time"

type GuestSide string

type GuestStatus string

const (
	GuestSideBride GuestSide = "bride"
	GuestSideGroom GuestSide = "groom"
	GuestSideBoth  GuestSide = "both"

	GuestStatusPending   GuestStatus = "pending"
	GuestStatusConfirmed GuestStatus = "confirmed"
	GuestStatusDeclined  GuestStatus = "declined"
)

type Guest struct {
	ID             string      `json:"id" bson:"_id,omitempty"`
	UserID         string      `json:"user_id" bson:"user_id"`
	WeddingID      string      `json:"wedding_id" bson:"wedding_id"`
	CaptainName    string      `json:"captain_name" bson:"captain_name"`
	Phone          string      `json:"phone,omitempty" bson:"phone,omitempty"`
	Side           GuestSide   `json:"side" bson:"side"`
	MembersInvited int         `json:"members_invited" bson:"members_invited"`
	MembersComing  int         `json:"members_coming" bson:"members_coming"`
	Status         GuestStatus `json:"status" bson:"status"`
	Notes          string      `json:"notes,omitempty" bson:"notes,omitempty"`
	CreatedAt      time.Time   `json:"created_at" bson:"created_at"`
}

type CreateGuestRequest struct {
	CaptainName    string      `json:"captain_name"`
	Phone          string      `json:"phone,omitempty"`
	Side           GuestSide   `json:"side"`
	MembersInvited *int        `json:"members_invited,omitempty"`
	MembersComing  *int        `json:"members_coming,omitempty"`
	Status         GuestStatus `json:"status,omitempty"`
	Notes          *string     `json:"notes,omitempty"`
}

type UpdateGuestRequest struct {
	CaptainName    *string      `json:"captain_name,omitempty"`
	Phone          *string      `json:"phone,omitempty"`
	Side           *GuestSide   `json:"side,omitempty"`
	MembersInvited *int         `json:"members_invited,omitempty"`
	MembersComing  *int         `json:"members_coming,omitempty"`
	Status         *GuestStatus `json:"status,omitempty"`
	Notes          *string      `json:"notes,omitempty"`
}
