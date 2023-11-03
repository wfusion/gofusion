package pulsar

import (
	"context"
	"sync"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/pkg/errors"
	"github.com/wfusion/gofusion/common/utils"

	"github.com/wfusion/gofusion/common/infra/watermill"
	"github.com/wfusion/gofusion/common/infra/watermill/message"
)

// SubscriberConfig is the configuration to create a subscriber
type SubscriberConfig struct {
	// URL is the URL to the broker
	URL string

	// QueueGroup is the JetStream queue group.
	//
	// All subscriptions with the same queue name (regardless of the connection they originate from)
	// will form a queue group. Each message will be delivered to only one subscriber per queue group,
	// using queuing semantics.
	//
	// It is recommended to set it with DurableName.
	// For non durable queue subscribers, when the last member leaves the group,
	// that group is removed. A durable queue group (DurableName) allows you to have all members leave
	// but still maintain state. When a member re-joins, it starts at the last position in that group.
	//
	// When QueueGroup is empty, subscribe without QueueGroup will be used.
	QueueGroup string

	Persistent bool

	Authentication pulsar.Authentication
}

// Subscriber provides the pulsar implementation for watermill subscribe operations
type Subscriber struct {
	conn     pulsar.Client
	logger   watermill.LoggerAdapter
	conf     *SubscriberConfig
	subsLock sync.RWMutex
	subs     map[string]pulsar.Consumer
	closed   bool
	closing  chan struct{}

	SubscribersCount int
	clientID         string
}

// NewSubscriber creates a new Subscriber.
func NewSubscriber(config *SubscriberConfig, logger watermill.LoggerAdapter) (*Subscriber, error) {
	conn, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:            config.URL,
		Authentication: config.Authentication,
	})
	if err != nil {
		return nil, errors.Wrap(err, "cannot connect to Pulsar")
	}
	return NewSubscriberWithPulsarClient(conn, config, logger)
}

// NewSubscriberWithPulsarClient creates a new Subscriber with the provided pulsar client.
func NewSubscriberWithPulsarClient(conn pulsar.Client, config *SubscriberConfig, logger watermill.LoggerAdapter) (
	*Subscriber, error) {
	if logger == nil {
		logger = watermill.NopLogger{}
	}

	return &Subscriber{
		conn:     conn,
		logger:   logger,
		conf:     config,
		closing:  make(chan struct{}),
		clientID: config.QueueGroup,
		subs:     make(map[string]pulsar.Consumer),
	}, nil
}

// Subscribe subscribes messages from JetStream.
func (s *Subscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	output := make(chan *message.Message)
	queueGroup := s.conf.QueueGroup

	s.subsLock.Lock()
	defer s.subsLock.Unlock()
	sub, found := s.subs[topic]
	if !found {
		if queueGroup == "" {
			queueGroup = topic + "-" + utils.ULID()
		}

		consumerOption := pulsar.ConsumerOptions{
			Topic:                       topic,
			SubscriptionName:            queueGroup,
			Type:                        pulsar.Exclusive,
			MessageChannel:              make(chan pulsar.ConsumerMessage, 10),
			AckWithResponse:             true,
			SubscriptionInitialPosition: pulsar.SubscriptionPositionLatest,
			SubscriptionMode:            pulsar.Durable,
		}

		if s.conf.QueueGroup != "" {
			consumerOption.Type = pulsar.Shared
		}

		if !s.conf.Persistent {
			consumerOption.SubscriptionMode = pulsar.NonDurable
		}

		sb, err := s.conn.Subscribe(consumerOption)
		if err != nil {
			return nil, err
		}
		s.subs[topic] = sb
		sub = sb
	}

	go func() {
		defer close(output)
		for !s.isClosed() {
			select {
			case <-ctx.Done():
				s.logger.Info("[Common] watermill pulsar exiting on context closure", nil)
				return
			case m := <-sub.Chan():
				go s.processMessage(ctx, output, m, sub)
			}
		}
	}()

	return output, nil
}

func (s *Subscriber) processMessage(ctx context.Context,
	output chan *message.Message, m pulsar.Message, sub pulsar.Consumer) {
	if s.isClosed() {
		return
	}

	logFields := watermill.LogFields{}
	s.logger.Trace("[Common] watermill pulsar received message", logFields)

	ctx = context.WithValue(ctx, watermill.ContextKeyMessageUUID, m.Key())
	ctx = context.WithValue(ctx, watermill.ContextKeyRawMessageID, m.ID().String())
	ctx, cancelCtx := context.WithCancel(ctx)
	defer cancelCtx()

	messageLogFields := logFields.Add(watermill.LogFields{
		"message_raw_id": m.ID().String(),
		"message_uuid":   m.Key(),
	})
	s.logger.Trace("[Common] watermill pulsar unmarshal message", messageLogFields)

	msg := message.NewMessage(m.Key(), m.Payload())
	msg.Metadata = m.Properties()
	msg.Metadata[watermill.ContextKeyMessageUUID] = msg.UUID
	msg.Metadata[watermill.ContextKeyRawMessageID] = m.ID().String()
	msg.SetContext(ctx)

	select {
	case <-s.closing:
		s.logger.Trace("[Common] watermill pulsar closing, message discarded", messageLogFields)
		return
	case <-ctx.Done():
		s.logger.Trace("[Common] watermill pulsar context cancelled, message discarded", messageLogFields)
		return
	// if this is first can risk 'send on closed channel' errors
	case output <- msg:
		s.logger.Trace("[Common] watermill pulsar message sent to consumer", messageLogFields)
	}

	select {
	case <-msg.Acked():
		if err := sub.Ack(m); err != nil {
			s.logger.Error("[Common] watermill pulsar message ack failed", err, messageLogFields)
		} else {
			s.logger.Trace("[Common] watermill pulsar message acked", messageLogFields)
		}
	case <-msg.Nacked():
		sub.Nack(m)
		s.logger.Trace("[Common] watermill pulsar message nacked", messageLogFields)
	case <-s.closing:
		s.logger.Trace("[Common] watermill pulsar closing, message discarded before ack", messageLogFields)
		return
	case <-ctx.Done():
		s.logger.Trace("[Common] watermill pulsar context cancelled, message discarded before ack", messageLogFields)
		return
	}
}

// Close closes the publisher and the underlying connection.
// It will attempt to wait for in-flight messages to complete.
func (s *Subscriber) Close() error {
	s.subsLock.Lock()
	defer s.subsLock.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	s.logger.Debug("Closing subscriber", nil)
	defer s.logger.Info("Subscriber closed", nil)

	close(s.closing)

	for _, sub := range s.subs {
		sub.Close()
	}
	s.conn.Close()

	return nil
}

func (s *Subscriber) isClosed() bool {
	s.subsLock.RLock()
	defer s.subsLock.RUnlock()

	return s.closed
}
