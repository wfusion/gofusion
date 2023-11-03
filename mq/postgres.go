package mq

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/infra/watermill"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/db"

	millSql "github.com/wfusion/gofusion/common/infra/watermill/pubsub/sql"
)

func newPostgres(ctx context.Context, appName, name string, conf *Conf, logger watermill.LoggerAdapter) (
	pub Publisher, sub Subscriber) {
	instance := db.Use(ctx, conf.Endpoint.Instance, db.AppName(appName))
	cli := utils.Must(instance.GetProxy().DB())
	if conf.Producer {
		pub = newPostgresPublisher(ctx, appName, name, conf, logger, cli)
	}

	if conf.Consumer {
		sub = newPostgresSubscriber(ctx, appName, name, conf, logger, cli)
	}

	return
}

type postgresPublisher struct {
	*abstractMQ
	publisher *millSql.Publisher
}

func newPostgresPublisher(ctx context.Context, appName, name string, conf *Conf, logger watermill.LoggerAdapter,
	cli *sql.DB) Publisher {
	cfg := millSql.PublisherConfig{
		SchemaAdapter: millSql.DefaultPostgreSQLSchema{
			GenerateMessagesTableName: func(topic string) string {
				return fmt.Sprintf("%s_%s", conf.MessageScheme, topic)
			},
			SubscribeBatchSize: conf.ConsumerConcurrency,
		},
		AutoInitializeSchema: true,
		AppID:                config.Use(appName).AppName(),
	}

	pub, err := millSql.NewPublisher(cli, cfg, logger)
	if err != nil {
		panic(errors.Wrapf(err, "initialize mq component postgres publisher failed: %s", err))
	}

	return &postgresPublisher{
		abstractMQ: newPub(ctx, pub, appName, name, conf, logger),
		publisher:  pub,
	}
}

func (p *postgresPublisher) close() (err error) {
	return p.publisher.Close()
}

type postgresSubscriber struct {
	*abstractMQ
	subscriber *millSql.Subscriber
}

func newPostgresSubscriber(ctx context.Context, appName, name string, conf *Conf, logger watermill.LoggerAdapter,
	cli *sql.DB) Subscriber {
	cfg := millSql.SubscriberConfig{
		ConsumerGroup:  conf.ConsumerGroup,
		PollInterval:   0,
		ResendInterval: 0,
		RetryInterval:  0,
		BackoffManager: nil,
		SchemaAdapter: millSql.DefaultPostgreSQLSchema{
			GenerateMessagesTableName: func(topic string) string {
				return fmt.Sprintf("%s_%s", conf.MessageScheme, topic)
			},
			SubscribeBatchSize: conf.ConsumerConcurrency, // fetch how many rows per query
		},
		OffsetsAdapter: millSql.DefaultPostgreSQLOffsetsAdapter{
			GenerateMessagesOffsetsTableName: func(topic string) string {
				return fmt.Sprintf("%s_offsets_%s", conf.MessageScheme, topic)
			},
		},
		InitializeSchema:  true,
		DisablePersistent: !conf.Persistent,
	}

	sub, err := millSql.NewSubscriber(cli, cfg, logger)
	if err != nil {
		panic(errors.Wrapf(err, "initialize mq component mysql subscriber failed: %s", err))
	}

	return &postgresSubscriber{
		abstractMQ: newSub(ctx, sub, appName, name, conf, logger),
		subscriber: sub,
	}
}

func (p *postgresSubscriber) close() (err error) {
	return p.subscriber.Close()
}
