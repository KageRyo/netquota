package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/KageRyo/netquota/internal/config"
	"github.com/KageRyo/netquota/internal/model"
)

type Store struct {
	ConfigPath string
	StatePath  string
}

func (s Store) LoadConfig() (model.Config, error) {
	var cfg model.Config
	if err := readJSON(s.ConfigPath, &cfg); err != nil {
		return model.Config{}, err
	}
	cfg = config.WithDefaults(cfg)
	if err := config.Validate(cfg); err != nil {
		return model.Config{}, fmt.Errorf("validate config: %w", err)
	}
	return cfg, nil
}

func (s Store) SaveConfig(cfg model.Config) error {
	cfg = config.WithDefaults(cfg)
	if err := config.Validate(cfg); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}
	return writeJSONAtomic(s.ConfigPath, cfg)
}

func (s Store) LoadState() (model.State, error) {
	var state model.State
	if err := readJSON(s.StatePath, &state); err != nil {
		return model.State{}, err
	}
	if state.Version == 0 {
		state.Version = model.StateVersion
	}
	if state.Version != model.StateVersion {
		return model.State{}, fmt.Errorf("unsupported state version %d", state.Version)
	}
	if state.AlertedThresholds == nil {
		state.AlertedThresholds = make(map[string]bool)
	}
	return state, nil
}

func (s Store) SaveState(state model.State) error {
	if state.Version == 0 {
		state.Version = model.StateVersion
	}
	return writeJSONAtomic(s.StatePath, state)
}

func readJSON(path string, target any) error {
	if path == "" {
		return errors.New("JSON path cannot be empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func writeJSONAtomic(path string, value any) error {
	if path == "" {
		return errors.New("JSON path cannot be empty")
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	data = append(data, '\n')
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create storage directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary JSON file: %w", err)
	}
	temporaryName := temporary.Name()
	defer func() {
		_ = os.Remove(temporaryName)
	}()
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set JSON permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write temporary JSON file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("flush temporary JSON file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary JSON file: %w", err)
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return fmt.Errorf("replace JSON file: %w", err)
	}
	return nil
}
