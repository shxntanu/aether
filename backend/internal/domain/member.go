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
	ID          MemberID
	Email       string
	DisplayName string
	Role        MemberRole
	Status      MemberStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
