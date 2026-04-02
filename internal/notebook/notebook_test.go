package notebook

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
)

func newTestNotebook(t *testing.T) *FileNotebook {
	t.Helper()
	path := filepath.Join(t.TempDir(), "notebook.json")
	return NewFileNotebook(path)
}

func TestWriteAndRead(t *testing.T) {
	nb := newTestNotebook(t)
	ctx := context.Background()

	err := nb.Write(ctx, Note{
		Type:    Reference,
		Title:   "Test Note",
		Content: "Some content",
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	notes, err := nb.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(notes))
	}

	note, err := nb.Read(ctx, notes[0].ID)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if note.Title != "Test Note" || note.Content != "Some content" {
		t.Errorf("unexpected note: %+v", note)
	}
	if note.ID == "" {
		t.Error("expected auto-generated ID")
	}
	if note.CreatedAt.IsZero() || note.UpdatedAt.IsZero() {
		t.Error("expected timestamps to be set")
	}
}

func TestUpsert(t *testing.T) {
	nb := newTestNotebook(t)
	ctx := context.Background()

	err := nb.Write(ctx, Note{
		ID:      "test-id",
		Type:    Reference,
		Title:   "Original",
		Content: "Original content",
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	err = nb.Write(ctx, Note{
		ID:      "test-id",
		Type:    Reference,
		Title:   "Updated",
		Content: "Updated content",
	})
	if err != nil {
		t.Fatalf("Write (update): %v", err)
	}

	notes, err := nb.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 note after upsert, got %d", len(notes))
	}

	note, err := nb.Read(ctx, "test-id")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if note.Title != "Updated" || note.Content != "Updated content" {
		t.Errorf("upsert did not update: %+v", note)
	}
}

func TestDelete(t *testing.T) {
	nb := newTestNotebook(t)
	ctx := context.Background()

	nb.Write(ctx, Note{ID: "del-me", Type: Reference, Title: "Delete Me", Content: "x"})

	err := nb.Delete(ctx, "del-me")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = nb.Read(ctx, "del-me")
	if err == nil {
		t.Fatal("expected error reading deleted note")
	}
}

func TestDeleteNotFound(t *testing.T) {
	nb := newTestNotebook(t)
	err := nb.Delete(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error deleting nonexistent note")
	}
}

func TestSearch(t *testing.T) {
	nb := newTestNotebook(t)
	ctx := context.Background()

	nb.Write(ctx, Note{ID: "1", Type: Reference, Title: "User Preferences", Content: "likes sci-fi"})
	nb.Write(ctx, Note{ID: "2", Type: Reference, Title: "Server Config", Content: "runs on port 8989"})
	nb.Write(ctx, Note{ID: "3", Type: Pinned, Title: "Important", Content: "user prefers 1080p"})

	results, err := nb.Search(ctx, "prefer")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	results, err = nb.Search(ctx, "port 8989")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 || results[0].ID != "2" {
		t.Errorf("unexpected search results: %+v", results)
	}
}

func TestPinned(t *testing.T) {
	nb := newTestNotebook(t)
	ctx := context.Background()

	nb.Write(ctx, Note{ID: "1", Type: Pinned, Title: "Pinned 1", Content: "p1"})
	nb.Write(ctx, Note{ID: "2", Type: Reference, Title: "Ref 1", Content: "r1"})
	nb.Write(ctx, Note{ID: "3", Type: Pinned, Title: "Pinned 2", Content: "p2"})

	pinned, err := nb.Pinned(ctx)
	if err != nil {
		t.Fatalf("Pinned: %v", err)
	}
	if len(pinned) != 2 {
		t.Fatalf("expected 2 pinned notes, got %d", len(pinned))
	}
}

func TestPinnedCountLimit(t *testing.T) {
	nb := newTestNotebook(t)
	ctx := context.Background()

	for i := range MaxPinnedNotes {
		err := nb.Write(ctx, Note{
			ID:      fmt.Sprintf("pin-%d", i),
			Type:    Pinned,
			Title:   fmt.Sprintf("Pinned %d", i),
			Content: "x",
		})
		if err != nil {
			t.Fatalf("Write pinned %d: %v", i, err)
		}
	}

	err := nb.Write(ctx, Note{
		Type:    Pinned,
		Title:   "One Too Many",
		Content: "x",
	})
	if err == nil {
		t.Fatal("expected error exceeding pinned count limit")
	}
}

func TestPinnedContentLimit(t *testing.T) {
	nb := newTestNotebook(t)
	ctx := context.Background()

	// Write a pinned note that nearly fills the content limit.
	bigContent := make([]byte, MaxPinnedContentLen-10)
	for i := range bigContent {
		bigContent[i] = 'a'
	}
	err := nb.Write(ctx, Note{
		ID:      "big",
		Type:    Pinned,
		Title:   "Big",
		Content: string(bigContent),
	})
	if err != nil {
		t.Fatalf("Write big pinned: %v", err)
	}

	// This should exceed the limit.
	err = nb.Write(ctx, Note{
		Type:    Pinned,
		Title:   "Overflow",
		Content: "this is way more than 10 chars",
	})
	if err == nil {
		t.Fatal("expected error exceeding pinned content limit")
	}
}

func TestReferenceCountLimit(t *testing.T) {
	nb := newTestNotebook(t)
	ctx := context.Background()

	for i := range MaxReferenceNotes {
		err := nb.Write(ctx, Note{
			ID:      fmt.Sprintf("ref-%d", i),
			Type:    Reference,
			Title:   fmt.Sprintf("Ref %d", i),
			Content: "x",
		})
		if err != nil {
			t.Fatalf("Write ref %d: %v", i, err)
		}
	}

	err := nb.Write(ctx, Note{
		Type:    Reference,
		Title:   "One Too Many",
		Content: "x",
	})
	if err == nil {
		t.Fatal("expected error exceeding reference count limit")
	}
}

func TestListTypes(t *testing.T) {
	nb := newTestNotebook(t)
	ctx := context.Background()

	nb.Write(ctx, Note{ID: "1", Type: Pinned, Title: "Pinned", Content: "p"})
	nb.Write(ctx, Note{ID: "2", Type: Reference, Title: "Reference", Content: "r"})

	summaries, err := nb.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}
	// Verify summaries don't contain content (they're NoteSummary structs).
	for _, s := range summaries {
		if s.ID == "" || s.Title == "" {
			t.Errorf("summary missing fields: %+v", s)
		}
	}
}

func TestPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notebook.json")
	ctx := context.Background()

	// Write with one instance.
	nb1 := NewFileNotebook(path)
	nb1.Write(ctx, Note{ID: "persist", Type: Reference, Title: "Persist", Content: "survives restart"})

	// Read with a fresh instance.
	nb2 := NewFileNotebook(path)
	note, err := nb2.Read(ctx, "persist")
	if err != nil {
		t.Fatalf("Read from fresh instance: %v", err)
	}
	if note.Content != "survives restart" {
		t.Errorf("unexpected content: %q", note.Content)
	}
}

func TestReadNotFound(t *testing.T) {
	nb := newTestNotebook(t)
	_, err := nb.Read(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error reading nonexistent note")
	}
}

func TestEmptyNotebook(t *testing.T) {
	nb := newTestNotebook(t)
	ctx := context.Background()

	notes, err := nb.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(notes) != 0 {
		t.Errorf("expected 0 notes, got %d", len(notes))
	}

	pinned, err := nb.Pinned(ctx)
	if err != nil {
		t.Fatalf("Pinned: %v", err)
	}
	if len(pinned) != 0 {
		t.Errorf("expected 0 pinned, got %d", len(pinned))
	}

	results, err := nb.Search(ctx, "anything")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}
