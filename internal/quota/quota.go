package quota

import (
	"fmt"

	"github.com/KageRyo/netquota/internal/model"
)

type Dimension string

const (
	Total    Dimension = "total"
	Download Dimension = "download"
	Upload   Dimension = "upload"
)

type MetricStatus struct {
	Dimension      Dimension
	UsedBytes      uint64
	LimitBytes     uint64
	RemainingBytes uint64
	Percent        float64
	Enabled        bool
	Exceeded       bool
}

type Status struct {
	Total    MetricStatus
	Download MetricStatus
	Upload   MetricStatus
}

type Alert struct {
	Key        string
	Dimension  Dimension
	Percentage uint8
	UsedBytes  uint64
	LimitBytes uint64
}

func Calculate(usage model.Usage, quotas model.Quotas) Status {
	total := saturatingAdd(usage.DownloadBytes, usage.UploadBytes)
	return Status{
		Total:    calculateMetric(Total, total, quotas.Total),
		Download: calculateMetric(Download, usage.DownloadBytes, quotas.Download),
		Upload:   calculateMetric(Upload, usage.UploadBytes, quotas.Upload),
	}
}

func DetectAlerts(_ model.Usage, current model.Usage, quotas model.Quotas, already map[string]bool) []Alert {
	currentTotal := saturatingAdd(current.DownloadBytes, current.UploadBytes)
	items := []struct {
		dimension Dimension
		current   uint64
		limit     model.Limit
	}{
		{dimension: Total, current: currentTotal, limit: quotas.Total},
		{dimension: Download, current: current.DownloadBytes, limit: quotas.Download},
		{dimension: Upload, current: current.UploadBytes, limit: quotas.Upload},
	}

	var alerts []Alert
	for _, item := range items {
		if item.limit.Bytes == 0 {
			continue
		}
		for _, percentage := range item.limit.AlertPercentages {
			key := AlertKey(item.dimension, item.limit.Bytes, percentage)
			if already != nil && already[key] {
				continue
			}
			if reached(item.current, item.limit.Bytes, percentage) {
				alerts = append(alerts, Alert{
					Key:        key,
					Dimension:  item.dimension,
					Percentage: percentage,
					UsedBytes:  item.current,
					LimitBytes: item.limit.Bytes,
				})
			}
		}
	}
	return alerts
}

func AlertKey(dimension Dimension, limitBytes uint64, percentage uint8) string {
	return fmt.Sprintf("%s:%d:%d", dimension, limitBytes, percentage)
}

func calculateMetric(dimension Dimension, used uint64, limit model.Limit) MetricStatus {
	if limit.Bytes == 0 {
		return MetricStatus{Dimension: dimension, UsedBytes: used}
	}
	remaining := uint64(0)
	if used < limit.Bytes {
		remaining = limit.Bytes - used
	}
	return MetricStatus{
		Dimension:      dimension,
		UsedBytes:      used,
		LimitBytes:     limit.Bytes,
		RemainingBytes: remaining,
		Percent:        float64(used) / float64(limit.Bytes) * 100,
		Enabled:        true,
		Exceeded:       used >= limit.Bytes,
	}
}

func reached(used, limit uint64, percentage uint8) bool {
	if limit == 0 {
		return false
	}
	return float64(used)/float64(limit)*100 >= float64(percentage)
}

func saturatingAdd(left, right uint64) uint64 {
	if ^uint64(0)-left < right {
		return ^uint64(0)
	}
	return left + right
}
