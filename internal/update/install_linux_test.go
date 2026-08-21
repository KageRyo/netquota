//go:build linux

package update

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractLinuxBinary(t *testing.T) {
	archivePath := writeUpdateArchive(t, []testArchiveEntry{
		{name: "netquota-linux-amd64/README.md", content: []byte("docs")},
		{name: "netquota-linux-amd64/netquota", content: []byte("new binary")},
	})
	destination := filepath.Join(t.TempDir(), "netquota")
	if err := extractLinuxBinary(archivePath, destination); err != nil {
		t.Fatalf("extractLinuxBinary: %v", err)
	}
	content, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read extracted binary: %v", err)
	}
	if string(content) != "new binary" {
		t.Fatalf("extracted binary = %q, want %q", content, "new binary")
	}
}

func TestExtractLinuxBinaryRejectsUnsafeEntry(t *testing.T) {
	archivePath := writeUpdateArchive(t, []testArchiveEntry{{name: "../netquota", content: []byte("unsafe")}})
	destination := filepath.Join(t.TempDir(), "netquota")
	if err := extractLinuxBinary(archivePath, destination); err == nil {
		t.Fatal("extractLinuxBinary accepted an unsafe archive entry")
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatalf("destination stat error = %v, want file not to exist", err)
	}
}

type testArchiveEntry struct {
	name    string
	content []byte
}

func writeUpdateArchive(t *testing.T, entries []testArchiveEntry) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "update.tar.gz")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	compressed := gzip.NewWriter(file)
	archive := tar.NewWriter(compressed)
	for _, entry := range entries {
		header := &tar.Header{Name: entry.name, Mode: 0o755, Size: int64(len(entry.content))}
		if err := archive.WriteHeader(header); err != nil {
			t.Fatalf("write archive header: %v", err)
		}
		if _, err := archive.Write(entry.content); err != nil {
			t.Fatalf("write archive entry: %v", err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("close tar archive: %v", err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatalf("close gzip archive: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}
	return path
}
