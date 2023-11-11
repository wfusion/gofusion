package lock

import (
	"context"
	"time"

	"github.com/wfusion/gofusion/common/utils"
)

const (
	ErrDuplicatedName       utils.Error = "duplicated lock name"
	ErrUnsupportedLockType  utils.Error = "unsupported lock type"
	ErrReentrantKeyNotFound utils.Error = "reentrant key for lock not found"
	ErrTimeout              utils.Error = "try to lock timeout"
	ErrContextDone          utils.Error = "try to lock when context done"

	// tolerance Default timeout to prevent deadlock
	tolerance = 2000 * time.Millisecond
)

type Lockable interface {
	Lock(ctx context.Context, key string, opts ...utils.OptionExtender) (err error)
	Unlock(ctx context.Context, key string, opts ...utils.OptionExtender) (err error)
}

type ReentrantLockable interface {
	Lockable
	ReentrantLock(ctx context.Context, key, reentrantKey string, opts ...utils.OptionExtender) (err error)
}

type lockType string

const (
	lockTypeRedisLua lockType = "redis_lua"
	lockTypeRedisNX  lockType = "redis_nx" // not support ReentrantKey
	lockTypeMySQL    lockType = "mysql"    // MariaDB versions >= 10.0.2 MySQL versions >= 5.7.5
	lockTypeMariaDB  lockType = "mariadb"  // MariaDB versions >= 10.0.2 MySQL versions >= 5.7.5
	lockTypeMongo    lockType = "mongo"    // mongo versions >= 3.6
)

// Conf lock configure
type Conf struct {
	Type     lockType `yaml:"type" json:"type" toml:"type"`
	Instance string   `yaml:"instance" json:"instance" toml:"instance"`
	Scheme   string   `yaml:"scheme" json:"scheme" toml:"scheme"`
}

type lockOption struct {
	expired      time.Duration // expired After locking, the timeout of the lock
	reentrantKey string        // reentrantKey Reentrant mark
}

func Expire(expired time.Duration) utils.OptionFunc[lockOption] {
	return func(l *lockOption) {
		l.expired = expired
	}
}

func ReentrantKey(key string) utils.OptionFunc[lockOption] {
	return func(l *lockOption) {
		l.reentrantKey = key
	}
}
