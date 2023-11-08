package cases

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/db"
	"github.com/wfusion/gofusion/log"

	testDB "github.com/wfusion/gofusion/test/db"
)

func TestTableSharding(t *testing.T) {
	testingSuite := &TableSharding{Test: new(testDB.Test)}
	testingSuite.Init(testingSuite)
	suite.Run(t, testingSuite)
}

type TableSharding struct {
	*testDB.Test
}

func (t *TableSharding) BeforeTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right before %s %s", suiteName, testName)
	})
}

func (t *TableSharding) AfterTest(suiteName, testName string) {
	t.Catch(func() {
		log.Info(context.Background(), "right after %s %s", suiteName, testName)
	})
}

func (t *TableSharding) TestMysql() {
	t.testDefault(nameMysqlRead, nameMysqlWrite)
}

func (t *TableSharding) TestPostgres() {
	t.testDefault(namePostgres, namePostgres)
}

func (t *TableSharding) TestOpengauss() {
	t.testDefault(nameOpenGauss, nameOpenGauss)
}

func (t *TableSharding) TestSqlserver() {
	t.testDefault(nameSqlserver, nameSqlserver)
}

func (t *TableSharding) testDefault(read, write string) {
	t.Run("Migrate", func() { t.testMigrate(read, write) })
	t.Run("Insert", func() { t.testInsert(read, write) })
	t.Run("InsertNested", func() { t.testInsertNested(read, write) })
	t.Run("BatchInsert", func() { t.testBatchInsert(read, write) })
	t.Run("BatchInInsert", func() { t.testBatchInInsert(read, write) })
	t.Run("Query", func() { t.testQuery(read, write) })
	t.Run("QueryNested", func() { t.testQueryNested(read, write) })
	t.Run("Delete", func() { t.testDelete(read, write) })
	t.Run("DAL", func() { t.testDAL(read, write) })
	t.Run("DALQueryWithDeletedAt", func() { t.testDALQueryWithDeletedAt(read, write) })
	t.Run("Joins", func() { t.testJoins(read, write) })
	t.Run("DALFindAndCount", func() { t.testDALFindAndCount(read, write) })
	t.Run("ShardingKeyByRawValue", func() { t.testShardingKeyByRawValue(read, write) })
}

func (t *TableSharding) testMigrate(read, write string) {
	t.Catch(func() {
		ctx := context.Background()
		orm := db.Use(ctx, write, db.AppName(t.AppName()))
		t.NoError(orm.Migrator().AutoMigrate(new(modelWithSharding)))
		t.NoError(orm.Migrator().DropTable(new(modelWithSharding)))

		t.NoError(orm.Migrator().AutoMigrate(new(modelWithShardingPtr)))
		t.NoError(orm.Migrator().DropTable(new(modelWithShardingPtr)))

		t.NoError(orm.Migrator().AutoMigrate(new(modelWithShardingByRawValue)))
		t.NoError(orm.Migrator().DropTable(new(modelWithShardingByRawValue)))
	})
}

func (t *TableSharding) testInsert(read, write string) {
	t.Catch(func() {
		ctx := context.Background()
		orm := db.Use(ctx, write, db.AppName(t.AppName()))
		t.NoError(orm.Migrator().AutoMigrate(new(modelWithSharding)))
		t.NoError(orm.Migrator().AutoMigrate(new(modelWithShardingEmbed)))
		t.NoError(orm.Migrator().AutoMigrate(new(modelWithShardingPtr)))
		defer func() {
			t.NoError(orm.Migrator().DropTable(new(modelWithSharding)))
			t.NoError(orm.Migrator().DropTable(new(modelWithShardingEmbed)))
			t.NoError(orm.Migrator().DropTable(new(modelWithShardingPtr)))
		}()

		m1 := &modelWithSharding{
			AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
			Name:   "sharding1",
		}
		t.NoError(orm.Create(m1).Error)
		t.NoError(orm.Delete(m1).Error)

		m2 := &modelWithShardingPtr{
			Name: "sharding_ptr1",
			Age:  18,
		}
		t.NoError(orm.Create(m2).Error)
		t.NoError(orm.Delete(m2).Error)
	})
}

func (t *TableSharding) testInsertNested(read, write string) {
	t.Catch(func() {
		ctx := context.Background()
		orm := db.Use(ctx, write, db.AppName(t.AppName()))
		t.NoError(orm.Migrator().AutoMigrate(new(modelWithSharding)))
		t.NoError(orm.Migrator().AutoMigrate(new(modelWithShardingEmbed)))
		defer func() {
			t.NoError(orm.Migrator().DropTable(new(modelWithSharding)))
			t.NoError(orm.Migrator().DropTable(new(modelWithShardingEmbed)))
		}()

		mws := &modelWithSharding{
			AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
			Name:   "sharding1",
			EmbedList: []*modelWithShardingEmbed{
				{
					AZBase:    AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
					OtherName: "other1",
				},
				{
					AZBase:    AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
					OtherName: "other2",
				},
				{
					AZBase:    AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
					OtherName: "other3",
				},
			},
		}

		t.NoError(orm.Create(mws).Error)
		t.NoError(orm.Delete(mws).Error)
		if read == nameOpenGauss {
			t.NoError(orm.Where("az_name = ? AND model_id = ?", mws.AZName, mws.ID).Find(&mws.EmbedList).Error)
		}

		t.NoError(orm.Delete(mws.EmbedList).Error)
	})
}

func (t *TableSharding) testBatchInsert(read, write string) {
	t.Catch(func() {
		ctx := context.Background()
		orm := db.Use(ctx, write, db.AppName(t.AppName()))
		t.NoError(orm.Migrator().AutoMigrate(new(modelWithSharding)))
		t.NoError(orm.Migrator().AutoMigrate(new(modelWithShardingEmbed)))
		t.NoError(orm.Migrator().AutoMigrate(new(modelWithShardingPtr)))
		defer func() {
			t.NoError(orm.Migrator().DropTable(new(modelWithSharding)))
			t.NoError(orm.Migrator().DropTable(new(modelWithShardingEmbed)))
			t.NoError(orm.Migrator().DropTable(new(modelWithShardingPtr)))
		}()
		mList1 := []*modelWithSharding{
			{
				AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
				Name:   "sharding1",
			},
			{
				AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
				Name:   "sharding2",
			},
			{
				AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
				Name:   "sharding3",
			},
		}
		t.NoError(orm.Create(mList1).Error)
		t.NoError(orm.Unscoped().Delete(mList1).Error)

		mList2 := []*modelWithShardingPtr{
			{
				Business: db.Business{Data: db.Data{ID: 0x110}},
				Name:     "sharding_ptr2",
				Age:      18,
			},
			{
				Business: db.Business{Data: db.Data{ID: 0x210}},
				Name:     "sharding_ptr3",
				Age:      18,
			},
			{
				Business: db.Business{Data: db.Data{ID: 0x310}},
				Name:     "sharding_ptr4",
				Age:      18,
			},
		}
		t.NoError(orm.Create(mList2).Error)
		t.NoError(orm.Unscoped().Delete(mList2).Error)
	})
}

func (t *TableSharding) testBatchInInsert(read, write string) {
	t.Catch(func() {
		ctx := context.Background()
		orm := db.Use(ctx, write, db.AppName(t.AppName()))
		t.NoError(orm.Migrator().AutoMigrate(new(modelWithSharding)))
		t.NoError(orm.Migrator().AutoMigrate(new(modelWithShardingEmbed)))
		t.NoError(orm.Migrator().AutoMigrate(new(modelWithShardingPtr)))
		defer func() {
			t.NoError(orm.Migrator().DropTable(new(modelWithSharding)))
			t.NoError(orm.Migrator().DropTable(new(modelWithShardingEmbed)))
			t.NoError(orm.Migrator().DropTable(new(modelWithShardingPtr)))
		}()
		mList1 := []*modelWithSharding{
			{
				AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
				Name:   "sharding1",
			},
			{
				AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
				Name:   "sharding2",
			},
			{
				AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
				Name:   "sharding3",
			},
		}
		t.NoError(orm.CreateInBatches(mList1, 2).Error)
		t.NoError(orm.Unscoped().Delete(mList1).Error)

		mList2 := []*modelWithShardingPtr{
			{
				Business: db.Business{Data: db.Data{ID: 0x110}},
				Name:     "sharding_ptr2",
				Age:      18,
			},
			{
				Business: db.Business{Data: db.Data{ID: 0x210}},
				Name:     "sharding_ptr3",
				Age:      18,
			},
			{
				Business: db.Business{Data: db.Data{ID: 0x310}},
				Name:     "sharding_ptr4",
				Age:      18,
			},
		}
		t.NoError(orm.CreateInBatches(mList2, 2).Error)
		t.NoError(orm.Unscoped().Delete(mList2).Error)
	})
}

// TestQuery may not pass if run all test cases cause table plugin only create table once while inserting
func (t *TableSharding) testQuery(read, write string) {
	t.Catch(func() {
		ctx := context.Background()
		orm := db.Use(ctx, write, db.AppName(t.AppName()))
		t.NoError(orm.Migrator().AutoMigrate(new(modelWithSharding)))
		t.NoError(orm.Migrator().AutoMigrate(new(modelWithShardingEmbed)))
		t.NoError(orm.Migrator().AutoMigrate(new(modelWithShardingPtr)))
		defer func() {
			t.NoError(orm.Migrator().DropTable(new(modelWithSharding)))
			t.NoError(orm.Migrator().DropTable(new(modelWithShardingEmbed)))
			t.NoError(orm.Migrator().DropTable(new(modelWithShardingPtr)))
		}()
		mList1 := []*modelWithSharding{
			{
				AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
				Name:   "sharding1",
			},
			{
				AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
				Name:   "sharding2",
			},
			{
				AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
				Name:   "sharding3",
			},
		}
		t.NoError(orm.Create(mList1).Error)
		var mList1Dup []*modelWithSharding
		t.NoError(orm.Where("az_name = 'az1'").Find(&mList1Dup).Error)
		t.EqualValues(mList1, mList1Dup)
		t.NoError(orm.Unscoped().Delete(mList1).Error)

		mList2 := []*modelWithShardingPtr{
			{
				Business: db.Business{Data: db.Data{ID: 0x110}},
				Name:     "sharding_ptr2",
				Age:      18,
			},
			{
				Business: db.Business{Data: db.Data{ID: 0x210}},
				Name:     "sharding_ptr3",
				Age:      18,
			},
			{
				Business: db.Business{Data: db.Data{ID: 0x310}},
				Name:     "sharding_ptr4",
				Age:      18,
			},
		}
		t.NoError(orm.Create(mList2).Error)
		var m2Dup *modelWithShardingPtr
		m2 := mList2[0]
		t.NoError(orm.Where("name = ? AND id = ? AND age = ?", m2.Name, m2.ID, m2.Age).First(&m2Dup).Error)
		t.EqualValues(m2, m2Dup)
		t.NoError(orm.Unscoped().Delete(mList2).Error)
	})
}

// TestQueryNested may not pass if run all test cases cause plugin only create table once while inserting
func (t *TableSharding) testQueryNested(read, write string) {
	t.Catch(func() {
		ctx := context.Background()
		orm := db.Use(ctx, write, db.AppName(t.AppName()))
		t.NoError(orm.Migrator().AutoMigrate(new(modelWithSharding)))
		t.NoError(orm.Migrator().AutoMigrate(new(modelWithShardingEmbed)))
		defer func() {
			t.NoError(orm.Migrator().DropTable(new(modelWithSharding)))
			t.NoError(orm.Migrator().DropTable(new(modelWithShardingEmbed)))
		}()
		mws := &modelWithSharding{
			AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
			Name:   "sharding1",
			EmbedList: []*modelWithShardingEmbed{
				{
					AZBase:    AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
					OtherName: "other1",
				},
				{
					AZBase:    AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
					OtherName: "other2",
				},
				{
					AZBase:    AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
					OtherName: "other3",
				},
			},
		}

		t.NoError(orm.Create(mws).Error)

		var mwsList []*modelWithSharding
		t.NoError(orm.
			Preload("EmbedList", "az_name = ?", "az1").
			Where("az_name = ?", "az1").
			Find(&mwsList).Error,
		)
		t.NotEmpty(mwsList)

		t.NoError(orm.Unscoped().Delete(mws).Error)
		if read == nameOpenGauss {
			t.NoError(orm.Where("az_name = ? AND model_id = ?", mws.AZName, mws.ID).Find(&mws.EmbedList).Error)
		}

		t.NoError(orm.Unscoped().Delete(mws.EmbedList).Error)
	})
}

func (t *TableSharding) testDelete(read, write string) {
	t.Catch(func() {
		ctx := context.Background()
		orm := db.Use(ctx, write, db.AppName(t.AppName()))
		t.NoError(orm.Migrator().AutoMigrate(new(modelWithSharding)))
		t.NoError(orm.Migrator().AutoMigrate(new(modelWithShardingEmbed)))
		t.NoError(orm.Migrator().AutoMigrate(new(modelWithShardingPtr)))
		defer func() {
			t.NoError(orm.Migrator().DropTable(new(modelWithSharding)))
			t.NoError(orm.Migrator().DropTable(new(modelWithShardingEmbed)))
			t.NoError(orm.Migrator().DropTable(new(modelWithShardingPtr)))
		}()
		m1 := &modelWithSharding{
			AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
			Name:   "sharding1",
		}
		t.NoError(orm.Create(m1).Error)
		t.NoError(orm.Unscoped().
			Where("az_name = ? AND name = ?", m1.AZName, m1.Name).
			Delete(new(modelWithSharding)).Error)

		m2 := &modelWithShardingPtr{
			Name: "sharding_ptr1",
			Age:  18,
		}
		t.NoError(orm.Create(m2).Error)
		t.NoError(orm.Unscoped().
			Where("name = ? AND id = ? AND age = ?", m2.Name, m2.ID, m2.Age).
			Delete(new(modelWithShardingPtr)).Error)
	})
}

func (t *TableSharding) testDAL(read, write string) {
	t.Catch(func() {
		ctx := context.Background()
		dal1 := db.NewDAL[modelWithSharding, []*modelWithSharding](read, write, db.AppName(t.AppName()))
		t.NoError(dal1.WriteDB(ctx).Migrator().AutoMigrate(new(modelWithSharding)))
		t.NoError(dal1.WriteDB(ctx).Migrator().AutoMigrate(new(modelWithShardingEmbed)))
		defer func() {
			t.NoError(dal1.WriteDB(ctx).Migrator().DropTable(new(modelWithSharding)))
			t.NoError(dal1.WriteDB(ctx).Migrator().DropTable(new(modelWithShardingEmbed)))
		}()

		mList1 := []*modelWithSharding{
			{
				AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
				Name:   "sharding1",
			},
			{
				AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
				Name:   "sharding2",
			},
			{
				AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az2"},
				Name:   "sharding3",
			},
		}
		t.NoError(dal1.InsertInBatches(ctx, mList1, 1))
		if read != nameOpenGauss {
			t.NoError(dal1.Save(ctx, mList1))
		}
		_, err := dal1.Delete(ctx, mList1, db.Unscoped())
		t.NoError(err)

		dal2 := db.NewDAL[modelWithShardingPtr, []*modelWithShardingPtr](read, write, db.AppName(t.AppName()))
		t.NoError(dal2.WriteDB(ctx).Migrator().AutoMigrate(new(modelWithShardingPtr)))
		defer func() {
			t.NoError(dal2.WriteDB(ctx).Migrator().DropTable(new(modelWithShardingPtr)))
		}()

		mList2 := []*modelWithShardingPtr{
			{
				Business: db.Business{Data: db.Data{ID: utils.Must(dal2.ShardingIDGen(ctx))}},
				Name:     "sharding_ptr2",
				Age:      18,
			},
			{
				Business: db.Business{Data: db.Data{ID: 299}},
				Name:     "sharding_ptr3",
				Age:      19,
			},
			{
				Business: db.Business{Data: db.Data{ID: 399}},
				Name:     "sharding_ptr4",
				Age:      20,
			},
		}
		t.NoError(dal2.InsertInBatches(ctx, mList2, 1))
		if read != nameOpenGauss {
			t.NoError(dal2.Save(ctx, mList2))
		}
		_, err = dal2.Delete(ctx, mList2, db.Unscoped())
		t.NoError(err)
	})
}

func (t *TableSharding) testDALQueryWithDeletedAt(read, write string) {
	t.Catch(func() {
		ctx := context.Background()

		dal := db.NewDAL[modelWithSharding, []*modelWithSharding](read, write, db.AppName(t.AppName()))
		t.NoError(dal.WriteDB(ctx).Migrator().AutoMigrate(new(modelWithSharding)))
		t.NoError(dal.WriteDB(ctx).Migrator().AutoMigrate(new(modelWithShardingEmbed)))
		defer func() {
			t.NoError(dal.WriteDB(ctx).Migrator().DropTable(new(modelWithSharding)))
			t.NoError(dal.WriteDB(ctx).Migrator().DropTable(new(modelWithShardingEmbed)))
		}()

		expected := []*modelWithSharding{
			{
				AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
				Name:   "sharding1",
			},
			{
				AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
				Name:   "sharding2",
			},
			{
				AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az2"},
				Name:   "sharding3",
			},
		}
		t.NoError(dal.InsertInBatches(ctx, expected, 1))

		tx := dal.ReadDB(ctx)
		tx = tx.Where("az_name = ?", "az1")
		tx = tx.Where("name like ?", "%sharding%")

		var actual []*modelWithSharding
		t.NoError(tx.Find(&actual).Error)
		t.NotEmpty(actual)

		_, err := dal.Delete(ctx, expected)
		t.NoError(err)
	})
}

func (t *TableSharding) testJoins(read, write string) {
	t.Catch(func() {
		ctx := context.Background()
		mwsDAL := db.NewDAL[modelWithSharding, []*modelWithSharding](read, write, db.AppName(t.AppName()))
		mwseDAL := db.NewDAL[modelWithShardingExtend, []*modelWithShardingExtend](
			read, write, db.AppName(t.AppName()))
		t.NoError(mwsDAL.WriteDB(ctx).Migrator().AutoMigrate(new(modelWithSharding)))
		t.NoError(mwsDAL.WriteDB(ctx).Migrator().AutoMigrate(new(modelWithShardingEmbed)))
		t.NoError(mwseDAL.WriteDB(ctx).Migrator().AutoMigrate(new(modelWithShardingExtend)))
		defer func() {
			t.NoError(mwsDAL.WriteDB(ctx).Migrator().DropTable(new(modelWithSharding)))
			t.NoError(mwsDAL.WriteDB(ctx).Migrator().DropTable(new(modelWithShardingEmbed)))
			t.NoError(mwseDAL.WriteDB(ctx).Migrator().DropTable(new(modelWithShardingExtend)))
		}()

		mws := &modelWithSharding{
			AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
			Name:   "sharding1",
		}
		t.NoError(mwsDAL.InsertOne(ctx, mws))
		defer func() { _, _ = mwsDAL.Delete(ctx, mws) }()
		mwsTblName := mws.TableName()

		mwse := &modelWithShardingExtend{
			AZBase:    AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
			OtherName: "sharding1",
			ModelID:   mws.ID,
		}
		t.NoError(mwseDAL.InsertOne(ctx, mwse))
		defer func() { _, _ = mwseDAL.Delete(ctx, mwse) }()
		mwseTblName := mwse.TableName()

		var mwsList []*modelWithSharding
		joins := mwsDAL.ReadDB(ctx).
			Joins(fmt.Sprintf("left join %s on %s.model_id = %s.id", mwseTblName, mwseTblName, mwsTblName)).
			Where(fmt.Sprintf("%s.id = ?", mwsTblName), mws.ID).
			Where(fmt.Sprintf("%s.az_name = ?", mwsTblName), "az1").
			Where(fmt.Sprintf("%s.az_name = ?", mwseTblName), "az1")
		t.NoError(joins.Find(&mwsList).Error)
		t.NotEmpty(mwsList)
	})
}

func (t *TableSharding) testDALFindAndCount(read, write string) {
	t.Catch(func() {
		ctx := context.Background()

		dal := db.NewDAL[modelWithSharding, []*modelWithSharding](read, write, db.AppName(t.AppName()))
		t.NoError(dal.WriteDB(ctx).Migrator().AutoMigrate(new(modelWithSharding)))
		defer func() {
			t.NoError(dal.WriteDB(ctx).Migrator().DropTable(new(modelWithSharding)))
		}()
		mList := []*modelWithSharding{
			{
				AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
				Name:   "sharding1",
			},
			{
				AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az2"},
				Name:   "sharding2",
			},
			{
				AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az3"},
				Name:   "sharding3",
			},
			{
				AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
				Name:   "sharding4",
			},
			{
				AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az2"},
				Name:   "sharding5",
			},
		}
		t.NoError(dal.InsertInBatches(ctx, mList, 100))
		defer dal.WriteDB(ctx).Unscoped().Delete(mList)

		var (
			count  int64
			result []*modelWithSharding
		)
		for _, azName := range []string{"az1", "az2", "az3"} {
			count = 0
			result = nil

			t.NoError(dal.ReadDB(ctx).Where("az_name = ?", azName).Find(&result).Error)
			t.NoError(dal.ReadDB(ctx).Where("az_name = ?", azName).Count(&count).Error)
			t.EqualValues(count, len(result))
		}
	})
}

func (t *TableSharding) testShardingKeyByRawValue(read, write string) {
	t.Catch(func() {
		ctx := context.Background()
		orm := db.Use(ctx, write, db.AppName(t.AppName()))
		t.NoError(orm.Migrator().AutoMigrate(new(modelWithShardingByRawValue)))
		defer func() {
			t.NoError(orm.Migrator().DropTable(new(modelWithShardingByRawValue)))
		}()

		m1 := &modelWithShardingByRawValue{
			AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az1"},
			Name:   "sharding_ptr1",
			Age:    18,
		}
		t.NoError(orm.Create(m1).Error)
		defer func() {
			t.NoError(orm.Unscoped().Delete(m1).Error)
		}()

		m2 := &modelWithShardingByRawValue{
			AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az2"},
			Name:   "sharding_ptr1",
			Age:    18,
		}
		t.NoError(orm.Create(m2).Error)
		defer func() {
			t.NoError(orm.Unscoped().Delete(m2).Error)
		}()

		m3 := &modelWithShardingByRawValue{
			AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az3"},
			Name:   "sharding_ptr1",
			Age:    18,
		}
		t.NoError(orm.Create(m3).Error)
		defer func() {
			t.NoError(orm.Unscoped().Delete(m3).Error)
		}()

		m4 := &modelWithShardingByRawValue{
			AZBase: AZBase{RegionBase: RegionBase{RegionID: "12345"}, AZName: "az4"},
			Name:   "sharding_ptr1",
			Age:    18,
		}
		t.NoError(orm.Create(m4).Error)
		defer func() {
			t.NoError(orm.Unscoped().Delete(m4).Error)
		}()
	})
}
