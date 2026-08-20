package startup

import "fmt"

func Configure(enabled bool, executable string) error {
	if executable == "" {
		return fmt.Errorf("executable path cannot be empty")
	}
	return configure(enabled, executable)
}

func Supported() bool {
	return supported()
}
