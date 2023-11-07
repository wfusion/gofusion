package mq

import (
	"context"
	"fmt"
	"reflect"

	"github.com/Rican7/retry"

	"github.com/wfusion/gofusion/common/infra/watermill"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/clone"
	"github.com/wfusion/gofusion/common/utils/compress"
	"github.com/wfusion/gofusion/common/utils/serialize"
	"github.com/wfusion/gofusion/routine"

	mw "github.com/wfusion/gofusion/common/infra/watermill/message"
	fmkCtx "github.com/wfusion/gofusion/context"
	pd "github.com/wfusion/gofusion/internal/util/payload"
)

type abstractMQ struct {
	pub mw.Publisher
	sub mw.Subscriber

	appName string
	ctx     context.Context
	name    string
	conf    *Conf
	logger  watermill.LoggerAdapter

	compressType  compress.Algorithm
	serializeType serialize.Algorithm
}

func newPub(ctx context.Context, pub mw.Publisher, appName, name string,
	conf *Conf, logger watermill.LoggerAdapter) *abstractMQ {
	mq := &abstractMQ{ctx: ctx, pub: pub, appName: appName, name: name, conf: clone.Slowly(conf), logger: logger}
	mq.serializeType = serialize.ParseAlgorithm(conf.SerializeType)
	mq.compressType = compress.ParseAlgorithm(conf.CompressType)
	return mq
}

func newSub(ctx context.Context, sub mw.Subscriber, appName, name string,
	conf *Conf, logger watermill.LoggerAdapter) *abstractMQ {
	mq := &abstractMQ{ctx: ctx, sub: sub, appName: appName, name: name, conf: clone.Slowly(conf), logger: logger}
	mq.serializeType = serialize.ParseAlgorithm(conf.SerializeType)
	mq.compressType = compress.ParseAlgorithm(conf.CompressType)
	return mq
}

func (a *abstractMQ) Publish(ctx context.Context, opts ...utils.OptionExtender) (err error) {
	opt := utils.ApplyOptions[pubOption](opts...)
	msgs := opt.watermillMessages
	for _, msg := range opt.messages {
		msg, err := a.newMessage(ctx, msg, opt)
		if err != nil {
			return err
		}
		msgs = append(msgs, msg)
	}
	for _, object := range opt.objects {
		msg, err := a.newObjectMessage(ctx, object, opt)
		if err != nil {
			return err
		}
		msgs = append(msgs, msg)
	}
	if len(msgs) == 0 {
		logInfo(ctx, a.logger, a.appName, a.name, "none messages to publish")
		return
	}

	if !opt.async {
		return a.pub.Publish(ctx, a.conf.Topic, msgs...)
	}

	routine.Goc(ctx, func() {
		idList := utils.SliceMapping(msgs, func(s *mw.Message) (id string) { return s.UUID })
		idList = utils.NewSet(idList...).Items()
		retryFunc := func(attempt uint) error {
			if attempt > 1 {
				logInfo(ctx, a.logger, a.appName, a.name,
					"retry to publish topic message [topic[%s] message%v[%v] attempt[%v]]",
					a.conf.Topic, idList, len(msgs), attempt-1)
			}
			return a.pub.Publish(ctx, a.conf.Topic, msgs...)
		}

		if err = retry.Retry(retryFunc, opt.asyncStrategies...); err != nil {
			logError(ctx, a.logger, a.appName, a.name,
				"retry to publish topic message failed [err[%s] topic[%s] message%v[%v]]",
				err, a.conf.Topic, idList, len(msgs))
		}
	}, routine.AppName(a.appName))
	return
}

func (a *abstractMQ) PublishRaw(ctx context.Context, opts ...utils.OptionExtender) (err error) {
	opt := utils.ApplyOptions[pubOption](opts...)
	msgs := opt.watermillMessages
	for _, msg := range opt.messages {
		wmsg := mw.NewMessage(msg.ID(), msg.Payload())
		wmsg.Metadata = fmkCtx.WatermillMetadata(ctx)
		wmsg.SetContext(ctx)
		msgs = append(msgs, wmsg)
	}
	if len(msgs) == 0 {
		logInfo(ctx, a.logger, a.appName, a.name, "none messages to publish")
		return
	}

	if !opt.async {
		return a.pub.Publish(ctx, a.conf.Topic, msgs...)
	}

	routine.Goc(ctx, func() {
		idList := utils.SliceMapping(msgs, func(s *mw.Message) (id string) { return s.UUID })
		idList = utils.NewSet(idList...).Items()
		retryFunc := func(attempt uint) error {
			if attempt > 1 {
				logInfo(ctx, a.logger, a.appName, a.name,
					"retry to publish topic message [topic[%s] message%v[%v] attempt[%v]]",
					a.conf.Topic, idList, len(msgs), attempt-1)
			}
			return a.pub.Publish(ctx, a.conf.Topic, msgs...)
		}

		if err = retry.Retry(retryFunc, opt.asyncStrategies...); err != nil {
			logError(ctx, a.logger, a.appName, a.name,
				"retry to publish topic message failed [err[%s] topic[%s] message%v[%v]]",
				err, a.conf.Topic, idList, len(msgs))
		}
	}, routine.AppName(a.appName))
	return
}

func (a *abstractMQ) SubscribeRaw(ctx context.Context, opts ...utils.OptionExtender) (dst <-chan Message, err error) {
	opt := utils.ApplyOptions[subOption](opts...)
	ch, err := a.sub.Subscribe(ctx, a.conf.Topic)
	if err != nil {
		return
	}

	msgCh := make(chan Message, opt.channelLength)
	routine.Go(func() {
		defer close(msgCh)
		for {
			select {
			case wmsg, ok := <-ch:
				if !ok {
					return
				}
				msg := rawMessageConvertFrom(wmsg)
				select {
				case msgCh <- msg:
				case <-ctx.Done():
					msg.Nack()
					a.logger.Info(fmt.Sprintf(
						"raw subscriber %s exited with a message nacked when business ctx done", a.name),
						watermill.LogFields{watermill.ContextLogFieldKey: ctx})
					return
				case <-a.ctx.Done():
					msg.Nack()
					a.logger.Info(fmt.Sprintf(
						"raw subscriber %s exited with a message nacked when app ctx done", a.name),
						watermill.LogFields{watermill.ContextLogFieldKey: ctx})
					return
				}
			case <-ctx.Done():
				return
			case <-a.ctx.Done():
				return
			}
		}
	}, routine.AppName(a.appName))
	return msgCh, err
}

func (a *abstractMQ) Subscribe(ctx context.Context, opts ...utils.OptionExtender) (dst <-chan Message, err error) {
	opt := utils.ApplyOptions[subOption](opts...)
	ch, err := a.sub.Subscribe(ctx, a.conf.Topic)
	if err != nil {
		return
	}

	msgCh := make(chan Message, opt.channelLength)
	routine.Go(func() {
		defer close(msgCh)
		for {
			select {
			case wmsg, ok := <-ch:
				if !ok {
					return
				}
				_, data, isRaw, err := pd.UnsealRaw(wmsg.Payload, pd.Compress(a.compressType))
				if err != nil {
					a.logger.Error("unseal message failed", err, watermill.LogFields{
						watermill.ContextLogFieldKey: ctx,
					})
					continue
				}
				wmsg.SetContext(fmkCtx.New(fmkCtx.Watermill(wmsg.Metadata)))
				msg := &message{Message: wmsg, payload: data}
				if !isRaw {
					_, msg.obj, _, err = pd.Unseal(wmsg.Payload,
						pd.Serialize(a.serializeType), pd.Compress(a.compressType))
					if err != nil {
						a.logger.Error("unseal message object failed", err, watermill.LogFields{
							watermill.ContextLogFieldKey: ctx,
						})
						continue
					}
				}
				select {
				case msgCh <- msg:
				case <-ctx.Done():
					msg.Nack()
					a.logger.Info(fmt.Sprintf(
						"subscriber %s exited with a message nacked when business ctx done", a.name),
						watermill.LogFields{watermill.ContextLogFieldKey: ctx})
					return
				case <-a.ctx.Done():
					msg.Nack()
					a.logger.Info(fmt.Sprintf(
						"subscriber %s exited with a message nacked when app ctx done", a.name),
						watermill.LogFields{watermill.ContextLogFieldKey: ctx})
					return
				}

			case <-ctx.Done():
				return
			case <-a.ctx.Done():
				return
			}
		}
	}, routine.AppName(a.appName))
	return msgCh, err
}

func (a *abstractMQ) close() error                             { panic(ErrNotImplement) }
func (a *abstractMQ) topic() string                            { return a.conf.Topic }
func (a *abstractMQ) watermillPublisher() mw.Publisher         { return a.pub }
func (a *abstractMQ) watermillSubscriber() mw.Subscriber       { return a.sub }
func (a *abstractMQ) watermillLogger() watermill.LoggerAdapter { return a.logger }

func (a *abstractMQ) newMessage(ctx context.Context, src Message, _ *pubOption) (
	msg *mw.Message, err error) {
	payload, err := pd.Seal(src.Payload(), pd.Compress(a.compressType))
	if err != nil {
		return
	}
	msg = mw.NewMessage(src.ID(), payload)
	msg.Metadata = fmkCtx.WatermillMetadata(ctx)
	msg.SetContext(ctx)
	return
}
func (a *abstractMQ) newObjectMessage(ctx context.Context, object any, opt *pubOption) (
	msg *mw.Message, err error) {
	payload, err := pd.Seal(object, pd.Compress(a.compressType), pd.Serialize(a.serializeType))
	if err != nil {
		return
	}
	uuid := utils.ULID()
	if opt.objectUUIDGenFunc.IsValid() && opt.objectUUIDGenFunc.Kind() == reflect.Func {
		inType := opt.objectUUIDGenFunc.Type().In(0)
		inParam := reflect.ValueOf(object).Convert(inType)
		uuid = opt.objectUUIDGenFunc.Call([]reflect.Value{inParam})[0].Interface().(string)
	}
	msg = mw.NewMessage(uuid, payload)
	msg.Metadata = fmkCtx.WatermillMetadata(ctx)
	msg.SetContext(ctx)
	return
}

func rawMessageConvertFrom(src *mw.Message) (dst Message) {
	return &message{Message: src, payload: src.Payload}
}

func messageConvertTo(src Message) (dst *mw.Message) {
	dst = src.(*message).Message
	return
}

func messageConvertFrom(src *mw.Message,
	serializeType serialize.Algorithm, compressType compress.Algorithm) (dst Message, err error) {
	_, data, isRaw, err := pd.UnsealRaw(src.Payload, pd.Compress(compressType))
	if err != nil {
		return
	}
	src.SetContext(fmkCtx.New(fmkCtx.Watermill(src.Metadata)))
	msg := &message{Message: src, payload: data}
	if !isRaw {
		_, msg.obj, _, err = pd.Unseal(src.Payload,
			pd.Serialize(serializeType), pd.Compress(compressType))
		if err != nil {
			return
		}
	}
	dst = msg
	return
}
