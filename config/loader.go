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

type loadConfigFunc func(out any, opts ...utils.OptionExtender)

// loadConfigFromFiles
// load files priority:
// 1. conf flag
// 2. Files()
// 3. app.local.yml > app{.ENV}.yml (if ENV specified, e.g. app.ci.yml)
func loadConfigFromFiles(out any, opts ...utils.OptionExtender) {
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
		dirs := [...]string{"", "conf", "config", "appConfigs"}
		prefixes := [...]string{
			"app", "application",
			"setting", "settings", "appsetting", "appsettings",
			"conf", "config", "configure", "configuration",
		}
		hyphens := [...]string{".", "-", "_"}
		extensions := [...]string{"yaml", "yml", "json", "toml"}
		for _, dir := range dirs {
			for _, prefix := range prefixes {
				for _, hyphen := range hyphens {
					for _, ext := range extensions {
						localFilename := filepath.Join(env.WorkDir, dir, prefix+hyphen+"local."+ext)
						if _, err := os.Stat(localFilename); err == nil {
							files = append(files, localFilename)
							continue
						}
					}
				}
			}
		}

		for _, dir := range dirs {
			for _, prefix := range prefixes {
				for _, ext := range extensions {
					filename := filepath.Join(env.WorkDir, dir, prefix+"."+ext)
					if _, err := os.Stat(filename); err == nil {
						files = append(files, filename)
						continue
					}
				}
			}
		}
	}

	if err := NewDefaultLoader(files...).Unmarshal(out); err != nil {
		panic(errors.Errorf("parse config file error! %s", err))
	}
}
