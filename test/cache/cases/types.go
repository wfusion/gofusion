package cases

import (
	"context"
	"reflect"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/serialize"
	"github.com/wfusion/gofusion/test/internal/mock"
)

var (
	// randomObjCallbackType FIXME: should not be deleted to avoid compiler optimized
	randomObjCallbackType = reflect.TypeOf(randomObjCallback)
	// commonObjCallbackType FIXME: should not be deleted to avoid compiler optimized
	commonObjCallbackType = reflect.TypeOf(commonObjCallback)
)

const (
	local                         = "local"
	localWithoutLog               = "local_without_log"
	localWithCallback             = "local_with_cb"
	localWithSerialize            = "local_with_serialize"
	localWithSerializeAndCompress = "local_with_serialize_and_compress"
	localWithZstdCompress         = "local_with_zstd_compress"
	localWithZlibCompress         = "local_with_zlib_compress"
	localWithS2Compress           = "local_with_s2_compress"
	localWithGzipCompress         = "local_with_gzip_compress"
	localWithDeflateCompress      = "local_with_deflate_compress"

	redis                    = "redis"
	redisJson                = "redis_json"
	redisWithZstdCompress    = "redis_with_zstd_compress"
	redisWithZlibCompress    = "redis_with_zlib_compress"
	redisWithS2Compress      = "redis_with_s2_compress"
	redisWithGzipCompress    = "redis_with_gzip_compress"
	redisWithDeflateCompress = "redis_with_deflate_compress"
)

func randomObjCallback(ctx context.Context, missed []string) (
	rs map[string]*mock.RandomObj, opts []utils.OptionExtender) {
	valList := mock.GenObjListBySerializeAlgo(serialize.AlgorithmGob, len(missed)).([]*mock.RandomObj)
	rs = make(map[string]*mock.RandomObj, len(missed))
	for idx, mis := range missed {
		rs[mis] = valList[idx]
	}
	return
}

func commonObjCallback(ctx context.Context, missed []string) (
	rs map[string]*mock.CommonObj, opts []utils.OptionExtender) {
	valList := mock.GenObjListBySerializeAlgo(serialize.AlgorithmJson, len(missed)).([]*mock.CommonObj)
	rs = make(map[string]*mock.CommonObj, len(missed))
	for idx, mis := range missed {
		rs[mis] = valList[idx]
	}
	return
}
