package agent

import "sync"

// HistoryStore maps chat IDs to per-chat conversation buffers.
type HistoryStore struct {
	mu      sync.RWMutex
	buffers map[string]*ConversationBuffer
	cap     int
}

// NewHistoryStore creates a store where each chat gets a buffer of turnCapacity turns.
func NewHistoryStore(turnCapacity int) *HistoryStore {
	return &HistoryStore{
		buffers: make(map[string]*ConversationBuffer),
		cap:     turnCapacity,
	}
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
	s.buffers[chatID] = buf
	return buf
}
