// Package store reads and writes state.json.
//
// state.json holds only the last-used profile, used as the default selection in
// the interactive picker. It is NOT authoritative "active" state: multiple
// profiles can run simultaneously in different terminals.
package store

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/raphaelneumann/claudectx/internal/paths"
)

// State is the on-disk shape of state.json.
type State struct {
	LastUsed  string `json:"lastUsed"`
	UpdatedAt string `json:"updatedAt"`
}

// Load reads state.json. A missing file is not an error (returns zero State).
func Load() (State, error) {
	var s State
	f, err := paths.StateFile()
	if err != nil {
		return s, err
	}
	b, err := os.ReadFile(f)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return s, err
	}
	return s, nil
}

// SetLastUsed records name as the most recently used profile (atomic write).
func SetLastUsed(name string) error {
	f, err := paths.StateFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f), 0o700); err != nil {
		return err
	}
	s := State{LastUsed: name, UpdatedAt: time.Now().UTC().Format(time.RFC3339)}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(f), ".state-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once renamed
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpName, f)
}
