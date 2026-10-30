package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

const encryptedFilePrefix = "BRISA_ENC_V1:"

// StoreOptions configures the local persistent desktop store.
type StoreOptions struct {
	// Path overrides the default file path (os.UserConfigDir()/<AppID>/store.json).
	Path string
	// EncryptionKey enables transparent AES-256-GCM encryption at rest when non-empty.
	EncryptionKey string
	// MemoryOnly keeps all data in RAM without writing to disk (useful for tests or ephemeral apps).
	MemoryOnly bool
}

// Store provides thread-safe persistent key-value and structured JSON storage for desktop applications.
type Store struct {
	mu         sync.RWMutex
	path       string
	aead       cipher.AEAD
	memoryOnly bool
	data       map[string]json.RawMessage
}

// NewStore initializes a Store with the given application ID and options.
func NewStore(appID string, opts StoreOptions) (*Store, error) {
	if appID == "" {
		appID = "brisa-app"
	}

	s := &Store{
		memoryOnly: opts.MemoryOnly,
		data:       make(map[string]json.RawMessage),
	}

	if opts.EncryptionKey != "" {
		hash := sha256.Sum256([]byte(opts.EncryptionKey))
		block, err := aes.NewCipher(hash[:])
		if err != nil {
			return nil, fmt.Errorf("brisa store cipher init: %w", err)
		}
		aead, err := cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("brisa store gcm init: %w", err)
		}
		s.aead = aead
	}

	if !opts.MemoryOnly {
		storePath := opts.Path
		if storePath == "" {
			cfgDir, err := os.UserConfigDir()
			if err != nil || cfgDir == "" {
				cfgDir = os.TempDir()
			}
			storePath = filepath.Join(cfgDir, appID, "store.json")
		}
		s.path = storePath
		if err := s.load(); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}

	return s, nil
}

// Path returns the filesystem path backing the store (empty if MemoryOnly).
func (s *Store) Path() string {
	return s.path
}

// Set stores any JSON-serializable value under key and persists to disk immediately.
func (s *Store) Set(key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("brisa store marshal %q: %w", key, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = raw
	return s.saveLocked()
}

// Get unmarshals the stored value for key into dest. Returns false if the key does not exist.
func (s *Store) Get(key string, dest any) (bool, error) {
	s.mu.RLock()
	raw, ok := s.data[key]
	s.mu.RUnlock()

	if !ok {
		return false, nil
	}
	if dest == nil {
		return true, nil
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return true, fmt.Errorf("brisa store unmarshal %q: %w", key, err)
	}
	return true, nil
}

// GetAny returns the decoded generic value for key, or nil if absent.
func (s *Store) GetAny(key string) (any, bool) {
	s.mu.RLock()
	raw, ok := s.data[key]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, false
	}
	return out, true
}

// GetString returns a string value for key, or fallback if not found or not a string.
func (s *Store) GetString(key string, fallback ...string) string {
	var def string
	if len(fallback) > 0 {
		def = fallback[0]
	}
	var val string
	ok, err := s.Get(key, &val)
	if !ok || err != nil {
		return def
	}
	return val
}

// GetInt returns an int value for key, or fallback if not found.
func (s *Store) GetInt(key string, fallback ...int) int {
	var def int
	if len(fallback) > 0 {
		def = fallback[0]
	}
	var val int
	ok, err := s.Get(key, &val)
	if !ok || err != nil {
		return def
	}
	return val
}

// GetBool returns a bool value for key, or fallback if not found.
func (s *Store) GetBool(key string, fallback ...bool) bool {
	var def bool
	if len(fallback) > 0 {
		def = fallback[0]
	}
	var val bool
	ok, err := s.Get(key, &val)
	if !ok || err != nil {
		return def
	}
	return val
}

// Has reports whether key exists in the store.
func (s *Store) Has(key string) bool {
	s.mu.RLock()
	_, ok := s.data[key]
	s.mu.RUnlock()
	return ok
}

// Delete removes key from the store and persists the change.
func (s *Store) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)
	return s.saveLocked()
}

// Clear removes all keys from the store and persists the empty state.
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = make(map[string]json.RawMessage)
	return s.saveLocked()
}

// Keys returns a sorted list of all keys currently in the store.
func (s *Store) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// All returns a map of all stored key-value pairs decoded as generic JSON values.
func (s *Store) All() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make(map[string]any, len(s.data))
	for k, raw := range s.data {
		var v any
		if err := json.Unmarshal(raw, &v); err == nil {
			out[k] = v
		}
	}
	return out
}

func (s *Store) load() error {
	if s.memoryOnly || s.path == "" {
		return nil
	}
	content, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	if len(content) == 0 {
		return nil
	}

	payload := content
	if s.aead != nil {
		str := string(content)
		if !hasPrefix(str, encryptedFilePrefix) {
			return errors.New("brisa store: file is not encrypted or has invalid header")
		}
		rawCipher, err := base64.StdEncoding.DecodeString(str[len(encryptedFilePrefix):])
		if err != nil {
			return fmt.Errorf("brisa store decode cipher: %w", err)
		}
		nonceSize := s.aead.NonceSize()
		if len(rawCipher) < nonceSize {
			return errors.New("brisa store: ciphertext too short")
		}
		nonce, ciphertext := rawCipher[:nonceSize], rawCipher[nonceSize:]
		plain, err := s.aead.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return fmt.Errorf("brisa store decrypt failed (invalid encryption key?): %w", err)
		}
		payload = plain
	}

	var loaded map[string]json.RawMessage
	if err := json.Unmarshal(payload, &loaded); err != nil {
		return fmt.Errorf("brisa store parse json: %w", err)
	}
	if loaded != nil {
		s.data = loaded
	}
	return nil
}

func (s *Store) saveLocked() error {
	if s.memoryOnly || s.path == "" {
		return nil
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("brisa store mkdir: %w", err)
	}

	rawJSON, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("brisa store encode: %w", err)
	}

	outBytes := rawJSON
	if s.aead != nil {
		nonce := make([]byte, s.aead.NonceSize())
		if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
			return fmt.Errorf("brisa store nonce: %w", err)
		}
		sealed := s.aead.Seal(nonce, nonce, rawJSON, nil)
		outBytes = []byte(encryptedFilePrefix + base64.StdEncoding.EncodeToString(sealed))
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, outBytes, 0o600); err != nil {
		return fmt.Errorf("brisa store write temp: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("brisa store atomic rename: %w", err)
	}
	return nil
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
