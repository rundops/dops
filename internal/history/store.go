package history

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"dops/internal/domain"
)

const defaultMaxRecords = 500
const defaultArchiveAfterDays = 7

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
type FileExecutionStore struct {
	dir             string
	maxRecords      int
	archiveAfterDays int
}

// NewFileStore creates a store backed by the given directory.
func NewFileStore(dir string, maxRecords int) *FileExecutionStore {
	if maxRecords <= 0 {
		maxRecords = defaultMaxRecords
	}
	return &FileExecutionStore{
		dir:              dir,
		maxRecords:       maxRecords,
		archiveAfterDays: defaultArchiveAfterDays,
	}
}

// SetArchiveDays configures how many days before logs are compressed.
func (s *FileExecutionStore) SetArchiveDays(days int) {
	if days > 0 {
		s.archiveAfterDays = days
	}
}

// Record writes an execution record to disk and enforces retention.
// If the record has a log file in a temporary directory, it is copied
// to the persistent logs/ subdirectory so it survives temp cleanup.
func (s *FileExecutionStore) Record(record *domain.ExecutionRecord) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("create history dir: %w", err)
	}

	// Copy log file to persistent storage if it's in a temp dir.
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

	s.enforceRetention()
	s.archiveOldLogs()
	return nil
}

// archiveOldLogs compresses log files older than archiveAfterDays.
func (s *FileExecutionStore) archiveOldLogs() {
	cutoff := time.Now().AddDate(0, 0, -s.archiveAfterDays)

	files, err := s.listFiles()
	if err != nil {
		return
	}

	for _, f := range files {
		rec, err := s.loadRecord(f)
		if err != nil || rec.LogPath == "" {
			continue
		}
		// Skip already compressed.
		if strings.HasSuffix(rec.LogPath, ".gz") {
			continue
		}
		// Skip recent.
		if rec.StartTime.After(cutoff) {
			continue
		}
		// Compress the log file.
		gzPath := rec.LogPath + ".gz"
		if err := compressFile(rec.LogPath, gzPath); err != nil {
			continue
		}
		_ = os.Remove(rec.LogPath)

		// Update the record's log path.
		rec.LogPath = gzPath
		data, err := json.MarshalIndent(rec, "", "  ")
		if err != nil {
			continue
		}
		_ = os.WriteFile(filepath.Join(s.dir, f), data, 0o644)
	}
}

// persistLog moves a log file from a temporary directory to ~/.dops/history/logs/
// using the execution ID as filename (no runbook names in the cache).
// Deletes the temp file after copying. No-op if LogPath is empty or already persistent.
func (s *FileExecutionStore) persistLog(record *domain.ExecutionRecord) {
	if record.LogPath == "" {
		return
	}
	// Skip if already under our directory.
	if strings.HasPrefix(record.LogPath, s.dir) {
		return
	}

	logsDir := filepath.Join(s.dir, "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return
	}

	execDir := filepath.Join(logsDir, record.ID)
	if err := os.MkdirAll(execDir, 0o755); err != nil {
		return
	}
	destPath := filepath.Join(execDir, record.StartTime.Format("2006-01-02T15-04-05")+".log")

	if err := copyFile(record.LogPath, destPath); err != nil {
		return // best-effort — keep original path
	}

	// Remove temp file now that we have a persistent copy.
	_ = os.Remove(record.LogPath)
	record.LogPath = destPath
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

func compressFile(src, dst string) error {
	in, err := os.Open(src) // #nosec G304 -- src is internal log path from execution
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	gz := gzip.NewWriter(out)
	if _, err := io.Copy(gz, in); err != nil {
		gz.Close()
		return err
	}
	return gz.Close()
}

// ReadLog reads a log file, transparently decompressing .gz files.
// Returns the lines and true if the file was available.
func ReadLog(path string) ([]string, bool) {
	if path == "" {
		return nil, false
	}

	f, err := os.Open(path) // #nosec G304 -- path from internal history record
	if err != nil {
		return nil, false
	}
	defer f.Close()

	var reader io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return nil, false
		}
		defer gz.Close()
		reader = gz
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, false
	}

	content := strings.TrimRight(string(data), "\n")
	if content == "" {
		return []string{}, true
	}
	return strings.Split(content, "\n"), true
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

	// Files are sorted oldest-first by name; reverse for newest-first.
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

		// Apply filters.
		if opts.RunbookID != "" && rec.RunbookID != opts.RunbookID {
			continue
		}
		if opts.Status != "" && rec.Status != opts.Status {
			continue
		}

		// Apply offset.
		if skipped < opts.Offset {
			skipped++
			continue
		}

		records = append(records, rec)

		// Apply limit.
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

// Delete removes an execution record by ID.
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

func (s *FileExecutionStore) filename(r *domain.ExecutionRecord) string {
	ts := r.StartTime.Format("2006-01-02T15-04-05.000")
	// Sanitize runbook ID for filename.
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
	sort.Strings(files) // lexicographic = chronological (timestamp prefix)
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

func (s *FileExecutionStore) enforceRetention() {
	files, err := s.listFiles()
	if err != nil || len(files) <= s.maxRecords {
		return
	}

	// Delete oldest files (sorted oldest-first) and their log files.
	excess := len(files) - s.maxRecords
	for i := 0; i < excess; i++ {
		rec, err := s.loadRecord(files[i])
		if err == nil && rec.LogPath != "" && strings.HasPrefix(rec.LogPath, s.dir) {
			_ = os.RemoveAll(filepath.Dir(rec.LogPath)) // remove UUID directory + log
		}
		_ = os.Remove(filepath.Join(s.dir, files[i]))
	}
}

var _ ExecutionStore = (*FileExecutionStore)(nil)
