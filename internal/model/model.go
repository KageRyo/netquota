// Package model contains the data shared by NetQuota's core, storage, and UI.
package model

import (
	"time"

	"github.com/KageRyo/netquota/internal/i18n"
)

const (
	ConfigVersion = 1
	StateVersion  = 1
)

// InterfaceSelection identifies the network interface that should be tracked.
// Name is the primary selector; hardware address and IPv4 are retained to make
// reconnects and renamed interfaces easier to resolve.
type InterfaceSelection struct {
	Name            string `json:"name"`
	HardwareAddress string `json:"hardware_address"`
	IPv4            string `json:"ipv4"`
}

// Limit describes one independently configurable quota and its notification
// thresholds. Bytes == 0 disables that quota.
type Limit struct {
	Bytes            uint64  `json:"bytes"`
	AlertPercentages []uint8 `json:"alert_percentages"`
}

type Quotas struct {
	Total    Limit `json:"total"`
	Download Limit `json:"download"`
	Upload   Limit `json:"upload"`
}

type NotificationConfig struct {
	Enabled bool `json:"enabled"`
}

type Config struct {
	Version             int                `json:"version"`
	Language            i18n.Language      `json:"language"`
	Interface           InterfaceSelection `json:"interface"`
	Quotas              Quotas             `json:"quotas"`
	PollIntervalSeconds int                `json:"poll_interval_seconds"`
	Notifications       NotificationConfig `json:"notifications"`
	StartOnLogin        bool               `json:"start_on_login"`
}

type Usage struct {
	DownloadBytes uint64 `json:"download_bytes"`
	UploadBytes   uint64 `json:"upload_bytes"`
}

type Counters struct {
	DownloadBytes uint64 `json:"last_download_counter"`
	UploadBytes   uint64 `json:"last_upload_counter"`
	Initialized   bool   `json:"initialized"`
}

// State is written after each sample. AlertedThresholds prevents a threshold
// from producing repeated notifications during one day.
type State struct {
	Version           int             `json:"version"`
	Date              string          `json:"date"`
	Usage             Usage           `json:"usage"`
	Counters          Counters        `json:"counters"`
	AlertedThresholds map[string]bool `json:"alerted_thresholds"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

func (c Config) Clone() Config {
	c.Quotas.Total.AlertPercentages = clonePercentages(c.Quotas.Total.AlertPercentages)
	c.Quotas.Download.AlertPercentages = clonePercentages(c.Quotas.Download.AlertPercentages)
	c.Quotas.Upload.AlertPercentages = clonePercentages(c.Quotas.Upload.AlertPercentages)
	return c
}

func (s State) Clone() State {
	if s.AlertedThresholds != nil {
		alertedThresholds := s.AlertedThresholds
		s.AlertedThresholds = make(map[string]bool, len(alertedThresholds))
		for key, value := range alertedThresholds {
			s.AlertedThresholds[key] = value
		}
	}
	return s
}

func clonePercentages(values []uint8) []uint8 {
	if values == nil {
		return nil
	}
	return append([]uint8(nil), values...)
}
