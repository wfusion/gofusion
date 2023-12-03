package config

import (
	"log"
	"os"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/wfusion/gofusion/common/env"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/internal/configor"
)

type loader struct {
	files []string
}

// NewDefaultLoader front files overwrite the backs
func NewDefaultLoader(files ...string) *loader {
	return &loader{
		files: files,
	}
}

// Unmarshal support required and default tag setting
// be carefully we can only assign default value when value is a pointer in map or slice
func (l *loader) Unmarshal(out any) (err error) {
	return configor.New(&configor.Config{
		Environment:        env.GetEnv(),
		ENVPrefix:          "",
		Debug:              false,
		Verbose:            false,
		Silent:             true,
		AutoReload:         true,
		AutoReloadInterval: 0,
		AutoReloadCallback: func(config any) {
			log.Printf("%v [Gofusion] Config auto reload config successfully => \n%s",
				syscall.Getpid(), utils.Must(yaml.Marshal(config)))
		},
		ErrorOnUnmatchedKeys: false,
		FS:                   nil,
	}).Load(out, l.files...)
}

var profile string

type loadConfigFunc func(out any, opts ...utils.OptionExtender)

func loadConfig(out any, opts ...utils.OptionExtender) {
	parseFlags()

	opt := utils.ApplyOptions[initOption](opts...)

	files := make([]string, 0, 2)
	switch {
	case len(customConfigPath) > 0:
		for _, p := range customConfigPath {
			files = append(files, filepath.Clean(p))
		}
	case len(opt.filenames) > 0:
		files = append(files, opt.filenames...)
	default:
		defaultPathPrefix := filepath.Join(env.WorkDir, "configs", "app.")
		defaultLocal1PathPrefix := filepath.Join(env.WorkDir, "configs", "app.local.")
		defaultLocal2PathPrefix := filepath.Join(env.WorkDir, "configs", "app_local.")
		defaultLocal3PathPrefix := filepath.Join(env.WorkDir, "configs", "app-local.")
		extensions := []string{"yaml", "yml", "json", "toml"}
		for _, ext := range extensions {
			localFilename := defaultLocal1PathPrefix + ext
			if _, err := os.Stat(localFilename); err == nil {
				files = append(files, localFilename)
				continue
			}
			localFilename = defaultLocal2PathPrefix + ext
			if _, err := os.Stat(localFilename); err == nil {
				files = append(files, localFilename)
				continue
			}
			localFilename = defaultLocal3PathPrefix + ext
			if _, err := os.Stat(localFilename); err == nil {
				files = append(files, localFilename)
				continue
			}
		}
		for _, ext := range extensions {
			defaultFilename := defaultPathPrefix + ext
			files = append(files, defaultFilename)
		}
	}

	if profile != "" {
		if err := configor.New(&configor.Config{Environment: profile}).Load(out, files...); err != nil {
			panic(errors.Errorf("parse config file of config env %s error: %v", profile, err))
		}
		return
	}

	if err := NewDefaultLoader(files...).Unmarshal(out); err != nil {
		panic(errors.Errorf("parse config file error! %s", err))
	}
}
