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
	// CSRFHash stores the SHA-256 digest of the synchronizer token.
	CSRFHash  string
	MemberID  MemberID
	CreatedAt time.Time
	ExpiresAt time.Time
}

// AuthenticatedSession contains the member and raw CSRF token for an
// authenticated request. The session cookie token is intentionally absent.
type AuthenticatedSession struct {
	// Member is the active member associated with the server-side session.
	Member Member
	// CSRFToken is the raw synchronizer token accepted for this session.
	CSRFToken string
}

type AuthFlow struct {
	StateHash    string
	Nonce        string
	PKCEVerifier string
	CreatedAt    time.Time
	ExpiresAt    time.Time
}
