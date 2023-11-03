package mq

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"

	"github.com/wfusion/gofusion/common/infra/watermill"
	"github.com/wfusion/gofusion/common/infra/watermill/pubsub/amqp"
)

func newAMQP(ctx context.Context, appName, name string, conf *Conf, logger watermill.LoggerAdapter) (
	pub Publisher, sub Subscriber) {
	if conf.Producer {
		pub = newAMQPPublisher(ctx, appName, name, conf, logger)
	}

	if conf.Consumer {
		sub = newAMQPSubscriber(ctx, appName, name, conf, logger)
	}

	return
}

type _AMQPPublisher struct {
	*abstractMQ
	publisher *amqp.Publisher
}

func newAMQPPublisher(ctx context.Context, appName, name string, conf *Conf, logger watermill.LoggerAdapter) Publisher {
	ep := parseAMQPEndpoint(conf)

	var genFunc amqp.QueueNameGenerator
	if utils.IsStrNotBlank(conf.ConsumerGroup) {
		genFunc = amqp.GenerateQueueNameTopicNameWithSuffix(conf.ConsumerGroup)
	} else {
		genFunc = amqp.GenerateQueueNameTopicName
	}

	var cfg amqp.Config
	if conf.Persistent {
		cfg = amqp.NewDurablePubSubConfig(ep, genFunc)
	} else {
		cfg = amqp.NewNonDurablePubSubConfig(ep, genFunc)
	}

	cfg.Marshaler = amqp.DefaultMarshaler{
		NotPersistentDeliveryMode: !conf.Persistent,
		AppID:                     config.Use(appName).AppName(),
	}

	pub, err := amqp.NewPublisher(cfg, logger)
	if err != nil {
		panic(errors.Wrapf(err, "initialize mq component amqp publisher failed: %s", err))
	}

	return &_AMQPPublisher{
		abstractMQ: newPub(ctx, pub, appName, name, conf, logger),
		publisher:  pub,
	}
}

func (a *_AMQPPublisher) close() (err error) {
	return a.publisher.Close()
}

type _AMQPSubscriber struct {
	*abstractMQ
	subscriber *amqp.Subscriber
}

func newAMQPSubscriber(ctx context.Context, appName, name string, conf *Conf, logger watermill.LoggerAdapter) Subscriber {
	ep := parseAMQPEndpoint(conf)

	var genFunc amqp.QueueNameGenerator
	if utils.IsStrNotBlank(conf.ConsumerGroup) {
		genFunc = amqp.GenerateQueueNameTopicNameWithSuffix(conf.ConsumerGroup)
	} else {
		genFunc = amqp.GenerateQueueNameTopicName
	}

	var cfg amqp.Config
	if conf.Persistent {
		cfg = amqp.NewDurablePubSubConfig(ep, genFunc)
	} else {
		cfg = amqp.NewNonDurablePubSubConfig(ep, genFunc)
	}

	sub, err := amqp.NewSubscriber(cfg, logger)
	if err != nil {
		panic(errors.Wrapf(err, "initialize mq component amqp subscriber failed: %s", err))
	}

	utils.MustSuccess(sub.SubscribeInitialize(conf.Topic))

	return &_AMQPSubscriber{
		abstractMQ: newSub(ctx, sub, appName, name, conf, logger),
		subscriber: sub,
	}
}

func (a *_AMQPSubscriber) close() (err error) {
	return a.subscriber.Close()
}

func parseAMQPEndpoint(conf *Conf) (result string) {
	hasUser := utils.IsStrNotBlank(conf.Endpoint.User)
	hasPassword := utils.IsStrNotBlank(conf.Endpoint.Password)
	if hasUser && hasPassword {
		addr := strings.TrimPrefix(conf.Endpoint.Addresses[0], "amqp://")
		result = fmt.Sprintf("amqp://%s:%s@%s", conf.Endpoint.User, conf.Endpoint.Password, addr)
	} else if hasUser {
		addr := strings.TrimPrefix(conf.Endpoint.Addresses[0], "amqp://")
		result = fmt.Sprintf("amqp://%s@%s", conf.Endpoint.User, addr)
	} else {
		addr := strings.TrimPrefix(conf.Endpoint.Addresses[0], "amqp://")
		result = fmt.Sprintf("amqp://%s", addr)
	}

	return
}
