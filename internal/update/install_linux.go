//go:build linux

package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Install(ctx context.Context, artifactPath, executable string) error {
	if artifactPath == "" || executable == "" {
		return errors.New("install update: missing package or executable path")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	resolvedExecutable, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return fmt.Errorf("resolve current executable: %w", err)
	}
	if err := installLinuxArchive(ctx, artifactPath, resolvedExecutable); err != nil {
		return fmt.Errorf("install Linux update: %w", err)
	}
	return nil
}

func installLinuxArchive(ctx context.Context, artifactPath, executable string) error {
	currentInfo, err := os.Stat(executable)
	if err != nil {
		return fmt.Errorf("stat current executable: %w", err)
	}
	parent := filepath.Dir(executable)
	staging, err := os.MkdirTemp(parent, ".netquota-update-*")
	if err != nil {
		return fmt.Errorf("create staging directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(staging) }()

	stagedExecutable := filepath.Join(staging, "netquota")
	if err := extractLinuxBinary(artifactPath, stagedExecutable); err != nil {
		return err
	}
	mode := currentInfo.Mode().Perm() | 0o111
	if err := os.Chmod(stagedExecutable, mode); err != nil {
		return fmt.Errorf("set staged executable permissions: %w", err)
	}
	if err := replaceAndRestart(ctx, stagedExecutable, executable, parent); err != nil {
		return err
	}
	return nil
}

func extractLinuxBinary(archivePath, destination string) error {
	archive, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer archive.Close()
	compressed, err := gzip.NewReader(archive)
	if err != nil {
		return fmt.Errorf("read archive compression: %w", err)
	}
	defer compressed.Close()

	reader := tar.NewReader(compressed)
	found := false
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("read archive entry: %w", err)
		}
		entry, target, err := archiveEntry(header.Name)
		if err != nil {
			return err
		}
		if header.Typeflag == tar.TypeDir {
			continue
		}
		if header.Typeflag != tar.TypeReg {
			return fmt.Errorf("unsupported archive entry type for %s", entry)
		}
		if !target {
			continue
		}
		if found {
			return errors.New("archive contains more than one netquota executable")
		}
		if header.Size < 0 || header.Size > maxDownloadBytes {
			return errors.New("staged executable is too large")
		}
		file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o700)
		if err != nil {
			return fmt.Errorf("create staged executable: %w", err)
		}
		written, copyErr := io.Copy(file, io.LimitReader(reader, maxDownloadBytes+1))
		closeErr := file.Close()
		if copyErr != nil {
			return fmt.Errorf("extract executable: %w", copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close staged executable: %w", closeErr)
		}
		if written != header.Size {
			return errors.New("archive executable size does not match its header")
		}
		found = true
	}
	if !found {
		return errors.New("archive does not contain a netquota executable")
	}
	return nil
}

func archiveEntry(name string) (string, bool, error) {
	clean := filepath.Clean(name)
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", false, fmt.Errorf("unsafe archive entry %q", name)
	}
	parts := strings.Split(clean, string(filepath.Separator))
	return clean, len(parts) == 2 && parts[1] == "netquota", nil
}

func replaceAndRestart(ctx context.Context, stagedExecutable, executable, parent string) error {
	backup, err := os.CreateTemp(parent, ".netquota-backup-*")
	if err != nil {
		return fmt.Errorf("create executable backup: %w", err)
	}
	backupPath := backup.Name()
	if err := backup.Close(); err != nil {
		_ = os.Remove(backupPath)
		return fmt.Errorf("close executable backup: %w", err)
	}
	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("prepare executable backup: %w", err)
	}

	if err := os.Link(executable, backupPath); err != nil {
		_ = os.Remove(backupPath)
		return fmt.Errorf("backup current executable: %w", err)
	}
	if err := os.Rename(stagedExecutable, executable); err != nil {
		_ = os.Remove(backupPath)
		return fmt.Errorf("replace executable: %w", err)
	}

	process := exec.CommandContext(ctx, executable)
	if err := process.Start(); err != nil {
		_ = os.Remove(executable)
		if restoreErr := os.Rename(backupPath, executable); restoreErr != nil {
			return fmt.Errorf("start updated executable: %w; restore failed: %v (backup: %s)", err, restoreErr, backupPath)
		}
		return fmt.Errorf("start updated executable: %w", err)
	}
	_ = process.Process.Release()
	_ = os.Remove(backupPath)
	return nil
}
