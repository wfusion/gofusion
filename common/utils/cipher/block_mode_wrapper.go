package cipher

import (
	"crypto/cipher"

	"github.com/wfusion/gofusion/common/utils"
)

type blockModeWrapper struct {
	block     cipher.Block
	blockMode cipher.BlockMode
	stream    cipher.Stream
	blockSize int

	mode      Mode
	isEncoder bool
}

type blockModeOption struct {
	block     cipher.Block
	blockMode cipher.BlockMode
	stream    cipher.Stream
	mode      Mode
	isEncoder bool
}

func wrapBlock(block cipher.Block, mode Mode, isEncoder bool) utils.OptionFunc[blockModeOption] {
	return func(o *blockModeOption) {
		o.mode = mode
		o.block = block
		o.isEncoder = isEncoder
	}
}

func wrapBlockMode(bm cipher.BlockMode, mode Mode) utils.OptionFunc[blockModeOption] {
	return func(o *blockModeOption) {
		o.mode = mode
		o.blockMode = bm
	}
}

func wrapStream(stream cipher.Stream, mode Mode) utils.OptionFunc[blockModeOption] {
	return func(o *blockModeOption) {
		o.mode = mode
		o.stream = stream
	}
}

func newBlockModeWrapper(opts ...utils.OptionExtender) (w *blockModeWrapper) {
	opt := utils.ApplyOptions[blockModeOption](opts...)
	w = &blockModeWrapper{
		block:     opt.block,
		blockMode: opt.blockMode,
		stream:    opt.stream,
		blockSize: 0,
		mode:      opt.mode,
		isEncoder: opt.isEncoder,
	}

	switch {
	case opt.block != nil:
		w.blockSize = w.block.BlockSize()
	case opt.blockMode != nil:
		w.blockSize = w.blockMode.BlockSize()
	case opt.stream != nil:
		w.blockSize = defaultBlockSize * blockSizeTimes
	}

	return
}

func (b *blockModeWrapper) CipherMode() Mode     { return b.mode }
func (b *blockModeWrapper) PlainBlockSize() int  { return b.blockSize }
func (b *blockModeWrapper) CipherBlockSize() int { return b.blockSize }
func (b *blockModeWrapper) CryptBlocks(dst, src, buf []byte) (n int, err error) {
	if len(src) == 0 {
		return
	}
	n = len(src)
	switch {
	case b.blockMode != nil:
		b.blockMode.CryptBlocks(dst, src)
	case b.block != nil:
		if b.isEncoder {
			b.block.Encrypt(dst, src)
		} else {
			b.block.Decrypt(dst, src)
		}
	case b.stream != nil:
		b.stream.XORKeyStream(dst, src)
	default:
		copy(dst, src)
	}
	return
}
