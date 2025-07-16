package cases

import (
	"context"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/go-faker/faker/v4"
	"github.com/spf13/cast"
	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/cache"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/serialize"
	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/test/internal/mock"

	testCache "github.com/wfusion/gofusion/test/cache"
)

func TestRedis(t *testing.T) {
	testingSuite := &Redis{Test: new(testCache.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Redis struct {
	*testCache.Test
}

func (t *Redis) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Redis) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Redis) TestRedis() {
	t.Catch(func() {
		// Given
		num := 15
		ctx := context.Background()
		algo := serialize.AlgorithmGob
		instance := cache.New[string, *mock.RandomObj, []*mock.RandomObj](redis, cache.AppName(t.AppName()))
		objList := mock.GenObjListBySerializeAlgo(algo, num).([]*mock.RandomObj)
		stringObjMap := make(map[string]*mock.RandomObj, num)
		for i := 0; i < num; i++ {
			stringObjMap[cast.ToString(i+1)] = objList[i]
		}
		defer instance.Clear(ctx)

		// When
		instance.Set(ctx, stringObjMap)

		// Then
		keys := []string{"13"}
		rs := instance.Get(ctx, keys, t.randomObjCallback(stringObjMap, algo, false))
		t.Require().NotEmpty(rs)

		keys = []string{"1"}
		rs = instance.Get(ctx, keys, t.randomObjCallback(stringObjMap, algo, false))
		t.Require().NotEmpty(rs)

		keys = []string{"1"}
		rs = instance.Get(ctx, keys, t.randomObjCallback(stringObjMap, algo, false))
		t.Require().NotEmpty(rs)

		time.Sleep(5 * time.Second)
		keys = []string{"1"}
		rs = instance.Get(ctx, keys, t.randomObjCallback(stringObjMap, algo, true))
		t.Require().NotEmpty(rs)
	})
}

func (t *Redis) TestClear() {
	t.Catch(func() {
		// Given
		ctx := context.Background()
		algo := serialize.AlgorithmGob
		instance := cache.New[string, *mock.RandomObj, []*mock.RandomObj](redis, cache.AppName(t.AppName()))
		stringObjMap := map[string]*mock.RandomObj{
			"1": mock.GenObjBySerializeAlgo(algo).(*mock.RandomObj),
			"2": mock.GenObjBySerializeAlgo(algo).(*mock.RandomObj),
			"3": mock.GenObjBySerializeAlgo(algo).(*mock.RandomObj),
		}
		defer instance.Clear(ctx)

		// When
		instance.Set(ctx, stringObjMap)

		// Then
		keys := []string{"1", "2", "3"}
		rs := instance.Get(ctx, keys, t.randomObjCallback(stringObjMap, algo, false))
		t.Require().NotEmpty(rs)

		instance.Clear(ctx)
		rs = instance.Get(ctx, keys, t.randomObjCallback(stringObjMap, algo, true))
		t.Require().NotEmpty(rs)
	})
}

func (t *Redis) TestDel() {
	t.Catch(func() {
		// Given
		ctx := context.Background()
		algo := serialize.AlgorithmGob
		instance := cache.New[string, *mock.RandomObj, []*mock.RandomObj](redis, cache.AppName(t.AppName()))
		stringObjMap := map[string]*mock.RandomObj{
			"1": mock.GenObjBySerializeAlgo(algo).(*mock.RandomObj),
			"2": mock.GenObjBySerializeAlgo(algo).(*mock.RandomObj),
			"3": mock.GenObjBySerializeAlgo(algo).(*mock.RandomObj),
		}
		defer instance.Clear(ctx)

		// When
		instance.Set(ctx, stringObjMap)

		// Then
		keys := []string{"1", "2", "3"}
		rs := instance.Get(ctx, keys, t.randomObjCallback(stringObjMap, algo, false))
		t.Require().NotEmpty(rs)

		instance.Del(ctx, keys...)
		rs = instance.Get(ctx, keys, t.randomObjCallback(stringObjMap, algo, true))
		t.Require().NotEmpty(rs)
	})
}

func (t *Redis) TestDelWithFailureKeys() {
	t.Catch(func() {
		// Given
		ctx := context.Background()
		algo := serialize.AlgorithmGob
		instance := cache.New[string, *mock.RandomObj, []*mock.RandomObj](redis, cache.AppName(t.AppName()))
		stringObjMap := map[string]*mock.RandomObj{
			"1": mock.GenObjBySerializeAlgo(algo).(*mock.RandomObj),
			"2": mock.GenObjBySerializeAlgo(algo).(*mock.RandomObj),
			"3": mock.GenObjBySerializeAlgo(algo).(*mock.RandomObj),
		}
		defer instance.Clear(ctx)

		// When
		keys := []string{"1", "2"}
		instance.Set(ctx, stringObjMap)
		failureKeys := instance.Del(ctx, keys...)
		t.Empty(failureKeys)

		// Then
		failureKeys = instance.Del(ctx, keys...)
		t.Empty(failureKeys)
	})
}

func (t *Redis) TestSetGetInParallel() {
	t.Catch(func() {
		// Given
		ctx := context.Background()
		instance := cache.New[string, *mock.CommonObj, []*mock.CommonObj](redisJson, cache.AppName(t.AppName()))
		defer instance.Clear(ctx)

		wg := new(sync.WaitGroup)
		for i := 0; i < 500; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				key := faker.UUIDHyphenated()
				val := mock.GenObjBySerializeAlgo(serialize.AlgorithmJson).(*mock.CommonObj)
				instance.Set(ctx, map[string]*mock.CommonObj{key: val})
				rs := instance.Get(ctx, []string{key}, commonObjCallback)
				t.Require().EqualValues(val, rs[0])
			}()
		}
		wg.Wait()
	})
}

func (t *Redis) TestLocalWithCompress() {
	t.Catch(func() {
		// Given
		ctx := context.Background()

		type cases struct {
			name      string
			cacheName string
		}

		testCases := []cases{
			{
				name:      "zstd",
				cacheName: redisWithZstdCompress,
			},
			{
				name:      "zlib",
				cacheName: redisWithZlibCompress,
			},
			{
				name:      "s2",
				cacheName: redisWithS2Compress,
			},
			{
				name:      "gzip",
				cacheName: redisWithGzipCompress,
			},
			{
				name:      "deflate",
				cacheName: redisWithDeflateCompress,
			},
		}

		algo := serialize.AlgorithmGob
		for _, cs := range testCases {
			instance := cache.New[string, *mock.RandomObj, []*mock.RandomObj](
				cs.cacheName, cache.AppName(t.AppName()))
			t.Run(cs.name, func() {
				defer instance.Clear(ctx)
				t.runInParallel(func() {
					randomKey := faker.UUIDHyphenated()
					stringObjMap := map[string]*mock.RandomObj{
						randomKey: mock.GenObjBySerializeAlgo(algo).(*mock.RandomObj),
					}
					instance.Set(ctx, stringObjMap)

					// When
					randomKeys := [3]string{}
					t.Require().NoError(faker.FakeData(&randomKeys))
					keys := append([]string{randomKey}, randomKeys[:]...)
					rs := instance.Get(ctx, keys, t.randomObjCallback(stringObjMap, algo, true))

					// Then
					t.Require().Equal(len(rs), len(keys))
					t.Require().EqualValues(stringObjMap[randomKey], rs[0])
					for i := 0; i < len(rs); i++ {
						t.Require().NotEmpty(rs[i])
					}
				})
			})
		}
	})
}

func (t *Redis) runInParallel(exec func()) {
	wg := new(sync.WaitGroup)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			exec()
			time.Sleep(time.Duration(float64(time.Millisecond) * rand.Float64()))
		}()
	}
	wg.Wait()
}

func (t *Redis) randomObjCallback(origin map[string]*mock.RandomObj, algo serialize.Algorithm, mayMissing bool) (
	cb func(context.Context, []string) (map[string]*mock.RandomObj, []utils.OptionExtender)) {
	return func(ctx context.Context, missed []string) (rs map[string]*mock.RandomObj, opts []utils.OptionExtender) {
		if !mayMissing {
			t.FailNow("cache missing!", missed)
		}

		rs = make(map[string]*mock.RandomObj, len(missed))
		for _, key := range missed {
			if v, ok := origin[key]; ok {
				rs[key] = v
			} else {
				rs[key] = mock.GenObjBySerializeAlgo(algo).(*mock.RandomObj)
			}
		}
		return
	}
}

func (t *Redis) commonObjCallback(origin map[string]*mock.CommonObj, algo serialize.Algorithm, mayMissing bool) (
	cb func(context.Context, []string) (map[string]*mock.CommonObj, []utils.OptionExtender)) {
	return func(ctx context.Context, missed []string) (rs map[string]*mock.CommonObj, opts []utils.OptionExtender) {
		if !mayMissing {
			t.FailNow("cache missing!", missed)
		}

		rs = make(map[string]*mock.CommonObj, len(missed))
		for _, key := range missed {
			if v, ok := origin[key]; ok {
				rs[key] = v
			} else {
				rs[key] = mock.GenObjBySerializeAlgo(algo).(*mock.CommonObj)
			}
		}
		return
	}
}
