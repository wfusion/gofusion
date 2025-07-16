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
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path"
	"reflect"
	"runtime/debug"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/serialize/json"
)

// UnmatchedTomlKeysError errors are returned by the Load function when
// ErrorOnUnmatchedKeys is set to true and there are unmatched keys in the input
// toml config file. The string returned by Error() contains the names of the
// missing keys.
type UnmatchedTomlKeysError struct {
	Keys []toml.Key
}

func (e *UnmatchedTomlKeysError) Error() string {
	return fmt.Sprintf("There are keys in the config file that do not match any field in the given struct: %v", e.Keys)
}

func (c *Configor) getENVPrefix(config any) string {
	if c.Config.ENVPrefix == "" {
		if prefix := os.Getenv("CONFIGOR_ENV_PREFIX"); prefix != "" {
			return prefix
		}
		return "Configor"
	}
	return c.Config.ENVPrefix
}

func (c *Configor) getConfigurationFileWithENVPrefix(file, env string) (string, time.Time, string, error) {
	var (
		envFile string
		extname = path.Ext(file)
		hyphens = [...]string{".", "-", "_"}
	)

	for _, hyphen := range hyphens {
		if extname == "" {
			envFile = fmt.Sprintf("%v%s%v", file, hyphen, env)
		} else {
			envFile = fmt.Sprintf("%v%s%v%v", strings.TrimSuffix(file, extname), hyphen, env, extname)
		}

		if fileInfo, err := c.statFunc(envFile); err == nil && fileInfo.Mode().IsRegular() {
			fileHash, _ := c.hashFunc(envFile)
			return envFile, fileInfo.ModTime(), fileHash, nil
		}
	}

	return "", time.Now(), "", fmt.Errorf("failed to find file %v", file)
}

func (c *Configor) getConfigurationFiles(watchMode bool, files ...string) (
	[]string, map[string]time.Time, map[string]string) {
	resultKeys := make([]string, 0, len(files))
	hashResult := make(map[string]string, len(files))
	modTimeResult := make(map[string]time.Time, len(files))
	if !watchMode && (c.Config.Debug || c.Config.Verbose) {
		fmt.Printf("Current environment: '%v'\n", c.GetEnvironment())
	}

	for i := len(files) - 1; i >= 0; i-- {
		foundFile := false
		file := files[i]

		// check configuration
		if fileInfo, err := c.statFunc(file); err == nil && fileInfo.Mode().IsRegular() {
			foundFile = true
			resultKeys = append(resultKeys, file)
			modTimeResult[file] = fileInfo.ModTime()
			if hash, err := c.hashFunc(file); err == nil {
				hashResult[file] = hash
			}
		}

		// check configuration with env
		if file, modTime, hash, err := c.getConfigurationFileWithENVPrefix(file, c.GetEnvironment()); err == nil {
			foundFile = true
			resultKeys = append(resultKeys, file)
			modTimeResult[file] = modTime
			if hash != "" {
				hashResult[file] = hash
			}
		}

		// check example configuration
		if !foundFile {
			if example, modTime, hash, err := c.getConfigurationFileWithENVPrefix(file, "example"); err == nil {
				if !watchMode && !c.Silent {
					log.Printf("Failed to find configuration %v, using example file %v\n", file, example)
				}
				resultKeys = append(resultKeys, example)
				modTimeResult[example] = modTime
				if hash != "" {
					hashResult[file] = hash
				}
			} else if !c.Silent {
				fmt.Printf("Failed to find configuration %v\n", file)
			}
		}
	}
	return resultKeys, modTimeResult, hashResult
}

func (c *Configor) processFile(config any, file string, errorOnUnmatchedKeys bool) error {
	readFile := ioutil.ReadFile
	if c.FS != nil {
		readFile = func(filename string) ([]byte, error) {
			return fs.ReadFile(c.FS, filename)
		}
	}
	data, err := readFile(file)
	if err != nil {
		return err
	}

	switch {
	case strings.HasSuffix(file, ".yaml") || strings.HasSuffix(file, ".yml"):
		if errorOnUnmatchedKeys {
			decoder := yaml.NewDecoder(bytes.NewBuffer(data))
			decoder.KnownFields(true)
			return decoder.Decode(config)
		}
		return yaml.Unmarshal(data, config)
	case strings.HasSuffix(file, ".toml"):
		return unmarshalToml(data, config, errorOnUnmatchedKeys)
	case strings.HasSuffix(file, ".json"):
		return unmarshalJSON(data, config, errorOnUnmatchedKeys)
	default:
		if err := unmarshalToml(data, config, errorOnUnmatchedKeys); err == nil {
			return nil
		} else if errUnmatchedKeys, ok := err.(*UnmatchedTomlKeysError); ok {
			return errUnmatchedKeys
		}

		if err := unmarshalJSON(data, config, errorOnUnmatchedKeys); err == nil {
			return nil
		} else if strings.Contains(err.Error(), "json: unknown field") {
			return err
		}

		var yamlError error
		if errorOnUnmatchedKeys {
			decoder := yaml.NewDecoder(bytes.NewBuffer(data))
			decoder.KnownFields(true)
			yamlError = decoder.Decode(config)
		} else {
			yamlError = yaml.Unmarshal(data, config)
		}

		if yamlError == nil {
			return nil
		} else if yErr, ok := yamlError.(*yaml.TypeError); ok {
			return yErr
		}

		return errors.New("failed to decode config")
	}
}

func unmarshalToml(data []byte, config any, errorOnUnmatchedKeys bool) error {
	metadata, err := toml.Decode(string(data), config)
	if err == nil && len(metadata.Undecoded()) > 0 && errorOnUnmatchedKeys {
		return &UnmatchedTomlKeysError{Keys: metadata.Undecoded()}
	}
	return err
}

// unmarshalJSON unmarshals the given data into the config interface.
// If the errorOnUnmatchedKeys boolean is true, an error will be returned if there
// are keys in the data that do not match fields in the config interface.
func unmarshalJSON(data []byte, config any, errorOnUnmatchedKeys bool) error {
	reader := strings.NewReader(string(data))
	decoder := json.NewDecoder(reader)

	if errorOnUnmatchedKeys {
		decoder.DisallowUnknownFields()
	}

	err := decoder.Decode(config)
	if err != nil && err != io.EOF {
		return err
	}
	return nil
}

func getPrefixForStruct(prefixes []string, fieldStruct *reflect.StructField) []string {
	if fieldStruct.Anonymous && fieldStruct.Tag.Get("anonymous") == "true" {
		return prefixes
	}
	return append(prefixes, fieldStruct.Name)
}

func (c *Configor) processTags(config any, prefixes ...string) error {
	configValue := reflect.Indirect(reflect.ValueOf(config))
	if configValue.Kind() != reflect.Struct {
		return errors.New("invalid config, should be struct")
	}

	configType := configValue.Type()
	for i := 0; i < configType.NumField(); i++ {
		var (
			envNames    []string
			fieldStruct = configType.Field(i)
			field       = configValue.Field(i)
			envName     = fieldStruct.Tag.Get("env") // read configuration from shell env
		)

		if !field.CanAddr() || !field.CanInterface() {
			continue
		}

		if envName == "" {
			envNames = append(envNames,
				strings.Join(append(prefixes, fieldStruct.Name), "_")) // Configor_DB_Name
			envNames = append(envNames,
				strings.ToUpper(strings.Join(append(prefixes, fieldStruct.Name), "_"))) // CONFIGOR_DB_NAME
		} else {
			envNames = []string{envName}
		}

		if c.Config.Verbose {
			fmt.Printf("Trying to load struct `%v`'s field `%v` from env %v\n",
				configType.Name(), fieldStruct.Name, strings.Join(envNames, ", "))
		}

		// Load From Shell ENV
		for _, env := range envNames {
			if value := os.Getenv(env); value != "" {
				if c.Config.Debug || c.Config.Verbose {
					fmt.Printf("Loading configuration for struct `%v`'s field `%v` from env %v...\n",
						configType.Name(), fieldStruct.Name, env)
				}

				switch reflect.Indirect(field).Kind() {
				case reflect.Bool:
					switch strings.ToLower(value) {
					case "", "0", "f", "false":
						field.Set(reflect.ValueOf(false))
					default:
						field.Set(reflect.ValueOf(true))
					}
				case reflect.String:
					field.Set(reflect.ValueOf(value))
				default:
					if err := yaml.Unmarshal([]byte(value), field.Addr().Interface()); err != nil {
						return err
					}
				}
				break
			}
		}

		if isBlank := reflect.DeepEqual(field.Interface(), reflect.Zero(field.Type()).Interface()); isBlank &&
			fieldStruct.Tag.Get("required") == "true" {
			// return error if it is required but blank
			return errors.New(fieldStruct.Name + " is required, but blank")
		}

		field = utils.IndirectValue(field)
		if field.Kind() == reflect.Struct {
			if err := c.processTags(field.Addr().Interface(),
				getPrefixForStruct(prefixes, &fieldStruct)...); err != nil {
				return err
			}
		}

		if field.Kind() == reflect.Slice {
			if arrLen := field.Len(); arrLen > 0 {
				for i := 0; i < arrLen; i++ {
					if reflect.Indirect(field.Index(i)).Kind() == reflect.Struct {
						if err := c.processTags(field.Index(i).Addr().Interface(),
							append(getPrefixForStruct(prefixes, &fieldStruct), fmt.Sprint(i))...); err != nil {
							return err
						}
					}
				}
			} else {
				defer func(field reflect.Value, fieldStruct reflect.StructField) {
					if !configValue.IsZero() {
						// load slice from env
						newVal := reflect.New(field.Type().Elem()).Elem()
						if newVal.Kind() == reflect.Struct {
							idx := 0
							for {
								newVal = reflect.New(field.Type().Elem()).Elem()
								if err := c.processTags(newVal.Addr().Interface(), append(
									getPrefixForStruct(prefixes, &fieldStruct), fmt.Sprint(idx))...); err != nil {
									return // err
								} else if reflect.DeepEqual(newVal.Interface(),
									reflect.New(field.Type().Elem()).Elem().Interface()) {
									break
								} else {
									idx++
									field.Set(reflect.Append(field, newVal))
								}
							}
						}
					}
				}(field, fieldStruct)
			}
		}
	}
	return nil
}

func (c *Configor) load(config any, watchMode bool, files ...string) (err error, changed bool) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("panic %s =>\n%s", r, debug.Stack())
			return
		}
		if c.Config.Debug || c.Config.Verbose {
			if err != nil {
				fmt.Printf("Failed to load configuration from %v, got %v\n", files, err)
			}

			fmt.Printf("Configuration:\n  %#v\n", config)
		}
	}()

	configFiles, configModTimeMap, hashMap := c.getConfigurationFiles(watchMode, files...)
	if watchMode && len(configModTimeMap) == len(c.configModTimes) && len(hashMap) == len(c.configHash) {
		var changed bool
		for f, curModTime := range configModTimeMap {
			curHash := hashMap[f]
			preHash, ok1 := c.configHash[f]
			preModTime, ok2 := c.configModTimes[f]
			if changed = !ok1 || !ok2 || curModTime.After(preModTime) || curHash != preHash; changed {
				break
			}
		}

		if !changed {
			return nil, false
		}
	}

	type withBeforeCallback interface {
		BeforeLoad(opts ...utils.OptionExtender)
	}
	type withAfterCallback interface {
		AfterLoad(opts ...utils.OptionExtender)
	}
	if cb, ok := config.(withBeforeCallback); ok {
		cb.BeforeLoad()
	}
	if cb, ok := config.(withAfterCallback); ok {
		defer cb.AfterLoad()
	}

	for _, file := range configFiles {
		if c.Config.Debug || c.Config.Verbose {
			fmt.Printf("Loading configurations from file '%v'...\n", file)
		}
		if err = c.processFile(config, file, c.GetErrorOnUnmatchedKeys()); err != nil {
			return err, true
		}
	}

	// process defaults after process file because map struct should be assigned first
	_ = utils.ParseTag(config, utils.ParseTagName("default"), utils.ParseTagUnmarshalType(utils.MarshalTypeYaml))

	// process file again to ensure read config from file
	for _, file := range configFiles {
		if c.Config.Debug || c.Config.Verbose {
			fmt.Printf("Loading configurations from file '%v'...\n", file)
		}
		if err = c.processFile(config, file, c.GetErrorOnUnmatchedKeys()); err != nil {
			return err, true
		}
	}

	c.configHash = hashMap
	c.configModTimes = configModTimeMap

	if prefix := c.getENVPrefix(config); prefix == "-" {
		err = c.processTags(config)
	} else {
		err = c.processTags(config, prefix)
	}

	// process defaults
	_ = utils.ParseTag(config, utils.ParseTagName("default"), utils.ParseTagUnmarshalType(utils.MarshalTypeYaml))

	return err, true
}
