package encode

import (
	"bytes"
	"runtime"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/cipher"
	"github.com/wfusion/gofusion/common/utils/compress"
)

type codec struct {
	buf              *bytes.Buffer
	encOpts, decOpts []utils.OptionExtender

	queue   [][2]int
	release func()
}

// From a byte slice or a string, not concurrent safe
func From[T ~[]byte | ~string](src T) (c Codecable) {
	buf, cb := utils.BytesBufferPool.Get(nil)

	switch v := any(src).(type) {
	case []byte:
		buf.Write(v)
	case string:
		buf.WriteString(v)
	default:
		buf.Write([]byte(src))
	}

	c = &codec{
		buf:     buf,
		release: cb,
	}

	runtime.SetFinalizer(c, func(c *codec) { c.release() })
	return
}

func (c *codec) Encode(opts ...utils.OptionExtender) Codecable {
	for i, p := 0, len(c.encOpts); i < len(opts); i++ {
		c.queue = append(c.queue, [2]int{p + i, 0})
	}
	c.encOpts = append(c.encOpts, opts...)
	return c
}
func (c *codec) Decode(opts ...utils.OptionExtender) Codecable {
	for i, p := 0, len(c.decOpts); i < len(opts); i++ {
		c.queue = append(c.queue, [2]int{p + i, 1})
	}
	c.decOpts = append(c.decOpts, opts...)
	return c
}
func (c *codec) ToBytes() (dst []byte, err error) {
	if err = c.transform(); err != nil {
		return
	}
	dst = make([]byte, c.buf.Len())
	copy(dst, c.buf.Bytes())
	return
}
func (c *codec) ToString() (dst string, err error) {
	if err = c.transform(); err != nil {
		return
	}
	dst = c.buf.String()
	return
}

func (c *codec) transform() (err error) {
	for _, elem := range c.queue {
		idx, isEnc := elem[0], elem[1] == 0
		if isEnc {
			if err = c.encode(c.encOpts[idx]); err != nil {
				return
			}
		} else {
			if err = c.decode(c.decOpts[idx]); err != nil {
				return
			}

		}
	}

	return
}
func (c *codec) encode(opt utils.OptionExtender) (err error) {
	var enc func([]byte) ([]byte, error)

	switch option := utils.ApplyOptions[option](opt); parseEncodedType(opt) {
	case EncodedTypeCompress:
		enc = compress.EncodeBytesFunc(option.compressAlgo)
	case EncodedTypeCipher:
		enc, err = cipher.EncryptBytesFunc(option.cipherAlgo, option.cipherMode, option.key, option.iv)
	case EncodedTypeEncode:
		enc = NewEncodeFunc(option.printableAlgo)
	default:
		return ErrEncodeMethodNotFound
	}
	if err != nil {
		return
	}

	dst, err := enc(c.buf.Bytes())
	if err != nil {
		return
	}
	c.buf.Reset()
	c.buf.Write(dst)
	return
}
func (c *codec) decode(opt utils.OptionExtender) (err error) {
	var dec func([]byte) ([]byte, error)

	switch option := utils.ApplyOptions[option](opt); parseEncodedType(opt) {
	case EncodedTypeCompress:
		dec = compress.DecodeBytesFunc(option.compressAlgo)
	case EncodedTypeCipher:
		dec, err = cipher.DecryptBytesFunc(option.cipherAlgo, option.cipherMode, option.key, option.iv)
	case EncodedTypeEncode:
		dec = NewDecodeFunc(option.printableAlgo)
	default:
		return ErrEncodeMethodNotFound
	}
	if err != nil {
		return
	}

	dst, err := dec(c.buf.Bytes())
	if err != nil {
		return
	}
	c.buf.Reset()
	c.buf.Write(dst)
	return
}
