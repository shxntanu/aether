package domain

import (
	"errors"
	"fmt"
	"time"
)

// DocumentID uniquely identifies a catalog document.
type DocumentID string

// MemberID uniquely identifies an allowlisted vault member.
type MemberID string

// DocumentStatus describes the storage lifecycle of a document.
type DocumentStatus string

const (
	// DocumentStatusUploading indicates that catalog creation preceded object storage.
	DocumentStatusUploading DocumentStatus = "uploading"
	// DocumentStatusReady indicates that the immutable original is retrievable.
	DocumentStatusReady DocumentStatus = "ready"
	// DocumentStatusFailed indicates that document ingestion did not complete.
	DocumentStatusFailed DocumentStatus = "failed"
	// DocumentStatusDeleted indicates a soft-deleted document awaiting purge.
	DocumentStatusDeleted DocumentStatus = "deleted"
)

// IndexStatus describes the document content-processing lifecycle.
type IndexStatus string

const (
	// IndexStatusNotScheduled indicates that content processing has not been requested.
	IndexStatusNotScheduled IndexStatus = "not_scheduled"
	// IndexStatusQueued indicates that content processing is waiting for a worker.
	IndexStatusQueued IndexStatus = "queued"
	// IndexStatusExtracting indicates that source text is being extracted.
	IndexStatusExtracting IndexStatus = "extracting"
	// IndexStatusEnriching indicates that extracted content is being enriched.
	IndexStatusEnriching IndexStatus = "enriching"
	// IndexStatusIndexed indicates that searchable content is current.
	IndexStatusIndexed IndexStatus = "indexed"
	// IndexStatusFailed indicates that content processing failed visibly.
	IndexStatusFailed IndexStatus = "failed"
)

// ErrInvalidDocumentTransition indicates a disallowed lifecycle change.
var ErrInvalidDocumentTransition = errors.New("invalid document transition")

// Document is provider-neutral catalog metadata for an immutable original.
type Document struct {
	// ID uniquely identifies the catalog record.
	ID DocumentID `json:"id"`
	// Title is the user-editable display title.
	Title string `json:"title"`
	// OriginalFilename retains the upload's sanitized base filename.
	OriginalFilename string `json:"originalFilename"`
	// MediaType is detected from the original's signature.
	MediaType string `json:"mediaType"`
	// SizeBytes is the exact original byte count.
	SizeBytes int64 `json:"sizeBytes"`
	// SHA256 is the lowercase hexadecimal digest of the original bytes.
	SHA256 string `json:"sha256"`
	// StorageKey is an internal provider-neutral object key.
	StorageKey string `json:"-"`
	// Status is the document storage lifecycle state.
	Status DocumentStatus `json:"status"`
	// IndexStatus is the content-processing lifecycle state.
	IndexStatus IndexStatus `json:"indexStatus"`
	// UploaderID identifies the member who created the document.
	UploaderID MemberID `json:"uploaderId"`
	// Version supports optimistic metadata concurrency.
	Version int64 `json:"version"`
	// CreatedAt is the catalog creation time.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt is the latest catalog metadata change time.
	UpdatedAt time.Time `json:"updatedAt"`
	// DeletedAt records soft deletion when applicable.
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
	// PurgeAfter is the earliest permanent-deletion time when applicable.
	PurgeAfter *time.Time `json:"purgeAfter,omitempty"`
	// ManifestError exposes a failed manifest synchronization for repair.
	ManifestError string `json:"manifestError,omitempty"`
}

// TagMatch controls how document tag filters are combined.
type TagMatch string

const (
	// TagMatchAll requires every requested tag to be attached.
	TagMatchAll TagMatch = "all"
	// TagMatchAny requires at least one requested tag to be attached.
	TagMatchAny TagMatch = "any"
)

// DocumentListOptions controls catalog document listing.
type DocumentListOptions struct {
	// NormalizedTags contains case-normalized exact tag values.
	NormalizedTags []string
	// TagMatch selects all-tag or any-tag matching.
	TagMatch TagMatch
	// Statuses limits the list to the supplied lifecycle states.
	//
	// A nil slice preserves the legacy ready-only default. An empty non-nil
	// slice matches no documents.
	Statuses []DocumentStatus
	// PurgeDueBefore limits results to deleted documents whose purge time is
	// due at or before the supplied instant.
	//
	// Callers must also include DocumentStatusDeleted in Statuses.
	PurgeDueBefore *time.Time
	// Limit bounds the number of returned documents.
	//
	// Zero preserves the legacy unbounded behavior.
	Limit int
}

// TransitionTo applies an allowed lifecycle transition at now.
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

// SoftDelete marks the document deleted with a positive retention period.
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
