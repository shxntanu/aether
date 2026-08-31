package local

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/shxntanu/aether/backend/internal/storage"
)

// Store persists objects beneath a fixed filesystem root.
type Store struct {
	root string
}

// New creates a local store rooted at root and creates the root when needed.
func New(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("local storage root must not be empty")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve local storage root: %w", err)
	}
	if err := os.MkdirAll(absRoot, 0o700); err != nil {
		return nil, fmt.Errorf("create local storage root: %w", err)
	}
	return &Store{root: absRoot}, nil
}

// Put streams an object to a temporary file and atomically publishes it.
func (s *Store) Put(
	ctx context.Context,
	key string,
	body io.Reader,
	options storage.PutOptions,
) (storage.ObjectInfo, error) {
	path, err := s.path(key, false)
	if err != nil {
		return storage.ObjectInfo{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return storage.ObjectInfo{}, fmt.Errorf("create object directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".aether-upload-*")
	if err != nil {
		return storage.ObjectInfo{}, fmt.Errorf("create temporary object: %w", err)
	}
	temporaryName := temporary.Name()
	defer func() { _ = os.Remove(temporaryName) }()

	written, copyErr := copyContext(ctx, temporary, body)
	closeErr := temporary.Close()
	if copyErr != nil {
		return storage.ObjectInfo{}, copyErr
	}
	if closeErr != nil {
		return storage.ObjectInfo{}, fmt.Errorf("close temporary object: %w", closeErr)
	}
	if err := os.Chmod(temporaryName, 0o600); err != nil {
		return storage.ObjectInfo{}, fmt.Errorf("secure temporary object: %w", err)
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return storage.ObjectInfo{}, fmt.Errorf("publish object: %w", err)
	}
	storedInfo, err := os.Stat(path)
	if err != nil {
		return storage.ObjectInfo{}, fmt.Errorf("stat published object: %w", err)
	}
	objectInfo := storage.ObjectInfo{
		Key:          key,
		Size:         written,
		ContentType:  options.ContentType,
		LastModified: storedInfo.ModTime(),
	}
	if err := s.writeMetadata(key, false, objectInfo); err != nil {
		_ = os.Remove(path)
		return storage.ObjectInfo{}, err
	}
	return objectInfo, nil
}

// OpenRange opens an object and restricts reads to an inclusive byte range.
func (s *Store) OpenRange(
	_ context.Context,
	key string,
	byteRange *storage.ByteRange,
) (io.ReadCloser, storage.ObjectInfo, error) {
	path, err := s.path(key, false)
	if err != nil {
		return nil, storage.ObjectInfo{}, err
	}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, storage.ObjectInfo{}, storage.ErrNotFound
	}
	if err != nil {
		return nil, storage.ObjectInfo{}, fmt.Errorf("open object: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, storage.ObjectInfo{}, fmt.Errorf("stat object: %w", err)
	}
	objectInfo := storage.ObjectInfo{Key: key, Size: info.Size(), LastModified: info.ModTime()}
	objectInfo.ContentType, err = s.readContentType(key, false)
	if err != nil {
		_ = file.Close()
		return nil, storage.ObjectInfo{}, err
	}
	if byteRange == nil {
		return file, objectInfo, nil
	}
	if byteRange.Start < 0 || byteRange.End < byteRange.Start || byteRange.End >= info.Size() {
		_ = file.Close()
		return nil, storage.ObjectInfo{}, fmt.Errorf("invalid object byte range")
	}
	if _, err := file.Seek(byteRange.Start, io.SeekStart); err != nil {
		_ = file.Close()
		return nil, storage.ObjectInfo{}, fmt.Errorf("seek object: %w", err)
	}
	length := byteRange.End - byteRange.Start + 1
	return &limitedReadCloser{Reader: io.LimitReader(file, length), closer: file}, objectInfo, nil
}

// Stat returns metadata for an available object.
func (s *Store) Stat(_ context.Context, key string) (storage.ObjectInfo, error) {
	path, err := s.path(key, false)
	if err != nil {
		return storage.ObjectInfo{}, err
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return storage.ObjectInfo{}, storage.ErrNotFound
	}
	if err != nil {
		return storage.ObjectInfo{}, fmt.Errorf("stat object: %w", err)
	}
	contentType, err := s.readContentType(key, false)
	if err != nil {
		return storage.ObjectInfo{}, err
	}
	return storage.ObjectInfo{
		Key:          key,
		Size:         info.Size(),
		ContentType:  contentType,
		LastModified: info.ModTime(),
	}, nil
}

// Trash moves an available object into the store's private trash tree.
func (s *Store) Trash(_ context.Context, key string) error {
	return s.move(key, false, true)
}

// Restore moves a trashed object back to its original key.
func (s *Store) Restore(_ context.Context, key string) error {
	return s.move(key, true, false)
}

// Delete permanently removes available and trashed copies of an object.
func (s *Store) Delete(_ context.Context, key string) error {
	path, err := s.path(key, false)
	if err != nil {
		return err
	}
	trashPath, err := s.path(key, true)
	if err != nil {
		return err
	}
	for _, target := range []string{path, trashPath} {
		if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("delete object: %w", err)
		}
	}
	for _, trashed := range []bool{false, true} {
		metadataPath, pathErr := s.metadataPath(key, trashed)
		if pathErr != nil {
			return pathErr
		}
		if err := os.Remove(metadataPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("delete object metadata: %w", err)
		}
	}
	return nil
}

func (s *Store) move(key string, fromTrash, toTrash bool) error {
	from, err := s.path(key, fromTrash)
	if err != nil {
		return err
	}
	to, err := s.path(key, toTrash)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o700); err != nil {
		return fmt.Errorf("create object directory: %w", err)
	}
	if err := os.Rename(from, to); errors.Is(err, os.ErrNotExist) {
		return storage.ErrNotFound
	} else if err != nil {
		return fmt.Errorf("move object: %w", err)
	}
	fromMetadata, err := s.metadataPath(key, fromTrash)
	if err != nil {
		return err
	}
	toMetadata, err := s.metadataPath(key, toTrash)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(toMetadata), 0o700); err != nil {
		return fmt.Errorf("create metadata directory: %w", err)
	}
	if err := os.Rename(fromMetadata, toMetadata); err != nil {
		if rollbackErr := os.Rename(to, from); rollbackErr != nil {
			return errors.Join(
				fmt.Errorf("move object metadata: %w", err),
				fmt.Errorf("roll back object move: %w", rollbackErr),
			)
		}
		return fmt.Errorf("move object metadata: %w", err)
	}
	return nil
}

func (s *Store) writeMetadata(key string, trash bool, info storage.ObjectInfo) error {
	path, err := s.metadataPath(key, trash)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create metadata directory: %w", err)
	}
	body, err := json.Marshal(struct {
		ContentType string `json:"contentType"`
	}{ContentType: info.ContentType})
	if err != nil {
		return fmt.Errorf("encode object metadata: %w", err)
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return fmt.Errorf("write object metadata: %w", err)
	}
	return nil
}

func (s *Store) readContentType(key string, trash bool) (string, error) {
	path, err := s.metadataPath(key, trash)
	if err != nil {
		return "", err
	}
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read object metadata: %w", err)
	}
	var metadata struct {
		ContentType string `json:"contentType"`
	}
	if err := json.Unmarshal(body, &metadata); err != nil {
		return "", fmt.Errorf("decode object metadata: %w", err)
	}
	return metadata.ContentType, nil
}

func (s *Store) metadataPath(key string, trash bool) (string, error) {
	return s.path(filepath.ToSlash(filepath.Join(".metadata", key+".json")), trash)
}

func (s *Store) path(key string, trash bool) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash(key))
	if key == "" || filepath.IsAbs(cleaned) || cleaned == "." || cleaned == ".." ||
		strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid object key")
	}
	root := s.root
	if trash {
		root = filepath.Join(root, ".trash")
	}
	path := filepath.Join(root, cleaned)
	if path != root && !strings.HasPrefix(path, root+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid object key")
	}
	return path, nil
}

func copyContext(ctx context.Context, destination io.Writer, source io.Reader) (int64, error) {
	buffer := make([]byte, 32*1024)
	var written int64
	for {
		if err := ctx.Err(); err != nil {
			return written, fmt.Errorf("write object: %w", err)
		}
		count, readErr := source.Read(buffer)
		if count > 0 {
			output, writeErr := destination.Write(buffer[:count])
			written += int64(output)
			if writeErr != nil {
				return written, fmt.Errorf("write object: %w", writeErr)
			}
			if output != count {
				return written, io.ErrShortWrite
			}
		}
		if errors.Is(readErr, io.EOF) {
			return written, nil
		}
		if readErr != nil {
			return written, fmt.Errorf("read object: %w", readErr)
		}
	}
}

type limitedReadCloser struct {
	io.Reader
	closer io.Closer
}

func (r *limitedReadCloser) Close() error {
	return r.closer.Close()
}
