package env

import (
	"os"
	"path/filepath"
)

const (
	Dev     = "dev"
	Online  = "online"
	Staging = "staging"
	CI      = "ci"
)

var (
	WorkDir string
	env     string
	svcName string
)

// GetEnv get environment ENV
func GetEnv() string {
	if env != "" {
		return env
	}
	if env = os.Getenv("ENV"); env == "" {
		env = Dev
	}
	return env
}

// IsDEV is dev
func IsDEV() bool {
	return GetEnv() == Dev
}

// IsOnline is online
func IsOnline() bool {
	return GetEnv() == Online
}

// IsStaging is staging
func IsStaging() bool {
	return GetEnv() == Staging
}

// IsCI is ci
func IsCI() bool {
	return GetEnv() == CI
}

func SetSvcName(name string) {
	svcName = name
}

func SvcName() string {
	if svcName != "" {
		return svcName
	}
	if svcName = os.Getenv("SVC_NAME"); svcName != "" {
		return svcName
	}
	if svcName = os.Getenv("SERVICE_NAME"); svcName != "" {
		return svcName
	}
	return ""
}

func init() {
	GetEnv()
	SvcName()

	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	WorkDir = dir
}
