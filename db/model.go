package db

import (
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/serialize/json"
	"github.com/wfusion/gofusion/db/softdelete"
)

type Data struct {
	ID           uint64 `gorm:"column:id;primary_key;autoIncrement" json:"id"`
	CreateTimeMs int64  `gorm:"column:create_time;type:bigint;autoCreateTime:milli" json:"createTime"`
	ModifyTimeMs int64  `gorm:"column:modify_time;type:bigint;autoUpdateTime:milli" json:"modifyTime"`
}

func (d *Data) CreateTime() time.Time {
	return utils.GetTime(d.CreateTimeMs)
}
func (d *Data) ModifyTime() time.Time {
	return utils.GetTime(d.ModifyTimeMs)
}
func (d *Data) Clone() *Data {
	if d == nil {
		return nil
	}
	return &Data{
		ID:           d.ID,
		CreateTimeMs: d.CreateTimeMs,
		ModifyTimeMs: d.ModifyTimeMs,
	}
}
func (d *Data) Equals(o *Data) bool {
	if d == nil && o == nil {
		return true
	}
	if d == nil || o == nil {
		return false
	}
	return d.ID == o.ID &&
		d.CreateTimeMs == o.CreateTimeMs &&
		d.ModifyTimeMs == o.ModifyTimeMs
}

// DataSoftDeleted
//nolint: revive // struct tag too long issue
type DataSoftDeleted struct {
	ID           uint64               `gorm:"column:id;primary_key;autoIncrement" json:"id"`
	CreateTimeMs int64                `gorm:"column:create_time;type:bigint;autoCreateTime:milli" json:"createTime"`
	ModifyTimeMs int64                `gorm:"column:modify_time;type:bigint;autoUpdateTime:milli" json:"modifyTime"`
	DeleteTimeMs softdelete.Timestamp `gorm:"column:delete_time;type:bigint;index:,composite:soft_delete" json:"deleteTime"`
	Deleted      softdelete.Deleted   `gorm:"column:deleted;index:,composite:soft_delete;default:false" json:"deleted"`
}

func (d *DataSoftDeleted) CreateTime() time.Time {
	return utils.GetTime(d.CreateTimeMs)
}
func (d *DataSoftDeleted) ModifyTime() time.Time {
	return utils.GetTime(d.ModifyTimeMs)
}
func (d *DataSoftDeleted) DeleteTime() *time.Time {
	if !d.DeleteTimeMs.Valid {
		return nil
	}
	return utils.AnyPtr(utils.GetTime(d.DeleteTimeMs.Int64))
}
func (d *DataSoftDeleted) Clone() *DataSoftDeleted {
	if d == nil {
		return nil
	}
	return &DataSoftDeleted{
		ID:           d.ID,
		Deleted:      d.Deleted,
		CreateTimeMs: d.CreateTimeMs,
		ModifyTimeMs: d.ModifyTimeMs,
		DeleteTimeMs: d.DeleteTimeMs,
	}
}
func (d *DataSoftDeleted) Equals(o *DataSoftDeleted) bool {
	if d == nil && o == nil {
		return true
	}
	if d == nil || o == nil {
		return false
	}
	return d.ID == o.ID &&
		d.CreateTimeMs == o.CreateTimeMs &&
		d.ModifyTimeMs == o.ModifyTimeMs &&
		d.DeleteTimeMs == o.DeleteTimeMs &&
		d.Deleted == o.Deleted
}

type Business struct {
	Data

	UUID            uuid.UUID `gorm:"column:uuid;type:varchar(36);uniqueIndex" json:"uuid"`
	BizCreateTimeMs int64     `gorm:"column:biz_create_time;type:bigint" json:"bizCreateTime"`
	BizModifyTimeMs int64     `gorm:"column:biz_modify_time;type:bigint" json:"bizModifyTime"`
}

func (b *Business) BeforeCreate(tx *gorm.DB) (err error) {
	if b == nil {
		tx.Statement.SetColumn("uuid", utils.UUID())
		return
	}
	if b.UUID == [16]byte{0} {
		b.UUID = uuid.New()
	}
	return
}
func (b *Business) BeforeCreateFn(tx *gorm.DB) func() error {
	return func() (err error) {
		return b.BeforeCreate(tx)
	}
}
func (b *Business) BizCreateTime() time.Time {
	return utils.GetTime(b.BizCreateTimeMs)
}
func (b *Business) BizModifyTime() time.Time {
	return utils.GetTime(b.BizModifyTimeMs)
}
func (b *Business) Clone() *Business {
	if b == nil {
		return nil
	}
	return &Business{
		Data:            *b.Data.Clone(),
		UUID:            b.UUID,
		BizCreateTimeMs: b.BizCreateTimeMs,
		BizModifyTimeMs: b.BizModifyTimeMs,
	}
}
func (b *Business) Equals(o *Business) bool {
	if b == nil && o == nil {
		return true
	}
	if b == nil || o == nil {
		return false
	}
	return b.Data.Equals(&o.Data) &&
		b.UUID == o.UUID &&
		b.BizCreateTimeMs == o.BizCreateTimeMs &&
		b.BizModifyTimeMs == o.BizModifyTimeMs
}

type BusinessSoftDeleted struct {
	DataSoftDeleted

	UUID            uuid.UUID `gorm:"column:uuid;type:varchar(36);uniqueIndex:uniq_uuid" json:"uuid"`
	BizCreateTimeMs int64     `gorm:"column:biz_create_time;type:bigint" json:"bizCreateTime"`
	BizModifyTimeMs int64     `gorm:"column:biz_modify_time;type:bigint" json:"bizModifyTime"`
}

func (b *BusinessSoftDeleted) BeforeCreate(tx *gorm.DB) (err error) {
	if b == nil {
		tx.Statement.SetColumn("uuid", utils.UUID())
		tx.Statement.SetColumn("delete_time", nil)
		return
	}
	if b.UUID == [16]byte{0} {
		b.UUID = uuid.New()
	}
	return
}
func (b *BusinessSoftDeleted) BeforeCreateFn(tx *gorm.DB) func() error {
	return func() (err error) {
		return b.BeforeCreate(tx)
	}
}
func (b *BusinessSoftDeleted) BizCreateTime() time.Time {
	return utils.GetTime(b.BizCreateTimeMs)
}
func (b *BusinessSoftDeleted) BizModifyTime() time.Time {
	return utils.GetTime(b.BizModifyTimeMs)
}
func (b *BusinessSoftDeleted) Clone() *BusinessSoftDeleted {
	if b == nil {
		return nil
	}
	return &BusinessSoftDeleted{
		DataSoftDeleted: *b.DataSoftDeleted.Clone(),
		UUID:            b.UUID,
		BizCreateTimeMs: b.BizCreateTimeMs,
		BizModifyTimeMs: b.BizModifyTimeMs,
	}
}
func (b *BusinessSoftDeleted) Equals(o *BusinessSoftDeleted) bool {
	if b == nil && o == nil {
		return true
	}
	if b == nil || o == nil {
		return false
	}
	return b.DataSoftDeleted.Equals(&o.DataSoftDeleted) &&
		b.UUID == o.UUID &&
		b.BizCreateTimeMs == o.BizCreateTimeMs &&
		b.BizModifyTimeMs == o.BizModifyTimeMs
}

func CheckGormErrorFn(tx *gorm.DB) func() error { return func() error { return tx.Error } }
func JsonUnmarshalFn(tx *gorm.DB, obj, field any) func() error {
	return func() (err error) {
		if tx.Error != nil {
			return
		}

		var bs []byte
		switch v := field.(type) {
		case string:
			bs = []byte(v)
		case *string:
			if v == nil {
				return
			}
			bs = []byte(*v)
		case []byte:
			bs = v
		case *[]byte:
			if v == nil {
				return
			}
			bs = *v
		default:
			return tx.AddError(errors.New("unsupported unmarshal field type"))
		}
		// be compatible with empty value
		if len(bs) == 0 {
			bs = []byte("null")
		}

		return tx.AddError(json.Unmarshal(bs, &obj))
	}
}
func JsonMarshalFn(tx *gorm.DB, obj, field any) func() error {
	return func() (err error) {
		if tx.Error != nil {
			return
		}

		switch v := field.(type) {
		case **string, **[]byte, *string, *[]byte:
			if v == nil {
				return
			}
			bs, err := json.Marshal(obj)
			if err != nil {
				return tx.AddError(errors.Wrap(err, "gorm json marshal error"))
			}
			switch f := field.(type) {
			case *string:
				*f = string(bs)
			case **string:
				*f = utils.AnyPtr(string(bs))
			case *[]byte:
				*f = bs
			case **[]byte:
				*f = utils.AnyPtr(bs)
			}
		default:
			return tx.AddError(errors.New("unsupported marshal field type"))
		}
		return
	}

}
