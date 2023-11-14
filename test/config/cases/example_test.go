package cases

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/async"
	"github.com/wfusion/gofusion/common/env"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/cron"
	"github.com/wfusion/gofusion/db"
	"github.com/wfusion/gofusion/internal/configor"
	"github.com/wfusion/gofusion/lock"
	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/metrics"
	"github.com/wfusion/gofusion/mongo"
	"github.com/wfusion/gofusion/mq"
	"github.com/wfusion/gofusion/redis"
	"github.com/wfusion/gofusion/routine"
	"github.com/wfusion/gofusion/test/config"

	fusCfg "github.com/wfusion/gofusion/config"

	_ "github.com/wfusion/gofusion/cache"
	_ "github.com/wfusion/gofusion/http"
	_ "github.com/wfusion/gofusion/i18n"
)

func TestExample(t *testing.T) {
	testingSuite := &Example{Test: new(config.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Example struct {
	*config.Test
}

func (t *Example) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Example) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Example) TestDefault() {
	t.Catch(func() {
		defer t.RawCopy(t.AllConfigFiles(), 1)()

		appSetting := new(appConf)
		defer fusCfg.Registry.Init(&appSetting)()

		allConfigs := fusCfg.Registry.GetAllConfigs()
		log.Info(context.Background(), "get all configs: %+v", allConfigs)
		log.Info(context.Background(), "get all configs json: %s", utils.MustJsonMarshal(allConfigs))
		log.Info(context.Background(), "get app name: %s", fusCfg.Registry.AppName())
		log.Info(context.Background(), "get debug: %+v", fusCfg.Registry.Debug())
	})
}

func (t *Example) TestRequired() {
	t.Catch(func() {
		files := []string{
			"app.required.local.yml",
			"app.required.yml",
		}
		defer t.RawCopy(files, 1)()

		for i := 0; i < len(files); i++ {
			files[i] = filepath.Join(env.WorkDir, "configs", files[i])
		}

		appSetting := new(appConf)
		defer fusCfg.Registry.Init(&appSetting, fusCfg.Files(files))()
		allConfigs := fusCfg.Registry.GetAllConfigs()
		log.Info(context.Background(), "get all configs: %+v", allConfigs)
		log.Info(context.Background(), "get all configs json: %s", utils.MustJsonMarshal(allConfigs))
		log.Info(context.Background(), "get app name: %s", fusCfg.Registry.AppName())
		log.Info(context.Background(), "get debug: %+v", fusCfg.Registry.Debug())

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		lock.Use("default")
		db.Use(ctx, "default")
		mongo.Use("default")
		redis.Use(ctx, "default")
		log.Use("default")
		mq.Use("default")
		async.C("default")
		async.P("default")
		cron.Use("default")
		metrics.Use("prometheus", "test_config_required")
	})
}

func (t *Example) TestWithoutFiles() {
	t.Catch(func() {
		appSetting := new(appConf)
		defer fusCfg.Registry.Init(&appSetting, fusCfg.Files(nil))()
		allConfigs := fusCfg.Registry.GetAllConfigs()
		log.Info(context.Background(), "get all configs: %+v", allConfigs)
		log.Info(context.Background(), "get all configs json: %s", utils.MustJsonMarshal(allConfigs))
		log.Info(context.Background(), "get app name: %s", fusCfg.Registry.AppName())
		log.Info(context.Background(), "get debug: %+v", fusCfg.Registry.Debug())
		routine.Go(func() {})
	})
}

func (t *Example) TestLoadAppConfig() {
	t.Catch(func() {
		appSetting := new(appConf)
		defer fusCfg.New(t.AppName()).Init(&appSetting, fusCfg.Files(t.ConfigFiles()))()
		allConfigs := fusCfg.Use(t.AppName()).GetAllConfigs()
		log.Info(context.Background(), "get all configs: %+v", allConfigs)
		log.Info(context.Background(), "get all configs json: %s", utils.MustJsonMarshal(allConfigs))
		log.Info(context.Background(), "get app name: %s", fusCfg.Use(t.AppName()).AppName())
		log.Info(context.Background(), "get debug: %+v", fusCfg.Use(t.AppName()).Debug())
	})
}

func (t *Example) TestLoadMultiTimes() {
	t.Catch(func() {
		appSetting := new(appConf)
		fusCfg.New(t.AppName()).Init(&appSetting, fusCfg.Files(t.ConfigFiles()))()

		appSetting = new(appConf)
		defer fusCfg.New(t.AppName()).Init(&appSetting, fusCfg.Files(t.ConfigFiles()))()
		allConfigs := fusCfg.Use(t.AppName()).GetAllConfigs()
		log.Info(context.Background(), "get all configs json: %s", utils.MustJsonMarshal(allConfigs))
		log.Info(context.Background(), "get app name: %s", fusCfg.Use(t.AppName()).AppName())
		log.Info(context.Background(), "get debug: %+v", fusCfg.Use(t.AppName()).Debug())
	})
}

func (t *Example) TestLoadWithContext() {
	t.Catch(func() {
		ctx, cancel := context.WithCancel(context.Background())
		appSetting := new(appConf)
		defer fusCfg.New(t.AppName()).Init(&appSetting, fusCfg.Ctx(ctx), fusCfg.Files(t.ConfigFiles()))()

		allConfigs := fusCfg.Use(t.AppName()).GetAllConfigs()
		log.Info(context.Background(), "get all configs: %+v", allConfigs)
		log.Info(context.Background(), "get all configs json: %s", utils.MustJsonMarshal(allConfigs))
		log.Info(context.Background(), "get app name: %s", fusCfg.Use(t.AppName()).AppName())
		log.Info(context.Background(), "get debug: %+v", fusCfg.Use(t.AppName()).Debug())

		cancel()
		time.Sleep(time.Second)
	})
}

func (t *Example) TestLoadWithLoader() {
	t.Catch(func() {
		appSetting := new(appConf)
		defer fusCfg.New(t.AppName()).Init(&appSetting, fusCfg.Files(t.ConfigFiles()),
			fusCfg.Loader(func(a any, opts ...utils.OptionExtender) {
				log.Info(context.Background(), "enter custom loader")
				defer log.Info(context.Background(), "exit custom loader")
				files := make([]string, 0, 2)
				localConfPath := filepath.Join(env.WorkDir + "configs" +
					fmt.Sprintf("%s.app.local.yml", t.AppName()))
				defaultConfPath := filepath.Join(env.WorkDir + "configs" +
					fmt.Sprintf("%s.app.yml", t.AppName()))
				if _, err := os.Stat(localConfPath); err == nil {
					files = append(files, localConfPath)
				}
				files = append(files, defaultConfPath)
				t.Require().NoError(configor.New(&configor.Config{}).Load(a, files...))
			}))()

		allConfigs := fusCfg.Use(t.AppName()).GetAllConfigs()
		log.Info(context.Background(), "get all configs: %+v", allConfigs)
		log.Info(context.Background(), "get all configs json: %s", utils.MustJsonMarshal(allConfigs))
		log.Info(context.Background(), "get app name: %s", fusCfg.Use(t.AppName()).AppName())
		log.Info(context.Background(), "get debug: %+v", fusCfg.Use(t.AppName()).Debug())
	})
}

func (t *Example) TestConcurrency() {
	t.Catch(func() {
		defer t.RawCopy(t.AllConfigFiles(), 1)()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		testComponentsFn := func(appName string) {
			lock.Use("default", lock.AppName(appName))
			db.Use(ctx, "read", db.AppName(appName))
			db.Use(ctx, "write", db.AppName(appName))
			mongo.Use("default", mongo.AppName(appName))
			redis.Use(ctx, "default", redis.AppName(appName))
			log.Use("default", log.AppName(appName))
			mq.Use("mysql", mq.AppName(appName))
			async.C("default", async.AppName(appName))
			async.P("default", async.AppName(appName))
			cron.Use("default", cron.AppName(appName))
			metrics.Use("prometheus", "test_config_concurrency", metrics.AppName(appName))
		}

		wg := new(sync.WaitGroup)
		for i := 0; i < 1000; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)

				appSetting := new(appConf)
				defer fusCfg.New(t.AppName()).Init(&appSetting, fusCfg.Files(t.ConfigFiles()))()
				testComponentsFn(t.AppName())
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()

				time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)

				appSetting := new(appConf)
				defer fusCfg.Registry.Init(&appSetting)()
				testComponentsFn("")
			}()
		}
		wg.Wait()
	})
}
