package plugins

import (
	"context"

	"gorm.io/gorm"
)

type TableSharding interface {
	gorm.Plugin

	ShardingIDGen(ctx context.Context) (id uint64, err error)
	ShardingByValues(ctx context.Context, src []map[string]any) (dst map[string][]map[string]any, err error)
	ShardingByModelList(ctx context.Context, src ...any) (dst map[string][]any, err error)
}
