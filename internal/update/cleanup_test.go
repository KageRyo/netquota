package update

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanupStaleDownloadsRemovesOnlyOldUpdaterDirectories(t *testing.T) {
	root := t.TempDir()
	now := time.Now()
	oldTime := now.Add(-48 * time.Hour)

	oldDirectory := filepath.Join(root, updateDirectoryPrefix+"old")
	if err := os.Mkdir(oldDirectory, 0o700); err != nil {
		t.Fatalf("create old updater directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(oldDirectory, "installer.exe"), []byte("old"), 0o600); err != nil {
		t.Fatalf("write old updater file: %v", err)
	}
	if err := os.Chtimes(oldDirectory, oldTime, oldTime); err != nil {
		t.Fatalf("age old updater directory: %v", err)
	}

	recentDirectory := filepath.Join(root, updateDirectoryPrefix+"recent")
	if err := os.Mkdir(recentDirectory, 0o700); err != nil {
		t.Fatalf("create recent updater directory: %v", err)
	}
	unrelatedDirectory := filepath.Join(root, "other-update-old")
	if err := os.Mkdir(unrelatedDirectory, 0o700); err != nil {
		t.Fatalf("create unrelated directory: %v", err)
	}
	if err := os.Chtimes(unrelatedDirectory, oldTime, oldTime); err != nil {
		t.Fatalf("age unrelated directory: %v", err)
	}

	if err := cleanupStaleDownloadsIn(root, now); err != nil {
		t.Fatalf("cleanupStaleDownloadsIn: %v", err)
	}
	if _, err := os.Stat(oldDirectory); !os.IsNotExist(err) {
		t.Fatalf("old updater directory stat error = %v, want it removed", err)
	}
	if _, err := os.Stat(recentDirectory); err != nil {
		t.Fatalf("recent updater directory: %v", err)
	}
	if _, err := os.Stat(unrelatedDirectory); err != nil {
		t.Fatalf("unrelated directory: %v", err)
	}
}

func TestCleanupStaleDownloadsSkipsSymlinkDirectories(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatalf("create target directory: %v", err)
	}
	symlink := filepath.Join(root, updateDirectoryPrefix+"symlink")
	if err := os.Symlink(target, symlink); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if err := cleanupStaleDownloadsIn(root, time.Now().Add(48*time.Hour)); err != nil {
		t.Fatalf("cleanupStaleDownloadsIn: %v", err)
	}
	if _, err := os.Lstat(symlink); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("symlink target: %v", err)
	}
}
