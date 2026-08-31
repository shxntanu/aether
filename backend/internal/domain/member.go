package domain

import "time"

type MemberRole string

const (
	MemberRoleMember MemberRole = "member"
	MemberRoleAdmin  MemberRole = "admin"
)

type MemberStatus string

const (
	MemberStatusActive   MemberStatus = "active"
	MemberStatusDisabled MemberStatus = "disabled"
)

type Member struct {
	ID          MemberID     `json:"id"`
	Email       string       `json:"email"`
	DisplayName string       `json:"displayName"`
	OIDCSubject string       `json:"-"`
	Role        MemberRole   `json:"role"`
	Status      MemberStatus `json:"status"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

type Session struct {
	ID        string
	TokenHash string
	MemberID  MemberID
	CreatedAt time.Time
	ExpiresAt time.Time
}

type AuthFlow struct {
	StateHash    string
	Nonce        string
	PKCEVerifier string
	CreatedAt    time.Time
	ExpiresAt    time.Time
}
