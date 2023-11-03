package encode

import (
	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/common/utils"
)

type EncodedType int8

const (
	EncodedTypeUnknown EncodedType = iota
	EncodedTypeCipher
	EncodedTypeCompress
	EncodedTypeEncode
)

var (
	encodeTypeEnum = utils.NewEnumString[EncodedType, []EncodedType](
		map[EncodedType]string{
			EncodedTypeCipher:   "cipher",
			EncodedTypeCompress: "compress",
			EncodedTypeEncode:   "encode",
		},
	)
)

func (e EncodedType) Value() uint8 {
	return uint8(e)
}

func (e EncodedType) String() string {
	return encodeTypeEnum.String(e)
}

func (e EncodedType) IsValid() bool {
	return encodeTypeEnum.IsValid(e)
}

func ParseEncodedType(s any) EncodedType {
	switch v := s.(type) {
	case string:
		if enumList := encodeTypeEnum.Enum(v); len(enumList) > 0 {
			return enumList[0]
		}
	case EncodedType:
		return v
	default:
		return EncodedType(cast.ToInt(s))
	}
	return EncodedTypeUnknown
}

func parseEncodedType(one utils.OptionExtender) EncodedType {
	opts := utils.ApplyOptions[option](one)
	switch {
	case opts.cipherAlgo.IsValid():
		return EncodedTypeCipher
	case opts.compressAlgo.IsValid():
		return EncodedTypeCompress
	case opts.printableAlgo.IsValid():
		return EncodedTypeEncode
	default:
		return EncodedTypeUnknown
	}
}
