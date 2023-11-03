package compress

import (
	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/common/utils"
)

//go:generate stringer -type=Algorithm -trimprefix=Algorithm
type Algorithm uint8

const (
	AlgorithmUnknown Algorithm = iota
	AlgorithmZSTD
	AlgorithmZLib
	AlgorithmS2
	AlgorithmGZip
	AlgorithmDeflate
)

var (
	algorithmEnum = utils.NewEnumString[Algorithm, []Algorithm](
		map[Algorithm]string{
			AlgorithmZSTD:    "zstd",
			AlgorithmZLib:    "zlib",
			AlgorithmS2:      "s2",
			AlgorithmGZip:    "gzip",
			AlgorithmDeflate: "deflate",
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
