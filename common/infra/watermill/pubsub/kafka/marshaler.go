package kafka

import (
	"time"

	"github.com/IBM/sarama"
	"github.com/pkg/errors"
	"github.com/wfusion/gofusion/common/infra/watermill"

	"github.com/wfusion/gofusion/common/infra/watermill/message"
)

const UUIDHeaderKey = "_watermill_message_uuid"

// Marshaler marshals Watermill's message to Kafka message.
type Marshaler interface {
	Marshal(topic string, msg *message.Message) (*sarama.ProducerMessage, error)
}

// Unmarshaler unmarshals Kafka's message to Watermill's message.
type Unmarshaler interface {
	Unmarshal(*sarama.ConsumerMessage) (*message.Message, error)
}

type MarshalerUnmarshaler interface {
	Marshaler
	Unmarshaler
}

type DefaultMarshaler struct {
	AppID string
}

func (d DefaultMarshaler) Marshal(topic string, msg *message.Message) (*sarama.ProducerMessage, error) {
	if value := msg.Metadata.Get(UUIDHeaderKey); value != "" {
		return nil, errors.Errorf("metadata %s is reserved by watermill for message UUID", UUIDHeaderKey)
	}

	headers := []sarama.RecordHeader{
		{
			Key:   []byte(UUIDHeaderKey),
			Value: []byte(msg.UUID),
		},
		{
			Key:   []byte(watermill.MessageHeaderAppID),
			Value: []byte(d.AppID),
		},
	}
	for key, value := range msg.Metadata {
		headers = append(headers, sarama.RecordHeader{
			Key:   []byte(key),
			Value: []byte(value),
		})
	}

	return &sarama.ProducerMessage{
		Topic:     topic,
		Value:     sarama.ByteEncoder(msg.Payload),
		Headers:   headers,
		Timestamp: time.Now(),
	}, nil
}

func (d DefaultMarshaler) Unmarshal(kafkaMsg *sarama.ConsumerMessage) (*message.Message, error) {
	var messageID string
	metadata := make(message.Metadata, len(kafkaMsg.Headers))

	for _, header := range kafkaMsg.Headers {
		if string(header.Key) == UUIDHeaderKey {
			messageID = string(header.Value)
		} else {
			metadata.Set(string(header.Key), string(header.Value))
		}
	}

	msg := message.NewMessage(messageID, kafkaMsg.Value)
	msg.Metadata = metadata

	return msg, nil
}

type GeneratePartitionKey func(topic string, msg *message.Message) (string, error)

type kafkaJsonWithPartitioning struct {
	DefaultMarshaler

	generatePartitionKey GeneratePartitionKey
}

func NewWithPartitioningMarshaler(generatePartitionKey GeneratePartitionKey) MarshalerUnmarshaler {
	return kafkaJsonWithPartitioning{generatePartitionKey: generatePartitionKey}
}

func (j kafkaJsonWithPartitioning) Marshal(topic string, msg *message.Message) (*sarama.ProducerMessage, error) {
	kafkaMsg, err := j.DefaultMarshaler.Marshal(topic, msg)
	if err != nil {
		return nil, err
	}

	key, err := j.generatePartitionKey(topic, msg)
	if err != nil {
		return nil, errors.Wrap(err, "cannot generate partition key")
	}
	kafkaMsg.Key = sarama.ByteEncoder(key)

	return kafkaMsg, nil
}
