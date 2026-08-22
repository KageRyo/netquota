package update

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	updateDirectoryPrefix = "netquota-update-"
	staleDownloadAge      = 24 * time.Hour
)

func NewDownloadDirectory() (string, error) {
	return os.MkdirTemp("", updateDirectoryPrefix)
}

func CleanupStaleDownloads() error {
	return cleanupStaleDownloadsIn(os.TempDir(), time.Now())
}

func cleanupStaleDownloadsIn(root string, now time.Time) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("read updater temporary directory: %w", err)
	}
	cutoff := now.Add(-staleDownloadAge)
	var cleanupErr error
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), updateDirectoryPrefix) ||
			entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name())
		info, err := entry.Info()
		if err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("inspect %s: %w", path, err))
			continue
		}
		if !info.ModTime().Before(cutoff) {
			continue
		}
		if err := os.RemoveAll(path); err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("remove %s: %w", path, err))
		}
	}
	return cleanupErr
}
