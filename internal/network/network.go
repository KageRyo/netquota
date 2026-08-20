package network

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/KageRyo/netquota/internal/model"
	psnet "github.com/shirou/gopsutil/v4/net"
)

type Interface struct {
	Name            string
	HardwareAddress string
	IPv4            string
	Flags           []string
}

type Counters struct {
	DownloadBytes uint64
	UploadBytes   uint64
}

type Provider interface {
	Interfaces(context.Context) ([]Interface, error)
	Counters(context.Context, string) (Counters, error)
}

type GopsutilProvider struct{}

func (GopsutilProvider) Interfaces(ctx context.Context) ([]Interface, error) {
	stats, err := psnet.InterfacesWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("list network interfaces: %w", err)
	}

	result := make([]Interface, 0, len(stats))
	for _, stat := range stats {
		result = append(result, Interface{
			Name:            stat.Name,
			HardwareAddress: stat.HardwareAddr,
			IPv4:            firstIPv4(stat.Addrs),
			Flags:           append([]string(nil), stat.Flags...),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func (GopsutilProvider) Counters(ctx context.Context, name string) (Counters, error) {
	stats, err := psnet.IOCountersWithContext(ctx, true)
	if err != nil {
		return Counters{}, fmt.Errorf("read network counters: %w", err)
	}
	for _, stat := range stats {
		if stat.Name == name {
			return Counters{
				DownloadBytes: stat.BytesRecv,
				UploadBytes:   stat.BytesSent,
			}, nil
		}
	}
	return Counters{}, fmt.Errorf("network interface %q is not available", name)
}

// Select resolves a saved interface and falls back to the first non-loopback
// interface when no preference was configured.
func Select(selection model.InterfaceSelection, interfaces []Interface) (Interface, error) {
	if len(interfaces) == 0 {
		return Interface{}, fmt.Errorf("no network interfaces found")
	}
	if selection.Name != "" {
		for _, iface := range interfaces {
			if iface.Name == selection.Name && (selection.HardwareAddress == "" || strings.EqualFold(iface.HardwareAddress, selection.HardwareAddress)) {
				return iface, nil
			}
		}
		// A changed hardware address should not make a user lose a deliberately
		// selected interface name.
		for _, iface := range interfaces {
			if iface.Name == selection.Name {
				return iface, nil
			}
		}
	}
	if selection.HardwareAddress != "" {
		for _, iface := range interfaces {
			if strings.EqualFold(iface.HardwareAddress, selection.HardwareAddress) {
				return iface, nil
			}
		}
	}
	for _, iface := range interfaces {
		if !isLoopback(iface) && iface.IPv4 != "" {
			return iface, nil
		}
	}
	for _, iface := range interfaces {
		if !isLoopback(iface) {
			return iface, nil
		}
	}
	return interfaces[0], nil
}

func firstIPv4(addresses []psnet.InterfaceAddr) string {
	for _, address := range addresses {
		value := strings.TrimSpace(address.Addr)
		if host, _, err := net.ParseCIDR(value); err == nil {
			if ipv4 := host.To4(); ipv4 != nil {
				return ipv4.String()
			}
			continue
		}
		if ip := net.ParseIP(value); ip != nil && ip.To4() != nil {
			return ip.To4().String()
		}
	}
	return ""
}

func isLoopback(iface Interface) bool {
	if strings.EqualFold(iface.Name, "lo") || strings.EqualFold(iface.Name, "loopback") {
		return true
	}
	for _, flag := range iface.Flags {
		if strings.EqualFold(flag, "loopback") {
			return true
		}
	}
	return false
}
