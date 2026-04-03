package history

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"dops/internal/domain"
)

const defaultMaxBytes int64 = 50 * 1024 * 1024 // 50MB

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
	Get(id string) (*domain.ExecutionRecord, error)
	List(opts ListOpts) ([]*domain.ExecutionRecord, error)
	Delete(id string) error
}

// FileExecutionStore stores execution records as JSON files.
// Evicts oldest records when total directory size exceeds maxBytes.
type FileExecutionStore struct {
	dir      string
	maxBytes int64
}

// NewFileStore creates a store backed by the given directory.
// maxBytes controls the size cap; 0 uses the default (50MB).
func NewFileStore(dir string, maxBytes int64) *FileExecutionStore {
	if maxBytes <= 0 {
		maxBytes = defaultMaxBytes
	}
	return &FileExecutionStore{dir: dir, maxBytes: maxBytes}
}

// Record writes an execution record to disk and enforces the size cap.
func (s *FileExecutionStore) Record(record *domain.ExecutionRecord) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("create history dir: %w", err)
	}

	s.persistLog(record)

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal record: %w", err)
	}

	filename := s.filename(record)
	path := filepath.Join(s.dir, filename)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write record: %w", err)
	}

	s.enforceSize()
	return nil
}

// Get retrieves a single execution record by ID.
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

// List returns execution records matching the filter, sorted newest first.
func (s *FileExecutionStore) List(opts ListOpts) ([]*domain.ExecutionRecord, error) {
	files, err := s.listFiles()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// Reverse for newest-first (files are sorted oldest-first by name).
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

// Delete removes an execution record and its log by ID.
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
			s.removeLog(rec)
			return os.Remove(filepath.Join(s.dir, f))
		}
	}
	return fmt.Errorf("execution %q not found", id)
}

// --- Log persistence ---

// persistLog moves a log file from a temporary directory to persistent storage.
// Logs stored as {uuid}/datetime.log under the logs/ subdirectory.
func (s *FileExecutionStore) persistLog(record *domain.ExecutionRecord) {
	if record.LogPath == "" {
		return
	}
	if strings.HasPrefix(record.LogPath, s.dir) {
		return
	}

	execDir := filepath.Join(s.dir, "logs", record.ID)
	if err := os.MkdirAll(execDir, 0o755); err != nil {
		return
	}
	destPath := filepath.Join(execDir, record.StartTime.Format("2006-01-02T15-04-05")+".log")

	if err := copyFile(record.LogPath, destPath); err != nil {
		return
	}

	_ = os.Remove(record.LogPath)
	record.LogPath = destPath
}

func (s *FileExecutionStore) removeLog(rec *domain.ExecutionRecord) {
	if rec.LogPath != "" && strings.HasPrefix(rec.LogPath, s.dir) {
		_ = os.RemoveAll(filepath.Dir(rec.LogPath))
	}
}

// ReadLog reads a log file. Returns the lines and true if available.
func ReadLog(path string) ([]string, bool) {
	if path == "" {
		return nil, false
	}

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

// --- Size-based eviction ---

// enforceSize deletes oldest records (and their logs) until total
// directory size is under maxBytes.
func (s *FileExecutionStore) enforceSize() {
	size := dirSize(s.dir)
	if size <= s.maxBytes {
		return
	}

	files, err := s.listFiles()
	if err != nil {
		return
	}

	// Delete oldest first until under budget.
	for _, f := range files {
		if dirSize(s.dir) <= s.maxBytes {
			break
		}
		rec, err := s.loadRecord(f)
		if err == nil {
			s.removeLog(rec)
		}
		_ = os.Remove(filepath.Join(s.dir, f))
	}
}

// dirSize returns the total size in bytes of all files under dir (recursive).
func dirSize(dir string) int64 {
	var total int64
	_ = filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		total += info.Size()
		return nil
	})
	return total
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
	data, err := os.ReadFile(filepath.Join(s.dir, filename))
	if err != nil {
		return nil, err
	}
	var rec domain.ExecutionRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src) // #nosec G304 -- src is internal log path
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

var _ ExecutionStore = (*FileExecutionStore)(nil)
