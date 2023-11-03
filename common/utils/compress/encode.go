package compress

import (
	"io"

	"github.com/wfusion/gofusion/common/utils"
)

type encodable interface {
	io.WriteCloser
	Flush() error
	Reset(w io.Writer)
}

func EncodeBytesFunc(algo Algorithm) func(src []byte) (dst []byte, err error) {
	return func(src []byte) (dst []byte, err error) {
		var (
			enc      encodable
			recycles []func()
		)
		defer func() {
			for _, cb := range recycles {
				if cb != nil {
					cb()
				}
			}
		}()

		dstBuffer, bufferCb := utils.BytesBufferPool.Get(nil)
		recycles = append(recycles, bufferCb)

		_, err = utils.Catch(func() {
			var cb func()
			enc, cb = getEncoder(algo, dstBuffer)
			recycles = append(recycles, cb)
		})
		if err != nil || enc == nil {
			return
		}

		srcBuffer, bufferCb := utils.BytesBufferPool.Get(nil)
		recycles = append(recycles, bufferCb)
		srcBuffer.Write(src)
		if _, err = io.Copy(enc, srcBuffer); err != nil {
			if err = utils.ErrIgnore(err, eofErrs...); err != nil {
				return
			}
		}
		if err = enc.Flush(); err != nil {
			if err = utils.ErrIgnore(err, eofErrs...); err != nil {
				return
			}
		}
		dst = make([]byte, dstBuffer.Len())
		copy(dst, dstBuffer.Bytes())
		return
	}
}

func EncodeStreamFunc(algo Algorithm) func(dst io.Writer, src io.Reader) (err error) {
	return func(dst io.Writer, src io.Reader) (err error) {
		var (
			enc encodable
			cb  func()
		)
		if _, err = utils.Catch(func() { enc, cb = getEncoder(algo, dst) }); err != nil {
			return
		}
		defer func() {
			if cb != nil {
				cb()
			}
		}()

		if _, err = io.Copy(enc, src); err != nil {
			if err = utils.ErrIgnore(err, eofErrs...); err != nil {
				return
			}
		}
		err = enc.Flush()
		return utils.ErrIgnore(err, eofErrs...)
	}
}

func getEncoder(algo Algorithm, dst io.Writer) (enc encodable, recycle func()) {
	p, ok := encoderPools[algo]
	if !algo.IsValid() || !ok {
		panic(ErrUnknownAlgorithm)
	}

	sealer, recycle := p.Get(dst)
	enc = sealer.encodable
	return
}

type encoder struct {
	enc encodable
	w   io.Writer
	cb  func()
}

func NewEncFunc(algo Algorithm) func(w io.Writer) encodable {
	return func(w io.Writer) encodable {
		enc, cb := getEncoder(algo, w)
		if enc == nil {
			panic(ErrUnknownAlgorithm)
		}
		return &encoder{
			enc: enc,
			w:   w,
			cb:  cb,
		}
	}
}

func (e *encoder) Write(p []byte) (n int, err error) { return e.enc.Write(p) }
func (e *encoder) Reset(w io.Writer)                 { e.enc.Reset(w) }
func (e *encoder) Flush() (err error)                { defer utils.FlushAnyway(e.w); return e.enc.Flush() }
func (e *encoder) Close() (err error) {
	defer e.cb()
	defer utils.CloseAnyway(e.w)
	return e.Flush()
}
