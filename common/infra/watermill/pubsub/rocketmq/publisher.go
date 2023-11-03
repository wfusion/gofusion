package rocketmq

import (
	"context"
	"log"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/infra/watermill"
	"github.com/wfusion/gofusion/common/infra/watermill/message"
)

// Publisher the rocketmq publisher
type Publisher struct {
	config   PublisherConfig
	producer rocketmq.Producer
	logger   watermill.LoggerAdapter

	closed bool
}

// NewPublisher creates a new RocketMQ Publisher.
func NewPublisher(
	config PublisherConfig,
	logger watermill.LoggerAdapter,
) (*Publisher, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if logger == nil {
		logger = watermill.NopLogger{}
	}
	pub, err := rocketmq.NewProducer(config.Options()...)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create RocketMQ producer")
	}
	if config.SendMode == "" {
		config.SendMode = Sync
	}
	if config.SendAsyncCallback == nil {
		config.SendAsyncCallback = DefaultSendAsyncCallback
	}
	return &Publisher{
		config:   config,
		producer: pub,
		logger:   logger,
	}, nil
}

// PublisherConfig the rocketmq publisher config
type PublisherConfig struct {
	GroupName             string
	InstanceName          string
	Namespace             string
	SendMsgTimeout        time.Duration
	VIPChannelEnabled     bool
	RetryTimes            int
	Interceptors          []primitive.Interceptor
	Selector              producer.QueueSelector
	Credentials           *primitive.Credentials
	DefaultTopicQueueNums int
	CreateTopicKey        string
	// NsResolver            primitive.NsResolver
	// NameServer            primitive.NamesrvAddr
	// NameServerDomain      string

	SendMode SendMode // ["sync", "async", "oneway"]
	SendAsyncCallback

	// Marshaler is used to marshal messages from Watermill format into Rocketmq format.
	Marshaler Marshaler
}

// Options generate options
func (c *PublisherConfig) Options() []producer.Option {
	var opts []producer.Option
	if c.GroupName != "" {
		opts = append(opts, producer.WithGroupName(c.GroupName))
	}
	if c.InstanceName != "" {
		opts = append(opts, producer.WithInstanceName(c.InstanceName))
	}
	if c.Namespace != "" {
		opts = append(opts, producer.WithNamespace(c.Namespace))
	}
	if c.SendMsgTimeout > 0 {
		opts = append(opts, producer.WithSendMsgTimeout(c.SendMsgTimeout))
	}
	if c.VIPChannelEnabled {
		opts = append(opts, producer.WithVIPChannel(c.VIPChannelEnabled))
	}
	if c.RetryTimes > 0 {
		opts = append(opts, producer.WithRetry(c.RetryTimes))
	}
	if len(c.Interceptors) > 0 {
		opts = append(opts, producer.WithInterceptor(c.Interceptors...))
	}
	if c.Selector != nil {
		opts = append(opts, producer.WithQueueSelector(c.Selector))
	}
	if c.Credentials != nil {
		opts = append(opts, producer.WithCredentials(*c.Credentials))
	}
	if c.DefaultTopicQueueNums > 0 {
		opts = append(opts, producer.WithDefaultTopicQueueNums(c.DefaultTopicQueueNums))
	}
	if c.CreateTopicKey != "" {
		opts = append(opts, producer.WithCreateTopicKey(c.CreateTopicKey))
	}
	return nil
}

// Validate validate publisher config
func (c PublisherConfig) Validate() error {
	if c.SendMode != "" && c.SendMode != "sync" && c.SendMode != "async" && c.SendMode != "one_way" {
		return errors.Errorf("invalid send mode: %s", c.SendMode)
	}
	return nil
}

// Publish publishes message to RocketMQ.
//
// Publish is blocking and wait for ack from RocketMQ.
// When one of messages delivery fails - function is interrupted.
func (p *Publisher) Publish(ctx context.Context, topic string, msgs ...*message.Message) error {
	if p.closed {
		return errors.New("publisher closed")
	}
	logFields := make(watermill.LogFields, 4)
	logFields["topic"] = topic
	for _, msg := range msgs {
		logFields["message_uuid"] = msg.UUID
		p.logger.Trace("Sending message to RocketMQ", logFields)
		rocketmqMsgs, err := p.config.Marshaler.Marshal(topic, msg)
		if err != nil {
			return errors.Wrapf(err, "cannot marshal message %s", msg.UUID)
		}
		switch p.config.SendMode {
		case Async:
			err = p.sendAsync(ctx, msg, logFields, rocketmqMsgs...)
		case OneWay:
			err = p.producer.SendOneWay(ctx)
		default:
			err = p.sendSync(ctx, msg, logFields, rocketmqMsgs...)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Publisher) sendSync(ctx context.Context, wmsg *message.Message,
	fields map[string]any, rmsg ...*primitive.Message) error {
	result, err := p.producer.SendSync(ctx, rmsg...)
	if err != nil {
		return errors.WithMessagef(err, "send sync msg %s failed", wmsg.UUID)
	}
	fields["send_status"] = result.Status
	fields["msg_id"] = result.MsgID
	fields["offset_msg_id"] = result.OffsetMsgID
	fields["queue_offset"] = result.QueueOffset
	fields["message_queue"] = result.MessageQueue.String()
	fields["transaction_id"] = result.TransactionID
	fields["region_id"] = result.RegionID
	fields["trace_on"] = result.TraceOn
	p.logger.Trace("Message sent to RocketMQ", fields)
	return nil
}

func (p *Publisher) sendAsync(ctx context.Context, wmsg *message.Message,
	fields map[string]any, rmsg ...*primitive.Message) error {
	err := p.producer.SendAsync(ctx, p.config.SendAsyncCallback, rmsg...)
	if err != nil {
		return errors.WithMessagef(err, "send sync msg %s failed", wmsg.UUID)
	}
	return nil
}

// Close closes the publisher
func (p *Publisher) Close() error {
	if p.closed {
		return nil
	}
	p.closed = true

	if err := p.producer.Shutdown(); err != nil {
		return errors.Wrap(err, "cannot close Kafka producer")
	}

	return nil
}

// SendMode send mode
type SendMode string

const (
	// Sync the syns mode
	Sync SendMode = "sync"
	// Async the async mode
	Async SendMode = "async"
	// OneWay the one way mode, no resule
	OneWay SendMode = "one_way"
)

// SendAsyncCallback callback for each message send aysnc result
type SendAsyncCallback func(ctx context.Context, result *primitive.SendResult, err error)

// DefaultSendAsyncCallback default SendAsyncCallback
func DefaultSendAsyncCallback(ctx context.Context, result *primitive.SendResult, err error) {
	if err != nil {
		log.Printf("receive message error: %v\n", err)
	} else {
		log.Printf("send message success: result=%s\n", result.String())
	}
}
