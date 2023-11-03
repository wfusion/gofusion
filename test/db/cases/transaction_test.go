package cases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/db"
	"github.com/wfusion/gofusion/log"

	testDB "github.com/wfusion/gofusion/test/db"
)

func TestTransaction(t *testing.T) {
	testingSuite := &Transaction{Test: testDB.T}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Transaction struct {
	*testDB.Test
}

func (t *Transaction) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Transaction) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Transaction) TestMysql() {
	t.testDefault(nameMysqlRead, nameMysqlWrite)
}

func (t *Transaction) TestPostgres() {
	t.testDefault(namePostgres, namePostgres)
}

func (t *Transaction) TestOpenGauss() {
	t.testDefault(nameOpenGauss, nameOpenGauss)
}

func (t *Transaction) TestSqlserver() {
	t.testDefault(nameSqlserver, nameSqlserver)
}

func (t *Transaction) testDefault(read, write string) {
	t.Run("WithTx", func() { t.testWithTx(read, write) })
}

func (t *Transaction) testWithTx(read, write string) {
	t.Catch(func() {
		ctx := context.Background()
		ctx = db.SetCtxGormDB(ctx, db.Use(ctx, read, db.AppName(testDB.Component)))
		ctx = db.SetCtxGormDB(ctx, db.Use(ctx, write, db.AppName(testDB.Component)))
		t.NoError(db.WithinTx(ctx,
			func(ctx context.Context) (err error) { return },
			db.TxUse(write),
		))
	})
}
