package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/wfusion/gofusion/common/utils"
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
	Port            int         `yaml:"port" json:"port" toml:"port" default:"80"`
	TLS             bool        `yaml:"tls" json:"tls" toml:"tls" default:"false"`
	Cert            string      `yaml:"cert" json:"cert" toml:"cert"`
	Key             string      `yaml:"key" json:"key" toml:"key"`
	NextProtos      []string    `yaml:"next_protos" json:"next_protos" toml:"next_protos" default:"[http/1.1]"` // h2, http/1.1 is ok
	SuccessCode     int         `yaml:"success_code" json:"success_code" toml:"success_code"`
	Pprof           bool        `yaml:"pprof" json:"pprof" toml:"pprof"`
	XSSWhiteURLList []string    `yaml:"xss_white_url_list" json:"xss_white_url_list" toml:"xss_white_url_list" default:"[]"`
	ColorfulConsole bool        `yaml:"colorful_console" json:"colorful_console" toml:"colorful_console" default:"false"`
	ReadTimeout     string      `yaml:"read_timeout" json:"read_timeout" toml:"read_timeout" default:"10s"`
	WriteTimeout    string      `yaml:"write_timeout" json:"write_timeout" toml:"write_timeout" default:"10s"`
	Asynq           []asynqConf `yaml:"asynq" json:"asynq" toml:"asynq"`
	LogInstance     string      `yaml:"log_instance" json:"log_instance" toml:"log_instance" default:"default"`
}

type asynqConf struct {
	Path              string       `yaml:"path" json:"path" toml:"path"`
	Instance          string       `yaml:"instance" json:"instance" toml:"instance"`
	InstanceType      instanceType `yaml:"instance_type" json:"instance_type" toml:"instance_type"`
	Readonly          bool         `yaml:"readonly" json:"readonly" toml:"readonly"`
	PrometheusAddress string       `yaml:"prometheus_address" json:"prometheus_address" toml:"prometheus_address"`
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

type instanceType string

const (
	instanceTypeRedis instanceType = "redis"
)
