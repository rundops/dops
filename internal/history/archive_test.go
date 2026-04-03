package history

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestAppendToActiveArchive_NewArchive(t *testing.T) {
	dir := t.TempDir()

	archivePath, err := AppendToActiveArchive(dir, "abc123.log", []byte("hello world\n"))
	if err != nil {
		t.Fatalf("AppendToActiveArchive: %v", err)
	}
	if archivePath == "" {
		t.Fatal("archive path should not be empty")
	}

	data, err := ReadFromArchive(archivePath, "abc123.log")
	if err != nil {
		t.Fatalf("ReadFromArchive: %v", err)
	}
	if string(data) != "hello world\n" {
		t.Errorf("content = %q", string(data))
	}
}

func TestAppendToActiveArchive_ReusesSameArchive(t *testing.T) {
	dir := t.TempDir()

	path1, _ := AppendToActiveArchive(dir, "first.log", []byte("first\n"))
	path2, _ := AppendToActiveArchive(dir, "second.log", []byte("second\n"))

	if path1 != path2 {
		t.Errorf("should reuse same archive: %q vs %q", path1, path2)
	}

	first, _ := ReadFromArchive(path1, "first.log")
	second, _ := ReadFromArchive(path2, "second.log")
	if string(first) != "first\n" || string(second) != "second\n" {
		t.Error("both entries should be readable")
	}
}

func TestAppendToActiveArchive_ArchiveIsUUIDNamed(t *testing.T) {
	dir := t.TempDir()

	path, _ := AppendToActiveArchive(dir, "test.log", []byte("data"))
	name := filepath.Base(path)

	if len(name) < 40 { // uuid (36 chars) + .log.gz (7 chars)
		t.Errorf("archive name too short: %q", name)
	}
	if filepath.Ext(name) != ".gz" {
		t.Errorf("expected .log.gz extension, got %q", name)
	}
}

func TestReadFromArchive_MissingEntry(t *testing.T) {
	dir := t.TempDir()
	path, _ := AppendToActiveArchive(dir, "exists.log", []byte("data"))

	_, err := ReadFromArchive(path, "missing.log")
	if err == nil {
		t.Error("expected error for missing entry")
	}
}

func TestReadFromArchive_MissingArchive(t *testing.T) {
	_, err := ReadFromArchive("/nonexistent/archive.log.gz", "entry.log")
	if err == nil {
		t.Error("expected error for missing archive")
	}
}

func TestAppendToActiveArchive_AtomicWrite(t *testing.T) {
	dir := t.TempDir()

	AppendToActiveArchive(dir, "first.log", []byte("first"))
	path, _ := AppendToActiveArchive(dir, "second.log", []byte("second"))

	tmpPath := path + ".tmp"
	if _, err := os.Stat(tmpPath); err == nil {
		t.Error("temp file should not exist after successful write")
	}
}

func TestAppendToActiveArchive_ManyEntries(t *testing.T) {
	dir := t.TempDir()

	var archivePath string
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("entry-%d.log", i)
		content := fmt.Sprintf("content %d\n", i)
		path, err := AppendToActiveArchive(dir, name, []byte(content))
		if err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
		archivePath = path
	}

	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("entry-%d.log", i)
		data, err := ReadFromArchive(archivePath, name)
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		expected := fmt.Sprintf("content %d\n", i)
		if string(data) != expected {
			t.Errorf("entry %d = %q, want %q", i, string(data), expected)
		}
	}
}
