package domain

import (
	"errors"
	"fmt"
	"time"
)

type DocumentID string
type MemberID string

type DocumentStatus string

const (
	DocumentStatusUploading DocumentStatus = "uploading"
	DocumentStatusReady     DocumentStatus = "ready"
	DocumentStatusFailed    DocumentStatus = "failed"
	DocumentStatusDeleted   DocumentStatus = "deleted"
)

type IndexStatus string

const (
	IndexStatusNotScheduled IndexStatus = "not_scheduled"
	IndexStatusQueued       IndexStatus = "queued"
	IndexStatusExtracting   IndexStatus = "extracting"
	IndexStatusEnriching    IndexStatus = "enriching"
	IndexStatusIndexed      IndexStatus = "indexed"
	IndexStatusFailed       IndexStatus = "failed"
)

var ErrInvalidDocumentTransition = errors.New("invalid document transition")

type Document struct {
	ID               DocumentID
	Title            string
	OriginalFilename string
	MediaType        string
	SizeBytes        int64
	SHA256           string
	StorageKey       string
	Status           DocumentStatus
	IndexStatus      IndexStatus
	UploaderID       MemberID
	Version          int64
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
	PurgeAfter       *time.Time
}

func (d *Document) TransitionTo(next DocumentStatus, now time.Time) error {
	allowed := map[DocumentStatus]map[DocumentStatus]bool{
		DocumentStatusUploading: {
			DocumentStatusReady:   true,
			DocumentStatusFailed:  true,
			DocumentStatusDeleted: true,
		},
		DocumentStatusReady: {
			DocumentStatusDeleted: true,
		},
		DocumentStatusFailed: {
			DocumentStatusUploading: true,
			DocumentStatusDeleted:   true,
		},
		DocumentStatusDeleted: {
			DocumentStatusReady: true,
		},
	}

	if !allowed[d.Status][next] {
		return fmt.Errorf("%w: %s to %s", ErrInvalidDocumentTransition, d.Status, next)
	}

	d.Status = next
	d.UpdatedAt = now
	if next == DocumentStatusReady {
		d.DeletedAt = nil
		d.PurgeAfter = nil
	}
	return nil
}

func (d *Document) SoftDelete(now time.Time, retention time.Duration) error {
	if retention <= 0 {
		return fmt.Errorf("retention must be positive")
	}
	if err := d.TransitionTo(DocumentStatusDeleted, now); err != nil {
		return err
	}

	purgeAfter := now.Add(retention)
	d.DeletedAt = &now
	d.PurgeAfter = &purgeAfter
	return nil
}
