package compress

import (
	"errors"
	"io"

	"github.com/klauspost/compress/flate"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/s2"
	"github.com/klauspost/compress/zlib"
	"github.com/klauspost/compress/zstd"

	"github.com/wfusion/gofusion/common/utils"
)

const (
	RndSeed int64 = 9009760027768254931
)

var (
	ErrUnknownAlgorithm = errors.New("unknown compress algorithm")

	encoderPools = map[Algorithm]utils.Poolable[*writerPoolSealer]{
		AlgorithmZSTD: utils.NewPool(func() *writerPoolSealer {
			return &writerPoolSealer{utils.Must(zstd.NewWriter(nil))}
		}),
		AlgorithmZLib: utils.NewPool(func() *writerPoolSealer {
			return &writerPoolSealer{zlib.NewWriter(nil)}
		}),
		AlgorithmS2: utils.NewPool(func() *writerPoolSealer {
			return &writerPoolSealer{s2.NewWriter(nil)}
		}),
		AlgorithmGZip: utils.NewPool(func() *writerPoolSealer {
			return &writerPoolSealer{gzip.NewWriter(nil)}
		}),
		AlgorithmDeflate: utils.NewPool(func() *writerPoolSealer {
			return &writerPoolSealer{utils.Must(flate.NewWriter(nil, flate.DefaultCompression))}
		}),
	}

	decoderPools = map[Algorithm]utils.Poolable[*readerPoolSealer]{
		AlgorithmZSTD: utils.NewPool(func() *readerPoolSealer {
			return &readerPoolSealer{utils.Must(zstd.NewReader(nil))}
		}),
		AlgorithmZLib: utils.NewPool(func() *readerPoolSealer {
			return &readerPoolSealer{new(zlibDecodable)} // init when call reset method
		}),
		AlgorithmS2: utils.NewPool(func() *readerPoolSealer {
			return &readerPoolSealer{&s2Decodable{s2.NewReader(nil)}}
		}),
		AlgorithmGZip: utils.NewPool(func() *readerPoolSealer {
			return &readerPoolSealer{new(gzipDecodable)} // init when call reset method
		}),
		AlgorithmDeflate: utils.NewPool(func() *readerPoolSealer {
			return &readerPoolSealer{new(deflateDecodable)} // init when call reset method
		}),
	}

	eofErrs = []error{io.EOF, io.ErrUnexpectedEOF}
)

type readerPoolSealer struct{ decodable }

func (r *readerPoolSealer) Reset(obj any) error {
	return r.decodable.Reset(obj.(io.Reader))
}

type writerPoolSealer struct{ encodable }

func (w *writerPoolSealer) Reset(obj any) {
	w.encodable.Reset(obj.(io.Writer))
}
