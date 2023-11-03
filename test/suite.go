package test

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/common/env"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/log"
)

var (
	S = &Suite{Suite: new(suite.Suite)}
)

type Suite struct {
	*suite.Suite

	cleanLock sync.RWMutex
	cleanup   []func()
}

func (t *Suite) SetupSuite() {
	t.Catch(func() {
		log.Info(context.Background(), "============ setup test suite ============")
	})
}

func (t *Suite) TearDownSuite() {
	t.Catch(func() {
		log.Info(context.Background(), "============ tear down test suite ============")
		t.cleanLock.RLock()
		defer t.cleanLock.RUnlock()
		for i := len(t.cleanup) - 1; i >= 0; i-- {
			t.cleanup[i]()
		}
	})
}

func (t *Suite) Catch(f func()) {
	defer func() {
		if r := recover(); r != nil {
			t.FailNow(fmt.Sprintf("panic: %s =>\n%s", r, debug.Stack()))
		}
	}()

	f()
}

func (t *Suite) RawCopy(filenames []string, stackSkip int) (cleanFn func()) {
	stackSkip++
	t.clearAllFiles(filenames, stackSkip)
	for _, filename := range filenames {
		t.copyFile(filename, filename, stackSkip)
	}
	return func() {
		t.clearAllFiles(filenames, stackSkip)
	}
}

func (t *Suite) Copy(src []string, stackSkip int) (cleanFn func()) {
	_, filename, _, ok := runtime.Caller(stackSkip)
	t.True(ok)
	component := t.moduleName(filename)
	fileMapping, others := t.mappingFilenames(component, src)
	// allFilenames := append(others, utils.MapValues(fileMapping)...)

	stackSkip++
	// t.clearAllFiles(allFilenames, stackSkip)
	for src, dst := range fileMapping {
		t.copyFile(dst, src, stackSkip)
	}
	for _, filename := range others {
		t.copyFile(filename, filename, stackSkip)
	}
	return func() {
		// t.clearAllFiles(allFilenames, stackSkip)
	}
}

func (t *Suite) Init(src []string, stackSkip int) (cleanFn func()) {
	_, filename, _, ok := runtime.Caller(stackSkip)
	t.True(ok)
	component := t.moduleName(filename)
	fileMapping, _ := t.mappingFilenames(component, src)
	cfgNames := utils.MapValues(fileMapping)
	for i := 0; i < len(cfgNames); i++ {
		cfgNames[i] = env.WorkDir + "/configs/" + cfgNames[i]
	}

	appCfg := &struct{}{}
	gracefullyExitFn := config.New(component).Init(&appCfg, config.Files(cfgNames))
	return func() {
		gracefullyExitFn()
	}
}

func (t *Suite) Cleanup(c func()) {
	if c == nil {
		return
	}
	t.cleanLock.Lock()
	defer t.cleanLock.Unlock()
	t.cleanup = append(t.cleanup, c)
}

func (t *Suite) isConfigFile(name string) (ok bool) {
	return strings.Contains(name, "app")
}

func (t *Suite) mappingFilenames(component string, filenames []string) (cfgMapping map[string]string, others []string) {
	others = make([]string, 0, len(filenames))
	cfgMapping = make(map[string]string, len(filenames))
	for _, filename := range filenames {
		if t.isConfigFile(filename) {
			cfgMapping[filename] = component + "." + filename
		} else {
			others = append(others, filename)
		}
	}
	return
}

func (t *Suite) moduleName(filename string) (name string) {
	fpath := "github.com/wfusion/gofusion/test/"
	moduleDir := filepath.Dir(filename)
	component := moduleDir[strings.Index(moduleDir, fpath):]
	return component[len(fpath):]
}

func (t *Suite) clearAllFiles(filenames []string, stackSkip int) {
	// locate project conf path & current conf path
	_, filename, _, ok := runtime.Caller(stackSkip)
	t.Require().True(ok)
	projectRoot := filepath.Dir(filename)

	projectConfDir := filepath.Join(strings.TrimSuffix(projectRoot, "/cases"), "configs")
	currentConfDir := filepath.Join(env.WorkDir, "configs")
	if utils.IsStrBlank(currentConfDir) || projectConfDir == currentConfDir {
		return
	}

	files, err := filepath.Glob(currentConfDir + "/*")
	t.Require().NoError(err)

	toBeDeleted := func(filePath string) (ok bool) {
		filename := filepath.Base(filePath)
		for _, name := range filenames {
			if strings.EqualFold(filename, name) {
				return true
			}
		}
		return
	}

	for _, filePath := range files {
		f, err := os.Stat(filePath)
		if err != nil {
			continue
		}
		if !toBeDeleted(filePath) {
			continue
		}
		if f.IsDir() {
			t.Require().NoError(os.RemoveAll(filePath))
		} else {
			t.Require().NoError(os.Remove(filePath))
		}
	}
}

func (t *Suite) copyFile(to, from string, stackSkip int) {
	_, filename, _, ok := runtime.Caller(stackSkip)
	t.Require().True(ok)
	projectRoot := filepath.Dir(filename)

	projectConfDir := filepath.Join(strings.TrimSuffix(projectRoot, "/cases"), "configs")
	currentConfDir := filepath.Join(env.WorkDir, "configs")
	if utils.IsStrBlank(currentConfDir) || projectConfDir == currentConfDir {
		return
	}

	// create current conf dir
	err := os.MkdirAll(currentConfDir, os.ModePerm)
	if err != nil {
		t.Require().ErrorIs(err, os.ErrExist)
	}

	// copy files
	copyFileFn := func(dst, src string) {
		from, err := os.Open(src)
		t.Require().NoError(err)
		defer func() { t.Nil(from.Close()) }()

		to, err := os.Create(dst)
		t.Require().NoError(err)
		defer func() { t.Nil(to.Close()) }()

		_, err = io.Copy(to, from)
		t.Require().NoError(err)
	}

	currentConfPath := filepath.Join(currentConfDir, to)
	projectConfPath := filepath.Join(projectConfDir, from)

	var f os.FileInfo
	if f, err = os.Stat(projectConfPath); err == nil && !f.IsDir() {
		copyFileFn(currentConfPath, projectConfPath)
		return
	}
	t.Require().ErrorIs(err, os.ErrNotExist)

	files, err := filepath.Glob(projectConfPath)
	t.Require().NoError(err)
	for _, filePath := range files {
		filePath = strings.TrimPrefix(filePath, projectConfDir)

		// mkdir -p
		subDir := filepath.Dir(filepath.Join(currentConfDir, filePath))
		if err := os.MkdirAll(subDir, os.ModePerm); err != nil {
			t.Require().ErrorIs(err, os.ErrExist)
		}

		// skip dir
		projectConfPath := filepath.Join(projectConfDir, filePath)
		currentConfPath := filepath.Join(currentConfDir, filePath)
		if f, err = os.Stat(projectConfPath); err == nil && !f.IsDir() {
			copyFileFn(projectConfPath, currentConfPath)
			continue
		}
		t.Require().ErrorIs(err, os.ErrNotExist)
	}
}
