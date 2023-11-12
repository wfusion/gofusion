//go:build !linux && !darwin
// +build !linux,!darwin

package rotatelog

import (
	"os"
)

func chown(_ string, _ os.FileInfo) error {
	return nil
}
