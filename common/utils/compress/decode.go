package compress

import (
	"io"

	"github.com/wfusion/gofusion/common/utils"
)

type decodable interface {
	io.Reader
	Reset(r io.Reader) error
}

func DecodeBytesFunc(algo Algorithm) func(src []byte) (dst []byte, err error) {
	return func(src []byte) (dst []byte, err error) {
		var (
			dec      decodable
			recycles []func()
		)
		defer func() {
			for _, cb := range recycles {
				if cb != nil {
					cb()
				}
			}
		}()

		srcBuffer, bufferCb := utils.BytesBufferPool.Get(nil)
		recycles = append(recycles, bufferCb)
		srcBuffer.Write(src)
		_, err = utils.Catch(func() {
			var cb func()
			dec, cb = getDecoder(algo, srcBuffer)
			recycles = append(recycles, cb)
		})
		if err != nil || dec == nil {
			return
		}

		dstBuffer, bufferCb := utils.BytesBufferPool.Get(nil)
		recycles = append(recycles, bufferCb)
		if _, err = io.Copy(dstBuffer, dec); err != nil {
			if err = utils.ErrIgnore(err, eofErrs...); err != nil {
				return
			}
		}
		dst = make([]byte, dstBuffer.Len())
		copy(dst, dstBuffer.Bytes())
		return
	}
}

func DecodeStreamFunc(algo Algorithm) func(dst io.Writer, src io.Reader) (err error) {
	return func(dst io.Writer, src io.Reader) (err error) {
		var (
			dec decodable
			cb  func()
		)
		defer func() {
			if cb != nil {
				cb()
			}
		}()

		if _, err = utils.Catch(func() { dec, cb = getDecoder(algo, src) }); err != nil {
			return
		}

		_, err = io.Copy(dst, dec)
		return utils.ErrIgnore(err, eofErrs...)
	}
}

func getDecoder(algo Algorithm, src io.Reader) (dec decodable, recycle func()) {
	p, ok := decoderPools[algo]
	if !algo.IsValid() || !ok {
		panic(ErrUnknownAlgorithm)
	}

	sealer, recycle := p.Get(src)
	dec = sealer.decodable
	return
}

type decoder struct {
	dec decodable
	r   io.Reader
	cb  func()
}

func NewDecFunc(algo Algorithm) func(r io.Reader) io.ReadCloser {
	return func(r io.Reader) io.ReadCloser {
		dec, cb := getDecoder(algo, r)
		return &decoder{
			dec: dec,
			r:   r,
			cb:  cb,
		}
	}
}

func (e *decoder) Reset(r io.Reader) error          { return e.dec.Reset(r) }
func (e *decoder) Read(p []byte) (n int, err error) { return e.dec.Read(p) }
func (e *decoder) Close() (err error)               { defer e.cb(); utils.CloseAnyway(e.r); return }
