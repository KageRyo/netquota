//go:build !linux && !windows

package startup

import "fmt"

func configure(bool, string) error {
	return fmt.Errorf("automatic startup is not supported on this platform")
}

func supported() bool { return false }
