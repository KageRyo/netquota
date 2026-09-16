//go:build windows

package network

import (
	"context"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

func defaultRouteInterfaceIndexes(ctx context.Context) (map[int]struct{}, error) {
	indexes := make(map[int]struct{})
	var lastErr error
	for _, family := range []uint16{windows.AF_INET, windows.AF_INET6} {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var table *windows.MibIpForwardTable2
		if err := windows.GetIpForwardTable2(family, &table); err != nil {
			lastErr = fmt.Errorf("get IP forward table for family %d: %w", family, err)
			continue
		}
		if table == nil {
			continue
		}
		for _, row := range table.Rows() {
			if row.DestinationPrefix.PrefixLength == 0 {
				indexes[int(row.InterfaceIndex)] = struct{}{}
			}
		}
		windows.FreeMibTable(unsafe.Pointer(table))
	}
	if len(indexes) == 0 && lastErr != nil {
		return indexes, lastErr
	}
	return indexes, nil
}
