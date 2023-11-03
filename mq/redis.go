package mq

import (
	"context"

	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/infra/watermill"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/redis"

	rdsDrv "github.com/redis/go-redis/v9"

	millRds "github.com/wfusion/gofusion/common/infra/watermill/pubsub/redis"
)

func newRedis(ctx context.Context, appName, name string, conf *Conf, logger watermill.LoggerAdapter) (
	pub Publisher, sub Subscriber) {

	cli := redis.Use(ctx, conf.Endpoint.Instance, redis.AppName(appName))

	if conf.Producer {
		pub = newRedisPublisher(ctx, appName, name, conf, logger, cli)
	}

	if conf.Consumer {
		sub = newRedisSubscriber(ctx, appName, name, conf, logger, cli)
	}

	return
}

type redisPublisher struct {
	*abstractMQ
	publisher *millRds.Publisher
}

func newRedisPublisher(ctx context.Context, appName, name string, conf *Conf, logger watermill.LoggerAdapter,
	cli rdsDrv.UniversalClient) Publisher {
	cfg := millRds.PublisherConfig{
		Client:                cli,
		Marshaller:            millRds.DefaultMarshallerUnmarshaller{AppID: config.Use(appName).AppName()},
		Maxlens:               nil,
		DisableRedisConnClose: true,
	}

	pub, err := millRds.NewPublisher(cfg, logger)
	if err != nil {
		panic(errors.Wrapf(err, "initialize mq component redis publisher failed: %s", err))
	}

	return &redisPublisher{
		abstractMQ: newPub(ctx, pub, appName, name, conf, logger),
		publisher:  pub,
	}
}

func (r *redisPublisher) close() (err error) {
	return r.publisher.Close()
}

type redisSubscriber struct {
	*abstractMQ
	subscriber *millRds.Subscriber
}

func newRedisSubscriber(ctx context.Context, appName, name string, conf *Conf, logger watermill.LoggerAdapter,
	cli rdsDrv.UniversalClient) Subscriber {
	cfg := millRds.SubscriberConfig{
		Client:                    cli,
		Unmarshaller:              millRds.DefaultMarshallerUnmarshaller{AppID: config.Use(appName).AppName()},
		Consumer:                  "",
		ConsumerGroup:             conf.ConsumerGroup,
		NackResendSleep:           0,
		BlockTime:                 0,
		ClaimInterval:             0,
		ClaimBatchSize:            0,
		MaxIdleTime:               0,
		CheckConsumersInterval:    0,
		ConsumerTimeout:           0,
		OldestId:                  "",
		ShouldClaimPendingMessage: nil,
		DisableRedisConnClose:     true,
	}

	sub, err := millRds.NewSubscriber(cfg, logger)
	if err != nil {
		panic(errors.Wrapf(err, "initialize mq component redis subscriber failed: %s", err))
	}

	return &redisSubscriber{
		abstractMQ: newSub(ctx, sub, appName, name, conf, logger),
		subscriber: sub,
	}
}

func (r *redisSubscriber) close() (err error) {
	return r.subscriber.Close()
}
