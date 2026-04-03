package history

import (
	"dops/internal/domain"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestRecord(rbID, catalog, name string, status domain.ExecStatus, exitCode int) *domain.ExecutionRecord {
	r := domain.NewExecutionRecord(rbID, name, catalog, domain.ExecCLI)
	r.Complete(exitCode, 10, "done")
	if status == domain.ExecFailed {
		r.Status = domain.ExecFailed
	}
	return r
}

func TestFileStore_RecordAndGet(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir, 0)

	rec := domain.NewExecutionRecord("default.hello", "hello", "default", domain.ExecCLI)
	rec.Parameters = map[string]string{"greeting": "hi"}
	rec.Complete(0, 5, "Hello!")

	if err := store.Record(rec); err != nil {
		t.Fatalf("Record: %v", err)
	}

	got, err := store.Get(rec.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != rec.ID {
		t.Errorf("ID = %q, want %q", got.ID, rec.ID)
	}
	if got.RunbookID != "default.hello" {
		t.Errorf("RunbookID = %q", got.RunbookID)
	}
	if got.Status != domain.ExecSuccess {
		t.Errorf("Status = %q", got.Status)
	}
	if got.Parameters["greeting"] != "hi" {
		t.Errorf("Parameters[greeting] = %q", got.Parameters["greeting"])
	}
}

func TestFileStore_Get_NotFound(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir, 0)

	_, err := store.Get("nonexistent")
	if err == nil {
		t.Error("expected error for missing record")
	}
}

func TestFileStore_List_NewestFirst(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir, 0)

	// Create records with different timestamps.
	r1 := domain.NewExecutionRecord("default.first", "first", "default", domain.ExecCLI)
	r1.Complete(0, 1, "first")
	store.Record(r1)

	time.Sleep(time.Millisecond * 10) // ensure distinct timestamps

	r2 := domain.NewExecutionRecord("default.second", "second", "default", domain.ExecCLI)
	r2.Complete(0, 1, "second")
	store.Record(r2)

	records, err := store.List(ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].RunbookID != "default.second" {
		t.Errorf("first result should be newest: got %q", records[0].RunbookID)
	}
	if records[1].RunbookID != "default.first" {
		t.Errorf("second result should be oldest: got %q", records[1].RunbookID)
	}
}

func TestFileStore_List_FilterByRunbook(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir, 0)

	r1 := newTestRecord("default.hello", "default", "hello", domain.ExecSuccess, 0)
	r2 := newTestRecord("infra.deploy", "infra", "deploy", domain.ExecSuccess, 0)
	store.Record(r1)
	store.Record(r2)

	records, _ := store.List(ListOpts{RunbookID: "infra.deploy", Limit: 10})
	if len(records) != 1 {
		t.Fatalf("expected 1 filtered record, got %d", len(records))
	}
	if records[0].RunbookID != "infra.deploy" {
		t.Errorf("filtered record = %q", records[0].RunbookID)
	}
}

func TestFileStore_List_FilterByStatus(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir, 0)

	r1 := newTestRecord("default.hello", "default", "hello", domain.ExecSuccess, 0)
	r2 := newTestRecord("default.fail", "default", "fail", domain.ExecFailed, 1)
	store.Record(r1)
	store.Record(r2)

	records, _ := store.List(ListOpts{Status: domain.ExecFailed, Limit: 10})
	if len(records) != 1 {
		t.Fatalf("expected 1 failed record, got %d", len(records))
	}
	if records[0].Status != domain.ExecFailed {
		t.Errorf("status = %q", records[0].Status)
	}
}

func TestFileStore_List_LimitAndOffset(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir, 0)

	for i := 0; i < 5; i++ {
		r := newTestRecord("default.hello", "default", "hello", domain.ExecSuccess, 0)
		time.Sleep(time.Millisecond * 10)
		store.Record(r)
	}

	// Limit to 2
	records, _ := store.List(ListOpts{Limit: 2})
	if len(records) != 2 {
		t.Fatalf("expected 2 records with limit, got %d", len(records))
	}

	// Offset 2, limit 2
	records, _ = store.List(ListOpts{Limit: 2, Offset: 2})
	if len(records) != 2 {
		t.Fatalf("expected 2 records with offset, got %d", len(records))
	}

	// Offset past end
	records, _ = store.List(ListOpts{Limit: 10, Offset: 100})
	if len(records) != 0 {
		t.Fatalf("expected 0 records past offset, got %d", len(records))
	}
}

func TestFileStore_Retention(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir, 1024) // 1KB cap — forces eviction

	for i := 0; i < 5; i++ {
		r := newTestRecord("default.hello", "default", "hello", domain.ExecSuccess, 0)
		time.Sleep(time.Millisecond * 10)
		store.Record(r)
	}

	// Size cap should have evicted some — fewer than 5 records remain.
	entries, _ := os.ReadDir(dir)
	jsonCount := 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".json" {
			jsonCount++
		}
	}
	if jsonCount >= 5 {
		t.Errorf("expected eviction to reduce records below 5, got %d", jsonCount)
	}
}

func TestFileStore_Delete(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir, 0)

	rec := newTestRecord("default.hello", "default", "hello", domain.ExecSuccess, 0)
	store.Record(rec)

	if err := store.Delete(rec.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := store.Get(rec.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestFileStore_Delete_NotFound(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir, 0)

	err := store.Delete("nonexistent")
	if err == nil {
		t.Error("expected error for missing record")
	}
}

func TestFileStore_List_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir, 0)

	records, err := store.List(ListOpts{Limit: 10})
	if err != nil {
		t.Fatalf("List on empty dir: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}
