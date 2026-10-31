// Package library stores named JSON files on disk: whole shows and layer
// presets. Names are restricted to a safe character set so they can be
// used directly as file names.
package library

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"ledledled/internal/fsutil"
	"ledledled/internal/show"
)

// Kind is a library section.
type Kind string

const (
	Shows   Kind = "shows"
	Presets Kind = "presets"
)

// Valid reports whether k is a known section.
func (k Kind) Valid() bool { return k == Shows || k == Presets }

// Preset is a reusable layer: its effect setup plus its keyframe tracks.
// Layer and group IDs are reassigned when a preset is added to a show.
type Preset struct {
	Layer  show.Layer   `json:"layer"`
	Tracks []show.Track `json:"tracks"`
}

// Entry describes a stored file.
type Entry struct {
	Name     string    `json:"name"`
	Modified time.Time `json:"modified"`
	Size     int64     `json:"size"`
}

// ErrNotFound is returned for missing entries.
var ErrNotFound = errors.New("not found")

var nameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 _.()-]{0,63}$`)

// ValidName checks a library name. Names start with a letter or digit and
// may contain letters, digits, spaces and _ . ( ) -.
func ValidName(name string) error {
	if !nameRE.MatchString(name) || strings.Contains(name, "..") {
		return fmt.Errorf("invalid name %q: use letters, digits, spaces and _ . ( ) - (max 64)", name)
	}
	return nil
}

// Store is a library rooted at a directory.
type Store struct{ dir string }

// New creates the library directories if needed.
func New(dir string) (*Store, error) {
	for _, k := range []Kind{Shows, Presets} {
		if err := os.MkdirAll(filepath.Join(dir, string(k)), 0o755); err != nil {
			return nil, err
		}
	}
	return &Store{dir: dir}, nil
}

func (s *Store) path(k Kind, name string) (string, error) {
	if !k.Valid() {
		return "", fmt.Errorf("unknown library section %q", k)
	}
	if err := ValidName(name); err != nil {
		return "", err
	}
	return filepath.Join(s.dir, string(k), name+".json"), nil
}

// List returns the entries in a section, newest first.
func (s *Store) List(k Kind) ([]Entry, error) {
	if !k.Valid() {
		return nil, fmt.Errorf("unknown library section %q", k)
	}
	ents, err := os.ReadDir(filepath.Join(s.dir, string(k)))
	if err != nil {
		return nil, err
	}
	out := []Entry{}
	for _, e := range ents {
		name, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok || e.IsDir() || ValidName(name) != nil {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, Entry{Name: name, Modified: info.ModTime(), Size: info.Size()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Modified.After(out[j].Modified) })
	return out, nil
}

// Read returns an entry's raw JSON.
func (s *Store) Read(k Kind, name string) ([]byte, error) {
	p, err := s.path(k, name)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotFound
	}
	return b, err
}

// Write stores raw JSON under name, replacing any existing entry.
func (s *Store) Write(k Kind, name string, data []byte) error {
	p, err := s.path(k, name)
	if err != nil {
		return err
	}
	return fsutil.WriteAtomic(p, data)
}

// Delete removes an entry.
func (s *Store) Delete(k Kind, name string) error {
	p, err := s.path(k, name)
	if err != nil {
		return err
	}
	if err := os.Remove(p); errors.Is(err, fs.ErrNotExist) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	return nil
}
