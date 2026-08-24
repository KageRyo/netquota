package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/KageRyo/netquota/internal/i18n"
	"github.com/KageRyo/netquota/internal/model"
)

const (
	BytesPerGiB          = uint64(1024 * 1024 * 1024)
	DefaultPollInterval  = 2
	DefaultTotalQuotaGiB = 100
)

var defaultAlertPercentages = []uint8{70, 85, 95, 100}

func Default() model.Config {
	return model.Config{
		Version:  model.ConfigVersion,
		Language: i18n.English,
		Quotas: model.Quotas{
			Total: model.Limit{
				Bytes:            DefaultTotalQuotaGiB * BytesPerGiB,
				AlertPercentages: append([]uint8(nil), defaultAlertPercentages...),
			},
		},
		PollIntervalSeconds: DefaultPollInterval,
		Notifications: model.NotificationConfig{
			Enabled: true,
		},
	}
}

// WithDefaults fills fields introduced by the current config version while
// preserving explicitly supplied values.
func WithDefaults(cfg model.Config) model.Config {
	defaults := Default()
	if cfg.Version == 0 {
		cfg.Version = defaults.Version
	}
	if cfg.Language == "" {
		cfg.Language = defaults.Language
	}
	if cfg.PollIntervalSeconds == 0 {
		cfg.PollIntervalSeconds = defaults.PollIntervalSeconds
	}
	if cfg.Quotas.Total.Bytes > 0 && len(cfg.Quotas.Total.AlertPercentages) == 0 {
		cfg.Quotas.Total.AlertPercentages = append([]uint8(nil), defaultAlertPercentages...)
	}
	if cfg.Quotas.Download.Bytes > 0 && len(cfg.Quotas.Download.AlertPercentages) == 0 {
		cfg.Quotas.Download.AlertPercentages = append([]uint8(nil), defaultAlertPercentages...)
	}
	if cfg.Quotas.Upload.Bytes > 0 && len(cfg.Quotas.Upload.AlertPercentages) == 0 {
		cfg.Quotas.Upload.AlertPercentages = append([]uint8(nil), defaultAlertPercentages...)
	}
	return cfg
}

func Validate(cfg model.Config) error {
	if cfg.Version != model.ConfigVersion {
		return fmt.Errorf("unsupported config version %d", cfg.Version)
	}
	if cfg.PollIntervalSeconds < 1 || cfg.PollIntervalSeconds > 24*60*60 {
		return errors.New("poll_interval_seconds must be between 1 and 86400")
	}
	if cfg.Language != "" && !cfg.Language.Valid() {
		return i18n.NewError("error.language_invalid", nil)
	}
	if err := validateLimit("total", cfg.Quotas.Total); err != nil {
		return err
	}
	if err := validateLimit("download", cfg.Quotas.Download); err != nil {
		return err
	}
	if err := validateLimit("upload", cfg.Quotas.Upload); err != nil {
		return err
	}
	return nil
}

func validateLimit(name string, limit model.Limit) error {
	if limit.Bytes == 0 {
		return nil
	}
	if len(limit.AlertPercentages) == 0 {
		return fmt.Errorf("%s alert_percentages cannot be empty when the quota is enabled", name)
	}
	var previous uint8
	for index, percentage := range limit.AlertPercentages {
		if percentage == 0 || percentage > 100 {
			return fmt.Errorf("%s alert percentage %d must be between 1 and 100", name, percentage)
		}
		if index > 0 && percentage <= previous {
			return fmt.Errorf("%s alert percentages must be strictly increasing", name)
		}
		previous = percentage
	}
	return nil
}

// Paths returns the per-user paths used for configuration and daily state.
// The directory is intentionally outside the repository so usage data stays
// local to the user.
func Paths() (configPath, statePath string, err error) {
	directory, err := os.UserConfigDir()
	if err != nil {
		return "", "", fmt.Errorf("resolve user config directory: %w", err)
	}
	directory = filepath.Join(directory, "NetQuota")
	return filepath.Join(directory, "config.json"), filepath.Join(directory, "state.json"), nil
}
