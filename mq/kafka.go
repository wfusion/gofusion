package mq

import (
	"context"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/infra/watermill"
	"github.com/wfusion/gofusion/common/infra/watermill/pubsub/kafka"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
)

func newKafka(ctx context.Context, appName, name string, conf *Conf, logger watermill.LoggerAdapter) (
	pub Publisher, sub Subscriber) {
	if conf.Producer {
		pub = newKafkaPublisher(ctx, appName, name, conf, logger)
	}

	if conf.Consumer {
		sub = newKafkaSubscriber(ctx, appName, name, conf, logger)
	}

	return
}

type kafkaPublisher struct {
	*abstractMQ
	publisher *kafka.Publisher
}

func newKafkaPublisher(ctx context.Context, appName, name string,
	conf *Conf, logger watermill.LoggerAdapter) Publisher {
	cfg := kafka.PublisherConfig{
		Brokers:               conf.Endpoint.Addresses,
		Marshaler:             kafka.DefaultMarshaler{AppID: config.Use(appName).AppName()},
		OverwriteSaramaConfig: parseKafkaConf(appName, conf),
	}

	pub, err := kafka.NewPublisher(cfg, logger)
	if err != nil {
		panic(errors.Wrapf(err, "initialize mq component kafka publisher failed: %s", err))
	}

	return &kafkaPublisher{
		abstractMQ: newPub(ctx, pub, appName, name, conf, logger),
		publisher:  pub,
	}
}

func (k *kafkaPublisher) close() (err error) {
	return k.publisher.Close()
}

type kafkaSubscriber struct {
	*abstractMQ
	subscriber *kafka.Subscriber
}

func newKafkaSubscriber(ctx context.Context, appName, name string,
	conf *Conf, logger watermill.LoggerAdapter) Subscriber {
	cfg := kafka.SubscriberConfig{
		Brokers:               conf.Endpoint.Addresses,
		Unmarshaler:           kafka.DefaultMarshaler{AppID: config.Use(appName).AppName()},
		OverwriteSaramaConfig: parseKafkaConf(appName, conf),
		ConsumerGroup:         conf.ConsumerGroup,
		NackResendSleep:       100 * time.Millisecond,
		ReconnectRetrySleep:   time.Second,
		InitializeTopicDetails: &sarama.TopicDetail{
			NumPartitions:     -1,
			ReplicationFactor: -1,
			ReplicaAssignment: nil,
			ConfigEntries:     nil,
		},
	}

	sub, err := kafka.NewSubscriber(cfg, logger)
	if err != nil {
		panic(errors.Wrapf(err, "initialize mq component kafka subscriber failed: %s", err))
	}

	if err = sub.SubscribeInitialize(conf.Topic); err != nil {
		panic(errors.Wrapf(err, "initialize mq component kafka subscriber intialize: %s", err))
	}

	return &kafkaSubscriber{
		abstractMQ: newSub(ctx, sub, appName, name, conf, logger),
		subscriber: sub,
	}
}

func (k *kafkaSubscriber) close() (err error) {
	return k.subscriber.Close()
}

func parseKafkaConf(appName string, conf *Conf) (saramaCfg *sarama.Config) {
	saramaCfg = sarama.NewConfig()
	saramaCfg.Producer.Return.Errors = true
	saramaCfg.Producer.Return.Successes = true
	saramaCfg.Producer.RequiredAcks = sarama.WaitForLocal
	saramaCfg.Producer.Retry.Max = 10
	saramaCfg.Consumer.Fetch.Default = 16 * 1024 * 1024 // 16mb, default is 1mb
	saramaCfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	saramaCfg.Consumer.Offsets.AutoCommit.Enable = true
	saramaCfg.Consumer.Offsets.AutoCommit.Interval = time.Second // only work when auto commit disabled
	saramaCfg.Consumer.Return.Errors = true
	saramaCfg.Metadata.Retry.Backoff = time.Second * 2
	saramaCfg.ClientID = config.Use(appName).AppName()
	if utils.IsStrNotBlank(conf.Endpoint.Version) {
		saramaCfg.Version = utils.Must(sarama.ParseKafkaVersion(conf.Endpoint.Version))
	}
	if utils.IsStrNotBlank(conf.Endpoint.User) {
		saramaCfg.Net.SASL.Enable = true
		saramaCfg.Net.SASL.User = conf.Endpoint.User
		saramaCfg.Net.SASL.Password = conf.Endpoint.Password
		saramaCfg.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		switch {
		case strings.EqualFold(conf.Endpoint.AuthType, sarama.SASLTypeSCRAMSHA256):
			saramaCfg.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		case strings.EqualFold(conf.Endpoint.AuthType, sarama.SASLTypeSCRAMSHA512):
			saramaCfg.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		case strings.EqualFold(conf.Endpoint.AuthType, sarama.SASLTypeOAuth):
			saramaCfg.Net.SASL.Mechanism = sarama.SASLTypeOAuth
			saramaCfg.Net.SASL.TokenProvider = &kafkaOAuthProvider{token: saramaCfg.Net.SASL.Password}
		}
	}
	return
}

type kafkaOAuthProvider struct {
	token string
}

func (k *kafkaOAuthProvider) Token() (*sarama.AccessToken, error) {
	return &sarama.AccessToken{Token: k.token}, nil
}
