package history

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// archiveEntry holds a single entry read from a tar.gz archive.
type archiveEntry struct {
	name string
	data []byte
}

// AppendToArchive adds an entry to a tar.gz archive. If the archive doesn't
// exist, it creates a new one. If it exists, it reads all entries, adds the
// new one, and writes a new archive atomically (temp file + rename).
func AppendToArchive(archivePath, entryName string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o755); err != nil {
		return fmt.Errorf("create archive dir: %w", err)
	}

	// Read existing entries if archive exists.
	var entries []archiveEntry
	if _, err := os.Stat(archivePath); err == nil {
		existing, err := readAllEntries(archivePath)
		if err != nil {
			return fmt.Errorf("read existing archive: %w", err)
		}
		entries = existing
	}

	// Add the new entry.
	entries = append(entries, archiveEntry{name: entryName, data: content})

	// Write atomically: temp file → rename.
	tmpPath := archivePath + ".tmp"
	if err := writeArchive(tmpPath, entries); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write archive: %w", err)
	}

	return os.Rename(tmpPath, archivePath)
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

	return os.WriteFile(path, buf.Bytes(), 0o644)
}
