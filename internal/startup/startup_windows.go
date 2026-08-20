//go:build windows

package startup

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows/registry"
)

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`

func configure(enabled bool, executable string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open Windows startup registry key: %w", err)
	}
	defer key.Close()
	if enabled {
		if err := key.SetStringValue("NetQuota", executable); err != nil {
			return fmt.Errorf("enable Windows startup: %w", err)
		}
		return nil
	}
	if err := key.DeleteValue("NetQuota"); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return fmt.Errorf("disable Windows startup: %w", err)
	}
	return nil
}

func supported() bool { return true }
