// Fork from github.com/jinzhu/configor@v1.2.2-0.20230118083828-f7a0fc7c9fc6
// Here is the license:
//
// The MIT License (MIT)
//
// Copyright (c) 2013-NOW Jinzhu <wosmvp@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package configor

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"reflect"
	"regexp"
	"time"

	"github.com/wfusion/gofusion/common/utils"
)

type Configor struct {
	*Config
	configHash     map[string]string
	configModTimes map[string]time.Time

	statFunc func(name string) (fs.FileInfo, error)
	hashFunc func(name string) (string, error)
}

type Config struct {
	Environment        string
	ENVPrefix          string
	Debug              bool
	Verbose            bool
	Silent             bool
	AutoReload         bool
	AutoReloadInterval time.Duration
	AutoReloadCallback func(config any)

	// In case of json files, this field will be used only when compiled with
	// go 1.10 or later.
	// This field will be ignored when compiled with go versions lower than 1.10.
	ErrorOnUnmatchedKeys bool

	// You can use embed.FS or any other fs.FS to load configs from. Default - use "os" package
	FS fs.FS
}

// New initialize a Configor
func New(config *Config) *Configor {
	if config == nil {
		config = &Config{}
	}

	if os.Getenv("CONFIGOR_DEBUG_MODE") != "" {
		config.Debug = true
	}

	if os.Getenv("CONFIGOR_VERBOSE_MODE") != "" {
		config.Verbose = true
	}

	if os.Getenv("CONFIGOR_SILENT_MODE") != "" {
		config.Silent = true
	}

	if config.AutoReload && config.AutoReloadInterval == 0 {
		config.AutoReloadInterval = time.Second
	}

	cfg := &Configor{
		Config:         config,
		configHash:     make(map[string]string),
		configModTimes: make(map[string]time.Time),
		statFunc:       os.Stat,
		hashFunc: func(name string) (h string, err error) {
			var file fs.File
			if file, err = os.Open(name); err != nil {
				return
			}
			defer utils.CloseAnyway(file)
			sha256Hash := sha256.New()
			if _, err = io.Copy(sha256Hash, file); err != nil {
				return
			}
			h = string(sha256Hash.Sum(nil))
			return
		},
	}
	if cfg.FS != nil {
		cfg.statFunc = func(name string) (os.FileInfo, error) {
			return fs.Stat(cfg.FS, name)
		}
		cfg.hashFunc = func(name string) (h string, err error) {
			var file fs.File
			if file, err = cfg.FS.Open(name); err != nil {
				return
			}
			defer utils.CloseAnyway(file)
			sha256Hash := sha256.New()
			if _, err = io.Copy(sha256Hash, file); err != nil {
				return
			}
			h = string(sha256Hash.Sum(nil))
			return
		}
	}

	return cfg
}

var testRegexp = regexp.MustCompile(`_test|(\.test$)`)

// GetEnvironment get environment
func (c *Configor) GetEnvironment() string {
	if c.Environment == "" {
		if env := os.Getenv("CONFIGOR_ENV"); env != "" {
			return env
		}

		if testRegexp.MatchString(os.Args[0]) {
			return "test"
		}

		return "development"
	}
	return c.Environment
}

// GetErrorOnUnmatchedKeys returns a boolean indicating if an error should be
// thrown if there are keys in the config file that do not correspond to the
// config struct
func (c *Configor) GetErrorOnUnmatchedKeys() bool {
	return c.ErrorOnUnmatchedKeys
}

// Load will unmarshal configurations to struct from files that you provide
func (c *Configor) Load(config any, files ...string) (err error) {
	defaultValue := reflect.Indirect(reflect.ValueOf(config))
	if !defaultValue.CanAddr() {
		return fmt.Errorf("config %v should be addressable", config)
	}
	err, _ = c.load(config, false, files...)

	if c.Config.AutoReload {
		go func() {
			timer := time.NewTimer(c.Config.AutoReloadInterval)
			for range timer.C {
				reflectPtr := reflect.New(reflect.ValueOf(config).Elem().Type())
				reflectPtr.Elem().Set(defaultValue)

				var changed bool
				if err, changed = c.load(reflectPtr.Interface(), true, files...); err == nil && changed {
					reflect.ValueOf(config).Elem().Set(reflectPtr.Elem())
					if c.Config.AutoReloadCallback != nil {
						c.Config.AutoReloadCallback(config)
					}
				} else if err != nil {
					if !c.Silent {
						fmt.Printf("Failed to reload configuration from %v, got error %v\n", files, err)
					}
				}
				timer.Reset(c.Config.AutoReloadInterval)
			}
		}()
	}
	return
}

// ENV return environment
func ENV() string {
	return New(nil).GetEnvironment()
}

// Load will unmarshal configurations to struct from files that you provide
func Load(config any, files ...string) error {
	return New(nil).Load(config, files...)
}
