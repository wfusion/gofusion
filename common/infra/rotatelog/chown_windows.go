//go:build windows
// +build windows

package rotatelog

import (
	"os"
)

func chown(name string, info os.FileInfo) error {
	return nil
}
