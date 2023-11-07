package pd

import (
	"context"
	"reflect"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/compress"
	"github.com/wfusion/gofusion/common/utils/serialize"
)

var (
	// sealMagicNumber FF FF FF FB: magic number
	sealMagicNumber uint32 = 0xFFFFFFFB
	// sealTypeNumber 01 00 00 00: version, serialize type, compress type, raw flag, encrypted info length
	sealTypeNumber   = [8]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	sealPrefixLength = 4 + len(sealTypeNumber)
)

type option struct {
	ctx           context.Context
	serializeType serialize.Algorithm
	compressType  compress.Algorithm
	dataType      reflect.Type
	version       byte
}

func Context(ctx context.Context) utils.OptionFunc[option] {
	return func(o *option) {
		o.ctx = ctx
	}
}

func Serialize(serializeType serialize.Algorithm) utils.OptionFunc[option] {
	return func(o *option) {
		o.serializeType = serializeType
	}
}

func Compress(compressType compress.Algorithm) utils.OptionFunc[option] {
	return func(o *option) {
		o.compressType = compressType
	}
}

func Type(typ reflect.Type) utils.OptionFunc[option] {
	return func(o *option) {
		o.dataType = typ
	}
}

func Version(ver uint8) utils.OptionFunc[option] {
	return func(o *option) {
		o.version = ver
	}
}
