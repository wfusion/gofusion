package cron

import (
	"context"
	"reflect"
	"time"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/log"
)

const (
	ErrLockInstanceNotFound     utils.Error = "cron lock instance not found"
	ErrUnsupportedSchedulerType utils.Error = "unsupported cron scheduler type"

	errDiscardMessage     utils.Error = "discard message"
	tolerantOfTimeNotSync             = 5 * time.Second
)

var (
	customLoggerType = reflect.TypeOf((*customLogger)(nil)).Elem()
)

type IRouter interface {
	Use(mws ...routerMiddleware)
	Handle(pattern string, fn any, opts ...utils.OptionExtender)
	Serve() error
	Start() error
	shutdown() error
}

type Task interface {
	ID() string
	Name() string
	Payload() []byte
	RawMessage() any
}

// Conf conf
//nolint: revive // struct tag too long issue
type Conf struct {
	Type                 schedulerType        `yaml:"type" json:"type" toml:"type" default:"asynq"`
	Instance             string               `yaml:"instance" json:"instance" toml:"instance"`
	InstanceType         instanceType         `yaml:"instance_type" json:"instance_type" toml:"instance_type"`
	LockInstance         string               `yaml:"lock_instance" json:"lock_instance" toml:"lock_instance"`
	Queue                string               `yaml:"queue" json:"queue" toml:"queue"`
	Server               bool                 `yaml:"server" json:"server" toml:"server" default:"true"`
	Trigger              bool                 `yaml:"trigger" json:"trigger" toml:"trigger" default:"false"`
	ServerConcurrency    int                  `yaml:"server_concurrency" json:"server_concurrency" toml:"server_concurrency"`
	Timezone             string               `yaml:"timezone" json:"timezone" toml:"timezone" default:"Asia/Shanghai"`
	Tasks                map[string]*taskConf `yaml:"tasks" json:"tasks" toml:"tasks"`
	TaskLoader           string               `yaml:"task_loader" json:"task_loader" toml:"task_loader"`
	RefreshTasksInterval string               `yaml:"refresh_tasks_interval" json:"refresh_tasks_interval" toml:"refresh_tasks_interval" default:"3m"`

	EnableLogger bool   `yaml:"enable_logger" json:"enable_logger" toml:"enable_logger" default:"false"`
	LogLevel     string `yaml:"log_level" json:"log_level" toml:"log_level" default:"info"`
	Logger       string `yaml:"logger" json:"logger" toml:"logger" default:"github.com/wfusion/gofusion/log/customlogger.cronLogger"`
	LogInstance  string `yaml:"log_instance" json:"log_instance" toml:"log_instance" default:"default"`
}

type schedulerType string

const (
	schedulerTypeAsynq schedulerType = "asynq"
)

type instanceType string

const (
	instanceTypeRedis instanceType = "redis"
	instanceTypeMysql instanceType = "mysql"
)

type routerHandleFunc func(ctx context.Context, task Task) (err error)

type routerMiddleware func(next routerHandleFunc) routerHandleFunc

type taskConf struct {
	Crontab  string `yaml:"crontab" json:"crontab" toml:"crontab"`
	Callback string `yaml:"callback" json:"callback" toml:"callback"`
	Payload  string `yaml:"payload" json:"payload" toml:"payload"`
	Retry    int    `yaml:"retry" json:"retry" toml:"retry"`
	Timeout  string `yaml:"timeout" json:"timeout" toml:"timeout"`
	Deadline string `yaml:"deadline" json:"deadline" toml:"deadline"`
}

type task struct {
	id, name   string
	payload    []byte
	rawMessage any
}

func (t *task) ID() string {
	return t.id
}

func (t *task) Name() string {
	return t.name
}

func (t *task) Payload() []byte {
	return t.payload
}

func (t *task) RawMessage() any {
	return t.rawMessage
}

type customLogger interface {
	Init(log log.Loggable, appName, name string)
}
