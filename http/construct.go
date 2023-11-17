package http

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"sync"
	"syscall"
	"time"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	"golang.org/x/text/language"

	"github.com/wfusion/gofusion/common/di"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/clone"
	"github.com/wfusion/gofusion/common/utils/inspect"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/http/middleware"
	"github.com/wfusion/gofusion/i18n"

	fusLog "github.com/wfusion/gofusion/log"

	_ "github.com/wfusion/gofusion/log/customlogger"
)

var (
	Router IRouter

	locker          sync.RWMutex
	routers         = map[string]IRouter{}
	appClientMap    = map[string]map[string]*resty.Client{}
	appClientCfgMap = map[string]map[string]*cfg{}
)

func Construct(ctx context.Context, conf Conf, opts ...utils.OptionExtender) func() {
	opt := utils.ApplyOptions[config.InitOption](opts...)
	optU := utils.ApplyOptions[useOption](opts...)
	if opt.AppName == "" {
		opt.AppName = optU.appName
	}

	var logger resty.Logger
	if utils.IsStrNotBlank(conf.Logger) {
		logger = reflect.New(inspect.TypeOf(conf.Logger)).Interface().(resty.Logger)
		if custom, ok := logger.(customLogger); ok {
			l := fusLog.Use(conf.LogInstance, fusLog.AppName(opt.AppName))
			custom.Init(l, opt.AppName)
		}
	}

	exitRouterFn := addRouter(ctx, conf, logger, opt)
	exitI18nFn := addI18n(opt)
	exitClientFn := addClient(ctx, conf, logger, opt)

	// gracefully exit outside gofusion
	return func() {
		exitClientFn()
		exitRouterFn()
		exitI18nFn()
	}
}

func addRouter(ctx context.Context, conf Conf, logger resty.Logger, opt *config.InitOption) func() {
	engine := gin.New()
	engine.Use(
		gin.Recovery(),
		middleware.Gateway,
		middleware.Trace(),
		middleware.Logging(ctx, opt.AppName, logger),
		middleware.Cors(),
		middleware.XSS(conf.XSSWhiteURLList),
		middleware.Recover(opt.AppName, logger),
	)
	if config.Use(opt.AppName).Debug() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	if !conf.ColorfulConsole {
		gin.DisableConsoleColor()
	} else {
		gin.ForceConsoleColor()
	}

	tag := i18n.DefaultLang(i18n.AppName(opt.AppName))
	engine.NoMethod(func(c *gin.Context) {
		c.Status(http.StatusMethodNotAllowed)
		msg := fmt.Sprintf("找不到该方法, Method: %s", c.Request.Method)
		if tag != language.Chinese {
			msg = fmt.Sprintf("Cannot find method: %s", c.Request.Method)
		}

		rspError(c, opt.AppName, -1, nil, 0, 0, msg)
	})
	engine.NoRoute(func(c *gin.Context) {
		c.Status(http.StatusNotFound)
		msg := fmt.Sprintf("找不到该内容, URL: %s", c.Request.URL.String())
		if tag != language.Chinese {
			msg = fmt.Sprintf("Cannot find URL content: %s", c.Request.URL.String())
		}
		rspError(c, opt.AppName, -1, nil, 0, 0, msg)
	})

	if conf.Pprof {
		pprof.Register(engine)
	}
	instance := newRouter(ctx, engine, opt.AppName, conf.SuccessCode)

	locker.Lock()
	defer locker.Unlock()
	if len(conf.Asynq) > 0 {
		initAsynq(ctx, opt.AppName, instance, conf.Asynq)
	}
	if _, ok := routers[opt.AppName]; ok {
		panic(errors.Errorf("duplicated http name: %s", opt.AppName))
	}
	routers[opt.AppName] = instance
	if opt.AppName == "" {
		Router = instance
	}

	if opt.DI != nil {
		opt.DI.MustProvide(func() IRouter { return Use(AppName(opt.AppName)) })
	}

	return func() {
		locker.Lock()
		defer locker.Unlock()
		if routers != nil {
			if router, ok := routers[opt.AppName]; ok {
				router.shutdown()
				wg := new(sync.WaitGroup)
				wg.Add(1)
				go func() { defer wg.Done(); <-router.Closing() }()
				if utils.Timeout(15*time.Second, utils.TimeoutWg(wg)) {
					pid := syscall.Getpid()
					app := config.Use(opt.AppName).AppName()
					log.Printf("%v [Gofusion] %s %s close http server timeout", pid, app, config.ComponentHttp)
				}
			}

			delete(routers, opt.AppName)
		}
		if opt.AppName == "" {
			Router = nil
		}
	}
}

func addI18n(opt *config.InitOption) func() {
	bundle := i18n.NewBundle[Errcode](i18n.DefaultLang(i18n.AppName(opt.AppName)))
	if I18n == nil {
		I18n = bundle
	}
	if i18ns == nil {
		i18ns = make(map[string]i18n.Localizable[Errcode])
	}

	i18ns[opt.AppName] = bundle

	// initialize http internal error
	bundle.AddMessages(errParam, map[language.Tag]*i18n.Message{
		language.English: {Other: "Invalid request parameters{{.err}}"},
		language.Chinese: {Other: "请求参数错误{{.err}}"},
	}, i18n.Var("err"))

	if opt.DI != nil {
		opt.DI.MustProvide(func() i18n.Localizable[Errcode] { return bundle })
	}

	return func() {
		locker.Lock()
		defer locker.Unlock()
		if i18ns != nil {
			delete(i18ns, opt.AppName)
		}
		if opt.AppName == "" {
			I18n = nil
		}
	}
}

func addClient(ctx context.Context, conf Conf, logger resty.Logger, opt *config.InitOption) func() {
	if _, ok := appClientCfgMap[opt.AppName]; !ok {
		defaultCfg := &cfg{
			c:       clone.Clone(defaultClientConf),
			appName: opt.AppName,
			logger:  logger,
		}
		appClientCfgMap[opt.AppName] = map[string]*cfg{
			"":        defaultCfg,
			"default": defaultCfg,
		}
	}
	for name, cliConf := range conf.Clients {
		cliCfg := &cfg{
			c:       cliConf,
			appName: opt.AppName,
			logger:  logger,
		}
		appClientCfgMap[opt.AppName][name] = cliCfg
		if name == config.DefaultInstanceKey {
			appClientCfgMap[opt.AppName][""] = cliCfg
		}

		if opt.AppName == "" && name == config.DefaultInstanceKey {
			Client = New(AppName(opt.AppName), CName(name))
		}
	}

	if opt.DI != nil {
		for name := range conf.Clients {
			opt.DI.MustProvide(func() *resty.Client { return New(AppName(opt.AppName), CName(name)) }, di.Name(name))
		}
	}

	return func() {
		locker.Lock()
		defer locker.Unlock()
		if appClientMap != nil {
			delete(appClientMap, opt.AppName)
		}
		if appClientCfgMap != nil {
			delete(appClientCfgMap, opt.AppName)
		}
		if opt.AppName == "" {
			Client = nil
		}
	}
}

type useOption struct {
	appName string
}

func AppName(name string) utils.OptionFunc[useOption] {
	return func(o *useOption) {
		o.appName = name
	}
}

func Use(opts ...utils.OptionExtender) IRouter {
	opt := utils.ApplyOptions[useOption](opts...)
	locker.RLock()
	defer locker.RUnlock()

	router, ok := routers[opt.appName]
	if !ok {
		panic(errors.Errorf("router not found"))
	}
	return router
}

func init() {
	config.AddComponent(config.ComponentHttp, Construct)
}
