package pd

import (
	"context"
	"encoding/binary"
	"reflect"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/compress"
	"github.com/wfusion/gofusion/common/utils/encode"
	"github.com/wfusion/gofusion/common/utils/inspect"
	"github.com/wfusion/gofusion/common/utils/serialize"

	fusCtx "github.com/wfusion/gofusion/context"
)

func Unseal(src []byte, opts ...utils.OptionExtender) (ctx context.Context, dst any, ok bool, err error) {
	if len(src) <= sealPrefixLength {
		return defaultUnseal(src, opts...)
	}

	// unseal magic number
	magicNumber, next := src[:4], src[4:]
	if binary.LittleEndian.Uint32(magicNumber) != sealMagicNumber {
		return defaultUnseal(src, opts...)
	}

	// unseal info
	inf, next := next[:len(sealTypeNumber)], next[len(sealTypeNumber):]
	switch inf[0] {
	case 1:
		return unsealV1(inf, next, opts...)
	default:
		panic(errors.Errorf("unsupported message version for unseal: %v", inf[0]))
	}
}

func unsealV1(inf, src []byte, opts ...utils.OptionExtender) (ctx context.Context, dst any, ok bool, err error) {
	opt := utils.ApplyOptions[option](opts...)
	serializeType := serialize.ParseAlgorithm(serialize.Algorithm(inf[1]))
	if opt.serializeType.IsValid() {
		serializeType = opt.serializeType
	}
	compressType := compress.ParseAlgorithm(compress.Algorithm(inf[2]))
	if opt.compressType.IsValid() {
		compressType = opt.compressType
	}

	isRaw := false
	if inf[3] == 1 {
		isRaw = true
	}

	// unseal encrypted inf
	encryptedInfLength := binary.LittleEndian.Uint32(inf[4:])
	_, src = src[:encryptedInfLength], src[encryptedInfLength:]

	// unseal context
	contextLengthBytes, src := src[:8], src[8:]
	contextLength := binary.LittleEndian.Uint64(contextLengthBytes)
	if contextLength > 0 {
		var contextBytes []byte
		contextBytes, src = src[:contextLength], src[contextLength:]
		ctx = fusCtx.New(fusCtx.Context(contextBytes))
	}

	// unseal data type
	inf, src = src[:8], src[8:]
	structNameLength := binary.LittleEndian.Uint64(inf)
	structName, src := src[:structNameLength], src[structNameLength:]
	if !isRaw && opt.dataType == nil {
		if opt.dataType = inspect.TypeOf(string(structName)); opt.dataType == nil {
			opt.dataType = reflect.TypeOf((*any)(nil)).Elem()
		}
	}

	// unseal data
	// unseal data length
	_, src = src[:8], src[8:]
	// binary.LittleEndian.Uint64(src[:8])

	// unseal data
	var decoded []byte
	if compressType.IsValid() {
		decoded, err = encode.From(src).Decode(encode.Compress(compressType)).ToBytes()
	} else {
		decoded = src
	}
	if err != nil {
		return
	}

	if !serializeType.IsValid() {
		dst = decoded
	} else {
		dstBuffer, cb := utils.BytesBufferPool.Get(nil)
		defer cb()
		dstBuffer.Write(decoded)

		unmarshalFunc := serialize.UnmarshalStreamFuncByType(serializeType, opt.dataType)
		if dst, err = unmarshalFunc(dstBuffer); err != nil {
			return
		}
	}

	ok = true
	return
}

func UnsealT[T any](src []byte, opts ...utils.OptionExtender) (ctx context.Context, dst T, ok bool, err error) {
	if len(src) <= sealPrefixLength {
		return
	}

	// unseal magic number
	magicNumber, src := src[:4], src[4:]
	if binary.LittleEndian.Uint32(magicNumber) != sealMagicNumber {
		return
	}

	// unseal info
	inf, src := src[:len(sealTypeNumber)], src[len(sealTypeNumber):]
	switch inf[0] {
	case 1:
		return unsealV1T[T](inf, src, opts...)
	default:
		panic(errors.Errorf("unsupported message version for unseal: %v", inf[0]))
	}
}

func unsealV1T[T any](inf, src []byte, opts ...utils.OptionExtender) (ctx context.Context, dst T, ok bool, err error) {
	opt := utils.ApplyOptions[option](opts...)
	serializeType := serialize.ParseAlgorithm(serialize.Algorithm(inf[1]))
	if opt.serializeType.IsValid() {
		serializeType = opt.serializeType
	}
	compressType := compress.ParseAlgorithm(compress.Algorithm(inf[2]))
	if opt.compressType.IsValid() {
		compressType = opt.compressType
	}

	isRaw := false
	if inf[3] == 1 {
		isRaw = true
	}

	// unseal encrypted inf
	encryptedInfLength := binary.LittleEndian.Uint32(inf[4:])
	_, src = src[:encryptedInfLength], src[encryptedInfLength:]

	// unseal context
	contextLengthBytes, src := src[:8], src[8:]
	contextLength := binary.LittleEndian.Uint64(contextLengthBytes)
	if contextLength > 0 {
		var contextBytes []byte
		contextBytes, src = src[:contextLength], src[contextLength:]
		ctx = fusCtx.New(fusCtx.Context(contextBytes))
	}

	// unseal data type
	inf, src = src[:8], src[8:]
	structNameLength := binary.LittleEndian.Uint64(inf)
	structName, src := src[:structNameLength], src[structNameLength:]
	if !isRaw && opt.dataType == nil {
		if opt.dataType = inspect.TypeOf(string(structName)); opt.dataType == nil {
			opt.dataType = reflect.TypeOf((*any)(nil)).Elem()
		}
	}

	// unseal data
	// unseal data length
	_, src = src[:8], src[8:]
	// binary.LittleEndian.Uint64(src[:8])

	// unseal data
	var decoded []byte
	if compressType.IsValid() {
		decoded, err = encode.From(src).Decode(encode.Compress(compressType)).ToBytes()
	} else {
		decoded = src
	}
	if err != nil {
		return
	}

	if !serializeType.IsValid() {
		dst = reflect.ValueOf(decoded).Convert(reflect.TypeOf(new(T)).Elem()).Interface().(T)
	} else {
		dstBuffer, cb := utils.BytesBufferPool.Get(nil)
		defer cb()
		dstBuffer.Write(decoded)

		unmarshalFunc := serialize.UnmarshalStreamFunc[T](serializeType)
		if dst, err = unmarshalFunc(dstBuffer); err != nil {
			return
		}
	}

	ok = true
	return
}

func UnsealRaw(src []byte, opts ...utils.OptionExtender) (ctx context.Context, dst []byte, isRaw bool, err error) {
	if len(src) <= sealPrefixLength {
		return nil, src, true, nil
	}

	// unseal magic number
	magicNumber, next := src[:4], src[4:]
	if binary.LittleEndian.Uint32(magicNumber) != sealMagicNumber {
		return nil, src, true, nil
	}

	// unseal info
	inf, src := next[:len(sealTypeNumber)], next[len(sealTypeNumber):]
	switch inf[0] {
	case 1:
		return unsealRawV1(inf, src, opts...)
	default:
		panic(errors.Errorf("unsupported message version for unseal raw: %v", inf[0]))
	}
	return
}

func unsealRawV1(inf, src []byte, opts ...utils.OptionExtender) (
	ctx context.Context, dst []byte, isRaw bool, err error) {
	opt := utils.ApplyOptions[option](opts...)
	compressType := compress.ParseAlgorithm(compress.Algorithm(inf[2]))
	if opt.compressType.IsValid() {
		compressType = opt.compressType
	}
	if src[3] == 1 {
		isRaw = true
	}

	// unseal encrypted inf
	encryptedInfLength := binary.LittleEndian.Uint32(inf[4:])
	_, src = src[:encryptedInfLength], src[encryptedInfLength:]

	// unseal context
	contextLengthBytes, src := src[:8], src[8:]
	contextLength := binary.LittleEndian.Uint64(contextLengthBytes)
	if contextLength > 0 {
		var contextBytes []byte
		contextBytes, src = src[:contextLength], src[contextLength:]
		ctx = fusCtx.New(fusCtx.Context(contextBytes))
	}

	// unseal data type
	inf, src = src[:8], src[8:]
	structNameLength := binary.LittleEndian.Uint64(inf)
	_, src = src[:structNameLength], src[structNameLength:]

	// unseal data
	// unseal data length
	_, src = src[:8], src[8:]
	// binary.LittleEndian.Uint64(src[:8])

	// unseal data
	if compressType.IsValid() {
		dst, err = encode.From(src).Decode(encode.Compress(compressType)).ToBytes()
	} else {
		dst = src
	}
	return
}

func defaultUnseal(src []byte, opts ...utils.OptionExtender) (ctx context.Context, dst any, ok bool, err error) {
	opt := utils.ApplyOptions[option](opts...)
	if opt.compressType.IsValid() {
		if src, err = encode.From(src).Decode(encode.Compress(opt.compressType)).ToBytes(); err != nil {
			return
		}
	}
	if !opt.serializeType.IsValid() {
		// try to convert directly
		srcVal := reflect.ValueOf(src)
		if srcVal.CanConvert(opt.dataType) {
			return nil, srcVal.Convert(opt.dataType).Interface(), false, nil
		}

		// try to map the structure
		out := reflect.New(opt.dataType).Interface()
		if err = mapstructure.Decode(src, out); err != nil {
			return
		}
		dst = reflect.ValueOf(out).Elem()
		return
	}
	unmarshalFunc := serialize.UnmarshalFuncByType(opt.serializeType, opt.dataType, serialize.JsonEscapeHTML(false))
	dst, err = unmarshalFunc(src)
	return
}
