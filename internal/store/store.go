package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ErrNotFound is returned when a requested PR record does not exist.
var ErrNotFound = errors.New("pr record not found")

// Store provides file-based CRUD for PR records and the import config,
// rooted at a data directory on the host-mounted volume.
type Store struct {
	dataDir    string // <DATA_DIR>/data
	configPath string // <DATA_DIR>/config.json

	mu sync.Mutex // guards concurrent writes to the same files
}

// New creates a Store rooted at dataDirRoot (DATA_DIR), ensuring the PR-data
// subdirectory and a default config.json exist.
func New(dataDirRoot string) (*Store, error) {
	s := &Store{
		dataDir:    filepath.Join(dataDirRoot, "data"),
		configPath: filepath.Join(dataDirRoot, "config.json"),
	}
	if err := os.MkdirAll(s.dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}
	if _, err := os.Stat(s.configPath); errors.Is(err, os.ErrNotExist) {
		if err := s.SaveConfig(Config{ImportRepos: []string{}, ImportOrgs: []string{}, TagTitles: map[string]string{}}); err != nil {
			return nil, fmt.Errorf("writing default config: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("stat config: %w", err)
	}
	return s, nil
}

func fileName(owner, repo string, number int) string {
	return fmt.Sprintf("%s__%s__%d.json", owner, repo, number)
}

func (s *Store) path(owner, repo string, number int) string {
	return filepath.Join(s.dataDir, fileName(owner, repo, number))
}

// atomicWriteJSON marshals v and writes it to path via a temp file + rename,
// so a crash mid-write never leaves a corrupt/partial JSON file behind.
func atomicWriteJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Exists reports whether a PR record is already tracked.
func (s *Store) Exists(owner, repo string, number int) bool {
	_, err := os.Stat(s.path(owner, repo, number))
	return err == nil
}

// Load reads one PR record from disk.
func (s *Store) Load(owner, repo string, number int) (PRRecord, error) {
	data, err := os.ReadFile(s.path(owner, repo, number))
	if errors.Is(err, os.ErrNotExist) {
		return PRRecord{}, ErrNotFound
	}
	if err != nil {
		return PRRecord{}, err
	}
	var rec PRRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return PRRecord{}, err
	}
	return rec, nil
}

// Save persists a PR record, overwriting any existing file for it.
func (s *Store) Save(rec PRRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return atomicWriteJSON(s.path(rec.Owner, rec.Repo, rec.Number), rec)
}

// Delete permanently removes a PR record's file.
func (s *Store) Delete(owner, repo string, number int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := os.Remove(s.path(owner, repo, number))
	if errors.Is(err, os.ErrNotExist) {
		return ErrNotFound
	}
	return err
}

// List reads every tracked PR record from disk.
func (s *Store) List() ([]PRRecord, error) {
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		return nil, err
	}
	records := make([]PRRecord, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dataDir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", e.Name(), err)
		}
		var rec PRRecord
		if err := json.Unmarshal(data, &rec); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", e.Name(), err)
		}
		records = append(records, rec)
	}
	return records, nil
}

// LoadConfig reads the import allow-list fresh from disk (no caching), so
// hand-edits to config.json on the host take effect without a restart.
func (s *Store) LoadConfig() (Config, error) {
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// SaveConfig overwrites config.json.
func (s *Store) SaveConfig(cfg Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return atomicWriteJSON(s.configPath, cfg)
}
