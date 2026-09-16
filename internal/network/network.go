package network

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/KageRyo/netquota/internal/model"
	psnet "github.com/shirou/gopsutil/v4/net"
)

type Interface struct {
	Name            string
	Index           int
	HardwareAddress string
	IPv4            string
	IPv6            string
	DefaultRoute    bool
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

var ErrSelectedInterfaceUnavailable = errors.New("selected network interface is unavailable")

type GopsutilProvider struct{}

func (GopsutilProvider) Interfaces(ctx context.Context) ([]Interface, error) {
	stats, err := psnet.InterfacesWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("list network interfaces: %w", err)
	}
	standardInterfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("list standard network interfaces: %w", err)
	}
	standardByName := make(map[string]net.Interface, len(standardInterfaces))
	for _, iface := range standardInterfaces {
		standardByName[iface.Name] = iface
	}
	routeIndexes, routeErr := defaultRouteInterfaceIndexes(ctx)
	if routeErr != nil && ctx.Err() != nil {
		return nil, routeErr
	}

	result := make([]Interface, 0, len(stats))
	for _, stat := range stats {
		standard := standardByName[stat.Name]
		result = append(result, Interface{
			Name:            stat.Name,
			Index:           standard.Index,
			HardwareAddress: stat.HardwareAddr,
			IPv4:            firstIPv4(stat.Addrs),
			IPv6:            firstIPv6(stat.Addrs),
			DefaultRoute:    hasIndex(routeIndexes, standard.Index),
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

// Select resolves a saved interface without falling back to an unrelated
// interface. With no saved preference it prefers an active default route and
// then deterministic non-loopback candidates.
func Select(selection model.InterfaceSelection, interfaces []Interface) (Interface, error) {
	if len(interfaces) == 0 {
		if selectionConfigured(selection) {
			return Interface{}, fmt.Errorf("%w: %s", ErrSelectedInterfaceUnavailable, selectionDisplayName(selection))
		}
		return Interface{}, fmt.Errorf("no network interfaces found")
	}
	if selectionConfigured(selection) {
		for _, iface := range interfaces {
			if matchesSelection(selection, iface) {
				return iface, nil
			}
		}
		return Interface{}, fmt.Errorf("%w: %s", ErrSelectedInterfaceUnavailable, selectionDisplayName(selection))
	}
	for _, iface := range interfaces {
		if !isLoopback(iface) && iface.DefaultRoute {
			return iface, nil
		}
	}
	for _, iface := range interfaces {
		if !isLoopback(iface) && hasAddress(iface) {
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

func SelectionForInterface(iface Interface) model.InterfaceSelection {
	return model.InterfaceSelection{
		Name:            iface.Name,
		Index:           iface.Index,
		HardwareAddress: iface.HardwareAddress,
		IPv4:            iface.IPv4,
		IPv6:            iface.IPv6,
	}
}

func SameIdentity(left, right Interface) bool {
	if left.Name == "" && left.Index == 0 && left.HardwareAddress == "" && right.Name == "" && right.Index == 0 && right.HardwareAddress == "" {
		return true
	}
	if left.Index != 0 && right.Index != 0 && left.Index == right.Index {
		return true
	}
	if left.HardwareAddress != "" && right.HardwareAddress != "" && strings.EqualFold(left.HardwareAddress, right.HardwareAddress) {
		return true
	}
	if left.Name == "" || left.Name != right.Name {
		return false
	}
	if left.HardwareAddress != "" && right.HardwareAddress != "" {
		return false
	}
	return true
}

func SameSelection(left, right model.InterfaceSelection) bool {
	return SameIdentity(
		Interface{
			Name:            left.Name,
			Index:           left.Index,
			HardwareAddress: left.HardwareAddress,
		},
		Interface{
			Name:            right.Name,
			Index:           right.Index,
			HardwareAddress: right.HardwareAddress,
		},
	)
}

func selectionConfigured(selection model.InterfaceSelection) bool {
	return selection.Name != "" || selection.Index != 0 || selection.HardwareAddress != "" || selection.IPv4 != "" || selection.IPv6 != ""
}

func matchesSelection(selection model.InterfaceSelection, iface Interface) bool {
	if selection.Index != 0 && iface.Index != 0 && selection.Index == iface.Index {
		return true
	}
	if selection.HardwareAddress != "" && iface.HardwareAddress != "" && strings.EqualFold(iface.HardwareAddress, selection.HardwareAddress) {
		return true
	}
	if selection.Index == 0 && selection.HardwareAddress == "" {
		if selection.Name != "" && selection.Name == iface.Name {
			return true
		}
		if selection.IPv4 != "" && selection.IPv4 == iface.IPv4 {
			return true
		}
		if selection.IPv6 != "" && selection.IPv6 == iface.IPv6 {
			return true
		}
	}
	return false
}

func selectionDisplayName(selection model.InterfaceSelection) string {
	if selection.Name != "" {
		return fmt.Sprintf("%q", selection.Name)
	}
	if selection.HardwareAddress != "" {
		return fmt.Sprintf("%q", selection.HardwareAddress)
	}
	if selection.IPv4 != "" {
		return fmt.Sprintf("%q", selection.IPv4)
	}
	return fmt.Sprintf("interface index %d", selection.Index)
}

func hasAddress(iface Interface) bool {
	return iface.IPv4 != "" || iface.IPv6 != ""
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

func firstIPv6(addresses []psnet.InterfaceAddr) string {
	for _, address := range addresses {
		value := strings.TrimSpace(address.Addr)
		if host, _, err := net.ParseCIDR(value); err == nil {
			if host.To4() == nil && host.IsGlobalUnicast() {
				return host.String()
			}
			continue
		}
		if ip := net.ParseIP(value); ip != nil && ip.To4() == nil && ip.IsGlobalUnicast() {
			return ip.String()
		}
	}
	return ""
}

func hasIndex(indexes map[int]struct{}, index int) bool {
	_, ok := indexes[index]
	return index != 0 && ok
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
