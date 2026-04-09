package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"dops/internal/domain"
)

// ListOpts controls filtering and pagination for List.
type ListOpts struct {
	RunbookID string
	Status    domain.ExecStatus
	Limit     int
	Offset    int
}

// ExecutionStore persists and queries execution records.
type ExecutionStore interface {
	Record(record *domain.ExecutionRecord) error
	ArchiveLog(record *domain.ExecutionRecord, lines []string) error
	Get(id string) (*domain.ExecutionRecord, error)
	List(opts ListOpts) ([]*domain.ExecutionRecord, error)
	Delete(id string) error
}

// FileExecutionStore stores execution records as JSON files and logs
// in size-capped .log.gz archives.
//
// Each execution's output is appended to the active archive (< 10MB).
// When the active archive reaches 10MB, a new one is created.
type FileExecutionStore struct {
	dir string
}

// NewFileStore creates a store backed by the given directory.
func NewFileStore(dir string, _ int64) *FileExecutionStore {
	return &FileExecutionStore{dir: dir}
}

// Record writes an execution record to disk.
func (s *FileExecutionStore) Record(record *domain.ExecutionRecord) error {
	if err := os.MkdirAll(s.dir, 0o750); err != nil {
		return fmt.Errorf("create history dir: %w", err)
	}

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal record: %w", err)
	}

	filename := s.filename(record)
	path := filepath.Join(s.dir, filename)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write record: %w", err)
	}

	return nil
}

// ArchiveLog writes execution log lines into the active .log.gz archive
// and updates the record's LogPath.
func (s *FileExecutionStore) ArchiveLog(record *domain.ExecutionRecord, lines []string) error {
	if len(lines) == 0 {
		return nil
	}

	logsDir := filepath.Join(s.dir, "logs")
	entryName := record.ID + ".log"
	content := []byte(strings.Join(lines, "\n") + "\n")

	archivePath, err := AppendToActiveArchive(logsDir, entryName, content)
	if err != nil {
		return fmt.Errorf("archive log: %w", err)
	}

	record.LogPath = archivePath + "#" + entryName
	return nil
}

// ReadLog reads a log, transparently handling:
//   - Archive paths: "path/to/archive.log.gz#entry.log"
//   - Plain files: "path/to/file.log" (backward compat)
func ReadLog(path string) ([]string, bool) {
	if path == "" {
		return nil, false
	}

	if idx := strings.Index(path, "#"); idx > 0 {
		archivePath := path[:idx]
		entryName := path[idx+1:]
		data, err := ReadFromArchive(archivePath, entryName)
		if err != nil {
			return nil, false
		}
		content := strings.TrimRight(string(data), "\n")
		if content == "" {
			return []string{}, true
		}
		return strings.Split(content, "\n"), true
	}

	// Plain file fallback.
	data, err := os.ReadFile(path) // #nosec G304 -- path from internal history record
	if err != nil {
		return nil, false
	}
	content := strings.TrimRight(string(data), "\n")
	if content == "" {
		return []string{}, true
	}
	return strings.Split(content, "\n"), true
}

// --- Query methods ---

func (s *FileExecutionStore) Get(id string) (*domain.ExecutionRecord, error) {
	files, err := s.listFiles()
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		rec, err := s.loadRecord(f)
		if err != nil {
			continue
		}
		if rec.ID == id {
			return rec, nil
		}
	}
	return nil, fmt.Errorf("execution %q not found", id)
}

func (s *FileExecutionStore) List(opts ListOpts) ([]*domain.ExecutionRecord, error) {
	files, err := s.listFiles()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// Reverse for newest-first.
	for i, j := 0, len(files)-1; i < j; i, j = i+1, j-1 {
		files[i], files[j] = files[j], files[i]
	}

	var records []*domain.ExecutionRecord
	skipped := 0
	for _, f := range files {
		rec, err := s.loadRecord(f)
		if err != nil {
			continue
		}
		if opts.RunbookID != "" && rec.RunbookID != opts.RunbookID {
			continue
		}
		if opts.Status != "" && rec.Status != opts.Status {
			continue
		}
		if skipped < opts.Offset {
			skipped++
			continue
		}
		records = append(records, rec)
		limit := opts.Limit
		if limit <= 0 {
			limit = 50
		}
		if len(records) >= limit {
			break
		}
	}
	return records, nil
}

func (s *FileExecutionStore) Delete(id string) error {
	files, err := s.listFiles()
	if err != nil {
		return err
	}
	for _, f := range files {
		rec, err := s.loadRecord(f)
		if err != nil {
			continue
		}
		if rec.ID == id {
			return os.Remove(filepath.Join(s.dir, f))
		}
	}
	return fmt.Errorf("execution %q not found", id)
}

// --- Helpers ---

func (s *FileExecutionStore) filename(r *domain.ExecutionRecord) string {
	ts := r.StartTime.Format("2006-01-02T15-04-05.000")
	rbID := strings.ReplaceAll(r.RunbookID, "/", "-")
	return fmt.Sprintf("%s-%s-%s.json", ts, rbID, r.ID)
}

func (s *FileExecutionStore) listFiles() ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		files = append(files, e.Name())
	}
	sort.Strings(files)
	return files, nil
}

func (s *FileExecutionStore) loadRecord(filename string) (*domain.ExecutionRecord, error) {
	// filename comes from listRecordFilenames which reads s.dir via os.ReadDir
	// and filters to *.json entries; strip any path component defensively.
	data, err := os.ReadFile(filepath.Join(s.dir, filepath.Base(filename))) // #nosec G304 -- constrained to s.dir
	if err != nil {
		return nil, err
	}
	var rec domain.ExecutionRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

var _ ExecutionStore = (*FileExecutionStore)(nil)
