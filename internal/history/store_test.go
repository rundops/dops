package history

import (
	"dops/internal/domain"
	"os"
	"path/filepath"
	"strings"
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
}

func TestFileStore_ArchiveLog(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir, 0)

	rec := domain.NewExecutionRecord("default.hello", "hello", "default", domain.ExecCLI)
	rec.Complete(0, 3, "done")

	lines := []string{"line 1", "line 2", "line 3"}
	if err := store.ArchiveLog(rec, lines); err != nil {
		t.Fatalf("ArchiveLog: %v", err)
	}

	// LogPath should be archive#entry format.
	if !strings.Contains(rec.LogPath, "#") {
		t.Fatalf("LogPath = %q, expected archive#entry format", rec.LogPath)
	}
	if !strings.HasSuffix(rec.LogPath, rec.ID+".log") {
		t.Errorf("LogPath should end with %s.log, got %q", rec.ID, rec.LogPath)
	}

	// ReadLog should return the lines.
	got, ok := ReadLog(rec.LogPath)
	if !ok {
		t.Fatal("ReadLog should return true")
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(got))
	}
	if got[0] != "line 1" {
		t.Errorf("line 0 = %q", got[0])
	}
}

func TestFileStore_ArchiveLog_MultipleInSameDay(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir, 0)

	rec1 := domain.NewExecutionRecord("default.first", "first", "default", domain.ExecCLI)
	rec1.Complete(0, 1, "first done")
	store.ArchiveLog(rec1, []string{"first output"})

	rec2 := domain.NewExecutionRecord("default.second", "second", "default", domain.ExecCLI)
	rec2.Complete(0, 1, "second done")
	store.ArchiveLog(rec2, []string{"second output"})

	// Both should be readable.
	lines1, ok1 := ReadLog(rec1.LogPath)
	lines2, ok2 := ReadLog(rec2.LogPath)

	if !ok1 || !ok2 {
		t.Fatal("both logs should be readable")
	}
	if lines1[0] != "first output" {
		t.Errorf("first = %q", lines1[0])
	}
	if lines2[0] != "second output" {
		t.Errorf("second = %q", lines2[0])
	}

	// Should be in the same archive file.
	archive1 := strings.Split(rec1.LogPath, "#")[0]
	archive2 := strings.Split(rec2.LogPath, "#")[0]
	if archive1 != archive2 {
		t.Errorf("expected same archive, got %q and %q", archive1, archive2)
	}
}

func TestFileStore_ArchiveLog_EmptyLines(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir, 0)

	rec := domain.NewExecutionRecord("default.hello", "hello", "default", domain.ExecCLI)
	rec.Complete(0, 0, "")

	if err := store.ArchiveLog(rec, nil); err != nil {
		t.Fatalf("ArchiveLog with nil: %v", err)
	}
	if rec.LogPath != "" {
		t.Error("empty lines should not set LogPath")
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

	r1 := domain.NewExecutionRecord("default.first", "first", "default", domain.ExecCLI)
	r1.Complete(0, 1, "first")
	store.Record(r1)

	time.Sleep(time.Millisecond * 10)

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
}

func TestFileStore_List_FilterByRunbook(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir, 0)

	r1 := newTestRecord("default.hello", "default", "hello", domain.ExecSuccess, 0)
	r2 := newTestRecord("infra.deploy", "infra", "deploy", domain.ExecSuccess, 0)
	store.Record(r1)
	store.Record(r2)

	records, _ := store.List(ListOpts{RunbookID: "infra.deploy", Limit: 10})
	if len(records) != 1 || records[0].RunbookID != "infra.deploy" {
		t.Errorf("filter failed: got %d records", len(records))
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
	if len(records) != 1 || records[0].Status != domain.ExecFailed {
		t.Errorf("status filter failed")
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

func TestFileStore_SizeEviction(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir, 1024) // 1KB cap

	for i := 0; i < 5; i++ {
		r := newTestRecord("default.hello", "default", "hello", domain.ExecSuccess, 0)
		time.Sleep(time.Millisecond * 10)
		store.Record(r)
		store.ArchiveLog(r, []string{"some output line that takes up space"})
	}

	// Should have evicted some.
	entries, _ := os.ReadDir(dir)
	jsonCount := 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".json" {
			jsonCount++
		}
	}
	if jsonCount >= 5 {
		t.Errorf("expected eviction, got %d records", jsonCount)
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

func TestReadLog_PlainFile_BackwardCompat(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")
	os.WriteFile(logPath, []byte("plain line 1\nplain line 2\n"), 0o644)

	lines, ok := ReadLog(logPath)
	if !ok {
		t.Fatal("should read plain file")
	}
	if len(lines) != 2 || lines[0] != "plain line 1" {
		t.Errorf("lines = %v", lines)
	}
}

func TestReadLog_EmptyPath(t *testing.T) {
	_, ok := ReadLog("")
	if ok {
		t.Error("empty path should return false")
	}
}
