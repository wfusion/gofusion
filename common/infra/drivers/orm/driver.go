package orm

import (
	"github.com/wfusion/gofusion/common/infra/drivers/orm/sqlite"
)

type driver string

const (
	DriverMysql      driver = "mysql"
	DriverTiDB       driver = "tidb"
	DriverPostgres   driver = "postgres"
	DriverSqlite     driver = sqlite.DriverName
	DriverSqlserver  driver = "sqlserver"
	DriverClickhouse driver = "clickhouse"
)

type dialect string

const (
	DialectMysql      dialect = "mysql"
	DialectPostgres   dialect = "pgx" // or pgx/v5
	DialectOpenGauss  dialect = "opengauss"
	DialectSqlite     dialect = sqlite.DriverName
	DialectSqlserver  dialect = "sqlserver" // or mssql
	DialectClickhouse dialect = "clickhouse"
)

var (
	defaultDriverDialectMapping = map[driver]dialect{
		DriverMysql:      DialectMysql,
		DriverTiDB:       DialectMysql,
		DriverPostgres:   DialectPostgres,
		DriverSqlite:     DialectSqlite,
		DriverSqlserver:  DialectSqlserver,
		DriverClickhouse: DialectClickhouse,
	}
)
