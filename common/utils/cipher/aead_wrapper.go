package cipher

import (
	"crypto/cipher"
	"errors"

	"github.com/wfusion/gofusion/common/utils"
)

const (
	blockSizeTimes   = 1 << 10 // 1024 times
	defaultBlockSize = 16      // 16 bytes
)

type abstractAEADWrapper struct {
	cipher.AEAD
	overheadSize    int
	nonceSize       int
	blockSize       int
	plainBlockSize  int
	cipherBlockSize int
	sealedSize      int

	mode Mode
}

func newAbstractAEADWrapper(cipherAEAD cipher.AEAD, blockSize int, mode Mode) abstractAEADWrapper {
	plainBlockSize := blockSize * blockSizeTimes
	return abstractAEADWrapper{
		AEAD:            cipherAEAD,
		overheadSize:    cipherAEAD.Overhead(),
		nonceSize:       cipherAEAD.NonceSize(),
		blockSize:       blockSize,
		plainBlockSize:  plainBlockSize,
		cipherBlockSize: cipherAEAD.NonceSize() + cipherAEAD.Overhead() + plainBlockSize,
		sealedSize:      cipherAEAD.Overhead() + plainBlockSize,
		mode:            mode,
	}
}

func (a *abstractAEADWrapper) PlainBlockSize() int  { return a.plainBlockSize }
func (a *abstractAEADWrapper) CipherBlockSize() int { return a.cipherBlockSize }
func (a *abstractAEADWrapper) CipherMode() Mode     { return a.mode }

type aeadEncryptWrapper struct {
	abstractAEADWrapper
}

func newAEADEncryptWrapper(cipherAEAD cipher.AEAD, blockSize int, mode Mode) blockMode {
	return &aeadEncryptWrapper{abstractAEADWrapper: newAbstractAEADWrapper(cipherAEAD, blockSize, mode)}
}

func (a *aeadEncryptWrapper) CryptBlocks(dst, src, buf []byte) (n int, err error) {
	if len(src) == 0 {
		return
	}

	nonce, cb := utils.BytesPool.Get(a.nonceSize)
	defer cb()
	if _, err = utils.CryptoRandom(nonce); err != nil {
		return
	}

	sealed := a.Seal(buf[:0], nonce, src[:utils.Min(a.plainBlockSize, len(src))], nil)
	copy(dst[:a.nonceSize], nonce)
	copy(dst[a.nonceSize:], sealed)
	n = len(nonce) + len(sealed)
	return
}

type aeadDecryptWrapper struct {
	abstractAEADWrapper
}

func newAEADDecryptWrapper(cipherAEAD cipher.AEAD, blockSize int, mode Mode) blockMode {
	return &aeadDecryptWrapper{abstractAEADWrapper: newAbstractAEADWrapper(cipherAEAD, blockSize, mode)}
}

func (a *aeadDecryptWrapper) CryptBlocks(dst, src, buf []byte) (n int, err error) {
	if len(src) == 0 {
		return
	}
	sealedSize := utils.Min(a.sealedSize, len(src)-a.nonceSize)
	if sealedSize < 0 {
		return 0, errors.New("input not full blocks when decrypt")
	}

	nonce := src[:a.nonceSize]
	src = src[a.nonceSize:]
	unsealed, err := a.Open(buf[:0], nonce, src[:sealedSize], nil)
	if err != nil {
		return
	}

	copy(dst, unsealed)
	n = len(unsealed)
	return
}
