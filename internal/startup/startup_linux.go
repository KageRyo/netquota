//go:build linux

package startup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

func configure(enabled bool, executable string) error {
	configDirectory, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("resolve user config directory: %w", err)
	}
	path := filepath.Join(configDirectory, "autostart", "netquota.desktop")
	if !enabled {
		err := os.Remove(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("remove Linux autostart entry: %w", err)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create Linux autostart directory: %w", err)
	}
	content := fmt.Sprintf("[Desktop Entry]\nType=Application\nVersion=1.0\nName=NetQuota\nComment=Track daily network usage\nExec=%s\nTerminal=false\nX-GNOME-Autostart-enabled=true\n", strconv.Quote(executable))
	temporary, err := os.CreateTemp(filepath.Dir(path), ".netquota.desktop.tmp-*")
	if err != nil {
		return fmt.Errorf("create Linux autostart entry: %w", err)
	}
	temporaryName := temporary.Name()
	defer func() { _ = os.Remove(temporaryName) }()
	if _, err := temporary.WriteString(content); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write Linux autostart entry: %w", err)
	}
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set Linux autostart permissions: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close Linux autostart entry: %w", err)
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return fmt.Errorf("install Linux autostart entry: %w", err)
	}
	return nil
}

func supported() bool { return true }
