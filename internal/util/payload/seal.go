package pd

import (
	"encoding/binary"
	"reflect"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/encode"
	"github.com/wfusion/gofusion/common/utils/serialize"

	fmkCtx "github.com/wfusion/gofusion/context"
)

func Seal(data any, opts ...utils.OptionExtender) (dst []byte, err error) {
	opt := utils.ApplyOptions[option](opts...)

	dstBuffer, cb := utils.BytesBufferPool.Get(nil)
	defer cb()

	dbytes, ok1 := data.([]byte)
	dstring, ok2 := data.(string)
	isRaw := ok1 || ok2

	// magic number
	if err = binary.Write(dstBuffer, binary.LittleEndian, sealMagicNumber); err != nil {
		return
	}

	// seal info
	inf := sealTypeNumber
	if opt.version > 0 {
		inf[0] = opt.version
	}
	if isRaw {
		inf[3] = 1
	} else {
		inf[1] = opt.serializeType.Value()
	}
	inf[2] = opt.compressType.Value()
	if err = binary.Write(dstBuffer, binary.LittleEndian, inf[:]); err != nil {
		return
	}

	// seal encrypted info TODO

	// seal context
	if opt.ctx == nil {
		if err = binary.Write(dstBuffer, binary.LittleEndian, uint64(0)); err != nil {
			return
		}
	} else {
		ctxBytes := fmkCtx.Flatten(opt.ctx).Marshal()
		if err = binary.Write(dstBuffer, binary.LittleEndian, uint64(len(ctxBytes))); err != nil {
			return
		}
		if err = binary.Write(dstBuffer, binary.LittleEndian, ctxBytes); err != nil {
			return
		}
	}

	// seal data type
	dt := utils.IndirectValue(reflect.ValueOf(data)).Type()
	structName := dt.PkgPath() + "." + dt.Name()
	if err = binary.Write(dstBuffer, binary.LittleEndian, uint64(len(structName))); err != nil {
		return
	}
	if err = binary.Write(dstBuffer, binary.LittleEndian, []byte(structName)); err != nil {
		return
	}

	// seal data
	marshaledBuffer, cb := utils.BytesBufferPool.Get(nil)
	defer cb()

	if isRaw {
		switch {
		case ok1:
			marshaledBuffer.Write(dbytes)
		case ok2:
			marshaledBuffer.WriteString(dstring)
		}
	} else {
		marshalFunc := serialize.MarshalStreamFunc(opt.serializeType, serialize.JsonEscapeHTML(false))
		if err = marshalFunc(marshaledBuffer, data); err != nil {
			return
		}
	}

	var encoded []byte
	if opt.compressType.IsValid() {
		encoded, err = encode.From(marshaledBuffer.Bytes()).Encode(encode.Compress(opt.compressType)).ToBytes()
	} else {
		encoded = marshaledBuffer.Bytes()
	}
	if err != nil {
		return
	}

	if err = binary.Write(dstBuffer, binary.LittleEndian, uint64(len(encoded))); err != nil {
		return
	}
	if err = binary.Write(dstBuffer, binary.LittleEndian, encoded); err != nil {
		return
	}

	dst = make([]byte, dstBuffer.Len())
	copy(dst, dstBuffer.Bytes())
	return
}
