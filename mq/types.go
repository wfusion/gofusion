package mq

import (
	"context"
	"reflect"

	"github.com/Rican7/retry/strategy"

	"github.com/wfusion/gofusion/common/infra/watermill"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/log"

	mw "github.com/wfusion/gofusion/common/infra/watermill/message"
)

const (
	ErrDuplicatedSubscriberName utils.Error = "duplicated mq subscriber name"
	ErrDuplicatedPublisherName  utils.Error = "duplicated mq publisher name"
	ErrDuplicatedRouterName     utils.Error = "duplicated mq router name"
	ErrEventHandlerConflict     utils.Error = "conflict with event handler and message handler"
	ErrNotImplement             utils.Error = "mq not implement"
)

var (
	handlerFuncType                   = reflect.TypeOf((*HandlerFunc)(nil)).Elem()
	watermillHandlerFuncType          = reflect.TypeOf((*mw.HandlerFunc)(nil)).Elem()
	watermillNoPublishHandlerFuncType = reflect.TypeOf((*mw.NoPublishHandlerFunc)(nil)).Elem()
	watermillHandlerMiddlewareType    = reflect.TypeOf((*mw.HandlerMiddleware)(nil)).Elem()
	customLoggerType                  = reflect.TypeOf((*customLogger)(nil)).Elem()
	watermillLoggerType               = reflect.TypeOf((*watermill.LoggerAdapter)(nil)).Elem()

	newFn = map[mqType]func(context.Context, string, string, *Conf, watermill.LoggerAdapter) (Publisher, Subscriber){
		mqTypeGoChannel: newGoChannel,
		mqTypeAMQP:      newAMQP,
		mqTypeRabbitmq:  newAMQP,
		mqTypeKafka:     newKafka,
		mqTypePulsar:    newPulsar,
		mqTypeRedis:     newRedis,
		mqTypeMysql:     newMysql,
		mqTypePostgres:  newPostgres,
	}

	singleConsumerMQType = utils.NewSet(mqTypeGoChannel, mqTypeMysql, mqTypePostgres)
)

type Publisher interface {
	// Publish publishes provided messages to given topic.
	//
	// Publish can be synchronous or asynchronous - it depends on the implementation.
	//
	// Most publishers implementations don't support atomic publishing of messages.
	// This means that if publishing one of the messages fails, the next messages will not be published.
	//
	// Publish is thread safe.
	Publish(ctx context.Context, opts ...utils.OptionExtender) error

	// PublishRaw publishes provided raw messages to given topic.
	//
	// PublishRaw can be synchronous or asynchronous - it depends on the implementation.
	//
	// Most publishers implementations don't support atomic publishing of messages.
	// This means that if publishing one of the messages fails, the next messages will not be published.
	//
	// PublishRaw is thread safe.
	PublishRaw(ctx context.Context, opts ...utils.OptionExtender) error

	// close should flush unsent messages, if publisher is async.
	close() error
	topic() string
	watermillPublisher() mw.Publisher
}
type EventPublisher[T eventual] interface {
	// PublishEvent publishes provided messages to given topic.
	//
	// PublishEvent can be synchronous or asynchronous - it depends on the implementation.
	//
	// Most publishers implementations don't support atomic publishing of messages.
	// This means that if publishing one of the messages fails, the next messages will not be published.
	//
	// PublishEvent is thread safe.
	PublishEvent(ctx context.Context, opts ...utils.OptionExtender) error
}

type IRouter interface {
	Handle(handlerName string, hdr any, opts ...utils.OptionExtender)
	Serve() error
	Start()
	Running() <-chan struct{}
	close() error
}

type Subscriber interface {
	// Subscribe returns output channel with messages from provided topic.
	// Channel is closed, when Close() was called on the subscriber.
	//
	// When provided ctx is cancelled, subscriber will close subscribe and close output channel.
	// Provided ctx is set to all produced messages.
	Subscribe(ctx context.Context, opts ...utils.OptionExtender) (<-chan Message, error)

	// SubscribeRaw returns output channel with original messages from provided topic.
	// Channel is closed, when Close() was called on the subscriber.
	//
	// When provided ctx is cancelled, subscriber will close subscribe and close output channel.
	// Provided ctx is set to all produced messages.
	SubscribeRaw(ctx context.Context, opts ...utils.OptionExtender) (<-chan Message, error)

	// close closes all subscriptions with their output channels and flush offsets etc. when needed.
	close() error
	topic() string
	watermillLogger() watermill.LoggerAdapter
	watermillSubscriber() mw.Subscriber
}

type EventSubscriber[T eventual] interface {
	// SubscribeEvent returns output channel with events from provided topic.
	// Channel is closed, when Close() was called on the subscriber.
	//
	// When provided ctx is cancelled, subscriber will close subscribe and close output channel.
	// Provided ctx is set to all produced messages.
	SubscribeEvent(ctx context.Context, opts ...utils.OptionExtender) (<-chan Event[T], error)
}

type HandlerFunc func(msg Message) error

type Message interface {
	ID() string
	Payload() []byte
	RawMessage() any
	Context() context.Context
	Object() any
	Ack() bool
	Nack() bool
}

type pubOption struct {
	messages          []Message
	watermillMessages mw.Messages

	async           bool
	asyncStrategies []strategy.Strategy

	objects           []any
	objectUUIDGenFunc reflect.Value
}
type eventPubOption[T eventual] struct {
	events []Event[T]
}

func Objects[T any](objectUUIDGenFunc func(T) string, objects ...any) utils.OptionFunc[pubOption] {
	return func(o *pubOption) {
		o.objects = objects
		if objectUUIDGenFunc != nil {
			o.objectUUIDGenFunc = reflect.ValueOf(objectUUIDGenFunc)
		}
	}
}
func Messages(messages ...Message) utils.OptionFunc[pubOption] {
	return func(o *pubOption) { o.messages = messages }
}
func messages(messages ...*mw.Message) utils.OptionFunc[pubOption] {
	return func(o *pubOption) {
		o.watermillMessages = messages
	}
}
func Async(strategies ...strategy.Strategy) utils.OptionFunc[pubOption] {
	return func(o *pubOption) {
		o.async = true
		o.asyncStrategies = strategies
	}
}
func Events[T eventual](events ...Event[T]) utils.OptionFunc[eventPubOption[T]] {
	return func(o *eventPubOption[T]) {
		o.events = events
	}
}

type subOption struct {
	channelLength int
}

func ChannelLen(channelLength int) utils.OptionFunc[subOption] {
	return func(o *subOption) {
		o.channelLength = channelLength
	}
}

type routerOption struct {
	isEventSubscriber bool
}

func handleEventSubscriber() utils.OptionFunc[routerOption] {
	return func(o *routerOption) {
		o.isEventSubscriber = true
	}
}

type message struct {
	*mw.Message

	payload []byte
	obj     any
}

func NewMessage(uuid string, payload []byte) Message {
	return &message{Message: mw.NewMessage(uuid, payload), payload: payload}
}
func (m *message) ID() string      { return m.Message.UUID }
func (m *message) Payload() []byte { return m.payload }
func (m *message) RawMessage() any { return m.Message }
func (m *message) Object() any     { return m.obj }

// Conf mq config
//nolint: revive // struct tag too long issue
type Conf struct {
	Topic               string        `yaml:"topic" json:"topic" toml:"topic"`
	Type                mqType        `yaml:"type" json:"type" toml:"type"`
	Producer            bool          `yaml:"producer" json:"producer" toml:"producer" default:"true"`
	Consumer            bool          `yaml:"consumer" json:"consumer" toml:"consumer"`
	ConsumerGroup       string        `yaml:"consumer_group" json:"consumer_group" toml:"consumer_group"`
	ConsumerConcurrency int           `yaml:"consumer_concurrency" json:"consumer_concurrency" toml:"consumer_concurrency"`
	Endpoint            *endpointConf `yaml:"endpoint" json:"endpoint" toml:"endpoint"`
	Persistent          bool          `yaml:"persistent" json:"persistent" toml:"persistent"`
	SerializeType       string        `yaml:"serialize_type" json:"serialize_type" toml:"serialize_type"`
	CompressType        string        `yaml:"compress_type" json:"compress_type" toml:"compress_type"`

	EnableLogger bool   `yaml:"enable_logger" json:"enable_logger" toml:"enable_logger" default:"false"`
	Logger       string `yaml:"logger" json:"logger" toml:"logger" default:"github.com/wfusion/gofusion/log/customlogger.mqLogger"`
	LogInstance  string `yaml:"log_instance" json:"log_instance" toml:"log_instance" default:"default"`

	// mongo, mysql, mariadb option
	MessageScheme  string `yaml:"message_scheme" json:"message_scheme" toml:"message_scheme" default:"watermill_message"`
	SeriesScheme   string `yaml:"series_scheme" json:"series_scheme" toml:"series_scheme" default:"watermill_series"`
	ConsumerScheme string `yaml:"consumer_scheme" json:"consumer_scheme" toml:"consumer_scheme" default:"watermill_subscriber"`

	ConsumeMiddlewares []*middlewareConf `yaml:"consume_middlewares" json:"consume_middlewares" toml:"consume_middlewares"`
}

type endpointConf struct {
	Addresses    []string     `yaml:"addresses" json:"addresses" toml:"addresses"`
	User         string       `yaml:"user" json:"user" toml:"user"`
	Password     string       `yaml:"password" json:"password" toml:"password" encrypted:""`
	AuthType     string       `yaml:"auth_type" json:"auth_type" toml:"auth_type"`
	Instance     string       `yaml:"instance" json:"instance" toml:"instance"`
	InstanceType instanceType `yaml:"instance_type" json:"instance_type" toml:"instance_type"`
	Version      string       `yaml:"version" json:"version" toml:"version"`
}

// middlewareConf consume middleware config
//nolint: revive // struct tag too long issue
type middlewareConf struct {
	Type middlewareType `yaml:"type" json:"type" toml:"type"`

	// Throttle middleware
	// Example duration and count: NewThrottle(10, time.Second) for 10 messages per second
	ThrottleCount    int            `yaml:"throttle_count" json:"throttle_count" toml:"throttle_count"`
	ThrottleDuration utils.Duration `yaml:"throttle_duration" json:"throttle_duration" toml:"throttle_duration"`

	// Retry middleware
	// MaxRetries is maximum number of times a retry will be attempted.
	RetryMaxRetries int `yaml:"retry_max_retries" json:"retry_max_retries" toml:"retry_max_retries"`
	// RetryInitialInterval is the first interval between retries. Subsequent intervals will be scaled by Multiplier.
	RetryInitialInterval utils.Duration `yaml:"retry_initial_interval" json:"retry_initial_interval" toml:"retry_initial_interval"`
	// RetryMaxInterval sets the limit for the exponential backoff of retries. The interval will not be increased beyond MaxInterval.
	RetryMaxInterval utils.Duration `yaml:"retry_max_interval" json:"retry_max_interval" toml:"retry_max_interval"`
	// RetryMultiplier is the factor by which the waiting interval will be multiplied between retries.
	RetryMultiplier float64 `yaml:"retry_multiplier" json:"retry_multiplier" toml:"retry_multiplier"`
	// RetryMaxElapsedTime sets the time limit of how long retries will be attempted. Disabled if 0.
	RetryMaxElapsedTime utils.Duration `yaml:"retry_max_elapsed_time" json:"retry_max_elapsed_time" toml:"retry_max_elapsed_time"`
	// RetryRandomizationFactor randomizes the spread of the backoff times within the interval of:
	// [currentInterval * (1 - randomization_factor), currentInterval * (1 + randomization_factor)].
	RetryRandomizationFactor float64 `yaml:"retry_randomization_factor" json:"retry_randomization_factor" toml:"retry_randomization_factor"`

	// Poison middleware
	// PoisonTopic salvages unprocessable messages and published them on a separate topic
	PoisonTopic string `yaml:"poison_topic" json:"poison_topic" toml:"poison_topic"`

	// Timeout middleware
	Timeout utils.Duration `yaml:"timeout" json:"timeout" toml:"timeout"`

	// CircuitBreaker middleware
	// CircuitBreakerMaxRequests is the maximum number of requests allowed to pass through
	// when the CircuitBreaker is half-open.
	// If CircuitBreakerMaxRequests is 0, the CircuitBreaker allows only 1 request.
	CircuitBreakerMaxRequests uint `yaml:"circuit_breaker_max_requests" json:"circuit_breaker_max_requests" toml:"circuit_breaker_max_requests"`
	// CircuitBreakerInterval is the cyclic period of the closed state
	// for the CircuitBreaker to clear the internal Counts.
	// If CircuitBreakerInterval is less than or equal to 0, the CircuitBreaker doesn't clear internal Counts during the closed state.
	CircuitBreakerInterval utils.Duration `yaml:"circuit_breaker_interval" json:"circuit_breaker_interval" toml:"circuit_breaker_interval"`
	// CircuitBreakerTimeout is the period of the open state,
	// after which the state of the CircuitBreaker becomes half-open.
	// If CircuitBreakerTimeout is less than or equal to 0, the timeout value of the CircuitBreaker is set to 60 seconds.
	CircuitBreakerTimeout utils.Duration `yaml:"circuit_breaker_timeout" json:"circuit_breaker_timeout" toml:"circuit_breaker_timeout"`
	// CircuitBreakerTripExpr ready to trip expression
	// support params: requests, total_successes, total_failures, consecutive_successes, consecutive_failures
	CircuitBreakerTripExpr string `yaml:"circuit_breaker_trip_expr" json:"circuit_breaker_trip_expr" toml:"circuit_breaker_trip_expr"`
}

type mqType string

const (
	mqTypeAMQP      mqType = "amqp"
	mqTypeRabbitmq  mqType = "rabbitmq"
	mqTypeGoChannel mqType = "gochannel"
	mqTypeKafka     mqType = "kafka"
	mqTypePulsar    mqType = "pulsar"
	mqTypeRedis     mqType = "redis"
	mqTypeRocketmq  mqType = "rocketmq"
	mqTypeMysql     mqType = "mysql"
	mqTypePostgres  mqType = "postgres"
)

type instanceType string

const (
	instanceTypeDB    instanceType = "db"
	instanceTypeRedis instanceType = "redis"
	instanceTypeMongo instanceType = "mongo"
)

type middlewareType string

const (
	middlewareTypeThrottle       middlewareType = "throttle"
	middlewareTypeRetry          middlewareType = "retry"
	middlewareTypeInstanceAck    middlewareType = "instance_ack"
	middlewareTypePoison         middlewareType = "poison"
	middlewareTypeTimeout        middlewareType = "timeout"
	middlewareTypeCircuitBreaker middlewareType = "circuit_breaker"
)

type customLogger interface {
	Init(log log.Loggable, appName, name string)
}
