package orm

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/pkg/errors"
	"gorm.io/driver/clickhouse"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"

	"github.com/wfusion/gofusion/common/env"
	"github.com/wfusion/gofusion/common/infra/drivers/orm/opengauss"
	"github.com/wfusion/gofusion/common/infra/drivers/orm/sqlite"
	"github.com/wfusion/gofusion/common/utils"
)

var Gorm Dialect = new(gormDriver)

type gormDriver struct{}

type gormDriverOption struct {
	Driver       driver  `yaml:"driver"`
	Dialect      dialect `yaml:"dialect"`
	Timeout      string  `yaml:"timeout"`
	ReadTimeout  string  `yaml:"read_timeout"`
	WriteTimeout string  `yaml:"write_timeout"`
	User         string  `yaml:"user"`
	Password     string  `yaml:"password"`
	DBName       string  `yaml:"db_name"`
	DBCharset    string  `yaml:"db_charset"`
	DBHostname   string  `yaml:"db_hostname"`
	DBPort       string  `yaml:"db_port"`
	MaxIdleConns int     `yaml:"max_idle_conns"`
	MaxOpenConns int     `yaml:"max_open_conns"`
	Scheme       string  `yaml:"scheme"`
}

func (g *gormDriver) New(ctx context.Context, option Option, opts ...utils.OptionExtender) (db *DB, err error) {
	opt := g.parseDBOption(option)
	gormDB, dialector, err := g.open(opt.Driver, string(opt.Dialect), opt)
	if err != nil {
		return
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return
	}

	// optional
	if opt.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(opt.MaxOpenConns)
	}
	if opt.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(opt.MaxIdleConns)
	}
	if utils.IsStrNotBlank(option.ConnMaxLifeTime) {
		if liftTime, err := utils.ParseDuration(option.ConnMaxLifeTime); err == nil {
			sqlDB.SetConnMaxLifetime(liftTime)
		}
	}
	if utils.IsStrNotBlank(option.ConnMaxLifeTime) {
		if idleTime, err := utils.ParseDuration(option.ConnMaxIdleTime); err == nil {
			sqlDB.SetConnMaxIdleTime(idleTime)
		}
	}

	newOpt := utils.ApplyOptions[newOption](opts...)
	if newOpt.logger != nil {
		gormDB.Logger = newOpt.logger
	}

	return &DB{DB: gormDB.WithContext(ctx), dialector: dialector}, nil
}

func (g *gormDriver) open(driver driver, dialect string, opt *gormDriverOption) (
	db *gorm.DB, dialector gorm.Dialector, err error) {
	// alternative driver
	switch driver {
	case DriverMysql:
		dialector = mysql.New(mysql.Config{
			DriverName: dialect,
			DSN:        g.genMySqlDsn(opt),
		})

	case DriverPostgres:
		if dialect == string(DialectOpenGauss) {
			dialector = opengauss.New(opengauss.Config{
				DriverName: dialect,
				DSN:        g.genPostgresDsn(opt),
			})
		} else {
			dialector = postgres.New(postgres.Config{
				DriverName: dialect,
				DSN:        g.genPostgresDsn(opt),
			})
		}

	// sqlite dsn is filepath
	// or file::memory:?cache=shared is also available, see also https://www.sqlite.org/inmemorydb.html
	case DriverSqlite:
		dialector = sqlite.Open(path.Join(env.WorkDir, path.Clean(opt.DBName)))

	case DriverSqlserver:
		dialector = sqlserver.New(sqlserver.Config{
			DriverName: dialect,
			DSN:        g.genSqlServerDsn(opt),
		})

	// tidb is compatible with mysql protocol
	case DriverTiDB:
		dialector = mysql.New(mysql.Config{
			DriverName: dialect,
			DSN:        g.genMySqlDsn(opt),
		})

	case DriverClickhouse:
		dialector = clickhouse.New(clickhouse.Config{
			DriverName: dialect,
			DSN:        g.genClickhouseDsn(opt),
		})

	default:
		panic(errors.Errorf("unknown db driver or dialect: %s %s", driver, dialect))
	}

	db, err = gorm.Open(dialector, &gorm.Config{
		PrepareStmt:                              true,
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	return
}

func (g *gormDriver) parseDBOption(option Option) (parsed *gormDriverOption) {
	parsed = &gormDriverOption{
		Driver:       option.Driver,
		Dialect:      option.Dialect,
		Timeout:      option.Timeout,
		ReadTimeout:  option.ReadTimeout,
		WriteTimeout: option.WriteTimeout,
		User:         option.User,
		Password:     option.Password,
		DBName:       option.DB,
		DBCharset:    "utf8mb4,utf8",
		DBHostname:   option.Host,
		DBPort:       fmt.Sprintf("%v", option.Port),
		Scheme:       "tcp",
	}

	if option.Driver != "" {
		parsed.Driver = option.Driver
	}
	if option.MaxIdleConns > 0 {
		parsed.MaxIdleConns = option.MaxIdleConns
	}
	if option.MaxOpenConns > 0 {
		parsed.MaxOpenConns = option.MaxOpenConns
	}

	if utils.IsStrBlank(string(parsed.Dialect)) {
		parsed.Dialect = defaultDriverDialectMapping[parsed.Driver]
	}

	return
}

func (g *gormDriver) genMySqlDsn(opt *gormDriverOption) (dsn string) {
	if opt.DBCharset == "" {
		opt.DBCharset = "utf8"
	}
	if opt.Scheme == "" {
		opt.Scheme = "tcp"
	}

	const (
		dsnFormat = "%s:%s@%s(%s:%s)/%s?charset=%s&parseTime=True&loc=Local&timeout=%s&readTimeout=%s&writeTimeout=%s"
	)

	return fmt.Sprintf(dsnFormat, opt.User, opt.Password, opt.Scheme, opt.DBHostname, opt.DBPort, opt.DBName,
		opt.DBCharset, opt.Timeout, opt.ReadTimeout, opt.WriteTimeout)
}

func (g *gormDriver) genPostgresDsn(opt *gormDriverOption) (dsn string) {
	const (
		dsnFormat = "host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai"
	)

	return fmt.Sprintf(dsnFormat, opt.DBHostname, opt.User, opt.Password, opt.DBName, opt.DBPort)
}

func (g *gormDriver) genSqlServerDsn(opt *gormDriverOption) (dsn string) {
	const (
		dsnFormat = "sqlserver://%s:%s@%s:%s?database=%s&connection+timeout=%s"
	)

	timeout := "5" // seconds
	if utils.IsStrNotBlank(opt.Timeout) {
		if duration, err := utils.ParseDuration(opt.Timeout); err == nil {
			timeout = fmt.Sprintf("%v", int(duration/time.Second))
		}
	}

	return fmt.Sprintf(dsnFormat, opt.User, opt.Password, opt.DBHostname, opt.DBPort, opt.DBName, timeout)
}

func (g *gormDriver) genClickhouseDsn(opt *gormDriverOption) (dsn string) {
	const (
		dsnFormat = "tcp://%s:%s?database=%s&username=%s&password=%s&read_timeout=%s&write_timeout=%s"
	)

	readTimeout := "2" // seconds
	if utils.IsStrNotBlank(opt.ReadTimeout) {
		if duration, err := utils.ParseDuration(opt.ReadTimeout); err == nil {
			readTimeout = fmt.Sprintf("%v", int(duration/time.Second))
		}
	}
	writeTimeout := "2" // seconds
	if utils.IsStrNotBlank(opt.ReadTimeout) {
		if duration, err := utils.ParseDuration(opt.WriteTimeout); err == nil {
			writeTimeout = fmt.Sprintf("%v", int(duration/time.Second))
		}
	}

	return fmt.Sprintf(dsnFormat, opt.DBHostname, opt.DBPort, opt.DBName, opt.User, opt.Password,
		readTimeout, writeTimeout)
}
