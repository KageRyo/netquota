package config

import (
	"testing"

	"github.com/KageRyo/netquota/internal/i18n"
	"github.com/KageRyo/netquota/internal/model"
)

func TestDefaultConfigIsValid(t *testing.T) {
	t.Parallel()

	cfg := Default()
	if err := Validate(cfg); err != nil {
		t.Fatalf("default config should be valid: %v", err)
	}
	if cfg.Quotas.Total.Bytes != 100*BytesPerGiB {
		t.Fatalf("default total quota = %d, want %d", cfg.Quotas.Total.Bytes, 100*BytesPerGiB)
	}
	if cfg.Language != i18n.English {
		t.Fatalf("default language = %q, want %q", cfg.Language, i18n.English)
	}
}

func TestWithDefaultsAddsThresholdsToEnabledLimits(t *testing.T) {
	t.Parallel()

	cfg := WithDefaults(model.Config{
		Version:             model.ConfigVersion,
		PollIntervalSeconds: 5,
		Quotas: model.Quotas{
			Download: model.Limit{Bytes: BytesPerGiB},
		},
	})
	if got, want := cfg.Quotas.Download.AlertPercentages, []uint8{70, 85, 95, 100}; !equalPercentages(got, want) {
		t.Fatalf("download thresholds = %v, want %v", got, want)
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("config with defaults should be valid: %v", err)
	}
}

func TestValidateRejectsUnsortedThresholds(t *testing.T) {
	t.Parallel()

	cfg := Default()
	cfg.Quotas.Total.AlertPercentages = []uint8{95, 70}
	if err := Validate(cfg); err == nil {
		t.Fatal("Validate accepted unsorted alert thresholds")
	}
}

func TestWithDefaultsAddsEnglishToExistingConfig(t *testing.T) {
	t.Parallel()

	cfg := WithDefaults(model.Config{Version: model.ConfigVersion, PollIntervalSeconds: 2})
	if cfg.Language != i18n.English {
		t.Fatalf("language = %q, want %q", cfg.Language, i18n.English)
	}
}

func TestValidateRejectsUnsupportedLanguage(t *testing.T) {
	t.Parallel()

	cfg := Default()
	cfg.Language = "fr"
	if err := Validate(cfg); err == nil {
		t.Fatal("Validate accepted an unsupported language")
	}
}

func equalPercentages(left, right []uint8) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
