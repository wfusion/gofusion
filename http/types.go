package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/log"
)

type IRouter interface {
	Use(middlewares ...gin.HandlerFunc) IRouter

	Handle(uri string, fn routerHandler, opts ...utils.OptionExtender) IRouter
	Any(uri string, fn routerHandler, opts ...utils.OptionExtender) IRouter
	GET(uri string, fn routerHandler, opts ...utils.OptionExtender) IRouter
	POST(uri string, fn routerHandler, opts ...utils.OptionExtender) IRouter
	DELETE(uri string, fn routerHandler, opts ...utils.OptionExtender) IRouter
	PATCH(uri string, fn routerHandler, opts ...utils.OptionExtender) IRouter
	PUT(uri string, fn routerHandler, opts ...utils.OptionExtender) IRouter
	OPTIONS(uri string, fn routerHandler, opts ...utils.OptionExtender) IRouter
	HEAD(uri string, fn routerHandler, opts ...utils.OptionExtender) IRouter
	Group(relativePath string, handlers ...gin.HandlerFunc) IRouter

	StaticFile(string, string) IRouter
	StaticFileFS(string, string, http.FileSystem) IRouter
	Static(string, string) IRouter
	StaticFS(string, http.FileSystem) IRouter

	ServeHTTP(http.ResponseWriter, *http.Request)
	Config() OutputConf
	ListenAndServe() error
	Start()
}

// Conf http configure
//nolint: revive // struct field annotation issue
type Conf struct {
	Port            int                    `yaml:"port" json:"port" toml:"port" default:"80"`
	TLS             bool                   `yaml:"tls" json:"tls" toml:"tls" default:"false"`
	Cert            string                 `yaml:"cert" json:"cert" toml:"cert"`
	Key             string                 `yaml:"key" json:"key" toml:"key"`
	NextProtos      []string               `yaml:"next_protos" json:"next_protos" toml:"next_protos" default:"[http/1.1]"` // h2, http/1.1 is ok
	SuccessCode     int                    `yaml:"success_code" json:"success_code" toml:"success_code"`
	Pprof           bool                   `yaml:"pprof" json:"pprof" toml:"pprof"`
	XSSWhiteURLList []string               `yaml:"xss_white_url_list" json:"xss_white_url_list" toml:"xss_white_url_list" default:"[]"`
	ColorfulConsole bool                   `yaml:"colorful_console" json:"colorful_console" toml:"colorful_console" default:"false"`
	ReadTimeout     string                 `yaml:"read_timeout" json:"read_timeout" toml:"read_timeout" default:"10s"`
	WriteTimeout    string                 `yaml:"write_timeout" json:"write_timeout" toml:"write_timeout" default:"10s"`
	EnableLogger    bool                   `yaml:"enable_logger" json:"enable_logger" toml:"enable_logger"`
	LogInstance     string                 `yaml:"log_instance" json:"log_instance" toml:"log_instance" default:"default"`
	Logger          string                 `yaml:"logger" json:"logger" toml:"logger" default:"github.com/wfusion/gofusion/log/customlogger.httpLogger"`
	Asynq           []asynqConf            `yaml:"asynq" json:"asynq" toml:"asynq"`
	Clients         map[string]*clientConf `yaml:"clients" json:"clients" toml:"clients"`
}

type asynqConf struct {
	Path              string       `yaml:"path" json:"path" toml:"path"`
	Instance          string       `yaml:"instance" json:"instance" toml:"instance"`
	InstanceType      instanceType `yaml:"instance_type" json:"instance_type" toml:"instance_type"`
	Readonly          bool         `yaml:"readonly" json:"readonly" toml:"readonly"`
	PrometheusAddress string       `yaml:"prometheus_address" json:"prometheus_address" toml:"prometheus_address"`
}

// clientConf http client configure
//nolint: revive // struct field annotation issue
type clientConf struct {
	Mock                  bool     `yaml:"mock" json:"mock" toml:"mock"`
	Timeout               string   `yaml:"timeout" json:"timeout" toml:"timeout" default:"30s"`
	DialTimeout           string   `yaml:"dial_timeout" json:"dial_timeout" toml:"dial_timeout" default:"30s"`
	DialKeepaliveTime     string   `yaml:"dial_keepalive_time" json:"dial_keepalive_time" toml:"dial_keepalive_time" default:"30s"`
	ForceAttemptHTTP2     bool     `yaml:"force_attempt_http2" json:"force_attempt_http2" toml:"force_attempt_http2" default:"true"`
	TLSHandshakeTimeout   string   `yaml:"tls_handshake_timeout" json:"tls_handshake_timeout" toml:"tls_handshake_timeout" default:"10s"`
	DisableCompression    bool     `yaml:"disable_compression" json:"disable_compression" toml:"disable_compression"`
	MaxIdleConns          int      `yaml:"max_idle_conns" json:"max_idle_conns" toml:"max_idle_conns" default:"100"`
	MaxIdleConnsPerHost   int      `yaml:"max_idle_conns_per_host" json:"max_idle_conns_per_host" toml:"max_idle_conns_per_host" default:"100"`
	MaxConnsPerHost       int      `yaml:"max_conns_per_host" json:"max_conns_per_host" toml:"max_conns_per_host"`
	IdleConnTimeout       string   `yaml:"idle_conn_timeout" json:"idle_conn_timeout" toml:"idle_conn_timeout" default:"90s"`
	ExpectContinueTimeout string   `yaml:"expect_continue_timeout" json:"expect_continue_timeout" toml:"expect_continue_timeout" default:"1s"`
	RetryCount            int      `yaml:"retry_count" json:"retry_count" toml:"retry_count"`
	RetryWaitTime         string   `yaml:"retry_wait_time" json:"retry_wait_time" toml:"retry_wait_time" default:"100ms"`
	RetryMaxWaitTime      string   `yaml:"retry_max_wait_time" json:"retry_max_wait_time" toml:"retry_max_wait_time" default:"2s"`
	RetryConditionFuncs   []string `yaml:"retry_condition_funcs" json:"retry_condition_funcs" toml:"retry_condition_funcs"`
	RetryHooks            []string `yaml:"retry_hooks" json:"retry_hooks" toml:"retry_hooks"`
}

type OutputConf struct {
	Port         int
	TLS          bool
	Cert         string
	Key          string
	NextProtos   []string
	SuccessCode  int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	AsynqConf    []asynqConf
}

type cfg struct {
	c       *clientConf
	appName string
	logger  resty.Logger
}

type instanceType string

const (
	instanceTypeRedis instanceType = "redis"
)

type customLogger interface {
	Init(log log.Loggable, appName string)
}
