package cases

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/atomic"

	"github.com/wfusion/gofusion/async"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/serialize"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/redis"
	"github.com/wfusion/gofusion/test/internal/mock"

	fusCtx "github.com/wfusion/gofusion/context"
	testAsync "github.com/wfusion/gofusion/test/async"
)

func TestAsynq(t *testing.T) {
	testingSuite := &Asynq{Test: new(testAsync.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Asynq struct {
	*testAsync.Test
}

func (t *Asynq) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Asynq) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Asynq) TestDefault() {
	t.Catch(func() {
		// Given
		times := 2
		c := async.C(nameDefault, async.AppName(t.AppName()))
		p := async.P(nameDefault, async.AppName(t.AppName()))
		ctx := fusCtx.SetTraceID(context.Background(), utils.NginxID())
		t.cleanByQueue(ctx, "")
		defer t.cleanByQueue(ctx, "")

		ctx, cancel := context.WithTimeout(ctx, time.Minute)
		defer cancel()

		// When
		css := make([]*cs, 0)
		css = append(css, t.testDefault(ctx, c, p, times))
		css = append(css, t.testVariadicHandler(ctx, c, p, times))

		t.Require().NoError(c.Start())

		// Then
		wg := new(sync.WaitGroup)
		for _, item := range css {
			testCase := item
			wg.Add(1)
			go func() {
				defer wg.Done()
				testCase.subTest()
			}()
		}
		wg.Wait()
	})
}

func (t *Asynq) testDefault(ctx context.Context, c async.Consumable, p async.Producable, times int) *cs {
	// Given
	taskName := "testDefault"
	cnt := atomic.NewInt32(0)
	expect := time.Duration(times)
	obj := mock.GenObjBySerializeAlgo(serialize.AlgorithmGob)

	// When
	c.Handle(taskName, func(ctx context.Context, arg *mock.RandomObj) (err error) {
		cnt.Add(1)
		deadline, ok := ctx.Deadline()
		log.Info(ctx, "testDefault get async task args: ctx(%s,%v)", deadline, ok)
		t.Require().EqualValues(obj, arg)
		return
	})

	// Then
	return &cs{
		name: "default",
		subTest: func() {
			for i := 0; i < int(expect); i++ {
				t.Require().NoError(
					p.Send(ctx, taskName, obj),
				)
			}
			time.Sleep(time.Duration(times) * time.Second)

			t.NotZero(cnt.Load())
			t.LessOrEqual(cnt.Load(), int32(expect))
		},
	}
}

func (t *Asynq) testVariadicHandler(ctx context.Context, c async.Consumable, p async.Producable, times int) *cs {
	// Given
	expect := time.Duration(times)
	cnt := atomic.NewInt32(0)
	obj1 := mock.GenObjBySerializeAlgo(serialize.AlgorithmGob)
	obj2 := mock.GenObjBySerializeAlgo(serialize.AlgorithmGob)
	obj3 := mock.GenObjBySerializeAlgo(serialize.AlgorithmGob)
	obj4 := 10

	hdr := func(ctx context.Context, a1, a2, a3 *mock.RandomObj, a4 int) (err error) {
		cnt.Add(1)
		deadline, ok := ctx.Deadline()
		log.Info(ctx, "testVariadicHandler get async task args: ctx(%s,%v)", deadline, ok)
		t.Require().EqualValues(obj1, a1)
		t.Require().EqualValues(obj2, a2)
		t.Require().EqualValues(obj3, a3)
		t.Require().EqualValues(obj4, a4)
		return
	}

	// When
	c.HandleFunc(hdr)

	// Then
	return &cs{
		name: "variadic_handler",
		subTest: func() {
			for i := 0; i < int(expect); i++ {
				t.Require().NoError(
					p.Goc(ctx, hdr, async.Args(obj1, obj2, obj3, obj4)),
				)
			}
			time.Sleep(time.Duration(times) * time.Second)

			t.NotZero(cnt.Load())
			t.LessOrEqual(cnt.Load(), int32(expect))
		},
	}
}

func (t *Asynq) TestWithQueue() {
	t.Catch(func() {
		// Given
		queue := "gofusion:async:with_queues"
		expect := time.Duration(2)
		cnt := atomic.NewInt32(0)
		ctx := fusCtx.SetTraceID(context.Background(), utils.NginxID())
		t.cleanByQueue(ctx, queue)
		defer t.cleanByQueue(ctx, queue)

		obj1 := mock.GenObjBySerializeAlgo(serialize.AlgorithmGob)
		obj2 := mock.GenObjBySerializeAlgo(serialize.AlgorithmGob)
		obj3 := mock.GenObjBySerializeAlgo(serialize.AlgorithmGob)
		obj4 := 10

		c := async.C(nameWithQueue, async.AppName(t.AppName()))
		p := async.P(nameWithQueue, async.AppName(t.AppName()))
		hdr := func(ctx context.Context, a1, a2, a3 *mock.RandomObj, a4 int) (err error) {
			cnt.Add(1)
			deadline, ok := ctx.Deadline()
			log.Info(ctx, "TestWithQueue get async task args: ctx(%s,%v)", deadline, ok)
			t.Require().EqualValues(obj1, a1)
			t.Require().EqualValues(obj2, a2)
			t.Require().EqualValues(obj3, a3)
			t.Require().EqualValues(obj4, a4)
			return
		}

		c.Handle("TestWithQueue", hdr)
		ctx, cancel := context.WithTimeout(ctx, time.Minute)
		defer cancel()
		for i := 0; i < int(expect); i++ {
			t.Require().NoError(
				p.Go(hdr, async.Args(obj1, obj2, obj3, obj4), async.Queue(queue)),
			)
		}

		// When
		t.Require().NoError(c.Start())
		time.Sleep(expect * time.Second)

		// Then
		t.NotZero(cnt.Load())
		t.LessOrEqual(cnt.Load(), int32(expect))
	})
}

func (t *Asynq) cleanByQueue(ctx context.Context, queue string) {
	pattern := fmt.Sprintf("asynq:{%s}:*", queue)
	if queue == "" {
		pattern = fmt.Sprintf("asynq:{%s:async}:*", config.Use(t.AppName()).AppName())
	}

	rdsCli := redis.Use(ctx, "default", redis.AppName(t.AppName()))
	keys, err := rdsCli.Keys(ctx, pattern).Result()
	t.Require().NoError(err)

	if len(keys) > 0 {
		rdsCli.Del(ctx, keys...)
	}
}
