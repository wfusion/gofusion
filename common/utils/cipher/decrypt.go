package cipher

import (
	"crypto/cipher"
	"errors"
	"fmt"
	"io"

	"github.com/wfusion/gofusion/common/utils"
)

func DecryptBytesFunc(algo Algorithm, mode Mode, key, iv []byte) (
	enc func(src []byte) (dst []byte, err error), err error) {
	bm, err := getDecrypter(algo, mode, key, iv)
	if err != nil {
		return
	}
	mode = bm.CipherMode()
	return func(src []byte) (dst []byte, err error) {
		dst, err = decrypt(bm, src)
		if err != nil {
			return
		}
		if mode.ShouldPadding() {
			dst, err = PKCS7Unpad(dst)
		}
		return
	}, err
}

func DecryptStreamFunc(algo Algorithm, mode Mode, key, iv []byte) (
	dec func(dst io.Writer, src io.Reader) (err error), err error) {
	var fn func(src io.Reader) io.ReadCloser

	_, err = utils.Catch(func() { fn = NewDecFunc(algo, mode, key, iv) })
	if err != nil {
		return
	}

	dec = func(dst io.Writer, src io.Reader) (err error) {
		buf, cb := utils.BytesPool.Get(defaultBlockSize * blockSizeTimes)
		defer cb()

		wrapper := fn(src)
		defer utils.CloseAnyway(wrapper)
		_, err = io.CopyBuffer(dst, wrapper, buf)
		return
	}

	return
}

func decrypt(bm blockMode, src []byte) (dst []byte, err error) {
	plainBlockSize := bm.PlainBlockSize()
	cipherBlockSize := bm.CipherBlockSize()
	defers := make([]func(), 0, 3)

	w, cb := utils.BytesBufferPool.Get(nil)
	defers = append(defers, cb)

	if plainBlockSize != 0 && cipherBlockSize != 0 {
		w.Grow((len(src) / cipherBlockSize) * plainBlockSize)
	} else {
		plainBlockSize, cipherBlockSize = len(src), len(src)
		w.Grow(plainBlockSize)
	}

	unsealed, cb := utils.BytesPool.Get(plainBlockSize)
	defers = append(defers, cb)

	buf, cb := utils.BytesPool.Get(plainBlockSize)
	defers = append(defers, cb)

	var n, blockSize int
	for len(src) > 0 {
		blockSize = utils.Min(cipherBlockSize, len(src))
		n, err = bm.CryptBlocks(unsealed[:plainBlockSize], src[:blockSize], buf)
		if err != nil {
			return
		}
		if _, err = w.Write(unsealed[:n]); err != nil {
			return
		}
		src = src[blockSize:]
	}

	bs := w.Bytes()
	dst = make([]byte, len(bs))
	copy(dst, bs)
	return
}

func getDecrypter(algo Algorithm, mode Mode, key, iv []byte) (bm blockMode, err error) {
	if blockMapping, ok := cipherBlockMapping[algo]; ok {
		var cipherBlock cipher.Block
		cipherBlock, err = blockMapping(key)
		if err != nil {
			return
		}
		modeMapping, ok := decryptModeMapping[mode]
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
		blockMapping, ok := streamDecryptMapping[algo]
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

type dec struct {
	bm blockMode
	r  io.Reader

	buf  []byte
	cb   func()
	n    int // current position in buf
	end  int // end of data in buf
	size int

	eof                 bool
	unsealed, unsealBuf []byte
}

func NewDecFunc(algo Algorithm, mode Mode, key, iv []byte) func(r io.Reader) io.ReadCloser {
	bm, err := getDecrypter(algo, mode, key, iv)
	if err != nil {
		panic(err)
	}
	if !bm.CipherMode().SupportStream() {
		panic(ErrNotSupportStream)
	}

	var (
		buf, unsealed, unsealBuf []byte
		cb                       func()
		size                     int
	)

	if size = bm.CipherBlockSize(); size > 0 {
		var bcb, ucb, ubcb func()
		buf, bcb = utils.BytesPool.Get(size)
		unsealed, ucb = utils.BytesPool.Get(bm.PlainBlockSize())
		unsealBuf, ubcb = utils.BytesPool.Get(bm.PlainBlockSize())
		cb = func() { bcb(); ucb(); ubcb() }
	}

	return func(r io.Reader) io.ReadCloser {
		return &dec{
			bm:        bm,
			r:         r,
			buf:       buf,
			cb:        cb,
			n:         0,
			end:       0,
			size:      size,
			eof:       false,
			unsealed:  unsealed,
			unsealBuf: unsealBuf,
		}
	}
}

func (d *dec) Read(p []byte) (n int, err error) {
	var (
		nr, nc int
		dst    []byte
	)

	if d.buf == nil {
		n, err = d.r.Read(p)
		d.eof = errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
		if err != nil && !d.eof {
			return
		}
		dst, err = decrypt(d.bm, p[:n])
		if err != nil {
			return
		}
		n = copy(p[:len(dst)], dst)
		if d.eof {
			err = io.EOF
		}
		return
	}

	for len(p) > 0 {
		// read from buffer
		if length := d.end - d.n; length > 0 {
			copied := utils.Min(length, len(p))
			n += copy(p[:copied], d.buf[d.n:d.n+copied])
			d.n += copied
			p = p[copied:]
			continue
		}

		// buffer is empty, write new buffer
		if d.eof {
			return n, io.EOF
		}
		d.n = 0
		d.end = 0
		for {
			nr, err = d.r.Read(d.buf[d.n:d.size])
			d.eof = errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
			if err != nil && !d.eof {
				return
			}

			d.n += nr
			if d.n < d.size && !d.eof {
				continue
			}

			nc, err = d.bm.CryptBlocks(d.unsealed, d.buf[:d.n], d.unsealBuf)
			if err != nil {
				return
			}

			d.end += copy(d.buf[:nc], d.unsealed[:nc])
			d.n = 0
			break
		}
	}

	return
}

func (d *dec) Close() (err error) {
	utils.CloseAnyway(d.r)
	if d.cb != nil {
		d.cb()
	}
	return
}
