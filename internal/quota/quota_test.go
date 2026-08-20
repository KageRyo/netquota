package quota

import (
	"math"
	"testing"

	"github.com/KageRyo/netquota/internal/model"
)

func TestCalculateSupportsTotalAndSeparateLimits(t *testing.T) {
	t.Parallel()

	status := Calculate(model.Usage{DownloadBytes: 70, UploadBytes: 20}, model.Quotas{
		Total:    model.Limit{Bytes: 100},
		Download: model.Limit{Bytes: 80},
		Upload:   model.Limit{Bytes: 25},
	})
	if status.Total.Percent != 90 {
		t.Fatalf("total percentage = %v, want 90", status.Total.Percent)
	}
	if status.Total.RemainingBytes != 10 {
		t.Fatalf("total remaining = %d, want 10", status.Total.RemainingBytes)
	}
	if status.Download.Percent != 87.5 || status.Upload.Percent != 80 {
		t.Fatalf("separate percentages = download:%v upload:%v", status.Download.Percent, status.Upload.Percent)
	}
}

func TestCalculateDisablesZeroLimit(t *testing.T) {
	t.Parallel()

	status := Calculate(model.Usage{DownloadBytes: 42}, model.Quotas{})
	if status.Download.Enabled || status.Total.Enabled || status.Upload.Enabled {
		t.Fatal("zero limits should be disabled")
	}
	if status.Download.UsedBytes != 42 {
		t.Fatalf("disabled metric lost usage: %+v", status.Download)
	}
}

func TestDetectAlertsForTotalAndSeparateDimensions(t *testing.T) {
	t.Parallel()

	quotas := model.Quotas{
		Total:    model.Limit{Bytes: 100, AlertPercentages: []uint8{70, 95}},
		Download: model.Limit{Bytes: 80, AlertPercentages: []uint8{70}},
		Upload:   model.Limit{Bytes: 40, AlertPercentages: []uint8{75}},
	}
	alerts := DetectAlerts(
		model.Usage{DownloadBytes: 40, UploadBytes: 20},
		model.Usage{DownloadBytes: 50, UploadBytes: 35},
		quotas,
		nil,
	)
	if len(alerts) != 2 {
		t.Fatalf("got %d alerts, want total and upload", len(alerts))
	}
	if alerts[0].Dimension != Total || alerts[0].Percentage != 70 {
		t.Fatalf("first alert = %+v", alerts[0])
	}
	if alerts[1].Dimension != Upload || alerts[1].Percentage != 75 {
		t.Fatalf("second alert = %+v", alerts[1])
	}
}

func TestDetectAlertsDoesNotRepeatAlreadyAlertedThreshold(t *testing.T) {
	t.Parallel()

	quotas := model.Quotas{Total: model.Limit{Bytes: 100, AlertPercentages: []uint8{70}}}
	key := AlertKey(Total, 100, 70)
	alerts := DetectAlerts(model.Usage{DownloadBytes: 69}, model.Usage{DownloadBytes: 80}, quotas, map[string]bool{key: true})
	if len(alerts) != 0 {
		t.Fatalf("got repeated alerts: %+v", alerts)
	}
}

func TestDetectAlertsReportsCurrentUsageAfterLimitIsLowered(t *testing.T) {
	t.Parallel()

	quotas := model.Quotas{Total: model.Limit{Bytes: 50, AlertPercentages: []uint8{70}}}
	alerts := DetectAlerts(model.Usage{DownloadBytes: 80}, model.Usage{DownloadBytes: 80}, quotas, nil)
	if len(alerts) != 1 || alerts[0].Dimension != Total {
		t.Fatalf("alerts after lowering limit = %+v", alerts)
	}
}

func TestCalculateSaturatesTotal(t *testing.T) {
	t.Parallel()

	status := Calculate(model.Usage{DownloadBytes: math.MaxUint64, UploadBytes: 1}, model.Quotas{Total: model.Limit{Bytes: math.MaxUint64}})
	if status.Total.UsedBytes != math.MaxUint64 || !status.Total.Exceeded {
		t.Fatalf("total overflow handling = %+v", status.Total)
	}
}
