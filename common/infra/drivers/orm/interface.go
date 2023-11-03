package orm

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/wfusion/gofusion/common/utils"
)

type Dialect interface {
	New(ctx context.Context, option Option, opts ...utils.OptionExtender) (db *DB, err error)
}

type newOption struct {
	logger logger.Interface
}

// Option db option
//nolint: revive // struct tag too long issue
type Option struct {
	Driver          driver  `yaml:"driver" json:"driver" toml:"driver"`
	Dialect         dialect `yaml:"dialect" json:"dialect" toml:"dialect"`
	DB              string  `yaml:"db" json:"db" toml:"db"`
	Host            string  `yaml:"host" json:"host" toml:"host"`
	Port            uint    `yaml:"port" json:"port" toml:"port"`
	User            string  `yaml:"user" json:"user" toml:"user"`
	Password        string  `yaml:"password" json:"password" toml:"password" encrypted:""`
	Timeout         string  `yaml:"timeout" json:"timeout" toml:"timeout" default:"5s"`
	ReadTimeout     string  `yaml:"read_timeout" json:"read_timeout" toml:"read_timeout" default:"2s"`
	WriteTimeout    string  `yaml:"write_timeout" json:"write_timeout" toml:"write_timeout" default:"2s"`
	MaxIdleConns    int     `yaml:"max_idle_conns" json:"max_idle_conns" toml:"max_idle_conns" default:"20"`
	MaxOpenConns    int     `yaml:"max_open_conns" json:"max_open_conns" toml:"max_open_conns" default:"20"`
	ConnMaxLifeTime string  `yaml:"conn_max_life_time" json:"conn_max_life_time" toml:"conn_max_life_time" default:"30m"`
	ConnMaxIdleTime string  `yaml:"conn_max_idle_time" json:"conn_max_idle_time" toml:"conn_max_idle_time" default:"15m"`
}

type DB struct {
	*gorm.DB
	dialector gorm.Dialector
}

func (d *DB) GetProxy() *gorm.DB {
	return d.DB
}

func (d *DB) GetDialector() gorm.Dialector {
	return d.dialector
}

func (d *DB) WithContext(ctx context.Context) *DB {
	d.DB = d.DB.WithContext(ctx)
	return d
}
