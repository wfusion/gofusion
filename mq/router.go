package mq

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/PaesslerAG/gval"
	"github.com/pkg/errors"
	"github.com/sony/gobreaker"
	"go.uber.org/multierr"

	"github.com/wfusion/gofusion/common/infra/watermill"
	"github.com/wfusion/gofusion/common/infra/watermill/message/router/middleware"
	"github.com/wfusion/gofusion/common/infra/watermill/message/router/plugin"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/compress"
	"github.com/wfusion/gofusion/common/utils/inspect"
	"github.com/wfusion/gofusion/common/utils/serialize"
	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/routine"

	millMsg "github.com/wfusion/gofusion/common/infra/watermill/message"
	fmkCtx "github.com/wfusion/gofusion/context"
	pd "github.com/wfusion/gofusion/util/payload"
)

const (
	defaultRouterEventHandlerName = "__router_event_handler"
)

type router struct {
	*millMsg.Router

	appName string

	c   *Conf
	pub Publisher
	sub Subscriber

	compressType  compress.Algorithm
	serializeType serialize.Algorithm

	ctx context.Context

	locker                  sync.RWMutex
	eventHandlers           map[string]*handler
	eventSubscriberHandlers map[string]*handler
}
type handler struct {
	fn             reflect.Value
	evtType        reflect.Type
	evtPayloadType reflect.Type
}

func newRouter(ctx context.Context, appName, name string, conf *Conf,
	pub Publisher, sub Subscriber, logger watermill.LoggerAdapter) *router {
	r := utils.Must(millMsg.NewRouter(millMsg.RouterConfig{CloseTimeout: 15 * time.Second}, logger))
	r.AddPlugin(plugin.SignalsHandler)
	r.AddMiddleware(
		middleware.Recoverer,
		middleware.CorrelationID,
	)
	for _, mwsConf := range conf.ConsumeMiddlewares {
		switch mwsConf.Type {
		case middlewareTypeRetry:
			r.AddMiddleware(middleware.Retry{
				MaxRetries:          mwsConf.RetryMaxRetries,
				InitialInterval:     utils.Must(time.ParseDuration(mwsConf.RetryInitialInterval)),
				MaxInterval:         utils.Must(time.ParseDuration(mwsConf.RetryMaxInterval)),
				Multiplier:          mwsConf.RetryMultiplier,
				MaxElapsedTime:      utils.Must(time.ParseDuration(mwsConf.RetryMaxElapsedTime)),
				RandomizationFactor: mwsConf.RetryRandomizationFactor,
				OnRetryHook: func(attempt int, delay time.Duration) {
					logTrace(ctx, logger, appName, name,
						"retry to consume message [attempt[%v] delay[%s]]", attempt, delay)
				},
				Logger: logger,
			}.Middleware)
		case middlewareTypeThrottle:
			r.AddMiddleware(
				middleware.NewThrottle(
					int64(mwsConf.ThrottleCount),
					utils.Must(time.ParseDuration(mwsConf.ThrottleDuration)),
				).Middleware,
			)
		case middlewareTypeInstanceAck:
			r.AddMiddleware(middleware.InstantAck)
		case middlewareTypePoison:
			shouldGoToPoisonQueue := func(err error) bool { return err != nil }
			r.AddMiddleware(
				utils.Must(middleware.PoisonQueueWithFilter(
					pub.watermillPublisher(),
					mwsConf.PoisonTopic,
					shouldGoToPoisonQueue,
				)),
			)
		case middlewareTypeTimeout:
			r.AddMiddleware(middleware.Timeout(utils.Must(time.ParseDuration(mwsConf.Timeout))))
		case middlewareTypeCircuitBreaker:
			var expr gval.Evaluable
			if utils.IsStrNotBlank(mwsConf.CircuitBreakerTripExpr) {
				expr = utils.Must(gval.Full().NewEvaluable(mwsConf.CircuitBreakerTripExpr))
			}
			r.AddMiddleware(middleware.NewCircuitBreaker(gobreaker.Settings{
				Name:        name,
				MaxRequests: uint32(mwsConf.CircuitBreakerMaxRequests),
				Interval:    utils.Must(time.ParseDuration(mwsConf.CircuitBreakerInterval)),
				Timeout:     utils.Must(time.ParseDuration(mwsConf.CircuitBreakerTimeout)),
				ReadyToTrip: func(counts gobreaker.Counts) bool {
					// fallback to default ready to trip expression
					if expr == nil {
						return counts.ConsecutiveFailures > 5
					}
					if ok, err := expr.EvalBool(ctx, map[string]uint32{
						"requests":              counts.Requests,
						"total_successes":       counts.TotalSuccesses,
						"total_failures":        counts.TotalFailures,
						"consecutive_successes": counts.ConsecutiveSuccesses,
						"consecutive_failures":  counts.ConsecutiveFailures,
					}); err == nil {
						return ok
					}
					// fallback to default ready to trip expression
					return counts.ConsecutiveFailures > 5
				},
				OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
					logInfo(ctx, logger, appName, name, "circuit breaker state changed: %s -> %s", from, to)
				},
				IsSuccessful: func(err error) bool { return err == nil },
			}).Middleware)
		default:
			typ := inspect.TypeOf(string(mwsConf.Type))
			if typ == nil || typ.ConvertibleTo(watermillHandlerMiddlewareType) {
				panic(errors.Errorf("unknown mq middleware: %s", mwsConf.Type))
			}
			mws := reflect.New(typ).Elem().Convert(watermillHandlerMiddlewareType).Interface()
			r.AddMiddleware(mws.(millMsg.HandlerMiddleware))
		}
	}

	rr := &router{
		ctx:                     ctx,
		appName:                 appName,
		c:                       conf,
		pub:                     pub,
		sub:                     sub,
		Router:                  r,
		eventHandlers:           make(map[string]*handler),
		eventSubscriberHandlers: make(map[string]*handler),
	}
	rr.serializeType = serialize.ParseAlgorithm(conf.SerializeType)
	rr.compressType = compress.ParseAlgorithm(conf.CompressType)

	return rr
}

func (r *router) Handle(handlerName string, hdr any, opts ...utils.OptionExtender) {
	opt := utils.ApplyOptions[routerOption](opts...)
	if opt.isEventSubscriber || singleConsumerMQType.Contains(r.c.Type) {
		r.addHandler(handlerName, handlerName, hdr, opt)
		return
	}
	for i := 0; i < r.c.ConsumerConcurrency; i++ {
		consumerName := fmt.Sprintf("%s_%v", handlerName, i)
		r.addHandler(handlerName, consumerName, hdr, opt)
	}
}

func (r *router) Serve() (err error) {
	if r.isHandlerConflict() {
		panic(ErrEventHandlerConflict)
	}
	if len(r.eventHandlers) > 0 {
		r.runEventHandlers()
	}
	if err = r.run(); err != nil {
		return
	}
	<-r.Router.ClosedCh
	return
}

func (r *router) Start() {
	if r.isHandlerConflict() {
		panic(ErrEventHandlerConflict)
	}
	if len(r.eventHandlers) > 0 {
		r.runEventHandlers()
	}
	routine.Go(r.run, routine.AppName(r.appName))
}

func (r *router) Running() <-chan struct{} {
	return r.Router.Running()
}

func (r *router) run() (err error) {
	if r.Router.IsRunning() {
		return r.Router.RunHandlers(r.ctx)
	}
	if err = r.Router.Run(r.ctx); err != nil {
		if errors.Is(err, millMsg.ErrRouterIsAlreadyRunning) {
			return r.Router.RunHandlers(r.ctx)
		}
	}
	return
}

func (r *router) addHandler(handlerName, consumerName string, hdr any, opt *routerOption) {
	switch fn := hdr.(type) {
	case HandlerFunc:
		r.Router.AddNoPublisherHandler(
			consumerName,
			r.sub.topic(),
			r.sub.watermillSubscriber(),
			func(wmsg *millMsg.Message) (err error) {
				msg, err := messageConvertFrom(wmsg, r.serializeType, r.compressType)
				if err != nil {
					return
				}
				return fn(msg)
			},
		)
	case millMsg.NoPublishHandlerFunc:
		r.Router.AddNoPublisherHandler(
			consumerName,
			r.sub.topic(),
			r.sub.watermillSubscriber(),
			fn,
		)
	case millMsg.HandlerFunc:
		r.Router.AddHandler(
			consumerName,
			r.sub.topic(),
			r.sub.watermillSubscriber(),
			r.pub.topic(),
			r.pub.watermillPublisher(),
			fn,
		)
	default:
		fnVal := reflect.ValueOf(hdr)
		switch {
		case fnVal.CanConvert(watermillHandlerFuncType):
			r.Router.AddNoPublisherHandler(
				consumerName,
				r.sub.topic(),
				r.sub.watermillSubscriber(),
				func(msg *millMsg.Message) error {
					rets := fnVal.Convert(watermillHandlerFuncType).Call(
						[]reflect.Value{reflect.ValueOf(rawMessageConvertFrom(msg))},
					)
					return utils.ParseVariadicFuncResult[error](rets, 0)
				},
			)
		case fnVal.CanConvert(watermillNoPublishHandlerFuncType):
			r.Router.AddNoPublisherHandler(
				consumerName,
				r.sub.topic(),
				r.sub.watermillSubscriber(),
				func(msg *millMsg.Message) error {
					rets := fnVal.
						Convert(watermillNoPublishHandlerFuncType).
						Call([]reflect.Value{reflect.ValueOf(msg)})
					return utils.ParseVariadicFuncResult[error](rets, 0)
				},
			)
		case fnVal.CanConvert(handlerFuncType):
			r.Router.AddHandler(
				consumerName,
				r.sub.topic(),
				r.sub.watermillSubscriber(),
				r.pub.topic(),
				r.pub.watermillPublisher(),
				func(wmsg *millMsg.Message) (msgs []*millMsg.Message, err error) {
					msg, err := messageConvertFrom(wmsg, r.serializeType, r.compressType)
					if err != nil {
						return
					}
					rets := fnVal.Convert(handlerFuncType).Call([]reflect.Value{reflect.ValueOf(msg)})
					msgs = utils.ParseVariadicFuncResult[[]*millMsg.Message](rets, 0)
					err = utils.ParseVariadicFuncResult[error](rets, 0)
					return
				},
			)
		case isEventHandler(fnVal):
			r.handleEvent(handlerName, fnVal, opt)
		default:
			r.Router.AddNoPublisherHandler(
				consumerName,
				r.sub.topic(),
				r.sub.watermillSubscriber(),
				r.handle(hdr),
			)
		}
	}
}

func (r *router) handleEvent(eventType string, fnVal reflect.Value, opt *routerOption) {
	// FIXME: Translating generics to corresponding implemented generic types like this is too hacky.
	//        If this set becomes invalid, switch to the implementation of event payload as any without generics,
	//        and the router can continue to provide it using the current method of storing reflect.Type.
	evtType := fnVal.Type().In(1)
	eventName := strings.Replace(evtType.Name(), "Event[", "event[", 1)
	eventTypeName := fmt.Sprintf(mqPackageSignFormat, eventName)
	et := inspect.TypeOf(eventTypeName)
	if et == nil {
		panic(errors.Errorf("unknown event generic object type: %s", eventTypeName))
	}
	eventPayloadName := strings.Replace(evtType.Name(), "Event[", "eventPayload[", 1)
	eventPayloadTypeName := fmt.Sprintf(mqPackageSignFormat, eventPayloadName)
	ept := inspect.TypeOf(eventPayloadTypeName)
	if ept == nil {
		panic(errors.Errorf("unknown event payload generic object type: %s", eventPayloadTypeName))
	}

	hdr := &handler{
		fn:             fnVal,
		evtType:        et,
		evtPayloadType: reflect.PtrTo(ept),
	}

	r.locker.Lock()
	defer r.locker.Unlock()
	if !opt.isEventSubscriber {
		r.eventHandlers[eventType] = hdr
	} else {
		r.eventSubscriberHandlers[eventType] = hdr
		r.addEventDispatchHandler(defaultRouterEventHandlerName + "_" + eventType)
		routine.Go(r.run, routine.AppName(r.appName))
	}
}

func (r *router) handle(hdr any) millMsg.NoPublishHandlerFunc {
	typ := wrapParams(hdr)
	fn := utils.WrapFunc1[error](hdr)
	return func(msg *millMsg.Message) (err error) {
		_, data, _, err := pd.Unseal(msg.Payload,
			pd.Serialize(r.serializeType), pd.Compress(r.compressType), pd.Type(typ))
		if err != nil {
			return
		}
		params := unwrapParams(typ, data)
		ctx := fmkCtx.New(fmkCtx.Watermill(msg.Metadata))
		return fn(append([]any{ctx}, params...)...)
	}
}

func (r *router) runEventHandlers() {
	if singleConsumerMQType.Contains(r.c.Type) {
		r.addEventDispatchHandler(defaultRouterEventHandlerName)
		return
	}
	for i := 0; i < r.c.ConsumerConcurrency; i++ {
		consumerName := fmt.Sprintf("%s_%v", defaultRouterEventHandlerName, i)
		r.addEventDispatchHandler(consumerName)
	}
}

func (r *router) addEventDispatchHandler(consumerName string) {
	r.Router.AddHandler(
		consumerName,
		r.sub.topic(),
		r.sub.watermillSubscriber(),
		r.pub.topic(),
		r.pub.watermillPublisher(),
		func(msg *millMsg.Message) (pubMsgs []*millMsg.Message, err error) {
			eventType := msg.Metadata[keyEventType]
			r.locker.RLock()
			hdr, ok1 := r.eventHandlers[eventType]
			subhdr, ok2 := r.eventSubscriberHandlers[eventType]
			r.locker.RUnlock()
			if !ok1 && !ok2 {
				rawID := "unknown"
				if msg.Metadata != nil {
					rawID = msg.Metadata[watermill.ContextKeyRawMessageID]
				}
				return nil, errors.Errorf(
					"handle unknown event message [type[%s] message_uuid[%s] message_raw_id[%s]]",
					eventType, msg.UUID, rawID)
			}
			handlers := []*handler{hdr, subhdr}

			wg := new(sync.WaitGroup)
			futures := make([]*routine.Future, 0, len(handlers))
			for _, hdr := range handlers {
				if hdr == nil {
					continue
				}
				wg.Add(1)
				f := routine.Promise(
					func(hdr handler) (msgs any, err error) {
						_, data, _, err := pd.Unseal(msg.Payload,
							pd.Serialize(r.serializeType), pd.Compress(r.compressType), pd.Type(hdr.evtPayloadType))
						if err != nil {
							return
						}
						event := reflect.New(hdr.evtType).Interface()
						inspect.SetField(event, "pd", data)

						ctx := fmkCtx.New(fmkCtx.Watermill(msg.Metadata))
						ctx = log.SetContextFields(ctx, log.Fields{
							keyEntityID:  msg.Metadata[keyEntityID],
							keyEventType: msg.Metadata[keyEventType],
						})
						inspect.SetField(event, "ctx", ctx)
						inspect.SetField(event, "ackfn", msg.Ack)
						inspect.SetField(event, "nackfn", msg.Nack)

						rets := hdr.fn.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(event)})
						msgs = utils.ParseVariadicFuncResult[[]Message](rets, 0)
						err = utils.ParseVariadicFuncResult[error](rets, 0)
						return
					},
					true,
					routine.Args(hdr),
					routine.WaitGroup(wg),
					routine.AppName(r.appName),
				)
				futures = append(futures, f)
			}
			wg.Wait()

			pubMsgs = make([]*millMsg.Message, 0, len(handlers))
			for _, f := range futures {
				msgsAny, msgErr := f.Get()
				err = multierr.Append(err, msgErr)
				if msgsAny != nil {
					msgs, _ := msgsAny.([]Message)
					for _, m := range msgs {
						pubMsgs = append(pubMsgs, messageConvertTo(m))
					}
				}
			}
			return
		},
	)
}

func (r *router) isHandlerConflict() (conflict bool) {
	if len(r.eventHandlers) == 0 {
		return
	}
	for name := range r.Handlers() {
		if !strings.Contains(name, defaultRouterEventHandlerName) {
			return true
		}
	}
	return
}

func (r *router) close() (err error) {
	return r.Close()
}
