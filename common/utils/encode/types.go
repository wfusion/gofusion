package encode

import (
	"errors"
	"io"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/cipher"
	"github.com/wfusion/gofusion/common/utils/clone"
	"github.com/wfusion/gofusion/common/utils/compress"
)

const (
	RndSeed int64 = -6544809196299914340
)

var (
	ErrUnknownAlgorithm     = errors.New("unknown encode algorithm")
	ErrEncodeMethodNotFound = errors.New("not found encode method")

	defaultBufferSize = 4 * 1024 // 4kb
)

type Streamable interface {
	Encode(dst io.Writer, src io.Reader) (n int64, err error)
	Decode(dst io.Writer, src io.Reader) (n int64, err error)
}

type Codecable interface {
	Encode(opts ...utils.OptionExtender) Codecable
	Decode(opts ...utils.OptionExtender) Codecable
	ToBytes() (dst []byte, err error)
	ToString() (dst string, err error)
}

type option struct {
	key, iv    []byte
	cipherMode cipher.Mode
	cipherAlgo cipher.Algorithm

	compressAlgo  compress.Algorithm
	printableAlgo Algorithm
}

func Cipher(algo cipher.Algorithm, mode cipher.Mode, key, iv []byte) utils.OptionFunc[option] {
	return func(o *option) {
		o.cipherAlgo = algo
		o.cipherMode = mode
		o.key = clone.SliceComparable(key)
		o.iv = clone.SliceComparable(iv)
	}
}

func Compress(algo compress.Algorithm) utils.OptionFunc[option] {
	return func(o *option) {
		o.compressAlgo = algo
	}
}

func Encode(algo Algorithm) utils.OptionFunc[option] {
	return func(o *option) {
		o.printableAlgo = algo
	}
}
