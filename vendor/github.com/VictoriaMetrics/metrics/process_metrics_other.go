//go:build !linux
// +build !linux

package metrics

import (
	"io"
)

func writeProcessMetrics(w io.Writer) {
	// TODO: implement it
}
