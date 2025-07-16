package cases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/db"
	"github.com/wfusion/gofusion/log"

	testDB "github.com/wfusion/gofusion/test/db"
)

func TestScan(t *testing.T) {
	testingSuite := &Scan{Test: new(testDB.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Scan struct {
	*testDB.Test
}

func (t *Scan) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Scan) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Scan) TestMysql() {
	t.testDefault(nameMysqlRead, nameMysqlWrite)
}

func (t *Scan) TestPostgres() {
	t.testDefault(namePostgres, namePostgres)
}

func (t *Scan) TestOpengauss() {
	t.testDefault(nameOpenGauss, nameOpenGauss)
}

func (t *Scan) TestSqlserver() {
	t.testDefault(nameSqlserver, nameSqlserver)
}

func (t *Scan) testDefault(read, write string) {
	t.Catch(func() {
		ctx := context.Background()
		orm := db.Use(ctx, write, db.AppName(t.AppName()))

		t.Require().NoError(orm.Migrator().AutoMigrate(new(modelWithData)))
		defer func() {
			t.Require().NoError(orm.Migrator().DropTable(new(modelWithData)))
		}()

		expected := []*modelWithData{
			{Name: "test1"}, {Name: "test2"}, {Name: "test3"},
			{Name: "test4"}, {Name: "test5"}, {Name: "test6"},
			{Name: "test7"}, {Name: "test8"},
		}
		t.Require().NoError(orm.Create(expected).Error)
		defer func() { t.Require().NoError(orm.Delete(expected).Error) }()

		actual := make([]*modelWithData, 0, len(expected))
		t.Require().NoError(
			db.Scan[modelWithData, []*modelWithData](
				ctx,
				func(mList []*modelWithData) bool { actual = append(actual, mList...); return true },
				db.ScanDAL[modelWithData, []*modelWithData](modelWithDataDAL(read, write, t.AppName())),
				db.ScanBatch(3),
				db.ScanCursor("id > ?", []string{"id"}, 0),
				db.ScanOrder("id ASC"),
				db.AppName(t.AppName()),
			),
		)

		t.Require().EqualValues(expected, actual)
	})
}
