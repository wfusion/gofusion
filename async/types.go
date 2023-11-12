package async

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/log"
)

const (
	ErrDuplicatedHandlerName    utils.Error = "duplicated async handler name"
	ErrDuplicatedInstanceName   utils.Error = "duplicated async instance name"
	ErrDuplicatedQueueName      utils.Error = "duplicated async queue name"
	ErrConsumerDisabled         utils.Error = "async consumer is disabled"
	ErrUnsupportedSchedulerType utils.Error = "unsupported async type"
)

var (
	// callbackMap taskName:callback function
	funcNameToTaskName = map[string]map[string]string{}
	callbackMap        = map[string]map[string]any{}
	callbackMapLock    = sync.RWMutex{}

	customLoggerType = reflect.TypeOf((*customLogger)(nil)).Elem()
)

type Producable interface {
	Go(fn any, opts ...utils.OptionExtender) error
	Goc(ctx context.Context, fn any, opts ...utils.OptionExtender) error
	Send(ctx context.Context, taskName string, data any, opts ...utils.OptionExtender) (err error)
}

type Consumable interface {
	Use(mws ...routerMiddleware)
	Handle(pattern string, fn any, opts ...utils.OptionExtender)
	HandleFunc(fn any, opts ...utils.OptionExtender)
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

// Conf async conf
//nolint: revive // struct tag too long issue
type Conf struct {
	Type                 asyncType    `yaml:"type" json:"type" toml:"type"`
	Instance             string       `yaml:"instance" json:"instance" toml:"instance"`
	InstanceType         instanceType `yaml:"instance_type" json:"instance_type" toml:"instance_type"`
	Producer             bool         `yaml:"producer" json:"producer" toml:"producer" default:"true"`
	Consumer             bool         `yaml:"consumer" json:"consumer" toml:"consumer" default:"false"`
	ConsumerConcurrency  int          `yaml:"consumer_concurrency" json:"consumer_concurrency" toml:"consumer_concurrency"`
	MessageSerializeType string       `yaml:"message_serialize_type" json:"message_serialize_type" toml:"message_serialize_type" default:"gob"`
	MessageCompressType  string       `yaml:"message_compress_type" json:"message_compress_type" toml:"message_compress_type"`
	Queues               []*queueConf `yaml:"queues" json:"queues" toml:"queues"`
	StrictPriority       bool         `yaml:"strict_priority" json:"strict_priority" toml:"strict_priority"`

	EnableLogger bool   `yaml:"enable_logger" json:"enable_logger" toml:"enable_logger" default:"false"`
	LogLevel     string `yaml:"log_level" json:"log_level" toml:"log_level" default:"info"`
	Logger       string `yaml:"logger" json:"logger" toml:"logger" default:"github.com/wfusion/gofusion/log/customlogger.asyncLogger"`
	LogInstance  string `yaml:"log_instance" json:"log_instance" toml:"log_instance" default:"default"`
}

type queueConf struct {
	Name  string `yaml:"name" json:"name" toml:"name"`
	Level int    `yaml:"level" json:"level" toml:"level"`
}

type instanceType string

const (
	instanceTypeRedis instanceType = "redis"
	instanceTypeDB    instanceType = "db"
)

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

type produceOption struct {
	id                string
	args              []any
	queue             string
	maxRetry          int
	deadline          time.Time
	timeout           time.Duration
	delayDuration     time.Duration
	delayTime         time.Time
	retentionDuration time.Duration
}

func TaskID(id string) utils.OptionFunc[produceOption] {
	return func(o *produceOption) { o.id = id }
}
func Args(args ...any) utils.OptionFunc[produceOption] {
	return func(o *produceOption) { o.args = append(o.args, args...) }
}
func Queue(queue string) utils.OptionFunc[produceOption] {
	return func(o *produceOption) { o.queue = queue }
}
func MaxRetry(n int) utils.OptionFunc[produceOption] {
	return func(o *produceOption) { o.maxRetry = n }
}
func Deadline(t time.Time) utils.OptionFunc[produceOption] {
	return func(o *produceOption) { o.deadline = t }
}
func Timeout(d time.Duration) utils.OptionFunc[produceOption] {
	return func(o *produceOption) { o.timeout = d }
}
func Delay(d time.Duration) utils.OptionFunc[produceOption] {
	return func(o *produceOption) { o.delayDuration = d }
}
func DelayAt(t time.Time) utils.OptionFunc[produceOption] {
	return func(o *produceOption) { o.delayTime = t }
}
func Retention(d time.Duration) utils.OptionFunc[produceOption] {
	return func(o *produceOption) { o.retentionDuration = d }
}

type asyncType string

const (
	asyncTypeAsynq asyncType = "asynq"
	asyncTypeMysql asyncType = "mysql"
)

type routerMiddlewareFunc func(ctx context.Context, task Task) (err error)

type routerMiddleware func(next routerMiddlewareFunc) routerMiddlewareFunc

type customLogger interface {
	Init(log log.Loggable, appName, name string)
}
