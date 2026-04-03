package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"dops/internal/domain"
)

const (
	defaultMaxBytes  int64 = 50 * 1024 * 1024 // 50MB size cap
	defaultExpireDays      = 90                // days before archive deletion
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
// in daily tar.gz archives.
//
// Lifecycle:
//   - On completion: log lines written to today's {date}.tar.gz as {uuid}.log entry
//   - After 90 days: archive deleted (unmodified for 90 days)
//   - Size cap: oldest archives deleted if total exceeds 50MB
type FileExecutionStore struct {
	dir        string
	maxBytes   int64
	expireDays int
}

// NewFileStore creates a store backed by the given directory.
func NewFileStore(dir string, maxBytes int64) *FileExecutionStore {
	if maxBytes <= 0 {
		maxBytes = defaultMaxBytes
	}
	return &FileExecutionStore{
		dir:        dir,
		maxBytes:   maxBytes,
		expireDays: defaultExpireDays,
	}
}

// Record writes an execution record to disk and runs maintenance.
func (s *FileExecutionStore) Record(record *domain.ExecutionRecord) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("create history dir: %w", err)
	}

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

// ArchiveLog writes execution log lines into today's daily tar.gz archive
// and updates the record's LogPath.
func (s *FileExecutionStore) ArchiveLog(record *domain.ExecutionRecord, lines []string) error {
	if len(lines) == 0 {
		return nil
	}

	logsDir := filepath.Join(s.dir, "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return fmt.Errorf("create logs dir: %w", err)
	}

	archiveName := record.StartTime.Format("2006-01-02") + ".tar.gz"
	archivePath := filepath.Join(logsDir, archiveName)
	entryName := record.ID + ".log"

	content := []byte(strings.Join(lines, "\n") + "\n")
	if err := AppendToArchive(archivePath, entryName, content); err != nil {
		return fmt.Errorf("archive log: %w", err)
	}

	record.LogPath = archivePath + "#" + entryName
	return nil
}

// ReadLog reads a log, transparently handling:
//   - Archive paths: "path/to/archive.tar.gz#entry.log"
//   - Plain files: "path/to/file.log" (backward compat)
func ReadLog(path string) ([]string, bool) {
	if path == "" {
		return nil, false
	}

	// Archive path: split on #
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

	// Plain file fallback (backward compat with old per-file logs).
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

// --- Maintenance ---

func (s *FileExecutionStore) maintain() {
	s.expireOld()
	s.enforceSize()
}

// expireOld deletes .tar.gz archives unmodified for > expireDays.
func (s *FileExecutionStore) expireOld() {
	logsDir := filepath.Join(s.dir, "logs")
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -s.expireDays)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".tar.gz") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(logsDir, e.Name()))
		}
	}

	// Also clean up old per-file log directories (backward compat migration).
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.RemoveAll(filepath.Join(logsDir, e.Name()))
		}
	}

	// Expire old JSON records whose logs are gone.
	files, err := s.listFiles()
	if err != nil {
		return
	}
	for _, f := range files {
		rec, err := s.loadRecord(f)
		if err != nil {
			continue
		}
		if rec.StartTime.Before(cutoff) {
			_ = os.Remove(filepath.Join(s.dir, f))
		}
	}
}

// enforceSize deletes oldest archives and records until under maxBytes.
func (s *FileExecutionStore) enforceSize() {
	if dirSize(s.dir) <= s.maxBytes {
		return
	}

	// Delete oldest archives first.
	logsDir := filepath.Join(s.dir, "logs")
	archives := listArchives(logsDir)
	for _, name := range archives {
		if dirSize(s.dir) <= s.maxBytes {
			break
		}
		_ = os.Remove(filepath.Join(logsDir, name))
	}

	// Then oldest records if still over.
	files, err := s.listFiles()
	if err != nil {
		return
	}
	for _, f := range files {
		if dirSize(s.dir) <= s.maxBytes {
			break
		}
		_ = os.Remove(filepath.Join(s.dir, f))
	}
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

func listArchives(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var archives []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".tar.gz") {
			archives = append(archives, e.Name())
		}
	}
	sort.Strings(archives) // oldest first (date prefix)
	return archives
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

var _ ExecutionStore = (*FileExecutionStore)(nil)
