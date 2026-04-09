package history

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxArchiveBytes int64 = 10 * 1024 * 1024 // 10MB

// archiveEntry holds a single entry read from a tar.gz archive.
type archiveEntry struct {
	name string
	data []byte
}

// AppendToActiveArchive finds or creates the active .log.gz archive (< 10MB)
// and appends the entry. Returns the archive path and entry name.
func AppendToActiveArchive(logsDir, entryName string, content []byte) (archivePath string, err error) {
	if err := os.MkdirAll(logsDir, 0o750); err != nil {
		return "", fmt.Errorf("create logs dir: %w", err)
	}

	archivePath = findActiveArchive(logsDir)
	if archivePath == "" {
		archivePath = filepath.Join(logsDir, generateArchiveID()+".log.gz")
	}

	return archivePath, appendToArchive(archivePath, entryName, content)
}

// ReadFromArchive reads a single entry from a tar.gz archive by name.
func ReadFromArchive(archivePath, entryName string) ([]byte, error) {
	f, err := os.Open(archivePath) // #nosec G304 -- path from internal history
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("gzip reader: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar next: %w", err)
		}
		if header.Name == entryName {
			data, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("read entry: %w", err)
			}
			return data, nil
		}
	}

	return nil, fmt.Errorf("entry %q not found in archive", entryName)
}

// findActiveArchive returns the newest .log.gz under maxArchiveBytes, or "".
func findActiveArchive(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	var archives []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".log.gz") {
			archives = append(archives, e.Name())
		}
	}
	if len(archives) == 0 {
		return ""
	}

	// Sort by mod time, newest first.
	sort.Slice(archives, func(i, j int) bool {
		infoI, _ := os.Stat(filepath.Join(dir, archives[i]))
		infoJ, _ := os.Stat(filepath.Join(dir, archives[j]))
		if infoI == nil || infoJ == nil {
			return false
		}
		return infoI.ModTime().After(infoJ.ModTime())
	})

	// Return newest if under size limit.
	newest := filepath.Join(dir, archives[0])
	info, err := os.Stat(newest)
	if err != nil || info.Size() >= maxArchiveBytes {
		return ""
	}
	return newest
}

func appendToArchive(archivePath, entryName string, content []byte) error {
	// Read existing entries if archive exists.
	var entries []archiveEntry
	if _, err := os.Stat(archivePath); err == nil {
		existing, err := readAllEntries(archivePath)
		if err != nil {
			return fmt.Errorf("read existing archive: %w", err)
		}
		entries = existing
	}

	entries = append(entries, archiveEntry{name: entryName, data: content})

	// Write atomically.
	tmpPath := archivePath + ".tmp"
	if err := writeArchive(tmpPath, entries); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write archive: %w", err)
	}

	return os.Rename(tmpPath, archivePath)
}

func readAllEntries(archivePath string) ([]archiveEntry, error) {
	f, err := os.Open(archivePath) // #nosec G304 -- path from internal history
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	var entries []archiveEntry
	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, err
		}
		entries = append(entries, archiveEntry{name: header.Name, data: data})
	}

	return entries, nil
}

func writeArchive(path string, entries []archiveEntry) error {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for _, e := range entries {
		header := &tar.Header{
			Name: e.name,
			Size: int64(len(e.data)),
			Mode: 0o644,
		}
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if _, err := tw.Write(e.data); err != nil {
			return err
		}
	}

	if err := tw.Close(); err != nil {
		return err
	}
	if err := gw.Close(); err != nil {
		return err
	}

	return os.WriteFile(path, buf.Bytes(), 0o600)
}

func generateArchiveID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
