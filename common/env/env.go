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

	Local = "local"
	Test  = "test"
	Sit   = "sit"
	Prod  = "prod"
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

func IsDEV() bool     { return GetEnv() == Dev }
func IsOnline() bool  { return GetEnv() == Online }
func IsStaging() bool { return GetEnv() == Staging }
func IsCI() bool      { return GetEnv() == CI }
func IsLocal() bool   { return GetEnv() == Local }
func IsTest() bool    { return GetEnv() == Test }
func IsSIT() bool     { return GetEnv() == Sit }
func IsProd() bool    { return GetEnv() == Prod }

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
	if svcName = os.Getenv("APP_NAME"); svcName != "" {
		return svcName
	}
	if svcName = os.Getenv("APPLICATION_NAME"); svcName != "" {
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
