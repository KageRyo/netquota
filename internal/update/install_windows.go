//go:build windows

package update

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

func Install(ctx context.Context, artifactPath, executable string) error {
	if artifactPath == "" {
		return errors.New("install update: missing installer path")
	}
	if _, err := os.Stat(artifactPath); err != nil {
		return fmt.Errorf("stat installer: %w", err)
	}
	// The installer is intentionally interactive. Inno Setup closes the running
	// application when the user proceeds, so the caller can exit after Start.
	_ = ctx
	_ = executable
	process := exec.Command(artifactPath)
	if err := process.Start(); err != nil {
		return fmt.Errorf("start installer: %w", err)
	}
	if err := process.Process.Release(); err != nil {
		return fmt.Errorf("detach installer: %w", err)
	}
	return nil
}
