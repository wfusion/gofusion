package cases

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/spf13/cast"
	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/common/constant"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/kv"
	"github.com/wfusion/gofusion/log"

	testKV "github.com/wfusion/gofusion/test/kv"
)

func TestKV(t *testing.T) {
	testingSuite := &KV{Test: new(testKV.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type KV struct {
	*testKV.Test
}

func (t *KV) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *KV) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *KV) TestRedis() {
	t.defaultTest(nameRedis, "redis:key", constant.Colon, time.Second, time.Millisecond)
}

func (t *KV) TestEtcd() {
	t.defaultTest(nameEtcd, "etcd_key", constant.Slash, time.Second, 3*time.Second)
}

func (t *KV) TestConsul() {
	t.defaultTest(nameConsul, "consul_key", constant.Slash, 10*time.Second, 21*time.Second)
}

func (t *KV) TestZookeeper() {
	t.defaultTest(nameZK, "/zk_key", constant.Slash, time.Second, time.Minute)
}

func (t *KV) defaultTest(name, key, sep string, expired, sleepTime time.Duration) {
	naming := func(n string) string { return name + "_" + n }
	t.Run(naming("Put"), func() { t.testPut(name, key+"put") })
	t.Run(naming("Set"), func() { t.testSet(name, key+"set") })
	t.Run(naming("Has"), func() { t.testHas(name, key+"has") })
	t.Run(naming("Del"), func() { t.testDel(name, key+"del") })
	t.Run(naming("Expired"), func() { t.testExpire(name, key+"expired", expired, sleepTime) })
	t.Run(naming("ExpiredDel"), func() { t.testExpireDel(name, key+"expireddel", expired) })
	t.Run(naming("QueryPrefix"), func() { t.testQueryPrefix(name, key+"qprefix", sep) })
	t.Run(naming("DeletePrefix"), func() { t.testDeletePrefix(name, key+"dprefix", sep) })
	t.Run(naming("KeysOnly"), func() { t.testKeysOnly(name, key+"keysonly", sep) })
	t.Run(naming("Paginate"), func() { t.testPaginate(name, key+"paginate", sep) })
	t.Run(naming("SetPageSize"), func() { t.testPaginateSetPageSize(name, key+"setpagesize", sep) })
	// FIXME: redis result is not stable when scan by count
	if name != nameRedis {
		t.Run(naming("FromCursor"), func() { t.testPaginateFromCursor(name, key+"fromcursor", sep) })
	}
}

func (t *KV) testPut(name, key string) {
	t.Catch(func() {
		// Given
		expect := "this is a value"
		ctx := context.Background()
		cli := kv.Use(ctx, name, kv.AppName(t.AppName()))

		// When
		t.Require().NoError(cli.Put(ctx, key, expect).Err())

		// Then
		result := cli.Get(ctx, key)
		t.Require().NoError(result.Err())
		t.Require().Equal(expect, result.String())

		t.Require().NoError(cli.Del(ctx, key).Err())
		result = cli.Get(ctx, key)
		t.Error(result.Err())
	})
}

func (t *KV) testSet(name, key string) {
	t.Catch(func() {
		// Given
		expect := "this is a value"
		ctx := context.Background()
		cli := kv.Use(ctx, name, kv.AppName(t.AppName()))

		// When
		t.Require().NoError(cli.Put(ctx, key, expect).Err())
		expect += "1"
		t.Require().NoError(cli.Put(ctx, key, expect).Err())
		expect += "2"
		t.Require().NoError(cli.Put(ctx, key, expect).Err())
		defer func() { t.Require().NoError(cli.Del(ctx, key).Err()) }()

		// Then
		result := cli.Get(ctx, key)
		t.Require().NoError(result.Err())
		t.Require().Equal(expect, result.String())
	})
}

func (t *KV) testHas(name, key string) {
	t.Catch(func() {
		// Given
		expect := "this is a value"
		ctx := context.Background()
		cli := kv.Use(ctx, name, kv.AppName(t.AppName()))
		t.Require().NoError(cli.Put(ctx, key, expect).Err())
		defer func() { t.Require().NoError(cli.Del(ctx, key).Err()) }()

		// When
		result := cli.Has(ctx, key)

		// Then
		t.Require().NoError(result.Err())
		t.Require().True(result.Bool())
	})
}

func (t *KV) testDel(name, key string) {
	t.Catch(func() {
		// Given
		expect := "this is a value"
		ctx := context.Background()
		cli := kv.Use(ctx, name, kv.AppName(t.AppName()))
		t.Require().NoError(cli.Put(ctx, key, expect).Err())
		result := cli.Has(ctx, key)
		t.Require().NoError(result.Err())
		t.Require().True(result.Bool())

		// When
		t.Require().NoError(cli.Del(ctx, key).Err())

		// Then
		result = cli.Has(ctx, key)
		t.Require().NoError(result.Err())
		t.False(result.Bool())
	})
}

func (t *KV) testExpire(name, key string, expired, sleepTime time.Duration) {
	t.Catch(func() {
		// Given
		expect := "this is a value"
		ctx := context.Background()
		cli := kv.Use(ctx, name, kv.AppName(t.AppName()))
		putActual := cli.Put(ctx, key, expect, kv.Expire(expired))
		t.Require().NoError(putActual.Err())
		defer func() {
			result := cli.Del(ctx, key, kv.LeaseID(putActual.LeaseID()))
			log.Info(ctx, "delete key(%s) result %+v after expired", key, result.Err())
		}()

		getActual := cli.Get(ctx, key)
		t.Require().NoError(getActual.Err())
		t.Require().Equal(expect, getActual.String())

		// When
		ti := time.NewTimer(sleepTime)
		defer ti.Stop()
		func() {
			begin := time.Now()
			for {
				select {
				case <-ti.C:
					return
				default:
					time.Sleep(time.Second)
					result := cli.Has(ctx, key, kv.Consistent())
					log.Info(ctx, "get key(%s) result(%v) after %s", key, result.Bool(), time.Since(begin))
					if !result.Bool() {
						return
					}
				}
			}
		}()

		// Then
		getActual = cli.Get(ctx, key)
		t.Require().Equal(kv.ErrNilValue, getActual.Err())

		existsActual := cli.Has(ctx, key)
		t.False(existsActual.Bool())
	})
}

func (t *KV) testExpireDel(name, key string, expired time.Duration) {
	t.Catch(func() {
		// Given
		expect := "this is a value"
		ctx := context.Background()

		cli := kv.Use(ctx, name, kv.AppName(t.AppName()))
		putActual := cli.Put(ctx, key, expect, kv.Expire(expired))
		t.Require().NoError(putActual.Err())
		getActual := cli.Get(ctx, key)
		t.Require().NoError(getActual.Err())
		t.Require().Equal(expect, getActual.String())

		// When
		t.Require().NoError(cli.Del(ctx, key, kv.LeaseID(putActual.LeaseID())).Err())

		// Then
		getActual = cli.Get(ctx, key)
		t.Require().Equal(kv.ErrNilValue, getActual.Err())

		existsActual := cli.Has(ctx, key)
		t.False(existsActual.Bool())
	})
}

func (t *KV) testQueryPrefix(name, key, sep string) {
	t.Catch(func() {
		// Given
		val := "this is a value"
		ctx := context.Background()
		cli := kv.Use(ctx, name, kv.AppName(t.AppName()))
		t.Require().NoError(cli.Put(ctx, key, val).Err())
		defer func() { t.Require().NoError(cli.Del(ctx, key).Err()) }()

		key1 := key + sep + "node1"
		t.Require().NoError(cli.Put(ctx, key1, val).Err())
		defer func() { t.Require().NoError(cli.Del(ctx, key1).Err()) }()

		key2 := key1 + sep + "node2"
		t.Require().NoError(cli.Put(ctx, key2, val).Err())
		defer func() { t.Require().NoError(cli.Del(ctx, key2).Err()) }()

		key3 := key + sep + "node3"
		t.Require().NoError(cli.Put(ctx, key3, val).Err())
		defer func() { t.Require().NoError(cli.Del(ctx, key3).Err()) }()

		// When
		existActual := cli.Has(ctx, key, kv.Prefix())

		// Then
		t.Require().NoError(existActual.Err())
		t.Require().True(existActual.Bool())

		getActual := cli.Get(ctx, key, kv.Prefix())
		t.Require().NoError(getActual.Err())

		kvs := getActual.KeyValues()
		t.Len(kvs, 4)
		actual := kvs.Keys()
		expect := []string{key, key1, key2, key3}
		sort.Strings(expect)
		sort.Strings(actual)
		t.Require().EqualValues(expect, actual)
		for _, item := range kvs {
			t.Require().EqualValues(val, item.Val)
		}
	})
}

func (t *KV) testDeletePrefix(name, key, sep string) {
	t.Catch(func() {
		// Given
		val := "this is a value"
		ctx := context.Background()
		cli := kv.Use(ctx, name, kv.AppName(t.AppName()))

		t.Require().NoError(cli.Put(ctx, key, val).Err())
		key1 := key + sep + "node1"
		t.Require().NoError(cli.Put(ctx, key1, val).Err())
		key2 := key1 + sep + "node2"
		t.Require().NoError(cli.Put(ctx, key2, val).Err())
		key3 := key + sep + "node3"
		t.Require().NoError(cli.Put(ctx, key3, val).Err())

		getActual := cli.Get(ctx, key, kv.Prefix())
		t.Require().NoError(getActual.Err())

		kvs := getActual.KeyValues()
		t.Len(kvs, 4)
		actual := kvs.Keys()
		expect := []string{key, key1, key2, key3}
		sort.Strings(expect)
		sort.Strings(actual)
		t.Require().EqualValues(expect, actual)
		for _, item := range kvs {
			t.Require().EqualValues(val, item.Val)
		}

		// When
		delActual := cli.Del(ctx, key, kv.Prefix())

		// Then
		t.Require().NoError(delActual.Err())

		getActual = cli.Get(ctx, key, kv.Prefix())
		t.Require().Equal(kv.ErrNilValue, getActual.Err())
	})
}

func (t *KV) testKeysOnly(name, key, sep string) {
	t.Catch(func() {
		// Given
		val := "this is a value"
		ctx := context.Background()
		cli := kv.Use(ctx, name, kv.AppName(t.AppName()))

		t.Require().NoError(cli.Put(ctx, key, val).Err())
		key1 := key + sep + "node1"
		t.Require().NoError(cli.Put(ctx, key1, val).Err())
		key2 := key1 + sep + "node2"
		t.Require().NoError(cli.Put(ctx, key2, val).Err())
		key3 := key + sep + "node3"
		t.Require().NoError(cli.Put(ctx, key3, val).Err())
		defer func() { t.Require().NoError(cli.Del(ctx, key, kv.Prefix()).Err()) }()

		// When
		getActual := cli.Get(ctx, key, kv.KeysOnly())
		getWithPrefixActual := cli.Get(ctx, key, kv.Prefix(), kv.KeysOnly())

		// Then
		t.Require().NoError(getActual.Err())
		t.Require().NoError(getWithPrefixActual.Err())

		t.Empty(getActual.String())

		kvs := getWithPrefixActual.KeyValues()
		t.Len(kvs, 4)
		actual := kvs.Keys()
		expect := []string{key, key1, key2, key3}
		sort.Strings(expect)
		sort.Strings(actual)
		t.Require().EqualValues(expect, actual)
		for _, item := range kvs {
			t.Empty(item.Val)
		}
	})
}

func (t *KV) testPaginate(name, key, sep string) {
	t.Catch(func() {
		// Given
		val := "this is a value"
		ctx := context.Background()
		cli := kv.Use(ctx, name, kv.AppName(t.AppName()))

		t.Require().NoError(cli.Put(ctx, key, val).Err())
		key1 := key + sep + "node1"
		t.Require().NoError(cli.Put(ctx, key1, val).Err())
		key2 := key1 + sep + "node2"
		t.Require().NoError(cli.Put(ctx, key2, val).Err())
		key3 := key + sep + "node3"
		t.Require().NoError(cli.Put(ctx, key3, val).Err())
		defer func() { t.Require().NoError(cli.Del(ctx, key, kv.Prefix()).Err()) }()

		// When
		iter1 := cli.Paginate(ctx, key, 1)
		iter2 := cli.Paginate(ctx, key, 2)
		iter3 := cli.Paginate(ctx, key, 3)
		iter4 := cli.Paginate(ctx, key, 4)
		iter5 := cli.Paginate(ctx, key, 5)
		iter6 := cli.Paginate(ctx, key, 1, kv.Consistent())
		iter7 := cli.Paginate(ctx, key, 1, kv.KeysOnly())
		iter8 := cli.Paginate(ctx, key, 1, kv.Consistent(), kv.KeysOnly())

		// Then
		checkFn := func(iter kv.Paginated, keysOnly bool) {
			var kvs kv.KeyValues
			for iter.More() {
				result, err := iter.Next()
				t.Require().NoError(err)
				if err != nil {
					return
				}
				kvs = append(kvs, result...)
			}

			t.Len(kvs, 4)

			expect := utils.NewSet([]string{key, key1, key2, key3}...)
			for _, item := range kvs {
				t.Require().True(expect.Contains(item.Key))
				if !keysOnly {
					t.Require().EqualValues(val, item.Val)
				}
				expect.Remove(item.Key)
			}
			t.Zero(expect.Size())
		}

		checkFn(iter1, false)
		checkFn(iter2, false)
		checkFn(iter3, false)
		checkFn(iter4, false)
		checkFn(iter5, false)
		checkFn(iter6, false)
		checkFn(iter7, true)
		checkFn(iter8, true)

	})
}

func (t *KV) testPaginateSetPageSize(name, key, sep string) {
	t.Catch(func() {
		// Given
		val := "this is a value"
		ctx := context.Background()
		cli := kv.Use(ctx, name, kv.AppName(t.AppName()))

		t.Require().NoError(cli.Put(ctx, key, val).Err())
		key1 := key + sep + "node1"
		t.Require().NoError(cli.Put(ctx, key1, val).Err())
		key2 := key1 + sep + "node2"
		t.Require().NoError(cli.Put(ctx, key2, val).Err())
		key3 := key + sep + "node3"
		t.Require().NoError(cli.Put(ctx, key3, val).Err())
		defer func() { t.Require().NoError(cli.Del(ctx, key, kv.Prefix()).Err()) }()

		expect := utils.NewSet([]string{key, key1, key2, key3}...)

		// When
		iter := cli.Paginate(ctx, key, 2)

		// Then
		var kvs kv.KeyValues
		for iter.More() {
			result, err := iter.Next()
			t.Require().NoError(err)
			if err != nil {
				return
			}
			kvs = append(kvs, result...)
			iter.SetPageSize(1)
		}

		t.Len(kvs, 4)
		for _, item := range kvs {
			t.Require().True(expect.Contains(item.Key))
			t.Require().EqualValues(val, item.Val)
			expect.Remove(item.Key)
		}
		t.Zero(expect.Size())
	})
}

func (t *KV) testPaginateFromCursor(name, key, sep string) {
	t.Catch(func() {
		// Given
		val := "this is a value"
		ctx := context.Background()
		cli := kv.Use(ctx, name, kv.AppName(t.AppName()))

		t.Require().NoError(cli.Put(ctx, key, val).Err())
		key1 := key + sep + "node1"
		t.Require().NoError(cli.Put(ctx, key1, val).Err())
		key2 := key1 + sep + "node2"
		t.Require().NoError(cli.Put(ctx, key2, val).Err())
		key3 := key + sep + "node3"
		t.Require().NoError(cli.Put(ctx, key3, val).Err())
		defer func() { t.Require().NoError(cli.Del(ctx, key, kv.Prefix()).Err()) }()

		// When
		checkFn := func(left int) {
			expect := utils.NewSet([]string{key, key1, key2, key3}...)
			total := expect.Size()

			iter := cli.Paginate(ctx, key, total-left)
			_, err := iter.Next()
			t.Require().NoError(err)

			cursor := iter.Cursor()
			iter = cli.Paginate(ctx, key, 1, kv.FromCursor(cast.ToString(cursor)))

			// Then
			var kvs kv.KeyValues
			for iter.More() {
				result, err := iter.Next()
				t.Require().NoError(err)
				if err != nil {
					return
				}
				kvs = append(kvs, result...)
			}

			t.Len(kvs, left)
			for _, item := range kvs {
				t.Require().True(expect.Contains(item.Key))
				t.Require().EqualValues(val, item.Val)
				expect.Remove(item.Key)
			}
			t.Require().EqualValues(expect.Size(), total-left)
		}

		// Then
		checkFn(0)
		checkFn(1)
		checkFn(2)
		checkFn(3)
	})
}
