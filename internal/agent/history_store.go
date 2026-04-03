package agent

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// HistoryStore maps chat IDs to per-chat conversation buffers.
type HistoryStore struct {
	mu      sync.RWMutex
	buffers map[string]*ConversationBuffer
	cap     int
	path    string // empty = in-memory only
}

// NewHistoryStore creates an in-memory store where each chat gets a buffer of turnCapacity turns.
func NewHistoryStore(turnCapacity int) *HistoryStore {
	return &HistoryStore{
		buffers: make(map[string]*ConversationBuffer),
		cap:     turnCapacity,
	}
}

// NewPersistentHistoryStore creates a store that persists conversation history to a JSON file.
// Existing history is loaded from the file on creation. If the file doesn't exist or is
// corrupt, the store starts empty.
func NewPersistentHistoryStore(turnCapacity int, path string) *HistoryStore {
	s := &HistoryStore{
		buffers: make(map[string]*ConversationBuffer),
		cap:     turnCapacity,
		path:    path,
	}
	s.load()
	return s
}

// Get returns the conversation buffer for the given chat ID, creating one if needed.
func (s *HistoryStore) Get(chatID string) *ConversationBuffer {
	s.mu.RLock()
	buf, ok := s.buffers[chatID]
	s.mu.RUnlock()
	if ok {
		return buf
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring write lock.
	if buf, ok := s.buffers[chatID]; ok {
		return buf
	}

	buf = NewConversationBuffer(s.cap)
	if s.path != "" {
		buf.onChange = func() { s.save() }
	}
	s.buffers[chatID] = buf
	s.saveLocked()
	return buf
}

// historyFile is the JSON structure persisted to disk.
type historyFile struct {
	Chats map[string][]Turn `json:"chats"`
}

// load reads persisted history from disk and populates buffers.
// Must be called before the store is used concurrently (i.e., during construction).
func (s *HistoryStore) load() {
	if s.path == "" {
		return
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("failed to read history file, starting empty", "path", s.path, "error", err)
		}
		return
	}

	var hf historyFile
	if err := json.Unmarshal(data, &hf); err != nil {
		slog.Warn("corrupt history file, starting empty", "path", s.path, "error", err)
		return
	}

	for chatID, turns := range hf.Chats {
		buf := NewConversationBuffer(s.cap)
		for _, t := range turns {
			buf.Add(t.Role, t.Content)
		}
		buf.onChange = func() { s.save() }
		s.buffers[chatID] = buf
	}
}

// save persists the full store state to disk. Acquires the store mutex.
func (s *HistoryStore) save() {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.saveLocked()
}

// saveLocked persists state to disk. Caller must hold at least a read lock on s.mu.
func (s *HistoryStore) saveLocked() {
	if s.path == "" {
		return
	}

	hf := historyFile{Chats: make(map[string][]Turn, len(s.buffers))}
	for chatID, buf := range s.buffers {
		hf.Chats[chatID] = buf.Turns()
	}

	data, err := json.MarshalIndent(hf, "", "  ")
	if err != nil {
		slog.Error("failed to marshal history", "error", err)
		return
	}

	// Atomic write: temp file + rename.
	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, ".history-*.tmp")
	if err != nil {
		slog.Error("failed to create temp file for history", "error", err)
		return
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		slog.Error("failed to write history temp file", "error", err)
		return
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		slog.Error("failed to close history temp file", "error", err)
		return
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		os.Remove(tmpName)
		slog.Error("failed to rename history file", "error", err)
		return
	}
}
