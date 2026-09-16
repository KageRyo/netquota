//go:build linux

package network

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

func defaultRouteInterfaceIndexes(ctx context.Context) (map[int]struct{}, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	names := make(map[string]struct{})
	for _, routeFile := range []struct {
		path  string
		parse func([]byte) (map[string]struct{}, error)
	}{
		{path: "/proc/net/route", parse: parseLinuxIPv4DefaultRoutes},
		{path: "/proc/net/ipv6_route", parse: parseLinuxIPv6DefaultRoutes},
	} {
		data, err := os.ReadFile(routeFile.path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", routeFile.path, err)
		}
		parsed, err := routeFile.parse(data)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", routeFile.path, err)
		}
		for name := range parsed {
			names[name] = struct{}{}
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("list network interfaces: %w", err)
	}
	indexes := make(map[int]struct{})
	for _, iface := range interfaces {
		if _, ok := names[iface.Name]; ok {
			indexes[iface.Index] = struct{}{}
		}
	}
	return indexes, nil
}

func parseLinuxIPv4DefaultRoutes(data []byte) (map[string]struct{}, error) {
	result := make(map[string]struct{})
	for lineNumber, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] == "Iface" {
			continue
		}
		if len(fields) < 8 {
			return nil, fmt.Errorf("line %d has %d columns, want at least 8", lineNumber+1, len(fields))
		}
		flags, err := strconv.ParseUint(fields[3], 16, 32)
		if err != nil {
			return nil, fmt.Errorf("line %d flags %q: %w", lineNumber+1, fields[3], err)
		}
		if fields[1] == "00000000" && fields[7] == "00000000" && flags&1 != 0 {
			result[fields[0]] = struct{}{}
		}
	}
	return result, nil
}

func parseLinuxIPv6DefaultRoutes(data []byte) (map[string]struct{}, error) {
	result := make(map[string]struct{})
	for lineNumber, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) < 10 {
			return nil, fmt.Errorf("line %d has %d columns, want at least 10", lineNumber+1, len(fields))
		}
		prefixLength, err := strconv.ParseUint(fields[1], 16, 8)
		if err != nil {
			return nil, fmt.Errorf("line %d prefix length %q: %w", lineNumber+1, fields[1], err)
		}
		if strings.Trim(fields[0], "0") == "" && prefixLength == 0 {
			result[fields[9]] = struct{}{}
		}
	}
	return result, nil
}
