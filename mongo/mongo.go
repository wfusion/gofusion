package mongo

import (
	"context"
	"fmt"
	"sync"

	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	"github.com/wfusion/gofusion/common/infra/drivers/mongo"
	"github.com/wfusion/gofusion/common/utils"

	mgoDrv "go.mongodb.org/mongo-driver/mongo"
)

var (
	rwlock       = new(sync.RWMutex)
	appInstances map[string]map[string]*instance
)

type instance struct {
	name     string
	database string
	mongo    *mongo.Mongo
}

func (d *instance) GetProxy() *mgoDrv.Client {
	return d.mongo.GetProxy()
}

func (d *instance) Database(opts ...*options.DatabaseOptions) *mgoDrv.Database {
	return d.mongo.Database(d.database, opts...)
}

type Mongo struct {
	*mgoDrv.Database
	Name string
}

type useOption struct {
	appName string

	// readConcern is the read concern to use for operations executed on the Database.
	// The default value is nil, which means that
	// the read concern of the Client used to configure the Database will be used.
	readConcern *readconcern.ReadConcern

	// writeConcern is the write concern to use for operations executed on the Database.
	// The default value is nil, which means that the
	// write concern of the Client used to configure the Database will be used.
	writeConcern *writeconcern.WriteConcern

	// readPreference is the read preference to use for operations executed on the Database.
	// The default value is nil, which means that
	// the read preference of the Client used to configure the Database will be used.
	readPreference *readpref.ReadPref

	// bsonOptions configures optional BSON marshaling and unmarshaling
	// behavior.
	bsonOptions *options.BSONOptions

	// registry is the BSON registry to marshal and unmarshal documents for operations executed on the Database.
	// The default value is nil,
	// which means that the registry of the Client used to configure the Database will be used.
	registry *bsoncodec.Registry
}

func AppName(name string) utils.OptionFunc[useOption] {
	return func(o *useOption) {
		o.appName = name
	}
}

func ReadConcern(readConcern *readconcern.ReadConcern) utils.OptionFunc[useOption] {
	return func(o *useOption) {
		o.readConcern = readConcern
	}
}
func WriteConcern(writeConcern *writeconcern.WriteConcern) utils.OptionFunc[useOption] {
	return func(o *useOption) {
		o.writeConcern = writeConcern
	}
}
func ReadPreference(readPreference *readpref.ReadPref) utils.OptionFunc[useOption] {
	return func(o *useOption) {
		o.readPreference = readPreference
	}
}
func BsonOptions(bsonOptions *options.BSONOptions) utils.OptionFunc[useOption] {
	return func(o *useOption) {
		o.bsonOptions = bsonOptions
	}
}
func Registry(registry *bsoncodec.Registry) utils.OptionFunc[useOption] {
	return func(o *useOption) {
		o.registry = registry
	}
}

func Use(ctx context.Context, name string, opts ...utils.OptionExtender) *Mongo {
	opt := utils.ApplyOptions[useOption](opts...)
	dbOpt := options.
		Database().
		SetReadConcern(opt.readConcern).
		SetWriteConcern(opt.writeConcern).
		SetReadPreference(opt.readPreference).
		SetBSONOptions(opt.bsonOptions).
		SetRegistry(opt.registry)

	rwlock.RLock()
	defer rwlock.RUnlock()
	instances, ok := appInstances[opt.appName]
	if !ok {
		panic(fmt.Errorf("mongo database instance not found for app: %s", opt.appName))
	}
	instance, ok := instances[name]
	if !ok {
		panic(fmt.Errorf("mongo database instance not found for name: %s", name))
	}
	return &Mongo{Database: instance.Database(dbOpt), Name: name}
}
