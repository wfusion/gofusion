package cipher

import (
	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/common/utils"
)

//go:generate stringer -type=Algorithm -trimprefix=Algorithm
type Algorithm uint8

const (
	AlgorithmUnknown Algorithm = iota
	AlgorithmDES
	Algorithm3DES
	AlgorithmAES
	AlgorithmRC4
	AlgorithmChaCha20poly1305
	AlgorithmXChaCha20poly1305
	AlgorithmSM4 // GM/T 0002-2012
)

var (
	algorithmEnum = utils.NewEnumString[Algorithm, []Algorithm](
		map[Algorithm]string{
			AlgorithmDES:               "des",
			Algorithm3DES:              "3des",
			AlgorithmAES:               "aes",
			AlgorithmRC4:               "rc4",
			AlgorithmChaCha20poly1305:  "chacha20poly1305",
			AlgorithmXChaCha20poly1305: "xchacha20poly1305",
			AlgorithmSM4:               "sm4",
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
