package pulsar

import (
	"context"
	"fmt"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/wfusion/gofusion/common/infra/watermill"
	"github.com/wfusion/gofusion/common/infra/watermill/message"
)

// PublisherConfig is the configuration to create a publisher
type PublisherConfig struct {
	// URL is the Pulsar URL.
	URL string

	Authentication pulsar.Authentication

	AppID string
}

// Publisher provides the pulsar implementation for watermill publish operations
type Publisher struct {
	conn pulsar.Client
	pubs map[string]pulsar.Producer

	logger watermill.LoggerAdapter
	config PublisherConfig
	closed bool
}

// NewPublisher creates a new Publisher.
func NewPublisher(config PublisherConfig, logger watermill.LoggerAdapter) (*Publisher, error) {
	conn, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:            config.URL,
		Authentication: config.Authentication,
	})
	if err != nil {
		return nil, errors.Wrap(err, "cannot connect to nats")
	}

	return NewPublisherWithPulsarClient(config, logger, conn)
}

// NewPublisherWithPulsarClient creates a new Publisher with the provided nats connection.
func NewPublisherWithPulsarClient(config PublisherConfig, logger watermill.LoggerAdapter,
	conn pulsar.Client) (*Publisher, error) {
	if logger == nil {
		logger = watermill.NopLogger{}
	}

	return &Publisher{
		conn:   conn,
		logger: logger,
		config: config,
		pubs:   make(map[string]pulsar.Producer),
	}, nil
}

// Publish publishes message to Pulsar.
//
// Publish will not return until an ack has been received from JetStream.
// When one of messages delivery fails - function is interrupted.
func (p *Publisher) Publish(ctx context.Context, topic string, messages ...*message.Message) (err error) {
	if p.closed {
		return errors.New("publisher closed")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	producer, found := p.pubs[topic]

	if !found {
		pr, err := p.conn.CreateProducer(pulsar.ProducerOptions{Topic: topic})

		if err != nil {
			return err
		}

		producer = pr
		p.pubs[topic] = producer
	}

	for _, msg := range messages {
		messageFields := watermill.LogFields{
			"message_uuid": msg.UUID,
			"topic_name":   topic,
		}
		msg.Metadata[watermill.MessageHeaderAppID] = p.config.AppID

		p.logger.Trace("Publishing message", messageFields)

		msgID, sendErr := producer.Send(ctx, &pulsar.ProducerMessage{
			Key:        msg.UUID,
			Payload:    msg.Payload,
			Properties: msg.Metadata,
			EventTime:  time.Now(),
		})
		if sendErr != nil {
			err = multierr.Append(err, sendErr)
			p.logger.Trace(fmt.Sprintf("Publishing message failed: %s", err), messageFields)
			continue
		}
		messageFields = messageFields.Add(watermill.LogFields{"message_raw_id": msgID.String()})

		p.logger.Trace("Publishing message success", messageFields)
	}

	return nil
}

// Close closes the publisher and the underlying connection
func (p *Publisher) Close() error {
	if p.closed {
		return nil
	}
	p.closed = true

	p.logger.Trace("Closing publisher", nil)
	defer p.logger.Trace("Publisher closed", nil)

	for _, pub := range p.pubs {
		pub.Close()
	}

	p.conn.Close()

	return nil
}
