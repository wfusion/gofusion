package cases

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

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
	t.defaultTest(nameRedis, "redis:key", time.Second)
}

func (t *KV) TestEtcd() {
	t.defaultTest(nameEtcd, "etcd_key", time.Second)
}

func (t *KV) TestConsul() {
	t.defaultTest(nameConsul, "consul_key", 10*time.Second)
}

func (t *KV) TestZookeeper() {
	t.defaultTest(nameZK, "/zk_key", time.Second)
}

func (t *KV) defaultTest(name, key string, expired time.Duration) {
	naming := func(n string) string { return name + "_" + n }
	t.Run(naming("GetPut"), func() { t.testGetPut(name, key, expired) })
	t.Run(naming("PutDel"), func() { t.testPutDel(name, key) })
	t.Run(naming("Set"), func() { t.testSet(name, key) })
}

func (t *KV) testGetPut(name, key string, expired time.Duration) {
	t.Catch(func() {
		// Given
		expect := "this is a value"
		ctx := context.Background()
		cli := kv.Use(ctx, name, kv.AppName(t.AppName()))

		// When
		putResult := cli.Put(ctx, key, expect, kv.Expire(expired))
		t.NoError(putResult.Err())

		defer func() { t.NoError(cli.Del(ctx, key).Err()) }()

		// Then
		result := cli.Get(ctx, key)
		t.NoError(result.Err())
		t.Equal(expect, result.String())
	})
}

func (t *KV) testPutDel(name, key string) {
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

		// Then
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

		// Then
		result := cli.Get(ctx, key)
		t.NoError(result.Err())
		t.Equal(expect, result.String())
	})
}
