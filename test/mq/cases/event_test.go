package cases

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/atomic"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/serialize"
	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/mq"
	"github.com/wfusion/gofusion/test/internal/mock"

	fmkCtx "github.com/wfusion/gofusion/context"
	testMq "github.com/wfusion/gofusion/test/mq"
)

func TestEvent(t *testing.T) {
	testingSuite := &Event{Test: new(testMq.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Event struct {
	*testMq.Test
}

func (t *Event) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Event) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Event) TestRabbitmq() {
	t.defaultTest(nameEventRabbitmq)
}

func (t *Event) TestKafka() {
	t.defaultTest(nameEventKafka)
}

func (t *Event) TestPulsar() {
	t.defaultTest(nameEventPulsar)
}

func (t *Event) TestRedis() {
	t.defaultTest(nameEventRedis)
}

func (t *Event) TestMysql() {
	t.defaultTest(nameEventMysql)
}

func (t *Event) TestPostgres() {
	t.defaultTest(nameEventPostgres)
}

func (t *Event) TestGoChannel() {
	t.defaultTest(nameEventGoChannel)
}

func (t *Event) defaultTest(name string) {
	naming := func(n string) string { return name + "_" + n }
	t.Run(naming("PubSubEvent"), func() { t.testPubSubEvent(name) })
	t.Run(naming("PubHandlerEvent"), func() { t.testPubHandlerEvent(name) })
}

func (t *Event) testPubSubEvent(name string) {
	t.Catch(func() {
		// Given
		expected := 5
		cnt := atomic.NewInt64(0)
		ctx := context.Background()
		traceID := utils.NginxID()
		ctx = fmkCtx.SetTraceID(ctx, traceID)
		ctx, cancel := context.WithTimeout(ctx, time.Duration(expected)*timeout)
		defer func() {
			time.Sleep(time.Second) // wait for ack
			cancel()
		}()

		structObjList := mock.GenObjList[*structCreated](expected)
		structEventType := (*structCreated).EventType(nil)
		structObjMap := utils.SliceToMap(structObjList, func(v *structCreated) string { return v.ID })

		// When
		wg := new(sync.WaitGroup)
		structSub := mq.NewEventSubscriber[*structCreated](name, mq.AppName(t.AppName()))
		structMsgCh, err := structSub.SubscribeEvent(ctx, mq.ChannelLen(expected))
		t.NoError(err)

		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case msg := <-structMsgCh:
					cnt.Add(1)

					t.True(msg.Ack())
					ctx := msg.Context()
					log.Info(ctx, "subscriber get struct created event consumed [event[%s]]", msg.ID())

					t.NotEmpty(msg.ID())
					t.EqualValues(msg.Type(), structEventType)
					t.EqualValues(structObjMap[msg.ID()], msg.Payload())
					if cnt.Load() == int64(len(structObjList)) {
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}()
		<-mq.Use(name, mq.AppName(t.AppName())).Running()
		t.publishStruct(ctx, name, structObjList, wg)

		// Then
		wg.Wait()
		t.EqualValues(len(structObjList), cnt.Load())
	})
}

func (t *Event) testPubHandlerEvent(name string) {
	t.Catch(func() {
		// Given
		expected := 5
		cnt := atomic.NewInt64(0)
		ctx := context.Background()
		traceID := utils.NginxID()
		ctx = fmkCtx.SetTraceID(ctx, traceID)
		ctx, cancel := context.WithTimeout(ctx, time.Duration(expected)*timeout)
		defer func() {
			time.Sleep(time.Second) // wait for ack
			cancel()
		}()

		randomObjList := mock.GenObjListBySerializeAlgo(serialize.AlgorithmGob, expected).([]*mock.RandomObj)
		randomEventType := (*mock.RandomObj).EventType(nil)
		randomObjMap := utils.SliceToMap(randomObjList, func(v *mock.RandomObj) string { return v.Str })

		// When
		wg := new(sync.WaitGroup)
		r := mq.Use(name, mq.AppName(t.AppName()))
		r.Handle(randomEventType, mq.EventHandler(
			func(ctx context.Context, event mq.Event[*mock.RandomObj]) (err error) {
				// Then
				cnt.Add(1)
				t.EqualValues(traceID, fmkCtx.GetTraceID(ctx))
				t.EqualValues(event.Type(), randomEventType)
				t.EqualValues(randomObjMap[event.ID()], event.Payload())

				log.Info(ctx, "router get random event consumed [event[%s]]", event.ID())
				return
			},
		))
		r.Start()

		<-r.Running()
		t.publishRandom(ctx, name, randomObjList, wg)

		// Then
		wg.Wait()
	BREAKING:
		for {
			select {
			case <-ctx.Done():
				break BREAKING
			default:
				if cnt.Load() == int64(len(randomObjList)) {
					break BREAKING
				}
			}
		}
		t.EqualValues(len(randomObjList), cnt.Load())
	})
}

func (t *Event) publishRandom(ctx context.Context, name string, objList []*mock.RandomObj, wg *sync.WaitGroup) {
	// publisher
	p := mq.NewEventPublisher[*mock.RandomObj](name, mq.AppName(t.AppName()))

	for i := 0; i < len(objList); i++ {
		event := mq.UntimedEvent(objList[i].Str, objList[i])
		wg.Add(1)
		go func() {
			defer wg.Done()
			t.NoError(p.PublishEvent(ctx, mq.Events(event)))
		}()
	}
}

func (t *Event) publishStruct(ctx context.Context, name string, objList []*structCreated, wg *sync.WaitGroup) {
	// publisher
	p := mq.NewEventPublisher[*structCreated](name, mq.AppName(t.AppName()))

	for i := 0; i < len(objList); i++ {
		event := mq.UntimedEvent(objList[i].ID, objList[i])
		wg.Add(1)
		go func() {
			defer wg.Done()
			t.NoError(p.PublishEvent(ctx, mq.Events(event)))
		}()
	}
}
