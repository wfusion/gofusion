package cases

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/wfusion/gofusion/cron"

	"github.com/wfusion/gofusion/async"
	"github.com/wfusion/gofusion/common/configor"
	"github.com/wfusion/gofusion/common/env"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/db"
	"github.com/wfusion/gofusion/lock"
	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/metrics"
	"github.com/wfusion/gofusion/mongo"
	"github.com/wfusion/gofusion/mq"
	"github.com/wfusion/gofusion/redis"
	"github.com/wfusion/gofusion/test/config"

	fmkCfg "github.com/wfusion/gofusion/config"

	_ "github.com/wfusion/gofusion/cache"
	_ "github.com/wfusion/gofusion/http"
	_ "github.com/wfusion/gofusion/i18n"
	_ "github.com/wfusion/gofusion/routine"
)

func TestExample(t *testing.T) {
	testingSuite := &Example{Test: config.T}
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
		files := []string{
			"app.local.yml",
			"app.yml",
			"app.json",
			"app.toml",
		}
		defer t.RawCopy(files, 1)()

		appSetting := new(appConf)
		defer fmkCfg.Registry.Init(&appSetting)()

		allConfigs := fmkCfg.Registry.GetAllConfigs()
		log.Info(context.Background(), "get all configs: %+v", allConfigs)
		log.Info(context.Background(), "get all configs json: %s", utils.MustJsonMarshal(allConfigs))
		log.Info(context.Background(), "get app name: %s", fmkCfg.Registry.AppName())
		log.Info(context.Background(), "get debug: %+v", fmkCfg.Registry.Debug())
	})
}

func (t *Example) TestLoadAppConfig() {
	t.Catch(func() {
		appSetting := new(appConf)
		defer fmkCfg.New(config.Component).Init(&appSetting, fmkCfg.Files(t.ConfigFiles()))()
		allConfigs := fmkCfg.Use(config.Component).GetAllConfigs()
		log.Info(context.Background(), "get all configs: %+v", allConfigs)
		log.Info(context.Background(), "get all configs json: %s", utils.MustJsonMarshal(allConfigs))
		log.Info(context.Background(), "get app name: %s", fmkCfg.Use(config.Component).AppName())
		log.Info(context.Background(), "get debug: %+v", fmkCfg.Use(config.Component).Debug())
	})
}

func (t *Example) TestLoadMultiTimes() {
	t.Catch(func() {
		appSetting := new(appConf)
		fmkCfg.New(config.Component).Init(&appSetting, fmkCfg.Files(t.ConfigFiles()))()

		appSetting = new(appConf)
		defer fmkCfg.New(config.Component).Init(&appSetting, fmkCfg.Files(t.ConfigFiles()))()
		allConfigs := fmkCfg.Use(config.Component).GetAllConfigs()
		log.Info(context.Background(), "get all configs json: %s", utils.MustJsonMarshal(allConfigs))
		log.Info(context.Background(), "get app name: %s", fmkCfg.Use(config.Component).AppName())
		log.Info(context.Background(), "get debug: %+v", fmkCfg.Use(config.Component).Debug())
	})
}

func (t *Example) TestLoadWithContext() {
	t.Catch(func() {
		ctx, cancel := context.WithCancel(context.Background())
		appSetting := new(appConf)
		defer fmkCfg.New(config.Component).Init(&appSetting, fmkCfg.Ctx(ctx), fmkCfg.Files(t.ConfigFiles()))()

		allConfigs := fmkCfg.Use(config.Component).GetAllConfigs()
		log.Info(context.Background(), "get all configs: %+v", allConfigs)
		log.Info(context.Background(), "get all configs json: %s", utils.MustJsonMarshal(allConfigs))
		log.Info(context.Background(), "get app name: %s", fmkCfg.Use(config.Component).AppName())
		log.Info(context.Background(), "get debug: %+v", fmkCfg.Use(config.Component).Debug())

		cancel()
		time.Sleep(time.Second)
	})
}

func (t *Example) TestLoadWithLoader() {
	t.Catch(func() {
		appSetting := new(appConf)
		defer fmkCfg.New(config.Component).Init(&appSetting, fmkCfg.Files(t.ConfigFiles()),
			fmkCfg.Loader(func(a any, opts ...utils.OptionExtender) {
				log.Info(context.Background(), "enter custom loader")
				defer log.Info(context.Background(), "exit custom loader")
				files := make([]string, 0, 2)
				localConfPath := env.WorkDir + fmt.Sprintf("/configs/%s.app.local.yml", config.Component)
				defaultConfPath := env.WorkDir + fmt.Sprintf("/configs/%s.app.yml", config.Component)
				if _, err := os.Stat(localConfPath); err == nil {
					files = append(files, localConfPath)
				}
				files = append(files, defaultConfPath)
				t.Require().NoError(configor.New(&configor.Config{}).Load(a, files...))
			}))()

		allConfigs := fmkCfg.Use(config.Component).GetAllConfigs()
		log.Info(context.Background(), "get all configs: %+v", allConfigs)
		log.Info(context.Background(), "get all configs json: %s", utils.MustJsonMarshal(allConfigs))
		log.Info(context.Background(), "get app name: %s", fmkCfg.Use(config.Component).AppName())
		log.Info(context.Background(), "get debug: %+v", fmkCfg.Use(config.Component).Debug())
	})
}

func (t *Example) TestConcurrency() {
	t.Catch(func() {
		files := []string{
			"app.local.yml",
			"app.yml",
			"app.json",
			"app.toml",
		}
		defer t.RawCopy(files, 1)()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		testComponentsFn := func(appName string) {
			lock.Use("default", lock.AppName(appName))
			db.Use(ctx, "read", db.AppName(appName))
			db.Use(ctx, "write", db.AppName(appName))
			mongo.Use(ctx, "default", mongo.AppName(appName))
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
				defer fmkCfg.New(config.Component).Init(&appSetting, fmkCfg.Files(t.ConfigFiles()))()
				testComponentsFn(config.Component)
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()

				time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)

				appSetting := new(appConf)
				defer fmkCfg.Registry.Init(&appSetting)()
				testComponentsFn("")
			}()
		}
		wg.Wait()
	})
}
