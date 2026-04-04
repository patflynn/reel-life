package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

func TestPersistentHistoryStoreSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")

	// Create store, add data, let it persist.
	store := NewPersistentHistoryStore(10, path)
	store.Get("chat-1").Add("user", "hello")
	store.Get("chat-1").Add("assistant", "hi there")
	store.Get("chat-2").Add("user", "hey")

	// Load into a new store and verify.
	store2 := NewPersistentHistoryStore(10, path)
	turns1 := store2.Get("chat-1").Turns()
	if len(turns1) != 2 {
		t.Fatalf("chat-1: got %d turns, want 2", len(turns1))
	}
	if turns1[0].Role != "user" || turns1[0].Content != "hello" {
		t.Errorf("chat-1 turns[0] = %+v, want user/hello", turns1[0])
	}
	if turns1[1].Role != "assistant" || turns1[1].Content != "hi there" {
		t.Errorf("chat-1 turns[1] = %+v, want assistant/hi there", turns1[1])
	}

	turns2 := store2.Get("chat-2").Turns()
	if len(turns2) != 1 || turns2[0].Content != "hey" {
		t.Errorf("chat-2 turns = %+v, want [user/hey]", turns2)
	}
}

func TestPersistentHistoryStoreFileNotExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.json")
	store := NewPersistentHistoryStore(10, path)
	turns := store.Get("chat-1").Turns()
	if len(turns) != 0 {
		t.Errorf("expected empty turns for new store, got %d", len(turns))
	}
}

func TestPersistentHistoryStoreCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	os.WriteFile(path, []byte("not json{{{"), 0644)

	store := NewPersistentHistoryStore(10, path)
	turns := store.Get("chat-1").Turns()
	if len(turns) != 0 {
		t.Errorf("expected empty turns for corrupt file, got %d", len(turns))
	}
}

func TestPersistentHistoryStoreAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history.json")

	store := NewPersistentHistoryStore(10, path)
	store.Get("chat-1").Add("user", "hello")

	// Verify the file exists and is valid JSON.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read history file: %v", err)
	}
	var hf struct {
		Chats map[string][]Turn `json:"chats"`
	}
	if err := json.Unmarshal(data, &hf); err != nil {
		t.Fatalf("history file is not valid JSON: %v", err)
	}
	if len(hf.Chats["chat-1"]) != 1 {
		t.Errorf("expected 1 turn in file, got %d", len(hf.Chats["chat-1"]))
	}

	// No temp files should remain.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "history.json" {
			t.Errorf("unexpected file in dir: %s", e.Name())
		}
	}
}

func TestPersistentHistoryStoreEvictionSurvivesReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")

	store := NewPersistentHistoryStore(3, path)
	buf := store.Get("chat-1")
	buf.Add("user", "a")
	buf.Add("assistant", "b")
	buf.Add("user", "c")
	buf.Add("assistant", "d") // evicts "a"

	store2 := NewPersistentHistoryStore(3, path)
	turns := store2.Get("chat-1").Turns()
	if len(turns) != 3 {
		t.Fatalf("got %d turns, want 3", len(turns))
	}
	if turns[0].Content != "b" {
		t.Errorf("oldest turn = %q, want %q", turns[0].Content, "b")
	}
	if turns[2].Content != "d" {
		t.Errorf("newest turn = %q, want %q", turns[2].Content, "d")
	}
}

func TestPersistentHistoryStoreNewChatAfterReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")

	store := NewPersistentHistoryStore(10, path)
	store.Get("chat-1").Add("user", "hello")

	// Reload and add a new chat.
	store2 := NewPersistentHistoryStore(10, path)
	store2.Get("chat-2").Add("user", "world")

	// Reload again — both chats should be present.
	store3 := NewPersistentHistoryStore(10, path)
	if turns := store3.Get("chat-1").Turns(); len(turns) != 1 {
		t.Errorf("chat-1: got %d turns, want 1", len(turns))
	}
	if turns := store3.Get("chat-2").Turns(); len(turns) != 1 {
		t.Errorf("chat-2: got %d turns, want 1", len(turns))
	}
}
