package gdrive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/shxntanu/aether/backend/internal/storage"
)

const (
	defaultAPIBaseURL    = "https://www.googleapis.com/drive/v3"
	defaultUploadBaseURL = "https://www.googleapis.com/upload/drive/v3"
	defaultChunkSize     = int64(8 * 1024 * 1024)
)

// Config configures the vault-owner Google Drive client.
type Config struct {
	// ClientID identifies the Google OAuth web client.
	ClientID string
	// ClientSecret authenticates the Google OAuth web client.
	ClientSecret string
	// RedirectURL is the exact owner-authorization callback URL.
	RedirectURL string
	// FolderID scopes all vault objects to one app-managed Drive folder.
	FolderID string
	// RefreshToken is the deployment secret for offline Drive access.
	RefreshToken string
	// HTTPClient optionally replaces the default client, mainly for local fakes.
	HTTPClient *http.Client
	// APIBaseURL overrides the Drive metadata API base URL.
	APIBaseURL string
	// UploadBaseURL overrides the Drive resumable-upload base URL.
	UploadBaseURL string
	// TokenURL overrides Google's OAuth token endpoint.
	TokenURL string
	// ChunkSize controls resumable upload chunks and must be 256 KiB aligned.
	ChunkSize int64
}

// Store implements storage.ObjectStore using an app-managed Drive folder.
type Store struct {
	client        *http.Client
	apiBaseURL    string
	uploadBaseURL string
	folderID      string
	chunkSize     int64
	ownerOAuth    oauthConfig
}

type oauthConfig struct {
	clientID     string
	clientSecret string
	redirectURL  string
	tokenURL     string
}

var _ storage.ObjectStore = (*Store)(nil)

// New creates an authenticated Google Drive object store.
func New(ctx context.Context, config Config) (*Store, error) {
	if strings.TrimSpace(config.ClientID) == "" ||
		strings.TrimSpace(config.ClientSecret) == "" ||
		strings.TrimSpace(config.FolderID) == "" ||
		strings.TrimSpace(config.RefreshToken) == "" {
		return nil, fmt.Errorf("Google Drive client credentials, folder, and refresh token are required")
	}
	if strings.TrimSpace(config.RedirectURL) == "" {
		return nil, fmt.Errorf("Google Drive redirect URL is required")
	}
	chunkSize := config.ChunkSize
	if chunkSize == 0 {
		chunkSize = defaultChunkSize
	}
	if chunkSize < 256*1024 || chunkSize%(256*1024) != 0 {
		return nil, fmt.Errorf("Google Drive chunk size must be a positive 256 KiB multiple")
	}
	apiBaseURL := strings.TrimRight(config.APIBaseURL, "/")
	if apiBaseURL == "" {
		apiBaseURL = defaultAPIBaseURL
	}
	uploadBaseURL := strings.TrimRight(config.UploadBaseURL, "/")
	if uploadBaseURL == "" {
		uploadBaseURL = defaultUploadBaseURL
	}
	tokenURL := config.TokenURL
	if tokenURL == "" {
		tokenURL = "https://oauth2.googleapis.com/token"
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	oauthContext := context.WithValue(ctx, oauth2.HTTPClient, httpClient)
	oauth := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL: tokenURL,
		},
		RedirectURL: config.RedirectURL,
		Scopes:      []string{"https://www.googleapis.com/auth/drive.file"},
	}
	tokenSource := oauth.TokenSource(oauthContext, &oauth2.Token{RefreshToken: config.RefreshToken})
	return &Store{
		client:        oauth2.NewClient(oauthContext, tokenSource),
		apiBaseURL:    apiBaseURL,
		uploadBaseURL: uploadBaseURL,
		folderID:      config.FolderID,
		chunkSize:     chunkSize,
		ownerOAuth: oauthConfig{
			clientID:     config.ClientID,
			clientSecret: config.ClientSecret,
			redirectURL:  config.RedirectURL,
			tokenURL:     tokenURL,
		},
	}, nil
}

// AuthorizationURL returns an owner authorization URL with drive.file scope.
func (s *Store) AuthorizationURL(state string) string {
	config := &oauth2.Config{
		ClientID: s.ownerOAuth.clientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:  s.ownerOAuth.authURL(),
			TokenURL: s.ownerOAuth.tokenURL,
		},
		RedirectURL: s.ownerOAuth.redirectURL,
		Scopes:      []string{"https://www.googleapis.com/auth/drive.file"},
	}
	return config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// ExchangeCode exchanges an owner authorization code for OAuth tokens.
func (s *Store) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	config := &oauth2.Config{
		ClientID:     s.ownerOAuth.clientID,
		ClientSecret: s.ownerOAuth.clientSecret,
		Endpoint:     oauth2.Endpoint{TokenURL: s.ownerOAuth.tokenURL},
		RedirectURL:  s.ownerOAuth.redirectURL,
	}
	return config.Exchange(ctx, code)
}

func (o oauthConfig) authURL() string {
	return "https://accounts.google.com/o/oauth2/v2/auth"
}

// Put streams bytes into a Drive resumable-upload session.
func (s *Store) Put(
	ctx context.Context,
	key string,
	body io.Reader,
	options storage.PutOptions,
) (storage.ObjectInfo, error) {
	if err := validateKey(key); err != nil {
		return storage.ObjectInfo{}, err
	}
	metadata := driveFile{
		Name:          key,
		MimeType:      options.ContentType,
		Parents:       []string{s.folderID},
		AppProperties: map[string]string{"aetherStorageKey": key},
	}
	existing, err := s.findFile(ctx, key, false)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return storage.ObjectInfo{}, err
	}
	if existing.ID != "" {
		metadata.ID = existing.ID
	}
	return s.resumableUpload(ctx, metadata, body, options.ContentType)
}

// OpenRange streams a complete Drive object or an inclusive byte range.
func (s *Store) OpenRange(
	ctx context.Context,
	key string,
	byteRange *storage.ByteRange,
) (io.ReadCloser, storage.ObjectInfo, error) {
	file, err := s.findFile(ctx, key, false)
	if err != nil {
		return nil, storage.ObjectInfo{}, err
	}
	endpoint := s.apiBaseURL + "/files/" + url.PathEscape(file.ID) + "?alt=media"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, storage.ObjectInfo{}, fmt.Errorf("create Drive download request: %w", err)
	}
	if byteRange != nil {
		request.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", byteRange.Start, byteRange.End))
	}
	response, err := s.client.Do(request)
	if err != nil {
		return nil, storage.ObjectInfo{}, fmt.Errorf("download Drive object: %w", err)
	}
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusPartialContent {
		defer response.Body.Close()
		return nil, storage.ObjectInfo{}, s.responseError("download Drive object", response)
	}
	return response.Body, file.objectInfo(key), nil
}

// Stat returns metadata for an available Drive object.
func (s *Store) Stat(ctx context.Context, key string) (storage.ObjectInfo, error) {
	file, err := s.findFile(ctx, key, false)
	if err != nil {
		return storage.ObjectInfo{}, err
	}
	return file.objectInfo(key), nil
}

// Trash marks a Drive object as trashed without deleting it permanently.
func (s *Store) Trash(ctx context.Context, key string) error {
	return s.updateTrashed(ctx, key, true)
}

// Restore clears the trashed flag on a Drive object.
func (s *Store) Restore(ctx context.Context, key string) error {
	return s.updateTrashed(ctx, key, false)
}

// Delete permanently removes a Drive object.
func (s *Store) Delete(ctx context.Context, key string) error {
	file, err := s.findFile(ctx, key, true)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		s.apiBaseURL+"/files/"+url.PathEscape(file.ID), nil)
	if err != nil {
		return fmt.Errorf("create Drive delete request: %w", err)
	}
	request.URL.RawQuery = "supportsAllDrives=true"
	response, err := s.client.Do(request)
	if err != nil {
		return fmt.Errorf("delete Drive object: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent && response.StatusCode != http.StatusOK {
		return s.responseError("delete Drive object", response)
	}
	return nil
}

func (s *Store) resumableUpload(ctx context.Context, metadata driveFile,
	body io.Reader, contentType string) (storage.ObjectInfo, error) {
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return storage.ObjectInfo{}, fmt.Errorf("encode Drive metadata: %w", err)
	}
	method := http.MethodPost
	endpoint := s.uploadBaseURL + "/files?uploadType=resumable&supportsAllDrives=true"
	if metadata.ID != "" {
		method = http.MethodPatch
		endpoint = s.uploadBaseURL + "/files/" + url.PathEscape(metadata.ID) +
			"?uploadType=resumable&supportsAllDrives=true"
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return storage.ObjectInfo{}, fmt.Errorf("create Drive upload session request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	request.Header.Set("X-Upload-Content-Type", contentType)
	response, err := s.client.Do(request)
	if err != nil {
		return storage.ObjectInfo{}, fmt.Errorf("start Drive upload session: %w", err)
	}
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		defer response.Body.Close()
		return storage.ObjectInfo{}, s.responseError("start Drive upload session", response)
	}
	sessionURL := response.Header.Get("Location")
	_ = response.Body.Close()
	if sessionURL == "" {
		return storage.ObjectInfo{}, fmt.Errorf("Drive upload session omitted Location")
	}
	return s.uploadChunks(ctx, sessionURL, body, metadata, contentType)
}

func (s *Store) uploadChunks(ctx context.Context, sessionURL string, body io.Reader,
	metadata driveFile, contentType string) (storage.ObjectInfo, error) {
	current, err := readChunk(body, s.chunkSize)
	if err != nil {
		return storage.ObjectInfo{}, err
	}
	var offset int64
	for {
		next, nextErr := readChunk(body, s.chunkSize)
		if nextErr != nil {
			return storage.ObjectInfo{}, nextErr
		}
		final := len(next) == 0
		total := "*"
		if final {
			total = strconv.FormatInt(offset+int64(len(current)), 10)
		}
		response, err := s.sendChunk(ctx, sessionURL, current, offset, total, contentType)
		if err != nil {
			return storage.ObjectInfo{}, err
		}
		if final {
			defer response.Body.Close()
			var file driveFile
			if err := json.NewDecoder(response.Body).Decode(&file); err != nil {
				return storage.ObjectInfo{}, fmt.Errorf("decode completed Drive upload: %w", err)
			}
			return file.objectInfo(metadata.AppProperties["aetherStorageKey"]), nil
		}
		_ = response.Body.Close()
		offset += int64(len(current))
		current = next
	}
}

func (s *Store) sendChunk(ctx context.Context, sessionURL string, chunk []byte,
	offset int64, total, contentType string) (*http.Response, error) {
	end := offset + int64(len(chunk)) - 1
	for attempt := 0; attempt < 3; attempt++ {
		request, err := http.NewRequestWithContext(ctx, http.MethodPut, sessionURL,
			bytes.NewReader(chunk))
		if err != nil {
			return nil, fmt.Errorf("create Drive chunk request: %w", err)
		}
		request.Header.Set("Content-Length", strconv.Itoa(len(chunk)))
		request.Header.Set("Content-Type", contentType)
		request.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%s", offset, end, total))
		response, err := s.client.Do(request)
		if err == nil && (response.StatusCode == http.StatusOK ||
			response.StatusCode == http.StatusCreated ||
			response.StatusCode == http.StatusPermanentRedirect) {
			return response, nil
		}
		if response != nil {
			_ = response.Body.Close()
		}
		if err != nil && !isTemporaryNetworkError(err) {
			return nil, fmt.Errorf("send Drive upload chunk: %w", err)
		}
		if err == nil && response.StatusCode != http.StatusTooManyRequests && response.StatusCode < 500 {
			return nil, s.responseError("send Drive upload chunk", response)
		}
		if err := waitRetry(ctx, attempt); err != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf("Drive upload chunk retries exhausted")
}

func (s *Store) findFile(ctx context.Context, key string, includeTrashed bool) (driveFile, error) {
	if err := validateKey(key); err != nil {
		return driveFile{}, err
	}
	query := "'" + escapeQueryValue(s.folderID) + "' in parents and " +
		"appProperties has { key='aetherStorageKey' and value='" +
		escapeQueryValue(key) + "' }"
	if !includeTrashed {
		query += " and trashed = false"
	}
	values := url.Values{}
	values.Set("q", query)
	values.Set("pageSize", "10")
	values.Set("fields", "files(id,name,mimeType,size,modifiedTime,trashed,appProperties)")
	request, err := http.NewRequestWithContext(ctx, http.MethodGet,
		s.apiBaseURL+"/files?"+values.Encode(), nil)
	if err != nil {
		return driveFile{}, fmt.Errorf("create Drive metadata request: %w", err)
	}
	response, err := s.client.Do(request)
	if err != nil {
		return driveFile{}, fmt.Errorf("list Drive object: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return driveFile{}, s.responseError("list Drive object", response)
	}
	var listing struct {
		Files []driveFile `json:"files"`
	}
	if err := json.NewDecoder(response.Body).Decode(&listing); err != nil {
		return driveFile{}, fmt.Errorf("decode Drive metadata: %w", err)
	}
	if len(listing.Files) == 0 {
		return driveFile{}, storage.ErrNotFound
	}
	return listing.Files[0], nil
}

func (s *Store) updateTrashed(ctx context.Context, key string, trashed bool) error {
	file, err := s.findFile(ctx, key, true)
	if err != nil {
		return err
	}
	body, err := json.Marshal(map[string]bool{"trashed": trashed})
	if err != nil {
		return fmt.Errorf("encode Drive trash update: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPatch,
		s.apiBaseURL+"/files/"+url.PathEscape(file.ID)+"?supportsAllDrives=true",
		bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create Drive trash update: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := s.client.Do(request)
	if err != nil {
		return fmt.Errorf("update Drive trash state: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return s.responseError("update Drive trash state", response)
	}
	return nil
}

type driveFile struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	MimeType      string            `json:"mimeType"`
	Size          int64             `json:"size,string"`
	ModifiedTime  time.Time         `json:"modifiedTime"`
	Trashed       bool              `json:"trashed"`
	Parents       []string          `json:"parents,omitempty"`
	AppProperties map[string]string `json:"appProperties,omitempty"`
}

func (f driveFile) objectInfo(key string) storage.ObjectInfo {
	return storage.ObjectInfo{Key: key, Size: f.Size, ContentType: f.MimeType,
		LastModified: f.ModifiedTime}
}

func (s *Store) responseError(operation string, response *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	return fmt.Errorf("%s: Drive returned HTTP %d: %s", operation, response.StatusCode,
		strings.TrimSpace(string(body)))
}

func readChunk(reader io.Reader, size int64) ([]byte, error) {
	if size <= 0 {
		return nil, fmt.Errorf("invalid upload chunk size")
	}
	chunk := make([]byte, size)
	read, err := io.ReadFull(reader, chunk)
	if errors.Is(err, io.EOF) {
		return nil, nil
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return chunk[:read], nil
	}
	if err != nil {
		return nil, fmt.Errorf("read Drive upload chunk: %w", err)
	}
	return chunk, nil
}

func validateKey(key string) error {
	if key == "" || strings.ContainsAny(key, "\r\n") || strings.Contains(key, "..") {
		return fmt.Errorf("invalid Drive object key")
	}
	return nil
}

func escapeQueryValue(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, `\`, `\\`), `'`, `\'`)
}

func waitRetry(ctx context.Context, attempt int) error {
	timer := time.NewTimer(time.Duration(1<<attempt) * 100 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func isTemporaryNetworkError(error) bool {
	return true
}
