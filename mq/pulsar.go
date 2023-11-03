package mq

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/infra/watermill"
	"github.com/wfusion/gofusion/common/infra/watermill/pubsub/pulsar"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"

	plsDrv "github.com/apache/pulsar-client-go/pulsar"
)

func newPulsar(ctx context.Context, appName, name string, conf *Conf, logger watermill.LoggerAdapter) (
	pub Publisher, sub Subscriber) {
	if conf.Producer {
		pub = newPulsarPublisher(ctx, appName, name, conf, logger)
	}

	if conf.Consumer {
		sub = newPulsarSubscriber(ctx, appName, name, conf, logger)
	}

	return
}

type pulsarPublisher struct {
	*abstractMQ
	publisher *pulsar.Publisher
}

func newPulsarPublisher(ctx context.Context, appName, name string,
	conf *Conf, logger watermill.LoggerAdapter) Publisher {
	cfg := pulsar.PublisherConfig{
		URL:   fmt.Sprintf("pulsar://%s", strings.TrimPrefix(conf.Endpoint.Addresses[0], "pulsar://")),
		AppID: config.Use(appName).AppName(),
	}
	hasUser := utils.IsStrNotBlank(conf.Endpoint.User)
	hasPassword := utils.IsStrNotBlank(conf.Endpoint.Password)
	hasAuthType := utils.IsStrNotBlank(conf.Endpoint.AuthType)
	if hasUser && hasPassword {
		cfg.Authentication = utils.Must(plsDrv.NewAuthenticationBasic(conf.Endpoint.User, conf.Endpoint.Password))
	}
	if hasAuthType {
		cfg.Authentication = utils.Must(plsDrv.NewAuthentication(conf.Endpoint.AuthType, conf.Endpoint.Password))
	}

	pub, err := pulsar.NewPublisher(cfg, logger)
	if err != nil {
		panic(errors.Wrapf(err, "initialize mq component pulsar publisher failed: %s", err))
	}

	return &pulsarPublisher{
		abstractMQ: newPub(ctx, pub, appName, name, conf, logger),
		publisher:  pub,
	}
}

func (p *pulsarPublisher) close() (err error) {
	return p.publisher.Close()
}

type pulsarSubscriber struct {
	*abstractMQ
	subscriber *pulsar.Subscriber
}

func newPulsarSubscriber(ctx context.Context, appName, name string,
	conf *Conf, logger watermill.LoggerAdapter) Subscriber {
	cfg := &pulsar.SubscriberConfig{
		URL:        fmt.Sprintf("pulsar://%s", strings.TrimPrefix(conf.Endpoint.Addresses[0], "pulsar://")),
		QueueGroup: conf.ConsumerGroup,
		Persistent: conf.Persistent,
	}
	hasUser := utils.IsStrNotBlank(conf.Endpoint.User)
	hasPassword := utils.IsStrNotBlank(conf.Endpoint.Password)
	hasAuthType := utils.IsStrNotBlank(conf.Endpoint.AuthType)
	if hasUser && hasPassword {
		cfg.Authentication = utils.Must(plsDrv.NewAuthenticationBasic(conf.Endpoint.User, conf.Endpoint.Password))
	}
	if hasAuthType {
		params := conf.Endpoint.Password
		switch conf.Endpoint.AuthType {
		case "basic", "org.apache.pulsar.client.impl.auth.AuthenticationBasic":
			params = utils.MustJsonMarshalString(map[string]string{
				"username": conf.Endpoint.User,
				"password": conf.Endpoint.Password,
			})
		}
		cfg.Authentication = utils.Must(plsDrv.NewAuthentication(conf.Endpoint.AuthType, params))
	}

	sub, err := pulsar.NewSubscriber(cfg, logger)
	if err != nil {
		panic(errors.Wrapf(err, "initialize mq component pulsar subscriber failed: %s", err))
	}

	return &pulsarSubscriber{
		abstractMQ: newSub(ctx, sub, appName, name, conf, logger),
		subscriber: sub,
	}
}
func (p *pulsarSubscriber) close() (err error) {
	return p.subscriber.Close()
}
