package http

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"golang.org/x/text/language"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/http/middleware"
	"github.com/wfusion/gofusion/i18n"
)

var (
	Router IRouter

	locker  sync.RWMutex
	routers = map[string]IRouter{}
)

func Construct(ctx context.Context, conf Conf, opts ...utils.OptionExtender) func() {
	opt := utils.ApplyOptions[config.InitOption](opts...)
	optU := utils.ApplyOptions[useOption](opts...)
	if opt.AppName == "" {
		opt.AppName = optU.appName
	}

	engine := gin.New()
	engine.Use(
		gin.Recovery(),
		middleware.Gateway,
		middleware.Trace(),
		middleware.Logging(opt.AppName, conf.LogInstance),
		middleware.Cors(),
		middleware.XSS(conf.XSSWhiteURLList),
		middleware.Recover(opt.AppName, conf.LogInstance),
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

	engine.NoMethod(func(c *gin.Context) {
		c.Status(http.StatusMethodNotAllowed)
		Error(c, opt.AppName, -1, nil, 0, 0, fmt.Sprintf("找不到该方法, Method: %s", c.Request.Method))
	})
	engine.NoRoute(func(c *gin.Context) {
		c.Status(http.StatusNotFound)
		Error(c, opt.AppName, -1, nil, 0, 0, fmt.Sprintf("找不到该内容, URL: %s", c.Request.URL.String()))
	})

	engine.GET("/health", func(c *gin.Context) {
		Success(c, opt.AppName, nil, 0, -1, "Api 访问正常")
	})

	if conf.Pprof {
		pprof.Register(engine)
	}
	router := newRouter(engine, opt.AppName)

	locker.Lock()
	defer locker.Unlock()
	if len(conf.Asynq) > 0 {
		initAsynq(ctx, opt.AppName, router, conf.Asynq)
	}
	if _, ok := routers[opt.AppName]; ok {
		panic(errors.Errorf("duplicated http name: %s", opt.AppName))
	}
	routers[opt.AppName] = router
	if opt.AppName == "" {
		Router = router
	}

	bundle := i18n.NewBundle[Errcode](i18n.DefaultLang(i18n.AppName(opt.AppName)))
	if I18n == nil {
		I18n = bundle
	}
	if i18ns == nil {
		i18ns = make(map[string]i18n.Localizable[Errcode])
	}
	i18ns[opt.AppName] = bundle

	// ioc
	if opt.DI != nil {
		opt.DI.MustProvide(func() i18n.Localizable[Errcode] { return bundle })
		opt.DI.MustProvide(func() IRouter { return Use(AppName(opt.AppName)) })
	}

	// initialize http internal error
	bundle.AddMessages(errParam, map[language.Tag]*i18n.Message{
		language.English: {Other: "Invalid request parameters{{.err}}"},
		language.Chinese: {Other: "请求参数错误{{.err}}"},
	}, i18n.Var("err"))

	// gracefully exit outside gofusion
	return func() {
		locker.Lock()
		defer locker.Unlock()
		if i18ns != nil {
			delete(i18ns, opt.AppName)
		}
		if routers != nil {
			delete(routers, opt.AppName)
		}
		if opt.AppName == "" {
			I18n = nil
			Router = nil
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
