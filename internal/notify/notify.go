package notify

import "github.com/gen2brain/beeep"

type Notifier interface {
	Notify(title, message string) error
}

type System struct {
	IconPath string
}

func (s System) Notify(title, message string) error {
	return beeep.Notify(title, message, s.IconPath)
}

type Noop struct{}

func (Noop) Notify(string, string) error { return nil }
