package history

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestAppendToArchive_NewArchive(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "2026-04-03.tar.gz")

	err := AppendToArchive(archivePath, "abc123.log", []byte("hello world\n"))
	if err != nil {
		t.Fatalf("AppendToArchive: %v", err)
	}

	if _, err := os.Stat(archivePath); err != nil {
		t.Fatal("archive should exist")
	}

	data, err := ReadFromArchive(archivePath, "abc123.log")
	if err != nil {
		t.Fatalf("ReadFromArchive: %v", err)
	}
	if string(data) != "hello world\n" {
		t.Errorf("content = %q, want %q", string(data), "hello world\n")
	}
}

func TestAppendToArchive_PreservesExisting(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "2026-04-03.tar.gz")

	AppendToArchive(archivePath, "first.log", []byte("first\n"))
	AppendToArchive(archivePath, "second.log", []byte("second\n"))

	// Both entries should be readable.
	first, err := ReadFromArchive(archivePath, "first.log")
	if err != nil {
		t.Fatalf("read first: %v", err)
	}
	if string(first) != "first\n" {
		t.Errorf("first = %q", string(first))
	}

	second, err := ReadFromArchive(archivePath, "second.log")
	if err != nil {
		t.Fatalf("read second: %v", err)
	}
	if string(second) != "second\n" {
		t.Errorf("second = %q", string(second))
	}
}

func TestReadFromArchive_MissingEntry(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "test.tar.gz")

	AppendToArchive(archivePath, "exists.log", []byte("data"))

	_, err := ReadFromArchive(archivePath, "missing.log")
	if err == nil {
		t.Error("expected error for missing entry")
	}
}

func TestReadFromArchive_MissingArchive(t *testing.T) {
	_, err := ReadFromArchive("/nonexistent/archive.tar.gz", "entry.log")
	if err == nil {
		t.Error("expected error for missing archive")
	}
}

func TestAppendToArchive_Atomic(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "test.tar.gz")

	// Write initial entry.
	AppendToArchive(archivePath, "first.log", []byte("first"))

	// Write second entry — should not leave .tmp file.
	AppendToArchive(archivePath, "second.log", []byte("second"))

	tmpPath := archivePath + ".tmp"
	if _, err := os.Stat(tmpPath); err == nil {
		t.Error("temp file should not exist after successful write")
	}
}

func TestAppendToArchive_MultipleEntries(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "test.tar.gz")

	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("entry-%d.log", i)
		content := fmt.Sprintf("content %d\n", i)
		if err := AppendToArchive(archivePath, name, []byte(content)); err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}

	// Verify all 10 entries.
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
