package lock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/mongo"

	mgoDrv "go.mongodb.org/mongo-driver/mongo"
)

var (
	mongoInitLocker sync.Mutex
)

type mongoLocker struct {
	ctx       context.Context
	appName   string
	mongoName string
	collName  string
}

func newMongoLocker(ctx context.Context, appName, mongoName, collName string) Lockable {
	mongoInitLocker.Lock()
	defer mongoInitLocker.Unlock()

	db := mongo.Use(mongoName, mongo.AppName(appName), mongo.WriteConcern(writeconcern.Majority()))
	coll := db.Collection(collName)
	colls, err := db.ListCollectionNames(ctx, bson.M{"name": collName})
	if err != nil {
		panic(errors.Errorf("%s lock component mongo %s parse collection %s failed: %s",
			appName, mongoName, collName, err))
	}
	if len(colls) == 0 {
		if err = db.CreateCollection(ctx, collName); err != nil {
			panic(errors.Errorf("%s lock component mongo %s create collection %s failed: %s",
				appName, mongoName, collName, err))
		}
	}

	ttlIdxModel := mgoDrv.IndexModel{
		Keys:    bson.M{"expires_at": 1},
		Options: options.Index().SetExpireAfterSeconds(0),
	}
	if _, err = coll.Indexes().CreateOne(ctx, ttlIdxModel); err != nil {
		panic(errors.Errorf("%s lock component mongo %s create ttl index %s failed: %s",
			appName, mongoName, collName, err))
	}

	indexModel := mgoDrv.IndexModel{
		Keys:    bson.M{"lock_key": 1},
		Options: options.Index().SetUnique(true),
	}
	if _, err = coll.Indexes().CreateOne(ctx, indexModel); err != nil {
		panic(errors.Errorf("%s lock component mongo %s create lock index %s failed: %s",
			appName, mongoName, collName, err))
	}

	return &mongoLocker{ctx: ctx, appName: appName, mongoName: mongoName, collName: collName}
}

func (m *mongoLocker) Lock(ctx context.Context, key string, opts ...utils.OptionExtender) (err error) {
	opt := utils.ApplyOptions[lockOption](opts...)
	expired := tolerance
	if opt.expired > 0 {
		expired = opt.expired
	}
	now := time.Now()
	lockKey := m.formatLockKey(key)
	filter := bson.M{
		"lock_key": bson.M{"$eq": lockKey, "$exists": true},
		"$or": []bson.M{
			{"expires_at": bson.M{"$lt": now, "$exists": true}},
			{"holder": bson.M{"$eq": opt.reentrantKey, "$exists": true}},
		},
	}
	update := bson.M{
		"$setOnInsert": bson.M{
			"lock_key": lockKey,
			"holder":   opt.reentrantKey,
		},
		"$inc": bson.M{"count": 1},
		"$max": bson.M{"expires_at": now.Add(expired)},
	}
	mopts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	updatedDoc := new(mongoLockDoc)
	err = mongo.Use(m.mongoName, mongo.AppName(m.appName), mongo.WriteConcern(writeconcern.Majority())).
		Collection(m.collName).
		FindOneAndUpdate(ctx, filter, update, mopts).
		Decode(updatedDoc)
	if err != nil {
		return
	}

	if updatedDoc.LockKey == lockKey && updatedDoc.Holder == opt.reentrantKey {
		return
	}

	return ErrTimeout
}

func (m *mongoLocker) Unlock(ctx context.Context, key string, opts ...utils.OptionExtender) (err error) {
	opt := utils.ApplyOptions[lockOption](opts...)
	filter := bson.M{
		"lock_key": m.formatLockKey(key),
		"holder":   opt.reentrantKey,
		"count":    bson.M{"$gt": 0},
	}
	update := bson.M{
		"$inc": bson.M{"count": -1},
		"$max": bson.M{"expires_at": time.Now().Add(opt.expired / 2)},
	}
	mopts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	updatedDoc := new(mongoLockDoc)
	coll := mongo.
		Use(m.mongoName, mongo.AppName(m.appName), mongo.WriteConcern(writeconcern.Majority())).
		Collection(m.collName)
	err = coll.FindOneAndUpdate(ctx, filter, update, mopts).Decode(&updatedDoc)
	if err != nil {
		return
	}
	if updatedDoc.Count <= 0 {
		_, err = coll.DeleteOne(ctx, bson.M{
			"lock_key": m.formatLockKey(key),
			"holder":   opt.reentrantKey,
			"count":    bson.M{"$lte": 0},
		})
	}

	return
}

func (m *mongoLocker) ReentrantLock(ctx context.Context, key, reentrantKey string,
	opts ...utils.OptionExtender) (err error) {
	opt := utils.ApplyOptions[lockOption](opts...)
	if utils.IsStrBlank(opt.reentrantKey) {
		return ErrReentrantKeyNotFound
	}
	return m.Lock(ctx, key, append(opts, ReentrantKey(reentrantKey))...)
}

func (m *mongoLocker) formatLockKey(key string) (format string) {
	return fmt.Sprintf("%s_%s", config.Use(m.appName).AppName(), key)
}

type mongoLockDoc struct {
	LockKey   string    `bson:"lock_key"`
	Holder    string    `bson:"holder"`
	ExpiresAt time.Time `bson:"expires_at"`
	Count     int       `bson:"count"`
}
