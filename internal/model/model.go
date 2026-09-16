// Package model contains the data shared by NetQuota's core, storage, and UI.
package model

import (
	"fmt"
	"time"

	"github.com/KageRyo/netquota/internal/i18n"
)

const (
	ConfigVersion = 2
	StateVersion  = 2
)

type BillingCycleKind string

const (
	BillingCycleDaily   BillingCycleKind = "daily"
	BillingCycleMonthly BillingCycleKind = "monthly"
	BillingCycleCustom  BillingCycleKind = "custom"
)

// BillingCycle controls when accumulated usage and alert marks start over.
// ResetDay is used only for the custom monthly cycle and is one-based.
type BillingCycle struct {
	Kind     BillingCycleKind `json:"kind"`
	ResetDay uint8            `json:"reset_day,omitempty"`
}

func (c BillingCycle) Normalized() BillingCycle {
	if c.Kind == "" {
		c.Kind = BillingCycleDaily
	}
	if c.Kind == BillingCycleMonthly {
		c.ResetDay = 0
	}
	return c
}

func (c BillingCycle) Valid() bool {
	switch c.Kind {
	case "", BillingCycleDaily, BillingCycleMonthly:
		return true
	case BillingCycleCustom:
		return c.ResetDay >= 1 && c.ResetDay <= 31
	default:
		return false
	}
}

func (c BillingCycle) Equal(other BillingCycle) bool {
	return c.Normalized() == other.Normalized()
}

func (c BillingCycle) Identity() string {
	c = c.Normalized()
	if c.Kind == BillingCycleCustom {
		return fmt.Sprintf("custom:%d", c.ResetDay)
	}
	return string(c.Kind)
}

// InterfaceSelection identifies the network interface that should be tracked.
// Index and hardware address are preferred stable identities. Name and
// addresses remain for backwards-compatible configuration and legacy hosts.
type InterfaceSelection struct {
	Name            string `json:"name"`
	Index           int    `json:"index"`
	HardwareAddress string `json:"hardware_address"`
	IPv4            string `json:"ipv4"`
	IPv6            string `json:"ipv6"`
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
	BillingCycle        BillingCycle       `json:"billing_cycle"`
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
	PeriodKey         string          `json:"period_key"`
	BillingCycleKey   string          `json:"billing_cycle_key"`
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
