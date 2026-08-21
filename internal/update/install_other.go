//go:build !linux && !windows

package update

import (
	"context"
)

func Install(ctx context.Context, artifactPath, executable string) error {
	_ = ctx
	_ = artifactPath
	_ = executable
	return ErrUnsupportedPlatform
}
