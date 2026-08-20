package app

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/KageRyo/netquota/internal/config"
	"github.com/KageRyo/netquota/internal/format"
	"github.com/KageRyo/netquota/internal/model"
	"github.com/KageRyo/netquota/internal/network"
	"github.com/KageRyo/netquota/internal/notify"
	"github.com/KageRyo/netquota/internal/quota"
	"github.com/KageRyo/netquota/internal/usage"
)

type ConfigSaver interface {
	SaveConfig(model.Config) error
}

type StateSaver interface {
	SaveState(model.State) error
}

type Sample struct {
	At            time.Time
	Interface     network.Interface
	Usage         model.Usage
	Quota         quota.Status
	Alerts        []quota.Alert
	NewDay        bool
	Baseline      bool
	DownloadReset bool
	UploadReset   bool
}

type Monitor struct {
	mu          sync.Mutex
	cfg         model.Config
	tracker     *usage.Tracker
	provider    network.Provider
	configSaver ConfigSaver
	stateSaver  StateSaver
	notifier    notify.Notifier
	logger      *slog.Logger
	selected    network.Interface
}

func NewMonitor(
	cfg model.Config,
	state model.State,
	provider network.Provider,
	configSaver ConfigSaver,
	stateSaver StateSaver,
	notifier notify.Notifier,
	logger *slog.Logger,
) *Monitor {
	if logger == nil {
		logger = slog.Default()
	}
	if notifier == nil {
		notifier = notify.Noop{}
	}
	return &Monitor{
		cfg:         cfg.Clone(),
		tracker:     usage.NewTracker(state, time.Local),
		provider:    provider,
		configSaver: configSaver,
		stateSaver:  stateSaver,
		notifier:    notifier,
		logger:      logger,
	}
}

func (m *Monitor) Config() model.Config {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cfg.Clone()
}

func (m *Monitor) SetConfig(cfg model.Config) error {
	if err := config.Validate(cfg); err != nil {
		return err
	}
	m.mu.Lock()
	interfaceChanged := m.cfg.Interface != cfg.Interface
	m.cfg = cfg.Clone()
	if interfaceChanged {
		m.tracker.ResetForInterface()
		m.selected = network.Interface{}
	}
	state := m.tracker.State()
	m.mu.Unlock()

	if m.configSaver != nil {
		if err := m.configSaver.SaveConfig(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
	}
	if interfaceChanged && m.stateSaver != nil {
		if err := m.stateSaver.SaveState(state); err != nil {
			return fmt.Errorf("save reset state: %w", err)
		}
	}
	return nil
}

func (m *Monitor) Sample(ctx context.Context, now time.Time) (Sample, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	interfaces, err := m.provider.Interfaces(ctx)
	if err != nil {
		return Sample{}, err
	}
	selected, err := network.Select(m.cfg.Interface, interfaces)
	if err != nil {
		return Sample{}, err
	}
	if m.selected.Name != "" && (m.selected.Name != selected.Name || m.selected.HardwareAddress != selected.HardwareAddress) {
		m.tracker.ResetForInterface()
	}
	m.selected = selected
	counters, err := m.provider.Counters(ctx, selected.Name)
	if err != nil {
		return Sample{}, err
	}
	result := m.tracker.Apply(now, counters)
	state := m.tracker.State()
	alerts := make([]quota.Alert, 0)
	if !result.Baseline && !result.NewDay {
		alerts = quota.DetectAlerts(result.PreviousUsage, result.Usage, m.cfg.Quotas, state.AlertedThresholds)
	}
	for _, alert := range alerts {
		m.tracker.MarkAlert(alert.Key)
	}
	state = m.tracker.State()
	if m.stateSaver != nil {
		if err := m.stateSaver.SaveState(state); err != nil {
			return Sample{}, fmt.Errorf("save state: %w", err)
		}
	}
	for _, alert := range alerts {
		if !m.cfg.Notifications.Enabled {
			continue
		}
		message := fmt.Sprintf("%s reached %d%% (%s of %s)", dimensionName(alert.Dimension), alert.Percentage, format.Bytes(alert.UsedBytes), format.Bytes(alert.LimitBytes))
		if err := m.notifier.Notify("NetQuota quota warning", message); err != nil {
			m.logger.Warn("send quota notification", "error", err, "dimension", alert.Dimension, "percentage", alert.Percentage)
		}
	}
	return Sample{
		At:            now,
		Interface:     selected,
		Usage:         result.Usage,
		Quota:         quota.Calculate(result.Usage, m.cfg.Quotas),
		Alerts:        alerts,
		NewDay:        result.NewDay,
		Baseline:      result.Baseline,
		DownloadReset: result.DownloadReset,
		UploadReset:   result.UploadReset,
	}, nil
}

func dimensionName(dimension quota.Dimension) string {
	switch dimension {
	case quota.Total:
		return "Total usage"
	case quota.Download:
		return "Download usage"
	case quota.Upload:
		return "Upload usage"
	default:
		return string(dimension)
	}
}
