// Package storage defines provider-neutral object persistence used by the
// document vault. Provider packages implement this contract without exposing
// provider credentials or URLs to callers.
package storage

import (
	"context"
	"errors"
	"io"
	"time"
)

// ErrNotFound indicates that an object key does not exist.
var ErrNotFound = errors.New("object not found")

// ByteRange is an inclusive range within an object.
type ByteRange struct {
	// Start is the zero-based first included byte.
	Start int64
	// End is the zero-based last included byte.
	End int64
}

// PutOptions describes metadata stored with an object.
type PutOptions struct {
	// ContentType is the trusted application-detected media type.
	ContentType string
}

// ObjectInfo describes a stored object's provider-neutral metadata.
type ObjectInfo struct {
	// Key is the opaque provider-neutral object key.
	Key string
	// Size is the stored object size in bytes.
	Size int64
	// ContentType is the application-supplied media type when known.
	ContentType string
	// LastModified is the provider's object modification time.
	LastModified time.Time
}

// ObjectLinks contains provider-hosted destinations for viewing or downloading
// an object. Providers may omit these links when they cannot serve them.
type ObjectLinks struct {
	// ViewURL opens the object in the provider's viewer.
	ViewURL string
	// DownloadURL downloads the object's original bytes from the provider.
	DownloadURL string
}

// Usage describes provider-reported account storage consumption. LimitBytes
// and RemainingBytes are nil when the provider grants unlimited storage.
type Usage struct {
	// UsedBytes is the provider account's current storage consumption.
	UsedBytes int64 `json:"usedBytes"`
	// LimitBytes is the provider account's storage limit when one applies.
	LimitBytes *int64 `json:"limitBytes"`
	// RemainingBytes is the non-negative capacity available under LimitBytes.
	RemainingBytes *int64 `json:"remainingBytes"`
}

// UsageReader is an optional object-store capability for account capacity.
type UsageReader interface {
	// Usage returns current provider-reported account storage consumption.
	Usage(context.Context) (Usage, error)
}

// ObjectLinker is an optional object-store capability for provider-hosted links.
type ObjectLinker interface {
	// Links returns authenticated-provider links for an object key.
	Links(context.Context, string) (ObjectLinks, error)
}

// ObjectStore persists immutable document objects behind opaque keys.
type ObjectStore interface {
	// Put streams an object to key, replacing an existing object atomically.
	Put(context.Context, string, io.Reader, PutOptions) (ObjectInfo, error)
	// OpenRange opens all bytes or the requested inclusive byte range.
	OpenRange(context.Context, string, *ByteRange) (io.ReadCloser, ObjectInfo, error)
	// Stat returns metadata without opening the object body.
	Stat(context.Context, string) (ObjectInfo, error)
	// Trash makes an object unavailable while preserving it for restoration.
	Trash(context.Context, string) error
	// Restore makes a previously trashed object available again.
	Restore(context.Context, string) error
	// Delete permanently removes an object and any trashed copy.
	Delete(context.Context, string) error
}
