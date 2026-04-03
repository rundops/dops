package history

import (
	"dops/internal/domain"
	"os"
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

	if !strings.Contains(rec.LogPath, "#") {
		t.Fatalf("LogPath = %q, expected archive#entry format", rec.LogPath)
	}
	if !strings.HasSuffix(rec.LogPath, rec.ID+".log") {
		t.Errorf("LogPath should end with %s.log", rec.ID)
	}
	if !strings.Contains(rec.LogPath, ".log.gz#") {
		t.Errorf("LogPath should contain .log.gz#, got %q", rec.LogPath)
	}

	got, ok := ReadLog(rec.LogPath)
	if !ok {
		t.Fatal("ReadLog should return true")
	}
	if len(got) != 3 || got[0] != "line 1" {
		t.Errorf("lines = %v", got)
	}
}

func TestFileStore_ArchiveLog_MultipleShareSameArchive(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir, 0)

	rec1 := domain.NewExecutionRecord("default.first", "first", "default", domain.ExecCLI)
	rec1.Complete(0, 1, "first done")
	store.ArchiveLog(rec1, []string{"first output"})

	rec2 := domain.NewExecutionRecord("default.second", "second", "default", domain.ExecCLI)
	rec2.Complete(0, 1, "second done")
	store.ArchiveLog(rec2, []string{"second output"})

	// Both readable.
	lines1, ok1 := ReadLog(rec1.LogPath)
	lines2, ok2 := ReadLog(rec2.LogPath)
	if !ok1 || !ok2 {
		t.Fatal("both logs should be readable")
	}
	if lines1[0] != "first output" || lines2[0] != "second output" {
		t.Errorf("first=%q second=%q", lines1[0], lines2[0])
	}

	// Same archive file.
	archive1 := strings.Split(rec1.LogPath, "#")[0]
	archive2 := strings.Split(rec2.LogPath, "#")[0]
	if archive1 != archive2 {
		t.Errorf("expected same archive: %q vs %q", archive1, archive2)
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

func TestFileStore_List_NewestFirst(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir, 0)

	r1 := newTestRecord("default.first", "default", "first", domain.ExecSuccess, 0)
	store.Record(r1)
	time.Sleep(time.Millisecond * 10)
	r2 := newTestRecord("default.second", "default", "second", domain.ExecSuccess, 0)
	store.Record(r2)

	records, _ := store.List(ListOpts{Limit: 10})
	if len(records) != 2 || records[0].RunbookID != "default.second" {
		t.Error("should be newest first")
	}
}

func TestFileStore_List_FilterByRunbook(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir, 0)

	store.Record(newTestRecord("default.hello", "default", "hello", domain.ExecSuccess, 0))
	store.Record(newTestRecord("infra.deploy", "infra", "deploy", domain.ExecSuccess, 0))

	records, _ := store.List(ListOpts{RunbookID: "infra.deploy", Limit: 10})
	if len(records) != 1 || records[0].RunbookID != "infra.deploy" {
		t.Error("filter failed")
	}
}

func TestFileStore_List_FilterByStatus(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir, 0)

	store.Record(newTestRecord("default.hello", "default", "hello", domain.ExecSuccess, 0))
	store.Record(newTestRecord("default.fail", "default", "fail", domain.ExecFailed, 1))

	records, _ := store.List(ListOpts{Status: domain.ExecFailed, Limit: 10})
	if len(records) != 1 || records[0].Status != domain.ExecFailed {
		t.Error("status filter failed")
	}
}

func TestFileStore_Delete(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir, 0)

	rec := newTestRecord("default.hello", "default", "hello", domain.ExecSuccess, 0)
	store.Record(rec)
	store.Delete(rec.ID)

	_, err := store.Get(rec.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestReadLog_PlainFile_BackwardCompat(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/test.log"
	os.WriteFile(path, []byte("plain\n"), 0o644)

	lines, ok := ReadLog(path)
	if !ok || len(lines) != 1 || lines[0] != "plain" {
		t.Errorf("plain file read failed: %v %v", ok, lines)
	}
}

func TestReadLog_EmptyPath(t *testing.T) {
	_, ok := ReadLog("")
	if ok {
		t.Error("empty path should return false")
	}
}
