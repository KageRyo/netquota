//go:build linux

package network

import "testing"

func TestParseLinuxIPv4DefaultRoutes(t *testing.T) {
	t.Parallel()

	data := []byte("Iface Destination Gateway Flags RefCnt Use Metric Mask MTU Window IRTT\n" +
		"eth0 00000000 0102A8C0 0003 0 0 100 00000000 0 0 0\n" +
		"vpn0 0008A8C0 00000000 0001 0 0 50 00FFFFFF 0 0 0\n")
	names, err := parseLinuxIPv4DefaultRoutes(data)
	if err != nil {
		t.Fatalf("parseLinuxIPv4DefaultRoutes: %v", err)
	}
	if _, ok := names["eth0"]; !ok {
		t.Fatalf("default routes = %v, want eth0", names)
	}
	if _, ok := names["vpn0"]; ok {
		t.Fatalf("non-default route vpn0 was selected: %v", names)
	}
}

func TestParseLinuxIPv6DefaultRoutes(t *testing.T) {
	t.Parallel()

	data := []byte("00000000000000000000000000000000 00000000 00000000000000000000000000000000 00000000 00000000000000000000000000000000 00000000 00000000 00000000 00000000 eth0\n")
	names, err := parseLinuxIPv6DefaultRoutes(data)
	if err != nil {
		t.Fatalf("parseLinuxIPv6DefaultRoutes: %v", err)
	}
	if _, ok := names["eth0"]; !ok {
		t.Fatalf("default routes = %v, want eth0", names)
	}
}

func TestParseLinuxIPv4DefaultRoutesRejectsMalformedRows(t *testing.T) {
	t.Parallel()

	if _, err := parseLinuxIPv4DefaultRoutes([]byte("Iface Destination Gateway Flags\neth0 00000000")); err == nil {
		t.Fatal("parseLinuxIPv4DefaultRoutes accepted a malformed row")
	}
}
