//go:build !linux && !windows

package network

import "context"

func defaultRouteInterfaceIndexes(context.Context) (map[int]struct{}, error) {
	return map[int]struct{}{}, nil
}
