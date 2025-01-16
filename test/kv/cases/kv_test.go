package cases

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/wfusion/gofusion/common/constant"

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
	t.Run(naming("Exists"), func() { t.testExists(name, key+"exists") })
	t.Run(naming("Expired"), func() { t.testExpire(name, key+"expired", expired, sleepTime) })
	t.Run(naming("ExpiredDel"), func() { t.testExpireDel(name, key+"expireddel", expired) })
	t.Run(naming("Prefix"), func() { t.testPrefix(name, key+"prefix", sep) })
}

func (t *KV) testPut(name, key string) {
	t.Catch(func() {
		// Given
		expect := "this is a value"
		ctx := context.Background()
		cli := kv.Use(ctx, name, kv.AppName(t.AppName()))

		// When
		t.NoError(cli.Put(ctx, key, expect).Err())

		// Then
		result := cli.Get(ctx, key)
		t.NoError(result.Err())
		t.Equal(expect, result.String())

		t.NoError(cli.Del(ctx, key).Err())
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
		t.NoError(cli.Put(ctx, key, expect).Err())
		expect += "1"
		t.NoError(cli.Put(ctx, key, expect).Err())
		expect += "2"
		t.NoError(cli.Put(ctx, key, expect).Err())
		defer func() { t.NoError(cli.Del(ctx, key).Err()) }()

		// Then
		result := cli.Get(ctx, key)
		t.NoError(result.Err())
		t.Equal(expect, result.String())
	})
}

func (t *KV) testExists(name, key string) {
	t.Catch(func() {
		// Given
		expect := "this is a value"
		ctx := context.Background()
		cli := kv.Use(ctx, name, kv.AppName(t.AppName()))
		t.NoError(cli.Put(ctx, key, expect).Err())
		defer func() { t.NoError(cli.Del(ctx, key).Err()) }()

		// When
		result := cli.Exists(ctx, key)

		// Then
		t.NoError(result.Err())
		t.True(result.Bool())
	})
}

func (t *KV) testExpire(name, key string, expired, sleepTime time.Duration) {
	t.Catch(func() {
		// Given
		expect := "this is a value"
		ctx := context.Background()
		cli := kv.Use(ctx, name, kv.AppName(t.AppName()))
		putActual := cli.Put(ctx, key, expect, kv.Expire(expired))
		t.NoError(putActual.Err())
		defer func() {
			result := cli.Del(ctx, key, kv.LeaseID(putActual.LeaseID()))
			log.Info(ctx, "delete key(%s) result %+v after expired", key, result.Err())
		}()

		getActual := cli.Get(ctx, key)
		t.NoError(getActual.Err())
		t.Equal(expect, getActual.String())

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
					getActual = cli.Get(ctx, key)
					log.Info(ctx, "get key(%s) result(%s) after %s", key, getActual, time.Since(begin))
					if getActual.String() != expect {
						return
					}
				}
			}
		}()

		// Then
		getActual = cli.Get(ctx, key)
		t.Equal(kv.ErrNilValue, getActual.Err())

		existsActual := cli.Exists(ctx, key)
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
		t.NoError(putActual.Err())
		getActual := cli.Get(ctx, key)
		t.NoError(getActual.Err())
		t.Equal(expect, getActual.String())

		// When
		t.NoError(cli.Del(ctx, key, kv.LeaseID(putActual.LeaseID())).Err())

		// Then
		getActual = cli.Get(ctx, key)
		t.Equal(kv.ErrNilValue, getActual.Err())

		existsActual := cli.Exists(ctx, key)
		t.False(existsActual.Bool())
	})
}

func (t *KV) testPrefix(name, key, sep string) {
	t.Catch(func() {
		// Given
		val := "this is a value"
		ctx := context.Background()
		cli := kv.Use(ctx, name, kv.AppName(t.AppName()))
		t.NoError(cli.Put(ctx, key, val).Err())
		defer func() { t.NoError(cli.Del(ctx, key).Err()) }()

		key1 := key + sep + "node1"
		t.NoError(cli.Put(ctx, key1, val).Err())
		defer func() { t.NoError(cli.Del(ctx, key1).Err()) }()

		key2 := key1 + sep + "node2"
		t.NoError(cli.Put(ctx, key2, val).Err())
		defer func() { t.NoError(cli.Del(ctx, key2).Err()) }()

		key3 := key + sep + "node3"
		t.NoError(cli.Put(ctx, key3, val).Err())
		defer func() { t.NoError(cli.Del(ctx, key3).Err()) }()

		// When
		existActual := cli.Exists(ctx, key, kv.Prefix())

		// Then
		t.NoError(existActual.Err())
		t.True(existActual.Bool())

		getActual := cli.Get(ctx, key, kv.Prefix())
		t.NoError(getActual.Err())

		kvs := getActual.KeyValues()
		t.Len(kvs, 4)
		actual := kvs.Keys()
		expect := []string{key, key1, key2, key3}
		sort.Strings(expect)
		sort.Strings(actual)
		t.EqualValues(expect, actual)
		for _, item := range kvs {
			t.EqualValues(val, item.Val)
		}
	})
}
