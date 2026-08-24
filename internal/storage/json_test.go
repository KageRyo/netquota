package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KageRyo/netquota/internal/config"
	"github.com/KageRyo/netquota/internal/i18n"
	"github.com/KageRyo/netquota/internal/model"
)

func TestStoreRoundTripsConfigAndStateAtomically(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	store := Store{
		ConfigPath: filepath.Join(directory, "nested", "config.json"),
		StatePath:  filepath.Join(directory, "nested", "state.json"),
	}
	cfg := config.Default()
	cfg.Quotas.Download = model.Limit{Bytes: 12 * config.BytesPerGiB, AlertPercentages: []uint8{80, 100}}
	state := model.State{
		Version: model.StateVersion,
		Date:    "2026-08-21",
		Usage:   model.Usage{DownloadBytes: 123, UploadBytes: 45},
		Counters: model.Counters{
			DownloadBytes: 999,
			UploadBytes:   888,
			Initialized:   true,
		},
		AlertedThresholds: map[string]bool{"total:1:70": true},
		UpdatedAt:         time.Date(2026, 8, 21, 1, 2, 3, 0, time.UTC),
	}
	if err := store.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	if err := store.SaveState(state); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	gotConfig, err := store.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	gotState, err := store.LoadState()
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if gotConfig.Quotas.Download.Bytes != cfg.Quotas.Download.Bytes {
		t.Fatalf("download limit = %d, want %d", gotConfig.Quotas.Download.Bytes, cfg.Quotas.Download.Bytes)
	}
	if gotState.Usage != state.Usage || gotState.Counters != state.Counters || gotState.Date != state.Date {
		t.Fatalf("state = %+v, want %+v", gotState, state)
	}
	if !gotState.AlertedThresholds["total:1:70"] {
		t.Fatal("alerted threshold was not persisted")
	}
}

func TestStoreMigratesAndPersistsMissingLanguage(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	configPath := filepath.Join(directory, "config.json")
	legacy := `{
  "version": 1,
  "interface": {},
  "quotas": {"total": {"bytes": 0}, "download": {"bytes": 0}, "upload": {"bytes": 0}},
  "poll_interval_seconds": 2,
  "notifications": {"enabled": true},
  "start_on_login": false
}`
	if err := os.WriteFile(configPath, []byte(legacy), 0o600); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}
	store := Store{ConfigPath: configPath, StatePath: filepath.Join(directory, "state.json")}
	cfg, err := store.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Language != i18n.English {
		t.Fatalf("loaded language = %q, want %q", cfg.Language, i18n.English)
	}
	cfg.Language = i18n.Japanese
	if err := store.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	contents, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read persisted config: %v", err)
	}
	if !strings.Contains(string(contents), `"language": "ja"`) {
		t.Fatalf("persisted config does not contain Japanese language: %s", contents)
	}
}
