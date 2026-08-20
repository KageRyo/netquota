package format

import "testing"

func TestBytesUsesBinaryUnits(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		value uint64
		want  string
	}{
		{value: 0, want: "0 B"},
		{value: 1024, want: "1.0 KiB"},
		{value: 1024 * 1024 * 1024, want: "1.0 GiB"},
	} {
		if got := Bytes(test.value); got != test.want {
			t.Errorf("Bytes(%d) = %q, want %q", test.value, got, test.want)
		}
	}
}

func TestParseGiBAndPercentages(t *testing.T) {
	t.Parallel()

	quota, err := ParseGiB("1.5")
	if err != nil {
		t.Fatalf("ParseGiB: %v", err)
	}
	if quota != 3*1024*1024*512 {
		t.Fatalf("ParseGiB = %d", quota)
	}
	percentages, err := ParsePercentages("95, 70, 100")
	if err != nil {
		t.Fatalf("ParsePercentages: %v", err)
	}
	if got, want := percentages, []uint8{70, 95, 100}; !samePercentages(got, want) {
		t.Fatalf("percentages = %v, want %v", got, want)
	}
}

func TestParsePercentagesRejectsDuplicates(t *testing.T) {
	t.Parallel()

	if _, err := ParsePercentages("70,70"); err == nil {
		t.Fatal("ParsePercentages accepted duplicate thresholds")
	}
}

func samePercentages(left, right []uint8) bool {
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
