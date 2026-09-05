package postgres

import (
	"time"

	"github.com/shxntanu/aether/backend/internal/domain"
)

type documentModel struct {
	ID               domain.DocumentID     `gorm:"column:id;primaryKey"`
	Title            string                `gorm:"column:title"`
	OriginalFilename string                `gorm:"column:original_filename"`
	MediaType        string                `gorm:"column:media_type"`
	SizeBytes        int64                 `gorm:"column:size_bytes"`
	SHA256           string                `gorm:"column:sha256"`
	StorageKey       string                `gorm:"column:storage_key"`
	Status           domain.DocumentStatus `gorm:"column:status"`
	IndexStatus      domain.IndexStatus    `gorm:"column:index_status"`
	UploaderID       domain.MemberID       `gorm:"column:uploader_id"`
	Version          int64                 `gorm:"column:version"`
	CreatedAt        time.Time             `gorm:"column:created_at"`
	UpdatedAt        time.Time             `gorm:"column:updated_at"`
	DeletedAt        *time.Time            `gorm:"column:deleted_at"`
	PurgeAfter       *time.Time            `gorm:"column:purge_after"`
	DeletionStatus   domain.DeletionStatus `gorm:"column:deletion_status"`
	DeletionError    string                `gorm:"column:deletion_error"`
	ManifestError    string                `gorm:"column:manifest_error"`
}

func (documentModel) TableName() string { return "documents_tbl" }

type tagModel struct {
	ID             domain.TagID `gorm:"column:id;primaryKey"`
	DisplayName    string       `gorm:"column:display_name"`
	NormalizedName string       `gorm:"column:normalized_name"`
}

func (tagModel) TableName() string { return "tags_tbl" }

type documentTagModel struct {
	DocumentID domain.DocumentID `gorm:"column:document_id;primaryKey"`
	TagID      domain.TagID      `gorm:"column:tag_id;primaryKey"`
}

func (documentTagModel) TableName() string { return "document_tags_tbl" }

type uploadRequestModel struct {
	MemberID   domain.MemberID   `gorm:"column:member_id;primaryKey"`
	KeyHash    string            `gorm:"column:key_hash;primaryKey"`
	DocumentID domain.DocumentID `gorm:"column:document_id"`
	CreatedAt  time.Time         `gorm:"column:created_at"`
}

func (uploadRequestModel) TableName() string { return "upload_requests_tbl" }

type memberModel struct {
	ID          domain.MemberID     `gorm:"column:id;primaryKey"`
	Email       string              `gorm:"column:email"`
	DisplayName string              `gorm:"column:display_name"`
	OIDCSubject *string             `gorm:"column:oidc_subject"`
	Role        domain.MemberRole   `gorm:"column:role"`
	Status      domain.MemberStatus `gorm:"column:status"`
	CreatedAt   time.Time           `gorm:"column:created_at"`
	UpdatedAt   time.Time           `gorm:"column:updated_at"`
}

func (memberModel) TableName() string { return "members_tbl" }

type sessionModel struct {
	ID        string          `gorm:"column:id;primaryKey"`
	TokenHash string          `gorm:"column:token_hash"`
	CSRFHash  string          `gorm:"column:csrf_hash"`
	MemberID  domain.MemberID `gorm:"column:member_id"`
	CreatedAt time.Time       `gorm:"column:created_at"`
	ExpiresAt time.Time       `gorm:"column:expires_at"`
}

func (sessionModel) TableName() string { return "sessions_tbl" }

type authFlowModel struct {
	StateHash    string    `gorm:"column:state_hash;primaryKey"`
	Nonce        string    `gorm:"column:nonce"`
	PKCEVerifier string    `gorm:"column:pkce_verifier"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	ExpiresAt    time.Time `gorm:"column:expires_at"`
}

func (authFlowModel) TableName() string { return "auth_flows_tbl" }

type auditEventModel struct {
	ID         string              `gorm:"column:id;primaryKey"`
	ActorID    *string             `gorm:"column:actor_id"`
	Action     string              `gorm:"column:action"`
	ObjectType string              `gorm:"column:object_type"`
	ObjectID   string              `gorm:"column:object_id"`
	Outcome    domain.AuditOutcome `gorm:"column:outcome"`
	OccurredAt time.Time           `gorm:"column:occurred_at"`
}

func (auditEventModel) TableName() string { return "audit_events_tbl" }

func documentModelFromDomain(document domain.Document) documentModel {
	return documentModel{
		ID:               document.ID,
		Title:            document.Title,
		OriginalFilename: document.OriginalFilename,
		MediaType:        document.MediaType,
		SizeBytes:        document.SizeBytes,
		SHA256:           document.SHA256,
		StorageKey:       document.StorageKey,
		Status:           document.Status,
		IndexStatus:      document.IndexStatus,
		UploaderID:       document.UploaderID,
		Version:          document.Version,
		CreatedAt:        document.CreatedAt,
		UpdatedAt:        document.UpdatedAt,
		DeletedAt:        document.DeletedAt,
		PurgeAfter:       document.PurgeAfter,
		DeletionStatus:   document.DeletionStatus,
		DeletionError:    document.DeletionError,
		ManifestError:    document.ManifestError,
	}
}

func (document documentModel) domain() domain.Document {
	return domain.Document{
		ID:               document.ID,
		Title:            document.Title,
		OriginalFilename: document.OriginalFilename,
		MediaType:        document.MediaType,
		SizeBytes:        document.SizeBytes,
		SHA256:           document.SHA256,
		StorageKey:       document.StorageKey,
		Status:           document.Status,
		IndexStatus:      document.IndexStatus,
		UploaderID:       document.UploaderID,
		Version:          document.Version,
		CreatedAt:        document.CreatedAt,
		UpdatedAt:        document.UpdatedAt,
		DeletedAt:        document.DeletedAt,
		PurgeAfter:       document.PurgeAfter,
		DeletionStatus:   document.DeletionStatus,
		DeletionError:    document.DeletionError,
		ManifestError:    document.ManifestError,
	}
}

func memberModelFromDomain(member domain.Member) memberModel {
	var subject *string
	if member.OIDCSubject != "" {
		subject = &member.OIDCSubject
	}
	return memberModel{
		ID:          member.ID,
		Email:       member.Email,
		DisplayName: member.DisplayName,
		OIDCSubject: subject,
		Role:        member.Role,
		Status:      member.Status,
		CreatedAt:   member.CreatedAt,
		UpdatedAt:   member.UpdatedAt,
	}
}

func (member memberModel) domain() domain.Member {
	var subject string
	if member.OIDCSubject != nil {
		subject = *member.OIDCSubject
	}
	return domain.Member{
		ID:          member.ID,
		Email:       member.Email,
		DisplayName: member.DisplayName,
		OIDCSubject: subject,
		Role:        member.Role,
		Status:      member.Status,
		CreatedAt:   member.CreatedAt,
		UpdatedAt:   member.UpdatedAt,
	}
}
