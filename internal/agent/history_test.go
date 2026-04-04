package agent

import (
	"fmt"
	"sync"
	"testing"
)

func TestConversationBufferAddAndTurns(t *testing.T) {
	buf := NewConversationBuffer(4)
	buf.Add("user", "hello")
	buf.Add("assistant", "hi there")
	buf.Add("user", "how are you?")
	buf.Add("assistant", "great!")

	turns := buf.Turns()
	if len(turns) != 4 {
		t.Fatalf("got %d turns, want 4", len(turns))
	}
	if turns[0].Role != "user" || turns[0].Content != "hello" {
		t.Errorf("turns[0] = %+v, want user/hello", turns[0])
	}
	if turns[3].Role != "assistant" || turns[3].Content != "great!" {
		t.Errorf("turns[3] = %+v, want assistant/great!", turns[3])
	}
}

func TestConversationBufferEviction(t *testing.T) {
	buf := NewConversationBuffer(3)
	buf.Add("user", "a")
	buf.Add("assistant", "b")
	buf.Add("user", "c")
	// Buffer is full. Next add should evict "a".
	buf.Add("assistant", "d")

	turns := buf.Turns()
	if len(turns) != 3 {
		t.Fatalf("got %d turns, want 3", len(turns))
	}
	if turns[0].Content != "b" {
		t.Errorf("oldest turn = %q, want %q (should have evicted 'a')", turns[0].Content, "b")
	}
	if turns[2].Content != "d" {
		t.Errorf("newest turn = %q, want %q", turns[2].Content, "d")
	}
}

func TestConversationBufferWraparound(t *testing.T) {
	buf := NewConversationBuffer(2)
	// Fill and wrap multiple times.
	for i := range 6 {
		buf.Add("user", fmt.Sprintf("msg-%d", i))
	}
	turns := buf.Turns()
	if len(turns) != 2 {
		t.Fatalf("got %d turns, want 2", len(turns))
	}
	if turns[0].Content != "msg-4" || turns[1].Content != "msg-5" {
		t.Errorf("turns = %+v, want msg-4 and msg-5", turns)
	}
}

func TestConversationBufferReset(t *testing.T) {
	buf := NewConversationBuffer(4)
	buf.Add("user", "hello")
	buf.Add("assistant", "hi")
	buf.Reset()

	turns := buf.Turns()
	if len(turns) != 0 {
		t.Errorf("got %d turns after reset, want 0", len(turns))
	}
}

func TestConversationBufferZeroCapacity(t *testing.T) {
	buf := NewConversationBuffer(0)
	buf.Add("user", "hello")
	buf.Add("assistant", "hi")

	turns := buf.Turns()
	if len(turns) != 0 {
		t.Errorf("got %d turns from zero-capacity buffer, want 0", len(turns))
	}

	// Reset should also be safe.
	buf.Reset()
}

func TestConversationBufferEmptyTurns(t *testing.T) {
	buf := NewConversationBuffer(4)
	turns := buf.Turns()
	if len(turns) != 0 {
		t.Errorf("got %d turns from empty buffer, want 0", len(turns))
	}
}

func TestConversationBufferConcurrency(t *testing.T) {
	buf := NewConversationBuffer(100)
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			buf.Add("user", fmt.Sprintf("msg-%d", n))
			buf.Turns() // concurrent read
		}(i)
	}
	wg.Wait()

	turns := buf.Turns()
	if len(turns) != 50 {
		t.Errorf("got %d turns, want 50", len(turns))
	}
}

func TestHistoryStoreIndependentBuffers(t *testing.T) {
	store := NewHistoryStore(10)

	buf1 := store.Get("chat-1")
	buf1.Add("user", "hello from chat 1")

	buf2 := store.Get("chat-2")
	buf2.Add("user", "hello from chat 2")

	if turns := store.Get("chat-1").Turns(); len(turns) != 1 || turns[0].Content != "hello from chat 1" {
		t.Errorf("chat-1 turns = %+v, unexpected", turns)
	}
	if turns := store.Get("chat-2").Turns(); len(turns) != 1 || turns[0].Content != "hello from chat 2" {
		t.Errorf("chat-2 turns = %+v, unexpected", turns)
	}
}

func TestHistoryStoreGetReturnsSameBuffer(t *testing.T) {
	store := NewHistoryStore(10)
	buf1 := store.Get("chat-1")
	buf2 := store.Get("chat-1")
	if buf1 != buf2 {
		t.Error("expected same buffer for same chat ID")
	}
}

func TestHistoryStoreConcurrentGet(t *testing.T) {
	store := NewHistoryStore(10)
	var wg sync.WaitGroup
	buffers := make([]*ConversationBuffer, 20)
	for i := range 20 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			buffers[n] = store.Get("same-chat")
		}(i)
	}
	wg.Wait()

	// All should be the same buffer.
	for i := 1; i < 20; i++ {
		if buffers[i] != buffers[0] {
			t.Fatalf("buffer[%d] differs from buffer[0]", i)
		}
	}
}
