package network

import (
	"errors"
	"testing"

	"github.com/KageRyo/netquota/internal/model"
	psnet "github.com/shirou/gopsutil/v4/net"
)

func TestSelectPrefersSavedNameAndHardwareAddress(t *testing.T) {
	t.Parallel()

	interfaces := []Interface{
		{Name: "lo", Flags: []string{"loopback"}},
		{Name: "Wi-Fi", HardwareAddress: "AA:BB:CC:DD:EE:FF", IPv4: "192.168.1.8"},
	}
	selected, err := Select(model.InterfaceSelection{
		Name:            "Wi-Fi",
		HardwareAddress: "aa:bb:cc:dd:ee:ff",
	}, interfaces)
	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}
	if selected.Name != "Wi-Fi" {
		t.Fatalf("selected %q, want Wi-Fi", selected.Name)
	}
}

func TestSelectFallsBackToNonLoopbackInterface(t *testing.T) {
	t.Parallel()

	interfaces := []Interface{
		{Name: "lo", Flags: []string{"loopback"}, IPv4: "127.0.0.1"},
		{Name: "Ethernet", IPv4: "10.0.0.4"},
	}
	selected, err := Select(model.InterfaceSelection{}, interfaces)
	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}
	if selected.Name != "Ethernet" {
		t.Fatalf("selected %q, want Ethernet", selected.Name)
	}
}

func TestSelectPrefersDefaultRouteBeforeSortedOrder(t *testing.T) {
	t.Parallel()

	interfaces := []Interface{
		{Name: "VPN", IPv4: "10.8.0.2"},
		{Name: "Wi-Fi", IPv4: "192.0.2.10", DefaultRoute: true},
	}
	selected, err := Select(model.InterfaceSelection{}, interfaces)
	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}
	if selected.Name != "Wi-Fi" {
		t.Fatalf("selected %q, want Wi-Fi", selected.Name)
	}
}

func TestSelectSupportsIPv6OnlyInterface(t *testing.T) {
	t.Parallel()

	selected, err := Select(model.InterfaceSelection{}, []Interface{
		{Name: "Tunnel", IPv6: "2001:db8::10"},
	})
	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}
	if selected.Name != "Tunnel" {
		t.Fatalf("selected %q, want Tunnel", selected.Name)
	}
}

func TestSelectDoesNotFallbackForUnavailableSavedSelection(t *testing.T) {
	t.Parallel()

	_, err := Select(model.InterfaceSelection{
		Name:            "Wi-Fi",
		HardwareAddress: "aa:bb:cc:dd:ee:ff",
	}, []Interface{{Name: "Ethernet", IPv4: "192.0.2.20"}})
	if !errors.Is(err, ErrSelectedInterfaceUnavailable) {
		t.Fatalf("Select error = %v, want ErrSelectedInterfaceUnavailable", err)
	}
}

func TestSelectMatchesStableIndexAfterRename(t *testing.T) {
	t.Parallel()

	selected, err := Select(model.InterfaceSelection{
		Index:           7,
		Name:            "Wi-Fi",
		HardwareAddress: "aa:bb:cc:dd:ee:ff",
	}, []Interface{{
		Index:           7,
		Name:            "Wireless",
		HardwareAddress: "11:22:33:44:55:66",
	}})
	if err != nil {
		t.Fatalf("Select returned error: %v", err)
	}
	if selected.Name != "Wireless" {
		t.Fatalf("selected %q, want Wireless", selected.Name)
	}
}

func TestSelectRejectsSameNameWithChangedHardwareIdentity(t *testing.T) {
	t.Parallel()

	_, err := Select(model.InterfaceSelection{
		Name:            "Wi-Fi",
		HardwareAddress: "aa:bb:cc:dd:ee:ff",
	}, []Interface{{
		Name:            "Wi-Fi",
		HardwareAddress: "11:22:33:44:55:66",
	}})
	if !errors.Is(err, ErrSelectedInterfaceUnavailable) {
		t.Fatalf("Select error = %v, want ErrSelectedInterfaceUnavailable", err)
	}
}

func TestFirstIPv4IgnoresIPv6(t *testing.T) {
	t.Parallel()

	addresses := []psnet.InterfaceAddr{
		{Addr: "fe80::1/64"},
		{Addr: "192.0.2.10/24"},
	}
	if got, want := firstIPv4(addresses), "192.0.2.10"; got != want {
		t.Fatalf("firstIPv4 = %q, want %q", got, want)
	}
}
