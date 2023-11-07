package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/wfusion/gofusion/common/utils"
)

type Dialect interface {
	New(ctx context.Context, option Option, opts ...utils.OptionExtender) (cli *Mongo, err error)
}

type newOption struct {
	monitor     *event.CommandMonitor
	poolMonitor *event.PoolMonitor
}

type Mongo struct {
	*mongo.Client
}

func (m *Mongo) GetProxy() *mongo.Client {
	return m.Client
}

func (m *Mongo) Database(name string, opts ...*options.DatabaseOptions) *mongo.Database {
	return m.Client.Database(name, opts...)
}
