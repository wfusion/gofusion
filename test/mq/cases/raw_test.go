package cases

import (
	"context"
	"fmt"
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

	fusCtx "github.com/wfusion/gofusion/context"
	testMq "github.com/wfusion/gofusion/test/mq"
)

func TestRaw(t *testing.T) {
	testingSuite := &Raw{Test: new(testMq.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Raw struct {
	*testMq.Test
}

func (t *Raw) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Raw) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		ctx := context.Background()
		log.Info(ctx, "right after %s %s", suiteName, testName)
	})
}

func (t *Raw) TestRabbitmq() {
	t.defaultTest(nameRawRabbitmq)
}

func (t *Raw) TestKafka() {
	t.defaultTest(nameRawKafka)
}

func (t *Raw) TestPulsar() {
	t.defaultTest(nameRawPulsar)
}

func (t *Raw) TestRedis() {
	t.defaultTest(nameRawRedis)
}

func (t *Raw) TestMysql() {
	t.defaultTest(nameRawMysql)
}

func (t *Raw) TestPostgres() {
	t.defaultTest(nameRawPostgres)
}

func (t *Raw) TestGoChannel() {
	t.defaultTest(nameRawGoChannel)
}

func (t *Raw) defaultTest(name string) {
	naming := func(n string) string { return name + "_" + n }
	t.Run(naming("PubSubRaw"), func() { t.testPubSubRaw(name) })
	t.Run(naming("PubHandleRaw"), func() { t.testPubHandleRaw(name) })
}

func (t *Raw) testPubSubRaw(name string) {
	t.Catch(func() {
		// Given
		expected := 5
		cnt := atomic.NewInt64(0)
		ctx := context.Background()
		traceID := utils.NginxID()
		ctx = fusCtx.SetTraceID(ctx, traceID)
		ctx, cancel := context.WithTimeout(ctx, time.Duration(expected)*timeout)
		defer func() {
			time.Sleep(ackTimeout) // wait for ack
			cancel()
		}()

		objList := mock.GenObjListBySerializeAlgo(serialize.AlgorithmJson, expected).([]*mock.CommonObj)
		objMap := utils.SliceToMap(objList, func(v *mock.CommonObj) string { return v.Str })

		// When
		sub := mq.Sub(name, mq.AppName(t.AppName()))
		msgCh, err := sub.SubscribeRaw(ctx, mq.ChannelLen(expected))
		t.Require().NoError(err)

		wg := new(sync.WaitGroup)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case msg := <-msgCh:
					cnt.Add(1)
					if msg == nil {
						t.Require().NotNil(msg)
						break
					}

					ctx := msg.Context()
					log.Info(ctx, "we get raw message consumed [raw_message[%s]]", msg.ID())
					t.Require().NotEmpty(msg.ID())
					actual := utils.MustJsonUnmarshal[mock.CommonObj](msg.Payload())
					t.Require().EqualValues(objMap[msg.ID()], actual)
					t.Require().True(msg.Ack())

					if cnt.Load() == int64(len(objList)) {
						return
					}

				case <-ctx.Done():
					return
				}
			}
		}()

		t.publishAsMessage(ctx, name, objList, wg)

		// Then
		wg.Wait()
		t.Require().EqualValues(len(objList), cnt.Load())
	})
}

func (t *Raw) testPubHandleRaw(name string) {
	t.Catch(func() {
		// Given
		expected := 5
		cnt := atomic.NewInt64(0)
		ctx := context.Background()
		traceID := utils.NginxID()
		ctx = fusCtx.SetTraceID(ctx, traceID)
		ctx, cancel := context.WithTimeout(ctx, time.Duration(expected)*timeout)
		defer func() {
			time.Sleep(ackTimeout) // wait for ack
			cancel()
		}()

		objList := mock.GenObjListBySerializeAlgo(serialize.AlgorithmJson, expected).([]*mock.CommonObj)
		objMap := utils.SliceToMap(objList, func(v *mock.CommonObj) string { return v.Str })

		// When
		wg := new(sync.WaitGroup)
		r := mq.Use(name, mq.AppName(t.AppName()))
		r.Handle(fmt.Sprintf("%s_raw_message_handler", name), func(msg mq.Message) (err error) {
			cnt.Add(1)

			log.Info(msg.Context(), "we get raw message consumed [raw_message[%s]]", msg.ID())
			actual := utils.MustJsonUnmarshal[mock.CommonObj](msg.Payload())
			t.Require().EqualValues(objMap[msg.ID()], actual)
			return
		})
		r.Start()

		<-r.Running()
		t.publishAsMessage(ctx, name, objList, wg)

		// Then
		wg.Wait()
	BREAKING:
		for {
			select {
			case <-ctx.Done():
				break BREAKING
			default:
				if cnt.Load() == int64(len(objList)) {
					break BREAKING
				}
			}
		}
		t.Require().EqualValues(len(objList), cnt.Load())
	})
}

func (t *Raw) publishAsMessage(ctx context.Context, name string, objList []*mock.CommonObj, wg *sync.WaitGroup) {
	// publisher
	p := mq.Pub(name, mq.AppName(t.AppName()))

	for i := 0; i < len(objList); i++ {
		msg := mq.NewMessage(objList[i].Str, utils.MustJsonMarshal(objList[i]))
		wg.Add(1)
		go func() {
			defer wg.Done()
			t.Require().NoError(p.PublishRaw(ctx, mq.Messages(msg)))
		}()
	}
}
