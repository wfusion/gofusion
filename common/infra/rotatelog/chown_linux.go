//go:build linux
// +build linux

package rotatelog

import (
	"os"
	"syscall"
)

func chown(name string, info os.FileInfo) error {
	stat := info.Sys().(*syscall.Stat_t)
	return os.Chown(name, int(stat.Uid), int(stat.Gid))
}
