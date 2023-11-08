package cases

import (
	"gorm.io/gorm"

	"github.com/wfusion/gofusion/db"
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

func modelWithDataDAL(read, write, appName string) db.DalInterface[modelWithData, []*modelWithData] {
	return db.NewDAL[modelWithData, []*modelWithData](read, write, db.AppName(appName))
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
	UserBase
	Name string `gorm:"column:name"`
}

func (*modelWithBusinessAndUser) TableName() string {
	return "model_with_business_and_user"
}

type modelWithSharding struct {
	db.Business
	AZBase
	gorm.DeletedAt

	Name string `gorm:"column:name"`

	EmbedList []*modelWithShardingEmbed `gorm:"foreignKey:ModelID;AssociationForeignKey:ID"`
}

func (modelWithSharding) TableName() string {
	return "model_with_sharding"
}

type modelWithShardingPtr struct {
	db.Business
	AZBase
	Name string `gorm:"column:name"`
	Age  int    `gorm:"column:age"`
}

func (*modelWithShardingPtr) TableName() string {
	return "model_with_sharding_ptr"
}

type modelWithShardingExtend struct {
	db.Data
	AZBase

	ModelID   uint64 `gorm:"column:model_id"`
	OtherName string `gorm:"column:other_name"`
}

func (*modelWithShardingExtend) TableName() string {
	return "model_with_sharding_extend"
}

type modelWithShardingEmbed struct {
	db.Data
	AZBase

	ModelID   uint64 `gorm:"column:model_id"`
	OtherName string `gorm:"column:other_name"`
}

func (*modelWithShardingEmbed) TableName() string {
	return "model_with_sharding_embed"
}

type modelWithShardingByRawValue struct {
	db.Business
	AZBase
	Name string `gorm:"column:name"`
	Age  int    `gorm:"column:age"`
}

func (*modelWithShardingByRawValue) TableName() string {
	return "model_with_sharding_by_raw_value"
}

type RegionBase struct {
	RegionID string `gorm:"column:region_id;type:varchar(64);index:,composite:base" json:"regionID"`
}

func (r *RegionBase) Clone() *RegionBase {
	if r == nil {
		return nil
	}
	return &RegionBase{
		RegionID: r.RegionID,
	}
}
func (r *RegionBase) Equals(o *RegionBase) bool {
	if r == nil && o == nil {
		return true
	}
	if r == nil || o == nil {
		return false
	}
	return r.RegionID == o.RegionID
}

type AZBase struct {
	RegionBase
	AZName string `gorm:"column:az_name;type:varchar(64);index:,composite:base" json:"azName"`
}

func (a *AZBase) Clone() *AZBase {
	if a == nil {
		return nil
	}
	return &AZBase{
		RegionBase: *a.RegionBase.Clone(),
		AZName:     a.AZName,
	}
}
func (a *AZBase) Equals(o *AZBase) bool {
	if a == nil && o == nil {
		return true
	}
	if a == nil || o == nil {
		return false
	}
	return a.RegionBase.Equals(&o.RegionBase) &&
		a.AZName == o.AZName
}

type UserBase struct {
	AZBase
	UserID string `gorm:"column:user_id;type:varchar(64);index:,composite:base" json:"userID"`
}

func (u *UserBase) Clone() *UserBase {
	if u == nil {
		return nil
	}
	return &UserBase{
		AZBase: *u.AZBase.Clone(),
		UserID: u.UserID,
	}
}
func (u *UserBase) Equals(o *UserBase) bool {
	if u == nil && o == nil {
		return true
	}
	if u == nil || o == nil {
		return false
	}
	return u.AZBase.Equals(&o.AZBase) &&
		u.UserID == o.UserID
}
