package cases

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/db"
	"github.com/wfusion/gofusion/log"

	testDB "github.com/wfusion/gofusion/test/db"
)

func TestModel(t *testing.T) {
	testingSuite := &Model{Test: new(testDB.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type Model struct {
	*testDB.Test
}

func (t *Model) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *Model) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *Model) TestMysql() {
	t.testDefault(nameMysqlRead, nameMysqlWrite)
}

func (t *Model) TestPostgres() {
	t.testDefault(namePostgres, namePostgres)
}

func (t *Model) TestOpengauss() {
	t.testDefault(nameOpenGauss, nameOpenGauss)
}

func (t *Model) TestSqlserver() {
	t.testDefault(nameSqlserver, nameSqlserver)
}

func (t *Model) testDefault(read, write string) {
	t.Run("DataModel", func() { t.testDataModel(read, write) })
	t.Run("Joins", func() { t.testJoins(read, write) })
	t.Run("SoftDelete", func() { t.testSoftDelete(read, write) })
	t.Run("BusinessSoftDelete", func() { t.testBusinessSoftDelete(read, write) })
	t.Run("SoftDeleteUnscoped", func() { t.testSoftDeleteUnscoped(read, write) })
}

func (t *Model) testDataModel(read, write string) {
	t.Catch(func() {
		ctx := context.Background()
		orm := db.Use(ctx, write, db.AppName(t.AppName())).WithContext(ctx)

		t.Require().NoError(orm.Migrator().AutoMigrate(new(modelWithData)))
		t.Require().NoError(orm.Migrator().AutoMigrate(new(modelWithBusiness)))
		t.Require().NoError(orm.Migrator().AutoMigrate(new(modelWithBusinessAndUser)))
		defer func() {
			t.Require().NoError(orm.Migrator().DropTable(new(modelWithData)))
			t.Require().NoError(orm.Migrator().DropTable(new(modelWithBusiness)))
			t.Require().NoError(orm.Migrator().DropTable(new(modelWithBusinessAndUser)))
		}()

		mwd1 := &modelWithData{Name: "az1"}
		t.Require().NoError(orm.Create(mwd1).Error)
		t.Require().NoError(orm.Where("name = 'az1'").Delete(mwd1).Error)

		mwd2 := &modelWithData{Name: "az2"}
		t.Require().NoError(orm.Create(mwd2).Error)
		t.Require().NoError(orm.Where("name = 'az2'").Delete(mwd2).Error)

		//ret := orm.Exec("insert into model_with_data(create_time, modify_time, name) values(1685702487694,1685702487694,'az3')")
		//t.Require().NoError(ret.Error)
		//id, _ := ret.Statement.Schema.PrioritizedPrimaryField.ValueOf(ret.Statement.Context, ret.Statement.ReflectValue)
		//t.Require().NoError(orm.Delete(mwd2, "name = 'az3' AND id = ?", id).Error)

		mwb := &modelWithBusiness{Name: "test"}
		t.Require().NoError(orm.Create(mwb).Error)
		t.Require().NoError(orm.Delete(mwb).Error)

		mwbu := &modelWithBusinessAndUser{Name: "test"}
		t.Require().NoError(orm.Create(mwbu).Error)
		t.Require().NoError(orm.Delete(mwbu).Error)
	})
}

func (t *Model) testJoins(read, write string) {
	t.Catch(func() {
		ctx := context.Background()
		orm := db.Use(ctx, write, db.AppName(t.AppName())).WithContext(ctx)

		t.Require().NoError(orm.Migrator().AutoMigrate(new(modelWithData)))
		t.Require().NoError(orm.Migrator().AutoMigrate(new(modelWithDataExtend)))
		defer func() {
			t.Require().NoError(orm.Migrator().DropTable(new(modelWithData)))
			t.Require().NoError(orm.Migrator().DropTable(new(modelWithDataExtend)))
		}()
		mwd := &modelWithData{Name: "az1"}
		t.Require().NoError(orm.Create(mwd).Error)
		mwdTableName := mwd.TableName()

		mwdt := &modelWithDataExtend{OtherName: "test", ModelID: mwd.ID}
		t.Require().NoError(orm.Create(mwdt).Error)
		mwdtTableName := mwdt.TableName()

		var mwdList []*modelWithData
		joins := modelWithDataDAL(read, write, t.AppName()).
			ReadDB(ctx).
			Joins(fmt.Sprintf("left join %s on %s.model_id = %s.id",
				mwdtTableName, mwdtTableName, mwdTableName))
		t.Require().NoError(joins.Find(&mwdList).Error)
		t.Require().NotEmpty(mwdList)

		t.Require().NoError(orm.Delete(mwd).Error)
		t.Require().NoError(orm.Delete(mwdt).Error)
	})
}

func (t *Model) testSoftDelete(read, write string) {
	t.Catch(func() {
		ctx := context.Background()
		orm := db.Use(ctx, write, db.AppName(t.AppName())).WithContext(ctx)
		t.Require().NoError(orm.Migrator().AutoMigrate(new(modelWithSoftDeleted)))
		defer func() {
			t.Require().NoError(orm.Migrator().DropTable(new(modelWithSoftDeleted)))
		}()
		mwb := &modelWithSoftDeleted{Name: "test"}
		t.Require().NoError(orm.Create(mwb).Error)
		defer func() {
			t.Require().NoError(orm.Unscoped().Model(mwb).Delete(nil, "id = ?", mwb.ID).Error)
		}()

		var found *modelWithSoftDeleted
		t.Require().NoError(orm.First(&found, "id = ?", mwb.ID).Error)
		t.Require().Equal(mwb.Name, found.Name)
		found = nil

		mwb.Name = "test2"
		t.Require().NoError(orm.Updates(mwb).Error)
		t.Require().NoError(orm.First(&found, "id = ?", mwb.ID).Error)
		t.Require().Equal(mwb.Name, found.Name)
		found = nil

		t.Require().NoError(orm.Delete(mwb).Error)
		t.Error(orm.First(&found, "id = ?", mwb.ID).Error)
	})
}

func (t *Model) testBusinessSoftDelete(read, write string) {
	t.Catch(func() {
		ctx := context.Background()
		orm := db.Use(ctx, write, db.AppName(t.AppName())).WithContext(ctx)
		t.Require().NoError(orm.Migrator().AutoMigrate(new(modelBizWithSoftDeleted)))
		defer func() {
			t.Require().NoError(orm.Migrator().DropTable(new(modelBizWithSoftDeleted)))
		}()
		mwb := &modelBizWithSoftDeleted{Name: "test"}
		t.Require().NoError(orm.Create(mwb).Error)
		defer func() {
			t.Require().NoError(orm.Unscoped().Model(mwb).Delete(nil, "id = ?", mwb.ID).Error)
		}()

		var found *modelBizWithSoftDeleted
		t.Require().NoError(orm.First(&found, "id = ?", mwb.ID).Error)
		t.Require().Equal(mwb.Name, found.Name)
		found = nil
		mwb.Name = "test2"
		t.Require().NoError(orm.Updates(mwb).Error)
		t.Require().NoError(orm.First(&found, "id = ?", mwb.ID).Error)
		t.Require().Equal(mwb.Name, found.Name)
		found = nil

		t.Require().NoError(orm.Delete(mwb).Error)
		t.Error(orm.First(&found, "id = ?", mwb.ID).Error)
	})
}

func (t *Model) testSoftDeleteUnscoped(read, write string) {
	t.Catch(func() {
		ctx := context.Background()
		orm := db.Use(ctx, write, db.AppName(t.AppName())).WithContext(ctx)
		t.Require().NoError(orm.Migrator().AutoMigrate(new(modelWithSoftDeleted)))
		defer func() {
			t.Require().NoError(orm.Migrator().DropTable(new(modelWithSoftDeleted)))
		}()
		dal := db.NewDAL[modelWithSoftDeleted, []*modelWithSoftDeleted](read, write, db.AppName(t.AppName()))

		mwb := &modelWithSoftDeleted{Name: "test"}
		t.Require().NoError(dal.InsertOne(ctx, mwb))
		defer func() {
			_, err := dal.Delete(ctx, "id = ?", mwb.ID, db.Unscoped())
			t.Require().NoError(err)
		}()

		found, err := dal.QueryFirst(ctx, "id = ?", mwb.ID)
		t.Require().NoError(err)
		t.Require().Equal(mwb.Name, found.Name)

		_, err = dal.Delete(ctx, "id = ?", mwb.ID)
		t.Require().NoError(err)

		found, err = dal.QueryFirst(ctx, "id = ?", mwb.ID)
		t.Require().NoError(err)
		t.Empty(found)

		found, err = dal.QueryFirst(ctx, "id = ?", mwb.ID, db.Unscoped())
		t.Require().NoError(err)
		t.Require().Equal(mwb.Name, found.Name)
	})
}
