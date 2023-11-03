package cases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/log"

	testDB "github.com/wfusion/gofusion/test/db"
)

// TestSqlite sqlite lite test cases, it should run in serial mode because sessions are too easy to race
func TestSqlite(t *testing.T) {
	testingSuite := &Sqlite{Test: testDB.T}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Sqlite struct {
	*testDB.Test
}

func (t *Sqlite) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Sqlite) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Sqlite) TestConn() {
	(&Conn{Test: t.Test}).testDefault(nameSqlite, nameSqlite)
}

func (t *Sqlite) TestModel() {
	(&Model{Test: t.Test}).testDefault(nameSqlite, nameSqlite)
}

func (t *Sqlite) TestScan() {
	(&Scan{Test: t.Test}).testDefault(nameSqlite, nameSqlite)
}

func (t *Sqlite) TestTableSharding() {
	(&TableSharding{Test: t.Test}).testDefault(nameSqlite, nameSqlite)
}

func (t *Sqlite) TestTransaction() {
	(&Transaction{Test: t.Test}).testDefault(nameSqlite, nameSqlite)
}
