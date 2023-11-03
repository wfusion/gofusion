package cache

import (
	"context"

	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/compress"
	"github.com/wfusion/gofusion/common/utils/inspect"
	"github.com/wfusion/gofusion/common/utils/serialize"
	"github.com/wfusion/gofusion/config"
)

// Construct cache only check some configures
func Construct(ctx context.Context, confs map[string]*Conf, _ ...utils.OptionExtender) func() {
	for _, conf := range confs {
		addInstance(ctx, conf)
	}

	return func() {

	}
}

func addInstance(ctx context.Context, conf *Conf) {
	switch conf.CacheType {
	case cacheTypeLocal:
	case cacheTypeRemote:
		if conf.RemoteType != remoteTypeRedis {
			panic(UnknownRemoteType)
		}

		parsedSerializeType := serialize.ParseAlgorithm(conf.SerializeType)
		parsedCompressType := compress.ParseAlgorithm(conf.Compress)
		if !parsedSerializeType.IsValid() && !parsedCompressType.IsValid() {
			panic(UnknownSerializeType)
		}
	case cacheTypeRemoteLocal:
		panic(ErrNotImplement)

	default:
		panic(UnknownCacheType)
	}

	if utils.IsStrNotBlank(conf.Callback) && inspect.FuncOf(conf.Callback) == nil {
		panic(errors.Errorf("not found callback function: %s", conf.Callback))
	}
}

func init() {
	config.AddComponent(config.ComponentCache, Construct)
}
