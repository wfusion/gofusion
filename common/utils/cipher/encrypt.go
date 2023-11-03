package cipher

import (
	"crypto/cipher"
	"fmt"
	"io"

	"github.com/wfusion/gofusion/common/utils"
)

func EncryptBytesFunc(algo Algorithm, mode Mode, key, iv []byte) (
	enc func(src []byte) (dst []byte, err error), err error) {
	bm, err := getEncrypter(algo, mode, key, iv)
	if err != nil {
		return
	}
	mode = bm.CipherMode()
	plainBlockSize := bm.PlainBlockSize()
	return func(src []byte) (dst []byte, err error) {
		if mode.ShouldPadding() {
			src = PKCS7Pad(src, plainBlockSize)
		}
		return encrypt(bm, src)
	}, err
}

func EncryptStreamFunc(algo Algorithm, mode Mode, key, iv []byte) (
	enc func(dst io.Writer, src io.Reader) (err error), err error) {
	var fn func(w io.Writer) io.WriteCloser

	_, err = utils.Catch(func() { fn = NewEncFunc(algo, mode, key, iv) })
	if err != nil {
		return nil, err
	}

	enc = func(dst io.Writer, src io.Reader) (err error) {
		buf, cb := utils.BytesPool.Get(defaultBlockSize * blockSizeTimes)
		defer cb()

		wrapper := fn(dst)
		defer utils.CloseAnyway(wrapper)

		_, err = io.CopyBuffer(wrapper, src, buf)
		return
	}

	return
}

func encrypt(bm blockMode, src []byte) (dst []byte, err error) {
	plainBlockSize := bm.PlainBlockSize()
	cipherBlockSize := bm.CipherBlockSize()
	defers := make([]func(), 0, 3)
	defer func() {
		for _, cb := range defers {
			cb()
		}
	}()

	w, cb := utils.BytesBufferPool.Get(nil)
	defers = append(defers, cb)

	if plainBlockSize != 0 && cipherBlockSize != 0 {
		w.Grow((len(src) / plainBlockSize) * cipherBlockSize)
	} else {
		plainBlockSize, cipherBlockSize = len(src), len(src)
		w.Grow(cipherBlockSize)
	}

	sealed, cb := utils.BytesPool.Get(cipherBlockSize)
	defers = append(defers, cb)

	buf, cb := utils.BytesPool.Get(plainBlockSize)
	defers = append(defers, cb)

	var n, blockSize int
	for len(src) > 0 {
		blockSize = utils.Min(plainBlockSize, len(src))
		n, err = bm.CryptBlocks(sealed[:cipherBlockSize], src[:blockSize], buf)
		if err != nil {
			return
		}
		if _, err = w.Write(sealed[:n]); err != nil {
			return
		}
		src = src[blockSize:]
	}

	bs := w.Bytes()
	dst = make([]byte, len(bs))
	copy(dst, bs)
	return
}

func getEncrypter(algo Algorithm, mode Mode, key, iv []byte) (bm blockMode, err error) {
	if blockMapping, ok := cipherBlockMapping[algo]; ok {
		var cipherBlock cipher.Block
		cipherBlock, err = blockMapping(key)
		if err != nil {
			return
		}
		modeMapping, ok := encryptModeMapping[mode]
		if !ok {
			return nil, fmt.Errorf("unknown cipher mode %+v", mode)
		}
		bm, err = modeMapping(cipherBlock, iv)
		if err != nil {
			return
		}
	}

	// stream
	if bm == nil {
		blockMapping, ok := streamEncryptMapping[algo]
		if !ok {
			return nil, fmt.Errorf("unknown cipher algorithm %+v", algo)
		}
		if bm, err = blockMapping(key); err != nil {
			return
		}
	}

	if bm == nil {
		return nil, fmt.Errorf("unknown cipher algorithm(%+v) or mode(%+v)", algo, mode)
	}

	return
}

type enc struct {
	bm blockMode
	w  io.Writer

	n   int
	buf []byte
	cb  func()

	sealed, sealBuf []byte
}

func NewEncFunc(algo Algorithm, mode Mode, key, iv []byte) func(w io.Writer) io.WriteCloser {
	bm, err := getEncrypter(algo, mode, key, iv)
	if err != nil {
		panic(err)
	}
	if !bm.CipherMode().SupportStream() {
		panic(ErrNotSupportStream)
	}

	var (
		buf, sealed, sealBuf []byte
		cb                   func()
	)
	if bm.PlainBlockSize() > 0 {
		var bcb, scb, sbcb func()
		buf, bcb = utils.BytesPool.Get(bm.PlainBlockSize())
		sealed, scb = utils.BytesPool.Get(bm.CipherBlockSize())
		sealBuf, sbcb = utils.BytesPool.Get(bm.PlainBlockSize())
		cb = func() { bcb(); scb(); sbcb() }
	}

	return func(w io.Writer) io.WriteCloser {
		return &enc{
			bm:      bm,
			w:       w,
			n:       0,
			buf:     buf,
			cb:      cb,
			sealed:  sealed,
			sealBuf: sealBuf,
		}
	}
}

func (e *enc) Write(p []byte) (n int, err error) {
	if e.buf == nil {
		var dst []byte
		dst, err = encrypt(e.bm, p)
		if err != nil {
			return
		}
		n = len(p)
		_, err = e.w.Write(dst)
		return
	}

	written := 0
	for len(p) > 0 {
		nCopy := utils.Min(len(p), len(e.buf)-e.n)
		copy(e.buf[e.n:], p[:nCopy])
		e.n += nCopy
		written += nCopy
		p = p[nCopy:]

		if e.n == len(e.buf) {
			if err = e.flush(); err != nil {
				return written, err
			}
		}
	}

	return written, nil
}

func (e *enc) flush() (err error) {
	if e.buf == nil {
		return
	}

	n, err := e.bm.CryptBlocks(e.sealed, e.buf[:e.n], e.sealBuf)
	if err != nil {
		return
	}

	_, err = e.w.Write(e.sealed[:n])
	if err != nil {
		return err
	}

	e.n = 0
	return
}

func (e *enc) Flush() (err error) {
	defer utils.FlushAnyway(e.w)
	if e.n > 0 {
		err = e.flush()
	}
	return
}

func (e *enc) Close() (err error) {
	defer func() {
		utils.CloseAnyway(e.w)
		if e.cb != nil {
			e.cb()
		}
	}()

	return e.Flush()
}
