package encode

import (
	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/common/utils"
)

//go:generate stringer -type=Algorithm -trimprefix=Algorithm
type Algorithm uint8

const (
	AlgorithmUnknown Algorithm = iota
	AlgorithmHex
	AlgorithmBase32Std
	AlgorithmBase32Hex
	AlgorithmBase64Std
	AlgorithmBase64URL
	AlgorithmBase64RawStd // without padding
	AlgorithmBase64RawURL // without padding
)

var (
	algorithmEnum = utils.NewEnumString[Algorithm, []Algorithm](
		map[Algorithm]string{
			AlgorithmHex:          "hex",
			AlgorithmBase32Std:    "base32",
			AlgorithmBase32Hex:    "base32-hex",
			AlgorithmBase64Std:    "base64",
			AlgorithmBase64URL:    "base64-url",
			AlgorithmBase64RawStd: "base64-raw",
			AlgorithmBase64RawURL: "base64-raw-url",
		},
	)
)

func (e Algorithm) Value() uint8 {
	return uint8(e)
}

func (e Algorithm) String() string {
	return algorithmEnum.String(e)
}

func (e Algorithm) IsValid() bool {
	return algorithmEnum.IsValid(e)
}

func ParseAlgorithm(s any) Algorithm {
	switch v := s.(type) {
	case string:
		if enumList := algorithmEnum.Enum(v); len(enumList) > 0 {
			return enumList[0]
		}
	case Algorithm:
		return v
	default:
		return Algorithm(cast.ToInt(s))
	}
	return AlgorithmUnknown
}
