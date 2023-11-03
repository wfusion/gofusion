package mq

import (
	"context"

	"github.com/wfusion/gofusion/common/infra/watermill"
	"github.com/wfusion/gofusion/common/infra/watermill/pubsub/gochannel"
	"github.com/wfusion/gofusion/config"
)

func newGoChannel(ctx context.Context, appName, name string, conf *Conf, logger watermill.LoggerAdapter) (
	pub Publisher, sub Subscriber) {
	cfg := gochannel.Config{
		OutputChannelBuffer:            int64(conf.ConsumerConcurrency),
		Persistent:                     conf.Persistent,
		ConsumerGroup:                  conf.ConsumerGroup,
		BlockPublishUntilSubscriberAck: false,
		AppID:                          config.Use(appName).AppName(),
	}

	native := gochannel.NewGoChannel(cfg, logger)
	if conf.Producer {
		pub = &goChannel{
			abstractMQ: newPub(ctx, native, appName, name, conf, logger),
			ch:         native,
		}
	}

	if conf.Consumer {
		sub = &goChannel{
			abstractMQ: newSub(ctx, native, appName, name, conf, logger),
			ch:         native,
		}
	}
	return
}

type goChannel struct {
	*abstractMQ
	ch *gochannel.GoChannel
}

func (g *goChannel) close() (err error) { return g.ch.Close() }
