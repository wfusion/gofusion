package encode

import (
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/cipher"
	"github.com/wfusion/gofusion/common/utils/compress"
)

func NewCodecStream(opts ...utils.OptionExtender) Streamable {
	encodeOrder := make([]EncodedType, 0, len(opts))
	for _, opt := range opts {
		if encType := parseEncodedType(opt); encType.IsValid() {
			encodeOrder = append(encodeOrder, encType)
		}
	}
	if len(encodeOrder) == 0 {
		panic(ErrEncodeMethodNotFound)
	}
	return &codecStream{
		o:           utils.ApplyOptions[option](opts...),
		encodeOrder: encodeOrder,
	}
}

type codecStream struct {
	o           *option
	encodeOrder []EncodedType
}

func (w *codecStream) Encode(dst io.Writer, src io.Reader) (n int64, err error) {
	for i := len(w.encodeOrder) - 1; i >= 0; i-- {
		switch w.encodeOrder[i] {
		case EncodedTypeCompress:
			dst = compress.NewEncFunc(w.o.compressAlgo)(dst)
		case EncodedTypeCipher:
			dst = cipher.NewEncFunc(w.o.cipherAlgo, w.o.cipherMode, w.o.key, w.o.iv)(dst)
		case EncodedTypeEncode:
			dst = NewWriter(w.o.printableAlgo, dst)
		}
	}

	buf, cb := utils.BytesPool.Get(defaultBufferSize)
	defer cb()

	defer utils.CloseAnyway(dst)
	n, err = io.CopyBuffer(dst, src, buf)
	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		err = nil
	}
	return
}

func (w *codecStream) Decode(dst io.Writer, src io.Reader) (n int64, err error) {
	for i := len(w.encodeOrder) - 1; i >= 0; i-- {
		switch w.encodeOrder[i] {
		case EncodedTypeCompress:
			src = compress.NewDecFunc(w.o.compressAlgo)(src)
		case EncodedTypeCipher:
			src = cipher.NewDecFunc(w.o.cipherAlgo, w.o.cipherMode, w.o.key, w.o.iv)(src)
		case EncodedTypeEncode:
			src = NewReader(w.o.printableAlgo, src)
		}
	}

	buf, cb := utils.BytesPool.Get(defaultBufferSize)
	defer cb()
	defer utils.CloseAnyway(src)
	n, err = io.CopyBuffer(dst, src, buf)
	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		err = nil
	}
	return
}

func NewWriter(algo Algorithm, w io.Writer) io.Writer {
	switch algo {
	case AlgorithmHex:
		return newEncoder(hex.NewEncoder(w), w)
	case AlgorithmBase32Std:
		return newEncoder(base32.NewEncoder(base32.StdEncoding, w), w)
	case AlgorithmBase32Hex:
		return newEncoder(base32.NewEncoder(base32.HexEncoding, w), w)
	case AlgorithmBase64Std:
		return newEncoder(base64.NewEncoder(base64.StdEncoding, w), w)
	case AlgorithmBase64URL:
		return newEncoder(base64.NewEncoder(base64.URLEncoding, w), w)
	case AlgorithmBase64RawStd:
		return newEncoder(base64.NewEncoder(base64.RawStdEncoding, w), w)
	case AlgorithmBase64RawURL:
		return newEncoder(base64.NewEncoder(base64.RawURLEncoding, w), w)
	default:
		panic(ErrUnknownAlgorithm)
	}
}

func NewEncodeFunc(algo Algorithm) func([]byte) ([]byte, error) {
	var (
		encodeFunc     func(src, dst []byte)
		encodedLenFunc func(int) int
	)
	switch algo {
	case AlgorithmHex:
		encodeFunc = func(src, dst []byte) { hex.Encode(src, dst) }
		encodedLenFunc = hex.EncodedLen
	case AlgorithmBase32Std:
		encodeFunc = base32.StdEncoding.Encode
		encodedLenFunc = base32.StdEncoding.EncodedLen
	case AlgorithmBase32Hex:
		encodeFunc = base32.HexEncoding.Encode
		encodedLenFunc = base32.HexEncoding.EncodedLen
	case AlgorithmBase64Std:
		encodeFunc = base64.StdEncoding.Encode
		encodedLenFunc = base64.StdEncoding.EncodedLen
	case AlgorithmBase64URL:
		encodeFunc = base64.URLEncoding.Encode
		encodedLenFunc = base64.URLEncoding.EncodedLen
	case AlgorithmBase64RawStd:
		encodeFunc = base64.RawStdEncoding.Encode
		encodedLenFunc = base64.RawStdEncoding.EncodedLen
	case AlgorithmBase64RawURL:
		encodeFunc = base64.RawURLEncoding.Encode
		encodedLenFunc = base64.RawURLEncoding.EncodedLen
	default:
		panic(ErrUnknownAlgorithm)
	}

	return func(src []byte) (dst []byte, err error) {
		dst = make([]byte, encodedLenFunc(len(src)))
		encodeFunc(dst, src)
		return
	}
}

func NewReader(algo Algorithm, r io.Reader) io.Reader {
	switch algo {
	case AlgorithmHex:
		return newDecoder(hex.NewDecoder(r), r)
	case AlgorithmBase32Std:
		return newDecoder(base32.NewDecoder(base32.StdEncoding, r), r)
	case AlgorithmBase32Hex:
		return newDecoder(base32.NewDecoder(base32.HexEncoding, r), r)
	case AlgorithmBase64Std:
		return newDecoder(base64.NewDecoder(base64.StdEncoding, r), r)
	case AlgorithmBase64URL:
		return newDecoder(base64.NewDecoder(base64.URLEncoding, r), r)
	case AlgorithmBase64RawStd:
		return newDecoder(base64.NewDecoder(base64.RawStdEncoding, r), r)
	case AlgorithmBase64RawURL:
		return newDecoder(base64.NewDecoder(base64.RawURLEncoding, r), r)
	default:
		panic(ErrUnknownAlgorithm)
	}
	return nil
}

func NewDecodeFunc(algo Algorithm) func([]byte) ([]byte, error) {
	var (
		decodeFunc     func(src, dst []byte) (int, error)
		decodedLenFunc func(int) int
	)
	switch algo {
	case AlgorithmHex:
		decodeFunc = hex.Decode
		decodedLenFunc = hex.DecodedLen
	case AlgorithmBase32Std:
		decodeFunc = base32.StdEncoding.Decode
		decodedLenFunc = base32.StdEncoding.DecodedLen
	case AlgorithmBase32Hex:
		decodeFunc = base32.HexEncoding.Decode
		decodedLenFunc = base32.HexEncoding.DecodedLen
	case AlgorithmBase64Std:
		decodeFunc = base64.StdEncoding.Decode
		decodedLenFunc = base64.StdEncoding.DecodedLen
	case AlgorithmBase64URL:
		decodeFunc = base64.URLEncoding.Decode
		decodedLenFunc = base64.URLEncoding.DecodedLen
	case AlgorithmBase64RawStd:
		decodeFunc = base64.RawStdEncoding.Decode
		decodedLenFunc = base64.RawStdEncoding.DecodedLen
	case AlgorithmBase64RawURL:
		decodeFunc = base64.RawURLEncoding.Decode
		decodedLenFunc = base64.RawURLEncoding.DecodedLen
	default:
		panic(ErrUnknownAlgorithm)
	}

	return func(src []byte) (dst []byte, err error) {
		dst = make([]byte, decodedLenFunc(len(src)))
		n, err := decodeFunc(dst, src)
		if err == nil {
			dst = dst[:n]
		}
		return
	}
}

type decoder struct {
	dec, r io.Reader
}

func newDecoder(dec, r io.Reader) io.Reader {
	return &decoder{dec: dec, r: r}
}

func (e *decoder) Read(p []byte) (n int, err error) { return e.dec.Read(p) }
func (e *decoder) Close()                           { utils.CloseAnyway(e.r); utils.CloseAnyway(e.dec) }

type encoder struct {
	enc, w io.Writer
}

func newEncoder(enc, w io.Writer) io.Writer {
	return &encoder{enc: enc, w: w}
}

func (e *encoder) Write(p []byte) (n int, err error) { return e.enc.Write(p) }
func (e *encoder) Close()                            { utils.CloseAnyway(e.enc); utils.CloseAnyway(e.w) }
