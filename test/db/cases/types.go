package cases

import (
	"gorm.io/gorm"

	"github.com/wfusion/gofusion/db"

	testDB "github.com/wfusion/gofusion/test/db"
)

const (
	nameMysqlWrite = "write"
	nameMysqlRead  = "read"
	namePostgres   = "postgres"
	nameOpenGauss  = "opengauss"
	nameSqlserver  = "sqlserver"
	nameSqlite     = "sqlite"
)

type modelWithData struct {
	db.Data
	Name string `gorm:"column:name"`
}

func (*modelWithData) TableName() string {
	return "model_with_data"
}

func modelWithDataDAL(read, write string) db.DalInterface[modelWithData, []*modelWithData] {
	return db.NewDAL[modelWithData, []*modelWithData](read, write, db.AppName(testDB.Component))
}

type modelWithDataExtend struct {
	db.Data
	ModelID   uint64 `gorm:"column:model_id"`
	OtherName string `gorm:"column:other_name"`
}

func (*modelWithDataExtend) TableName() string {
	return "model_with_data_extend"
}

type modelWithBusiness struct {
	db.Business
	Name string `gorm:"column:name"`
}

func (*modelWithBusiness) TableName() string {
	return "model_with_business"
}

type modelWithSoftDeleted struct {
	db.DataSoftDeleted
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at"`
	Name      string         `gorm:"column:name"`
}

func (*modelWithSoftDeleted) TableName() string {
	return "model_with_soft_deleted"
}

type modelBizWithSoftDeleted struct {
	db.BusinessSoftDeleted
	Name string `gorm:"column:name"`
}

func (*modelBizWithSoftDeleted) TableName() string {
	return "model_biz_with_soft_deleted"
}

type modelWithBusinessAndUser struct {
	db.Business
	db.UserBase
	Name string `gorm:"column:name"`
}

func (*modelWithBusinessAndUser) TableName() string {
	return "model_with_business_and_user"
}

type modelWithSharding struct {
	db.Business
	db.AZBase
	gorm.DeletedAt

	Name string `gorm:"column:name"`

	EmbedList []*modelWithShardingEmbed `gorm:"foreignKey:ModelID;AssociationForeignKey:ID"`
}

func (modelWithSharding) TableName() string {
	return "model_with_sharding"
}

type modelWithShardingPtr struct {
	db.Business
	db.AZBase
	Name string `gorm:"column:name"`
	Age  int    `gorm:"column:age"`
}

func (*modelWithShardingPtr) TableName() string {
	return "model_with_sharding_ptr"
}

type modelWithShardingExtend struct {
	db.Data
	db.AZBase

	ModelID   uint64 `gorm:"column:model_id"`
	OtherName string `gorm:"column:other_name"`
}

func (*modelWithShardingExtend) TableName() string {
	return "model_with_sharding_extend"
}

type modelWithShardingEmbed struct {
	db.Data
	db.AZBase

	ModelID   uint64 `gorm:"column:model_id"`
	OtherName string `gorm:"column:other_name"`
}

func (*modelWithShardingEmbed) TableName() string {
	return "model_with_sharding_embed"
}

type modelWithShardingByRawValue struct {
	db.Business
	db.AZBase
	Name string `gorm:"column:name"`
	Age  int    `gorm:"column:age"`
}

func (*modelWithShardingByRawValue) TableName() string {
	return "model_with_sharding_by_raw_value"
}
