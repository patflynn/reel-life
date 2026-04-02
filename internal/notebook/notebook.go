package notebook

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// NoteType distinguishes pinned notes (always loaded into prompt) from reference notes (looked up on demand).
type NoteType string

const (
	Pinned    NoteType = "pinned"
	Reference NoteType = "reference"
)

const (
	MaxPinnedNotes      = 10
	MaxPinnedContentLen = 2000
	MaxReferenceNotes   = 100
)

// Note is a single notebook entry.
type Note struct {
	ID        string    `json:"id"`
	Type      NoteType  `json:"type"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NoteSummary is the abbreviated form returned by List.
type NoteSummary struct {
	ID        string    `json:"id"`
	Type      NoteType  `json:"type"`
	Title     string    `json:"title"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Notebook is the interface for persistent note storage.
type Notebook interface {
	Write(ctx context.Context, note Note) error
	Read(ctx context.Context, id string) (Note, error)
	Search(ctx context.Context, query string) ([]Note, error)
	List(ctx context.Context) ([]NoteSummary, error)
	Delete(ctx context.Context, id string) error
	Pinned(ctx context.Context) ([]Note, error)
}

// FileNotebook stores notes as a single JSON file.
type FileNotebook struct {
	path string
	mu   sync.Mutex
}

// NewFileNotebook creates a FileNotebook backed by the given path.
func NewFileNotebook(path string) *FileNotebook {
	return &FileNotebook{path: path}
}

type store struct {
	Notes []Note `json:"notes"`
}

func (f *FileNotebook) load() (*store, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &store{}, nil
		}
		return nil, fmt.Errorf("reading notebook file: %w", err)
	}
	var s store
	if len(data) > 0 {
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf("parsing notebook file: %w", err)
		}
	}
	return &s, nil
}

func (f *FileNotebook) save(s *store) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling notebook: %w", err)
	}
	tmp := f.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("writing notebook temp file: %w", err)
	}
	_ = os.Remove(f.path)
	if err := os.Rename(tmp, f.path); err != nil {
		return fmt.Errorf("renaming notebook temp file: %w", err)
	}
	return nil
}

func generateID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to generate random ID: %v", err))
	}
	return hex.EncodeToString(b)
}

func pinnedContentLen(notes []Note) int {
	total := 0
	for _, n := range notes {
		if n.Type == Pinned {
			total += len(n.Content)
		}
	}
	return total
}

func pinnedCount(notes []Note) int {
	count := 0
	for _, n := range notes {
		if n.Type == Pinned {
			count++
		}
	}
	return count
}

func referenceCount(notes []Note) int {
	count := 0
	for _, n := range notes {
		if n.Type == Reference {
			count++
		}
	}
	return count
}

// Write creates or updates (upserts) a note.
func (f *FileNotebook) Write(_ context.Context, note Note) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	s, err := f.load()
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	// Upsert: find existing note by ID.
	idx := -1
	if note.ID != "" {
		for i, n := range s.Notes {
			if n.ID == note.ID {
				idx = i
				break
			}
		}
	}

	if idx >= 0 {
		// Update existing note.
		existing := s.Notes[idx]
		existing.Title = note.Title
		existing.Content = note.Content
		existing.Type = note.Type
		existing.UpdatedAt = now

		// Re-check limits if type changed to pinned.
		if existing.Type == Pinned {
			// Temporarily remove existing from the list for limit calculation.
			others := make([]Note, 0, len(s.Notes)-1)
			for i, n := range s.Notes {
				if i != idx {
					others = append(others, n)
				}
			}
			if pinnedCount(others) >= MaxPinnedNotes {
				return fmt.Errorf("cannot have more than %d pinned notes", MaxPinnedNotes)
			}
			if pinnedContentLen(others)+len(existing.Content) > MaxPinnedContentLen {
				return fmt.Errorf("pinned notes total content would exceed %d characters", MaxPinnedContentLen)
			}
		}
		s.Notes[idx] = existing
	} else {
		// Create new note.
		if note.ID == "" {
			note.ID = generateID()
		}
		note.CreatedAt = now
		note.UpdatedAt = now

		if note.Type == Pinned {
			if pinnedCount(s.Notes) >= MaxPinnedNotes {
				return fmt.Errorf("cannot have more than %d pinned notes", MaxPinnedNotes)
			}
			if pinnedContentLen(s.Notes)+len(note.Content) > MaxPinnedContentLen {
				return fmt.Errorf("pinned notes total content would exceed %d characters", MaxPinnedContentLen)
			}
		} else {
			if referenceCount(s.Notes) >= MaxReferenceNotes {
				return fmt.Errorf("cannot have more than %d reference notes", MaxReferenceNotes)
			}
		}

		s.Notes = append(s.Notes, note)
	}

	return f.save(s)
}

// Read returns a note by ID.
func (f *FileNotebook) Read(_ context.Context, id string) (Note, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	s, err := f.load()
	if err != nil {
		return Note{}, err
	}

	for _, n := range s.Notes {
		if n.ID == id {
			return n, nil
		}
	}
	return Note{}, fmt.Errorf("note %q not found", id)
}

// Search returns notes matching a substring in title or content.
func (f *FileNotebook) Search(_ context.Context, query string) ([]Note, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	s, err := f.load()
	if err != nil {
		return nil, err
	}

	q := strings.ToLower(query)
	var results []Note
	for _, n := range s.Notes {
		if strings.Contains(strings.ToLower(n.Title), q) || strings.Contains(strings.ToLower(n.Content), q) {
			results = append(results, n)
		}
	}
	return results, nil
}

// List returns summaries of all notes.
func (f *FileNotebook) List(_ context.Context) ([]NoteSummary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	s, err := f.load()
	if err != nil {
		return nil, err
	}

	summaries := make([]NoteSummary, len(s.Notes))
	for i, n := range s.Notes {
		summaries[i] = NoteSummary{
			ID:        n.ID,
			Type:      n.Type,
			Title:     n.Title,
			UpdatedAt: n.UpdatedAt,
		}
	}
	return summaries, nil
}

// Delete removes a note by ID.
func (f *FileNotebook) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	s, err := f.load()
	if err != nil {
		return err
	}

	for i, n := range s.Notes {
		if n.ID == id {
			s.Notes = append(s.Notes[:i], s.Notes[i+1:]...)
			return f.save(s)
		}
	}
	return fmt.Errorf("note %q not found", id)
}

// Pinned returns all pinned notes.
func (f *FileNotebook) Pinned(_ context.Context) ([]Note, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	s, err := f.load()
	if err != nil {
		return nil, err
	}

	var pinned []Note
	for _, n := range s.Notes {
		if n.Type == Pinned {
			pinned = append(pinned, n)
		}
	}
	return pinned, nil
}
