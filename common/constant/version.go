package constant

import (
	"runtime"
)

const (
	OS   = runtime.GOOS
	Arch = runtime.GOARCH
)

var (
	GoVersion = runtime.Version()
)
