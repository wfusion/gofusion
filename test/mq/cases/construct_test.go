package cases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/mq"

	testMq "github.com/wfusion/gofusion/test/mq"
)

func TestConstruct(t *testing.T) {
	testingSuite := &Construct{Test: testMq.T}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Construct struct {
	*testMq.Test
}

func (t *Construct) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Construct) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Construct) TestRaw() {
	t.Catch(func() {
		ctx := context.Background()
		name := "Construct_TestRaw"
		confs := map[string]*mq.Conf{
			name: {
				Topic:               name,
				Type:                "gochannel",
				Producer:            true,
				Consumer:            true,
				ConsumerGroup:       name,
				ConsumerConcurrency: 10,
				Endpoint:            nil,
				Persistent:          false,
				SerializeType:       "gob",
				CompressType:        "zstd",
				EnableLogger:        false,
				Logger:              "",
			},
		}
		mq.Construct(ctx, confs, config.AppName(testMq.Component))
		(&Raw{Test: t.Test}).defaultTest(name)
	})
}

func (t *Construct) TestEvent() {
	t.Catch(func() {
		ctx := context.Background()
		name := "Construct_TestEvent"
		confs := map[string]*mq.Conf{
			name: {
				Topic:               name,
				Type:                "gochannel",
				Producer:            true,
				Consumer:            true,
				ConsumerGroup:       name,
				ConsumerConcurrency: 10,
				Endpoint:            nil,
				Persistent:          false,
				SerializeType:       "gob",
				CompressType:        "zstd",
				EnableLogger:        false,
				Logger:              "",
			},
		}
		mq.Construct(ctx, confs, config.AppName(testMq.Component))
		(&Event{Test: t.Test}).defaultTest(name)
	})
}
