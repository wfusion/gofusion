package mq

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/wfusion/gofusion/common/constant"
	"github.com/wfusion/gofusion/common/infra/watermill"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/inspect"

	mw "github.com/wfusion/gofusion/common/infra/watermill/message"
)

const (
	keyEntityID                  = "entity_id"
	keyEventType                 = "event_type"
	eventHandlerSignature        = "github.com/wfusion/gofusion/mq.eventHandler["
	eventHandlerWithMsgSignature = "github.com/wfusion/gofusion/mq.eventHandlerWithMsg["
	mqPackageSignFormat          = "github.com/wfusion/gofusion/mq.%s"
)

func isEventHandler(f any) bool {
	sig := formatEventHandlerSignature(f)
	return strings.HasPrefix(sig, eventHandlerSignature) || strings.HasPrefix(sig, eventHandlerWithMsgSignature)
}
func formatEventHandlerSignature(f any) string {
	ft, ok := f.(reflect.Type)
	if !ok {
		fv, ok := f.(reflect.Value)
		if ok {
			ft = fv.Type()
		} else {
			ft = reflect.TypeOf(f)
		}
	}

	return ft.PkgPath() + "." + ft.Name()
}

type eventHandler[T eventual] func(ctx context.Context, event Event[T]) error
type eventHandlerWithMsg[T eventual] func(ctx context.Context, event Event[T]) ([]Message, error)

func EventHandler[T eventual](hdr eventHandler[T]) eventHandler[T] {
	return func(ctx context.Context, event Event[T]) error {
		// TODO: dedup & discard expired event
		return hdr(ctx, event)
	}
}

func EventHandlerWithMsg[T eventual](hdr eventHandlerWithMsg[T]) eventHandlerWithMsg[T] {
	return func(ctx context.Context, event Event[T]) ([]Message, error) {
		// TODO: dedup & discard expired event
		return hdr(ctx, event)
	}
}

func NewEventPublisherDI[T eventual](name string, opts ...utils.OptionExtender) func() EventPublisher[T] {
	return func() EventPublisher[T] {
		return NewEventPublisher[T](name, opts...)
	}
}

func NewEventPublisher[T eventual](name string, opts ...utils.OptionExtender) EventPublisher[T] {
	opt := utils.ApplyOptions[useOption](opts...)
	publisher := Pub(name, AppName(opt.appName))
	abstractMq := inspect.GetField[*abstractMQ](publisher, "abstractMQ")
	return &eventPublisher[T]{abstractMQ: abstractMq}
}

type eventPublisher[T eventual] struct {
	*abstractMQ
}

func (e *eventPublisher[T]) PublishEvent(ctx context.Context, opts ...utils.OptionExtender) (err error) {
	opt := utils.ApplyOptions[pubOption](opts...)
	optT := utils.ApplyOptions[eventPubOption[T]](opts...)
	msgs := make([]*mw.Message, 0, len(optT.events))
	for _, evt := range optT.events {
		msg, err := e.abstractMQ.newObjectMessage(ctx, evt.(*event[T]).pd, opt)
		if msg != nil {
			msg.Metadata[keyEntityID] = evt.ID()
			msg.Metadata[keyEventType] = evt.Type()
		}
		if err != nil {
			return err
		}
		msgs = append(msgs, msg)
	}
	return e.abstractMQ.Publish(ctx, messages(msgs...))
}

type eventSubscriber[T eventual] struct {
	*abstractMQ
	evtType string
}

func NewEventSubscriberDI[T eventual](name string, opts ...utils.OptionExtender) func() EventSubscriber[T] {
	return func() EventSubscriber[T] {
		return NewEventSubscriber[T](name, opts...)
	}
}

func NewEventSubscriber[T eventual](name string, opts ...utils.OptionExtender) EventSubscriber[T] {
	opt := utils.ApplyOptions[useOption](opts...)
	subscriber := Sub(name, AppName(opt.appName))
	abstractMq := inspect.GetField[*abstractMQ](subscriber, "abstractMQ")

	var m reflect.Value
	for tv := reflect.ValueOf(new(T)); tv.Kind() == reflect.Ptr; tv = tv.Elem() {
		if m = tv.MethodByName("EventType"); m.IsValid() {
			break
		}
	}
	eventType := m.Call(nil)[0].String()
	return &eventSubscriber[T]{abstractMQ: abstractMq, evtType: eventType}
}

func (e *eventSubscriber[T]) SubscribeEvent(ctx context.Context, opts ...utils.OptionExtender) (
	dst <-chan Event[T], err error) {
	opt := utils.ApplyOptions[subOption](opts...)
	out := make(chan Event[T], opt.channelLength)
	r := Use(e.name, AppName(e.appName)).(*router)
	r.Handle(
		e.evtType,
		EventHandlerWithMsg[T](func(ctx context.Context, event Event[T]) (msgs []Message, err error) {
			select {
			case out <- event:
			case <-r.Router.ClosingInProgressCh:
				event.Nack()
				e.logger.Info(fmt.Sprintf("event subscriber %s exited", e.name), nil)
				return
			case <-ctx.Done():
				event.Nack()
				e.logger.Info(fmt.Sprintf(
					"event subscriber %s exited with a message nacked when business ctx done", e.name),
					watermill.LogFields{watermill.ContextLogFieldKey: ctx})
				return
			case <-e.ctx.Done():
				event.Nack()
				e.logger.Info(fmt.Sprintf(
					"event subscriber %s exited with a message nacked when app ctx done", e.name),
					watermill.LogFields{watermill.ContextLogFieldKey: ctx})
				return
			}

			msgs = append(msgs,
				&message{Message: &mw.Message{Metadata: mw.Metadata{watermill.MessageRouterAck: ""}}})
			return
		}),
		handleEventSubscriber(),
	)
	return out, err
}

type eventual interface{ EventType() string }
type Event[T eventual] interface {
	ID() string
	Type() string
	CreatedAt() time.Time
	UpdatedAt() time.Time
	DeletedAt() time.Time
	Payload() T
	Context() context.Context
	Ack() bool
	Nack() bool
}

func NewEvent[T eventual](id string, createdAt, updatedAt, deletedAt time.Time, payload T) Event[T] {
	return newEvent[T](id, createdAt, updatedAt, deletedAt, payload)
}
func UntimedEvent[T eventual](id string, payload T) Event[T] {
	return newEvent[T](id, time.Time{}, time.Time{}, time.Time{}, payload)
}
func EventCreated[T eventual](id string, createdAt time.Time, payload T) Event[T] {
	return newEvent[T](id, createdAt, time.Time{}, time.Time{}, payload)
}
func EventUpdated[T eventual](id string, updatedAt time.Time, payload T) Event[T] {
	return newEvent[T](id, time.Time{}, updatedAt, time.Time{}, payload)
}
func EventDeleted[T eventual](id string, deletedAt time.Time, payload T) Event[T] {
	return newEvent[T](id, time.Time{}, time.Time{}, deletedAt, payload)
}

type eventPayload[T eventual] struct {
	I  string `json:"i,omitempty"`
	T  string `json:"t,omitempty"`
	P  T      `json:"p,omitempty"`
	C  string `json:"c,omitempty"`
	CL string `json:"cl,omitempty"`
	U  string `json:"u,omitempty"`
	UL string `json:"ul,omitempty"`
	D  string `json:"d,omitempty"`
	DL string `json:"dl,omitempty"`
	N  int64  `json:"n,omitempty"`
}
type event[T eventual] struct {
	ctx    context.Context
	ackfn  func() bool
	nackfn func() bool
	pd     *eventPayload[T]
}

func newEvent[T eventual](id string, createdAt, updatedAt, deletedAt time.Time, payload T) Event[T] {
	return &event[T]{
		pd: &eventPayload[T]{
			I:  id,
			T:  payload.EventType(),
			P:  payload,
			C:  createdAt.Format(time.RFC3339Nano),
			CL: createdAt.Location().String(),
			U:  updatedAt.Format(time.RFC3339Nano),
			UL: updatedAt.Location().String(),
			D:  deletedAt.Format(time.RFC3339Nano),
			DL: deletedAt.Location().String(),
			N:  time.Now().UnixNano(),
		},
	}
}
func (e *event[T]) ID() string               { return e.pd.I }
func (e *event[T]) Type() string             { return e.pd.T }
func (e *event[T]) Payload() T               { return e.pd.P }
func (e *event[T]) CreatedAt() time.Time     { return e.toTime(e.pd.C, e.pd.CL) }
func (e *event[T]) UpdatedAt() time.Time     { return e.toTime(e.pd.U, e.pd.UL) }
func (e *event[T]) DeletedAt() time.Time     { return e.toTime(e.pd.D, e.pd.DL) }
func (e *event[T]) Context() context.Context { return e.ctx }
func (e *event[T]) Ack() bool {
	if e.ackfn != nil {
		return e.ackfn()
	}
	return true
}
func (e *event[T]) Nack() bool {
	if e.nackfn != nil {
		return e.nackfn()
	}
	return true
}

func (e *event[T]) toTime(timestr, locationstr string) (t time.Time) {
	loc, err := time.LoadLocation(locationstr)
	if err != nil {
		loc = constant.DefaultLocation()
	}
	t, _ = time.ParseInLocation(time.RFC3339Nano, timestr, loc)
	return
}
