package cases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/log"
	"github.com/wfusion/gofusion/mongo"

	testMgo "github.com/wfusion/gofusion/test/mongo"
	mgoDrv "go.mongodb.org/mongo-driver/mongo"
)

func TestConn(t *testing.T) {
	testingSuite := &Conn{Test: new(testMgo.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Conn struct {
	*testMgo.Test
}

func (t *Conn) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Conn) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Conn) TestPing() {
	t.Catch(func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		mgoCli := mongo.Use(nameDefault, mongo.AppName(t.AppName()))
		err := mgoCli.Client().Ping(ctx, nil)
		t.NoError(err)
	})
}

func (t *Conn) TestCollections() {
	t.Catch(func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		name := "TestCollections"

		db := mongo.Use(nameDefault, mongo.AppName(t.AppName()))
		err := db.CreateCollection(ctx, name)
		t.NoError(err)

		coll := db.Collection(name)
		defer func() {
			t.NoError(coll.Drop(ctx))
		}()

		err = coll.FindOne(ctx, nil).Err()
		t.EqualValues(mgoDrv.ErrNilDocument, err)
	})
}
