package uploadstore

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Kind identifies the type of media held in the store.
type Kind string

const (
	KindImage Kind = "image"
	KindAudio Kind = "audio"
	KindVideo Kind = "video"
)

// Meta describes one locally persisted upload.
type Meta struct {
	ID               string `json:"id"`
	Kind             Kind   `json:"kind"`
	Ext              string `json:"ext,omitempty"`
	ContentType      string `json:"content_type,omitempty"`
	OriginalFilename string `json:"original_filename,omitempty"`
	Size             int64  `json:"size"`
}

// Store persists uploaded media bytes on disk so references by id can be
// re-hosted under whichever Leonardo account ends up running a generation.
type Store struct {
	dir   string
	mu    sync.RWMutex
	index map[string]Meta
}

// Open loads (or creates) a store rooted at dir, rebuilding its index from
// the meta files already on disk.
func Open(dir string) (*Store, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, fmt.Errorf("uploadstore: empty directory")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("uploadstore: create directory: %w", err)
	}
	s := &Store{dir: dir, index: make(map[string]Meta)}
	if err := s.rebuildIndex(); err != nil {
		return nil, err
	}
	return s, nil
}

// Dir returns the on-disk root of the store.
func (s *Store) Dir() string {
	return s.dir
}

// Save stores data under meta.ID. Callers must ensure meta.ID is non-empty.
func (s *Store) Save(meta Meta, data []byte) error {
	meta.ID = strings.TrimSpace(meta.ID)
	if meta.ID == "" {
		return fmt.Errorf("uploadstore: cannot save upload with empty id")
	}
	hash := objectKey(meta.ID)
	if err := os.WriteFile(filepath.Join(s.dir, hash+".bin"), data, 0o644); err != nil {
		return fmt.Errorf("uploadstore: write payload: %w", err)
	}
	meta.Size = int64(len(data))
	encoded, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("uploadstore: encode meta: %w", err)
	}
	if err := os.WriteFile(filepath.Join(s.dir, hash+".meta.json"), encoded, 0o644); err != nil {
		return fmt.Errorf("uploadstore: write meta: %w", err)
	}

	s.mu.Lock()
	s.index[meta.ID] = meta
	s.mu.Unlock()
	return nil
}

// Meta returns metadata for id, if present.
func (s *Store) Meta(id string) (Meta, bool) {
	if s == nil {
		return Meta{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	meta, ok := s.index[strings.TrimSpace(id)]
	return meta, ok
}

// Load returns the payload bytes and metadata for id.
func (s *Store) Load(id string) ([]byte, Meta, bool) {
	if s == nil {
		return nil, Meta{}, false
	}
	meta, ok := s.Meta(id)
	if !ok {
		return nil, Meta{}, false
	}
	data, err := os.ReadFile(filepath.Join(s.dir, objectKey(meta.ID)+".bin"))
	if err != nil {
		return nil, Meta{}, false
	}
	return data, meta, true
}

// Count returns the number of locally persisted uploads.
func (s *Store) Count() int {
	if s == nil {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.index)
}

func (s *Store) rebuildIndex() error {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return fmt.Errorf("uploadstore: read directory: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".meta.json") {
			continue
		}
		encoded, err := os.ReadFile(filepath.Join(s.dir, name))
		if err != nil {
			continue
		}
		var meta Meta
		if err := json.Unmarshal(encoded, &meta); err != nil {
			continue
		}
		if strings.TrimSpace(meta.ID) == "" {
			continue
		}
		s.index[meta.ID] = meta
	}
	return nil
}

// objectKey derives a stable, filesystem-safe name for an upload id.
func objectKey(id string) string {
	sum := sha256.Sum256([]byte(id))
	return hex.EncodeToString(sum[:])
}
