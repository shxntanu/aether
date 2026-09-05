package vault

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shxntanu/aether/backend/internal/domain"
	"github.com/shxntanu/aether/backend/internal/storage"
)

const (
	// DeletionRetention is the mandatory recovery window for soft-deleted
	// documents before permanent purge is allowed.
	DeletionRetention  = 30 * 24 * time.Hour
	maxPurgeBatch      = 100
	deletionRetryDelay = 30 * time.Second
)

var (
	// ErrRetentionActive indicates that a document cannot yet be permanently
	// removed because its recovery window has not expired.
	ErrRetentionActive = errors.New("document retention is still active")
	// ErrDeletionInProgress indicates that storage trashing must finish before
	// the document can be restored or permanently purged.
	ErrDeletionInProgress = errors.New("document deletion is still in progress")
)

// DeletionResult reports one bounded asynchronous storage-trash pass.
type DeletionResult struct {
	// Completed contains IDs whose original and manifest are now trashed.
	Completed []domain.DocumentID
	// Failed maps each attempted document ID to its retryable storage error.
	Failed map[domain.DocumentID]error
}

// PurgePolicy selects whether Purge must enforce the document's retention
// timestamp or may rely on a prior due-date selection.
type PurgePolicy uint8

const (
	// EnforceRetention requires PurgeAfter to be present and no later than now.
	EnforceRetention PurgePolicy = iota
	// RetentionAlreadyChecked permits a caller to purge a row selected by a
	// trusted due-date query.
	RetentionAlreadyChecked
)

// PurgeResult reports successful and failed documents from a bounded purge
// attempt. Failed values contain only document-scoped errors.
type PurgeResult struct {
	// Purged contains IDs whose objects and catalog rows were removed.
	Purged []domain.DocumentID
	// Failed maps each attempted document ID to its purge error.
	Failed map[domain.DocumentID]error
}

// Delete immediately soft-deletes a ready document and queues its storage
// objects for asynchronous trashing. Repeated deletion returns the current row.
func (s *Service) Delete(ctx context.Context, id domain.DocumentID) (DocumentRecord, error) {
	document, err := s.repository.GetDocument(ctx, id)
	if err != nil {
		return DocumentRecord{}, err
	}
	if document.Status == domain.DocumentStatusDeleted {
		return s.deletedRecord(ctx, document)
	}
	if document.Status != domain.DocumentStatusReady {
		return DocumentRecord{}, fmt.Errorf("delete document %q: %w", id, domain.ErrNotFound)
	}

	now := s.now().UTC()
	if err := document.SoftDelete(now, DeletionRetention); err != nil {
		return DocumentRecord{}, fmt.Errorf("delete document %q: %w", id, err)
	}
	updated, err := s.repository.UpdateDocument(ctx, document, document.Version)
	if err != nil {
		return DocumentRecord{}, fmt.Errorf("delete document %q: %w", id, err)
	}
	return s.deletedRecord(ctx, updated)
}

// Restore returns a deleted document to ready state, preserving its metadata
// and compensating restored objects if the catalog transition fails.
func (s *Service) Restore(ctx context.Context, id domain.DocumentID) (DocumentRecord, error) {
	document, err := s.repository.GetDocument(ctx, id)
	if err != nil {
		return DocumentRecord{}, err
	}
	if document.Status != domain.DocumentStatusDeleted {
		return DocumentRecord{}, fmt.Errorf("restore document %q: %w", id, domain.ErrConflict)
	}
	if document.DeletionStatus != domain.DeletionStatusComplete {
		return DocumentRecord{}, fmt.Errorf("restore document %q: %w", id, ErrDeletionInProgress)
	}

	restored, err := s.restoreDocumentObjects(ctx, document.StorageKey)
	if err != nil {
		return DocumentRecord{}, fmt.Errorf("restore document %q: %w", id, err)
	}

	now := s.now().UTC()
	if err := document.TransitionTo(domain.DocumentStatusReady, now); err != nil {
		return DocumentRecord{}, s.compensateRestore(ctx, id, restored, err)
	}
	updated, err := s.repository.UpdateDocument(ctx, document, document.Version)
	if err != nil {
		return DocumentRecord{}, s.compensateRestore(ctx, id, restored, err)
	}

	tags, err := s.repository.ListDocumentTags(ctx, id)
	if err != nil {
		return DocumentRecord{}, err
	}
	updated = s.writeManifest(ctx, updated, tags)
	return DocumentRecord{Document: updated, Tags: tags}, nil
}

// Purge permanently removes a deleted document's objects and then its
// catalog row. Missing objects are already in the desired terminal state.
func (s *Service) Purge(
	ctx context.Context,
	id domain.DocumentID,
	policy PurgePolicy,
) error {
	document, err := s.repository.GetDocument(ctx, id)
	if err != nil {
		return err
	}
	if document.Status != domain.DocumentStatusDeleted {
		return fmt.Errorf("purge document %q: %w", id, domain.ErrConflict)
	}
	if document.DeletionStatus != domain.DeletionStatusComplete {
		return fmt.Errorf("purge document %q: %w", id, ErrDeletionInProgress)
	}
	if policy != EnforceRetention && policy != RetentionAlreadyChecked {
		return fmt.Errorf("purge document %q: invalid purge policy", id)
	}
	if policy == EnforceRetention {
		now := s.now().UTC()
		if document.PurgeAfter == nil || document.PurgeAfter.After(now) {
			return fmt.Errorf("purge document %q: %w", id, ErrRetentionActive)
		}
	}

	if err := deleteDocumentObjects(ctx, s.objects, document.StorageKey); err != nil {
		return fmt.Errorf("purge document %q: %w", id, err)
	}
	if err := s.repository.PurgeDocument(ctx, id); err != nil {
		return fmt.Errorf("purge document %q: %w", id, err)
	}
	return nil
}

// ProcessDeletions trashes storage objects for at most batchSize queued or
// retryable deleted documents and persists visible progress for each attempt.
func (s *Service) ProcessDeletions(ctx context.Context, batchSize int) (DeletionResult, error) {
	if batchSize < 1 {
		batchSize = 1
	}
	if batchSize > maxPurgeBatch {
		batchSize = maxPurgeBatch
	}
	documents, err := s.repository.ListDocuments(ctx, domain.DocumentListOptions{
		Statuses: []domain.DocumentStatus{domain.DocumentStatusDeleted},
	})
	if err != nil {
		return DeletionResult{}, fmt.Errorf("list pending deletions: %w", err)
	}

	result := DeletionResult{
		Completed: make([]domain.DocumentID, 0, batchSize),
		Failed:    make(map[domain.DocumentID]error),
	}
	now := s.now().UTC()
	for _, document := range documents {
		if len(result.Completed)+len(result.Failed) >= batchSize {
			break
		}
		if document.DeletionStatus == domain.DeletionStatusComplete {
			continue
		}
		if document.DeletionStatus != domain.DeletionStatusQueued &&
			now.Sub(document.UpdatedAt) < deletionRetryDelay {
			continue
		}
		if err := s.processDeletion(ctx, document); err != nil {
			result.Failed[document.ID] = err
			continue
		}
		result.Completed = append(result.Completed, document.ID)
	}
	return result, nil
}

func (s *Service) processDeletion(ctx context.Context, document domain.Document) error {
	document.DeletionStatus = domain.DeletionStatusProcessing
	document.DeletionError = ""
	document.UpdatedAt = s.now().UTC()
	processing, err := s.repository.UpdateDocument(ctx, document, document.Version)
	if err != nil {
		return fmt.Errorf("claim deletion %q: %w", document.ID, err)
	}

	_, trashErr := s.trashDocumentObjects(ctx, processing.StorageKey)
	processing.UpdatedAt = s.now().UTC()
	if trashErr == nil {
		processing.DeletionStatus = domain.DeletionStatusComplete
		processing.DeletionError = ""
	} else {
		processing.DeletionStatus = domain.DeletionStatusFailed
		processing.DeletionError = trashErr.Error()
	}
	if _, err := s.repository.UpdateDocument(ctx, processing, processing.Version); err != nil {
		return errors.Join(trashErr, fmt.Errorf("persist deletion progress: %w", err))
	}
	if trashErr != nil {
		return fmt.Errorf("trash document %q: %w", document.ID, trashErr)
	}
	return nil
}

func (s *Service) deletedRecord(
	ctx context.Context,
	document domain.Document,
) (DocumentRecord, error) {
	tags, err := s.repository.ListDocumentTags(ctx, document.ID)
	if err != nil {
		return DocumentRecord{}, err
	}
	return DocumentRecord{Document: document, Tags: tags}, nil
}

// PurgeDue permanently purges at most 100 due deleted documents. It attempts
// every selected document and returns independent errors for failed attempts.
func (s *Service) PurgeDue(ctx context.Context, batchSize int) (PurgeResult, error) {
	if batchSize < 1 {
		batchSize = 1
	}
	if batchSize > maxPurgeBatch {
		batchSize = maxPurgeBatch
	}
	now := s.now().UTC()
	documents, err := s.repository.ListDocuments(ctx, domain.DocumentListOptions{
		Statuses:       []domain.DocumentStatus{domain.DocumentStatusDeleted},
		PurgeDueBefore: &now,
		Limit:          batchSize,
	})
	if err != nil {
		return PurgeResult{}, fmt.Errorf("list due documents: %w", err)
	}

	result := PurgeResult{
		Purged: make([]domain.DocumentID, 0, len(documents)),
		Failed: make(map[domain.DocumentID]error),
	}
	for _, document := range documents {
		if err := s.Purge(ctx, document.ID, RetentionAlreadyChecked); err != nil {
			result.Failed[document.ID] = fmt.Errorf("purge document %q: %w", document.ID, err)
			continue
		}
		result.Purged = append(result.Purged, document.ID)
	}
	return result, nil
}

func (s *Service) trashDocumentObjects(ctx context.Context, storageKey string) ([]string, error) {
	trashed := make([]string, 0, 2)
	if err := s.objects.Trash(ctx, storageKey); err != nil {
		return nil, fmt.Errorf("trash original object: %w", err)
	}
	trashed = append(trashed, storageKey)

	manifestKey := manifestKey(storageKey)
	if err := s.objects.Trash(ctx, manifestKey); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return trashed, nil
		}
		return trashed, errors.Join(
			fmt.Errorf("trash manifest object: %w", err),
			s.restoreObjectKeys(ctx, trashed),
		)
	}
	return append(trashed, manifestKey), nil
}

func (s *Service) restoreDocumentObjects(ctx context.Context, storageKey string) ([]string, error) {
	restored := make([]string, 0, 2)
	if err := s.objects.Restore(ctx, storageKey); err != nil {
		return nil, fmt.Errorf("restore original object: %w", err)
	}
	restored = append(restored, storageKey)

	manifestKey := manifestKey(storageKey)
	if err := s.objects.Restore(ctx, manifestKey); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return restored, nil
		}
		return restored, errors.Join(
			fmt.Errorf("restore manifest object: %w", err),
			s.trashObjectKeys(ctx, restored),
		)
	}
	return append(restored, manifestKey), nil
}

func (s *Service) compensateTrash(
	ctx context.Context,
	id domain.DocumentID,
	trashed []string,
	primary error,
) error {
	return fmt.Errorf(
		"delete document %q: %w",
		id,
		errors.Join(primary, s.restoreObjectKeys(ctx, trashed)),
	)
}

func (s *Service) compensateRestore(
	ctx context.Context,
	id domain.DocumentID,
	restored []string,
	primary error,
) error {
	return fmt.Errorf(
		"restore document %q: %w",
		id,
		errors.Join(primary, s.trashObjectKeys(ctx, restored)),
	)
}

func (s *Service) restoreObjectKeys(ctx context.Context, keys []string) error {
	var compensationErr error
	for index := len(keys) - 1; index >= 0; index-- {
		if err := s.objects.Restore(ctx, keys[index]); err != nil &&
			!errors.Is(err, storage.ErrNotFound) {
			compensationErr = errors.Join(
				compensationErr,
				fmt.Errorf("restore trashed object: %w", err),
			)
		}
	}
	return compensationErr
}

func (s *Service) trashObjectKeys(ctx context.Context, keys []string) error {
	var compensationErr error
	for index := len(keys) - 1; index >= 0; index-- {
		if err := s.objects.Trash(ctx, keys[index]); err != nil &&
			!errors.Is(err, storage.ErrNotFound) {
			compensationErr = errors.Join(
				compensationErr,
				fmt.Errorf("trash restored object: %w", err),
			)
		}
	}
	return compensationErr
}

func deleteDocumentObjects(
	ctx context.Context,
	objects storage.ObjectStore,
	storageKey string,
) error {
	for _, key := range []string{storageKey, manifestKey(storageKey)} {
		if err := objects.Delete(ctx, key); err != nil && !errors.Is(err, storage.ErrNotFound) {
			return fmt.Errorf("delete object: %w", err)
		}
	}
	return nil
}

func manifestKey(storageKey string) string {
	return storageKey + ".manifest.json"
}
