package http

import (
	"context"
	"net"
	"net/http"
	"sync"

	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/inspect"
	"github.com/wfusion/gofusion/common/utils/serialize/json"
	"github.com/wfusion/gofusion/config"

	fusCtx "github.com/wfusion/gofusion/context"
)

var (
	Client *resty.Client

	defaultClientConf = new(clientConf)
)

type clientOption struct {
	mu sync.Mutex

	appName         string
	name            string
	retryConditions []resty.RetryConditionFunc
	retryHooks      []resty.OnRetryFunc
}

func CName(name string) utils.OptionFunc[clientOption] {
	return func(o *clientOption) {
		o.name = name
	}
}

func RetryCondition(fn resty.RetryConditionFunc) utils.OptionFunc[clientOption] {
	return func(o *clientOption) {
		o.mu.Lock()
		defer o.mu.Unlock()
		o.retryConditions = append(o.retryConditions, fn)
	}
}

func RetryHook(fn resty.OnRetryFunc) utils.OptionFunc[clientOption] {
	return func(o *clientOption) {
		o.mu.Lock()
		defer o.mu.Unlock()
		o.retryHooks = append(o.retryHooks, fn)
	}
}

func New(opts ...utils.OptionExtender) *resty.Client {
	opt := utils.ApplyOptions[clientOption](opts...)
	opt.appName = utils.ApplyOptions[useOption](opts...).appName

	cli := useClient(opts...)
	if cli != nil {
		return applyClientOptions(cli, opt)
	}

	locker.Lock()
	defer locker.Unlock()

	c := resty.New().
		OnBeforeRequest(traceHeaderMiddleware).
		SetTransport(http.DefaultTransport).
		SetJSONMarshaler(json.Marshal).
		SetJSONUnmarshaler(json.Unmarshal).
		SetDebug(true)

	cfg, ok := appClientCfgMap[opt.appName][opt.name]
	if !ok {
		cfg = appClientCfgMap[opt.appName][config.DefaultInstanceKey]
	}

	if cfg.logger != nil {
		c.SetLogger(cfg.logger)
	}

	if cliCfg := cfg.c; cliCfg == nil || !cliCfg.Mock {
		c.EnableTrace()
	} else {
		c.SetTimeout(utils.Must(utils.ParseDuration(cliCfg.Timeout)))
		c.SetRetryCount(cliCfg.RetryCount)
		c.SetRetryWaitTime(utils.Must(utils.ParseDuration(cliCfg.RetryWaitTime)))
		c.SetRetryMaxWaitTime(utils.Must(utils.ParseDuration(cliCfg.RetryMaxWaitTime)))
		for _, funcName := range cliCfg.RetryConditionFuncs {
			c.AddRetryCondition(*(*resty.RetryConditionFunc)(inspect.FuncOf(funcName)))
		}
		for _, hookName := range cliCfg.RetryHooks {
			c.AddRetryHook(*(*resty.OnRetryFunc)(inspect.FuncOf(hookName)))
		}

		dialer := &net.Dialer{
			Timeout:   utils.Must(utils.ParseDuration(cliCfg.DialTimeout)),
			KeepAlive: utils.Must(utils.ParseDuration(cliCfg.DialKeepaliveTime)),
		}

		c.SetTransport(&http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           dialer.DialContext,
			ForceAttemptHTTP2:     cliCfg.ForceAttemptHTTP2,
			DisableCompression:    cliCfg.DisableCompression,
			IdleConnTimeout:       utils.Must(utils.ParseDuration(cliCfg.IdleConnTimeout)),
			TLSHandshakeTimeout:   utils.Must(utils.ParseDuration(cliCfg.TLSHandshakeTimeout)),
			ExpectContinueTimeout: utils.Must(utils.ParseDuration(cliCfg.TLSHandshakeTimeout)),
			MaxIdleConns:          cliCfg.MaxIdleConns,
			MaxIdleConnsPerHost:   cliCfg.MaxIdleConnsPerHost,
			MaxConnsPerHost:       cliCfg.MaxConnsPerHost,
		})

		if cliCfg.Mock {
			httpmock.ActivateNonDefault(c.GetClient())
		}
	}

	if _, ok := appClientMap[opt.appName]; !ok {
		appClientMap[opt.appName] = make(map[string]*resty.Client)
	}
	appClientMap[opt.appName][opt.name] = c
	return applyClientOptions(c, opt)
}

func NewRequest(ctx context.Context, opts ...utils.OptionExtender) *resty.Request {
	return New(opts...).R().SetContext(ctx)
}

func traceHeaderMiddleware(cli *resty.Client, req *resty.Request) (err error) {
	ctx := req.Context()
	if userID := fusCtx.GetUserID(ctx); utils.IsStrNotBlank(userID) {
		req.SetHeader("userid", userID)
	}
	if traceID := fusCtx.GetTraceID(ctx); utils.IsStrNotBlank(traceID) {
		req.SetHeader("traceid", traceID)
	}
	return
}

func applyClientOptions(src *resty.Client, opt *clientOption) (dst *resty.Client) {
RetryConditionLoop:
	for _, cond := range opt.retryConditions {
		condName := utils.GetFuncName(cond)
		for _, exist := range src.RetryConditions {
			if condName == utils.GetFuncName(exist) {
				break RetryConditionLoop
			}
		}
		src.AddRetryCondition(cond)
	}

RetryHookLoop:
	for _, hook := range opt.retryHooks {
		condName := utils.GetFuncName(hook)
		for _, exist := range src.RetryHooks {
			if condName == utils.GetFuncName(exist) {
				break RetryHookLoop
			}
		}
		src.AddRetryHook(hook)
	}

	return src
}

func useClient(opts ...utils.OptionExtender) (cli *resty.Client) {
	locker.RLock()
	defer locker.RUnlock()

	opt := utils.ApplyOptions[clientOption](opts...)
	opt.appName = utils.ApplyOptions[useOption](opts...).appName
	locker.RLock()
	defer locker.RUnlock()
	appClients, ok := appClientMap[opt.appName]
	if !ok {
		return
	}
	return appClients[opt.name]
}

func init() {
	_ = utils.ParseTag(defaultClientConf,
		utils.ParseTagName("default"),
		utils.ParseTagUnmarshalType(utils.UnmarshalTypeYaml))
}
