package agent

import "sync"

// Turn represents a single conversation turn (user message or assistant reply).
type Turn struct {
	Role    string // "user" or "assistant"
	Content string
}

// ConversationBuffer is a thread-safe ring buffer for conversation turns.
type ConversationBuffer struct {
	mu    sync.Mutex
	turns []Turn
	head  int
	count int
	cap   int
}

// NewConversationBuffer creates a buffer that holds up to capacity turns.
func NewConversationBuffer(capacity int) *ConversationBuffer {
	return &ConversationBuffer{
		turns: make([]Turn, capacity),
		cap:   capacity,
	}
}

// Add appends a turn, overwriting the oldest when full.
func (b *ConversationBuffer) Add(role, content string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	idx := (b.head + b.count) % b.cap
	if b.count == b.cap {
		// Buffer full — overwrite oldest and advance head.
		b.turns[b.head] = Turn{Role: role, Content: content}
		b.head = (b.head + 1) % b.cap
	} else {
		b.turns[idx] = Turn{Role: role, Content: content}
		b.count++
	}
}

// Turns returns the stored turns in chronological order.
func (b *ConversationBuffer) Turns() []Turn {
	b.mu.Lock()
	defer b.mu.Unlock()

	result := make([]Turn, b.count)
	for i := range b.count {
		result[i] = b.turns[(b.head+i)%b.cap]
	}
	return result
}

// Reset clears all stored turns.
func (b *ConversationBuffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.head = 0
	b.count = 0
}
