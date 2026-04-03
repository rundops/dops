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

const (
	defaultMaxBytes      int64 = 50 * 1024 * 1024 // 50MB size cap
	defaultCompressAfter       = 7                  // days before gzip
	defaultExpireAfter         = 90                 // days before deletion
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
	Get(id string) (*domain.ExecutionRecord, error)
	List(opts ListOpts) ([]*domain.ExecutionRecord, error)
	Delete(id string) error
}

// FileExecutionStore stores execution records as JSON files.
//
// Lifecycle:
//   - Fresh (< 7 days): plain .log files
//   - Compressed (7–90 days): .log.gz — still queryable
//   - Expired (> 90 days): deleted entirely
//   - Size cap (50MB): oldest deleted regardless of age
type FileExecutionStore struct {
	dir           string
	maxBytes      int64
	compressAfter int // days
	expireAfter   int // days
}

// NewFileStore creates a store backed by the given directory.
func NewFileStore(dir string, maxBytes int64) *FileExecutionStore {
	if maxBytes <= 0 {
		maxBytes = defaultMaxBytes
	}
	return &FileExecutionStore{
		dir:           dir,
		maxBytes:      maxBytes,
		compressAfter: defaultCompressAfter,
		expireAfter:   defaultExpireAfter,
	}
}

// Record writes an execution record to disk and runs maintenance.
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

	s.maintain()
	return nil
}

// maintain runs all retention passes: expire, compress, size cap.
func (s *FileExecutionStore) maintain() {
	s.expireOld()
	s.compressOld()
	s.enforceSize()
}

// expireOld deletes only compressed logs that haven't been modified in
// expireAfter days. Uncompressed logs are left alone — they'll get
// compressed first by compressOld, then expired on a future pass.
func (s *FileExecutionStore) expireOld() {
	cutoff := time.Now().AddDate(0, 0, -s.expireAfter)

	files, err := s.listFiles()
	if err != nil {
		return
	}

	for _, f := range files {
		rec, err := s.loadRecord(f)
		if err != nil {
			continue
		}
		// Only expire compressed logs past the cutoff.
		if rec.LogPath == "" || !strings.HasSuffix(rec.LogPath, ".gz") {
			continue
		}
		info, err := os.Stat(rec.LogPath)
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			s.removeLog(rec)
			_ = os.Remove(filepath.Join(s.dir, f))
		}
	}
}

// compressOld gzips log files older than compressAfter days.
func (s *FileExecutionStore) compressOld() {
	cutoff := time.Now().AddDate(0, 0, -s.compressAfter)

	files, err := s.listFiles()
	if err != nil {
		return
	}

	for _, f := range files {
		rec, err := s.loadRecord(f)
		if err != nil || rec.LogPath == "" {
			continue
		}
		// Skip already compressed or not yet old enough.
		if strings.HasSuffix(rec.LogPath, ".gz") || rec.StartTime.After(cutoff) {
			continue
		}

		gzPath := rec.LogPath + ".gz"
		if err := compressFile(rec.LogPath, gzPath); err != nil {
			continue
		}
		_ = os.Remove(rec.LogPath)

		// Update record with new path.
		rec.LogPath = gzPath
		data, _ := json.MarshalIndent(rec, "", "  ")
		_ = os.WriteFile(filepath.Join(s.dir, f), data, 0o644)
	}
}

// enforceSize deletes oldest records until under maxBytes.
func (s *FileExecutionStore) enforceSize() {
	if dirSize(s.dir) <= s.maxBytes {
		return
	}

	files, err := s.listFiles()
	if err != nil {
		return
	}

	for _, f := range files {
		if dirSize(s.dir) <= s.maxBytes {
			break
		}
		rec, _ := s.loadRecord(f)
		if rec != nil {
			s.removeLog(rec)
		}
		_ = os.Remove(filepath.Join(s.dir, f))
	}
}

// --- Log persistence ---

func (s *FileExecutionStore) persistLog(record *domain.ExecutionRecord) {
	if record.LogPath == "" || strings.HasPrefix(record.LogPath, s.dir) {
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

// ReadLog reads a log file, transparently decompressing .gz files.
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
			s.removeLog(rec)
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

	gz := gzip.NewWriter(out)
	if _, err := io.Copy(gz, in); err != nil {
		gz.Close()
		return err
	}
	return gz.Close()
}

var _ ExecutionStore = (*FileExecutionStore)(nil)
