package format

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/KageRyo/netquota/internal/config"
	"github.com/KageRyo/netquota/internal/i18n"
)

var units = []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB"}

func Bytes(value uint64) string {
	if value < 1024 {
		return fmt.Sprintf("%d B", value)
	}
	amount := float64(value)
	unit := 0
	for amount >= 1024 && unit < len(units)-1 {
		amount /= 1024
		unit++
	}
	return fmt.Sprintf("%.1f %s", amount, units[unit])
}

func Quota(value uint64) string {
	if value == 0 {
		return "Disabled"
	}
	return Bytes(value)
}

func ParseGiB(input string) (uint64, error) {
	value, err := strconv.ParseFloat(strings.TrimSpace(input), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, i18n.NewError("error.quota_not_number", nil)
	}
	if value < 0 {
		return 0, i18n.NewError("error.quota_negative", nil)
	}
	if value > float64(^uint64(0))/float64(config.BytesPerGiB) {
		return 0, i18n.NewError("error.quota_too_large", nil)
	}
	return uint64(math.Round(value * float64(config.BytesPerGiB))), nil
}

func ParsePercentages(input string) ([]uint8, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, nil
	}
	parts := strings.Split(input, ",")
	result := make([]uint8, 0, len(parts))
	for _, part := range parts {
		value, err := strconv.ParseUint(strings.TrimSpace(part), 10, 8)
		if err != nil || value == 0 || value > 100 {
			return nil, i18n.NewError("error.alert_invalid", nil)
		}
		result = append(result, uint8(value))
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	for index := 1; index < len(result); index++ {
		if result[index] == result[index-1] {
			return nil, i18n.NewError("error.alert_duplicates", nil)
		}
	}
	return result, nil
}

func Percent(value float64) string {
	return fmt.Sprintf("%.1f%%", value)
}
