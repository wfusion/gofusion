package cipher

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/rc4"
	"errors"

	"github.com/tjfoc/gmsm/sm4"
	"golang.org/x/crypto/chacha20poly1305"
)

type blockMode interface {
	CipherMode() Mode
	PlainBlockSize() int
	CipherBlockSize() int
	CryptBlocks(dst, src, buf []byte) (n int, err error)
}

const (
	RndSeed int64 = -3250455768463625913
)

var (
	// ErrNotSupportStream for stream encryption
	ErrNotSupportStream = errors.New("not support stream encryption mode")
)

var (
	cipherBlockMapping = map[Algorithm]func(key []byte) (cipher.Block, error){
		AlgorithmDES:  func(key []byte) (cipher.Block, error) { return des.NewCipher(key) },
		Algorithm3DES: func(key []byte) (cipher.Block, error) { return des.NewTripleDESCipher(key) },
		AlgorithmAES:  func(key []byte) (cipher.Block, error) { return aes.NewCipher(key) },
		AlgorithmSM4:  func(key []byte) (cipher.Block, error) { return sm4.NewCipher(key) },
	}

	encryptModeMapping = map[Mode]func(b cipher.Block, iv []byte) (e blockMode, err error){
		ModeECB: func(b cipher.Block, iv []byte) (e blockMode, err error) {
			return wrapErr(newBlockModeWrapper(
				wrapBlock(b, ModeECB, true)),
			)
		},
		ModeCBC: func(b cipher.Block, iv []byte) (e blockMode, err error) {
			return wrapErr(newBlockModeWrapper(
				wrapBlockMode(cipher.NewCBCEncrypter(b, iv), ModeCBC)),
			)
		},
		ModeCFB: func(b cipher.Block, iv []byte) (e blockMode, err error) {
			return wrapErr(newBlockModeWrapper(
				wrapStream(cipher.NewCFBEncrypter(b, iv), ModeCFB)),
			)
		},
		ModeCTR: func(b cipher.Block, iv []byte) (e blockMode, err error) {
			return wrapErr(newBlockModeWrapper(
				wrapStream(cipher.NewCTR(b, iv), ModeCTR)),
			)
		},
		ModeOFB: func(b cipher.Block, iv []byte) (e blockMode, err error) {
			return wrapErr(newBlockModeWrapper(
				wrapStream(cipher.NewOFB(b, iv), ModeOFB)),
			)
		},
		ModeGCM: func(b cipher.Block, iv []byte) (e blockMode, err error) {
			aead, err := cipher.NewGCM(b)
			if err != nil {
				return
			}
			e = newAEADEncryptWrapper(aead, b.BlockSize(), ModeGCM)
			return
		},
	}
	decryptModeMapping = map[Mode]func(b cipher.Block, iv []byte) (bm blockMode, err error){
		ModeECB: func(b cipher.Block, iv []byte) (e blockMode, err error) {
			return wrapErr(newBlockModeWrapper(
				wrapBlock(b, ModeECB, false)),
			)
		},
		ModeCBC: func(b cipher.Block, iv []byte) (bm blockMode, err error) {
			return wrapErr(newBlockModeWrapper(
				wrapBlockMode(cipher.NewCBCDecrypter(b, iv), ModeCBC)),
			)
		},
		ModeCFB: func(b cipher.Block, iv []byte) (bm blockMode, err error) {
			return wrapErr(newBlockModeWrapper(
				wrapStream(cipher.NewCFBDecrypter(b, iv), ModeCFB)),
			)
		},
		ModeCTR: func(b cipher.Block, iv []byte) (bm blockMode, err error) {
			return wrapErr(newBlockModeWrapper(
				wrapStream(cipher.NewCTR(b, iv), ModeCTR)),
			)
		},
		ModeOFB: func(b cipher.Block, iv []byte) (bm blockMode, err error) {
			return wrapErr(newBlockModeWrapper(
				wrapStream(cipher.NewOFB(b, iv), ModeOFB)),
			)
		},
		ModeGCM: func(b cipher.Block, iv []byte) (bm blockMode, err error) {
			cipherAEAD, err := cipher.NewGCM(b)
			if err != nil {
				return
			}
			bm = newAEADDecryptWrapper(cipherAEAD, b.BlockSize(), ModeGCM)
			return
		},
	}

	streamEncryptMapping = map[Algorithm]func(key []byte) (bm blockMode, err error){
		AlgorithmRC4: func(key []byte) (bm blockMode, err error) {
			cipher, err := rc4.NewCipher(key)
			if err != nil {
				return
			}
			bm = &rc4Wrapper{rc: cipher}
			return
		},
		AlgorithmChaCha20poly1305: func(key []byte) (bm blockMode, err error) {
			cipherAEAD, err := chacha20poly1305.New(key)
			if err != nil {
				return
			}
			bm = newAEADEncryptWrapper(cipherAEAD, defaultBlockSize, modeStream)
			return
		},
		AlgorithmXChaCha20poly1305: func(key []byte) (bm blockMode, err error) {
			cipherAEAD, err := chacha20poly1305.NewX(key)
			if err != nil {
				return
			}
			bm = newAEADEncryptWrapper(cipherAEAD, defaultBlockSize, modeStream)
			return
		},
	}
	streamDecryptMapping = map[Algorithm]func(key []byte) (bm blockMode, err error){
		AlgorithmRC4: func(key []byte) (bm blockMode, err error) {
			cipher, err := rc4.NewCipher(key)
			if err != nil {
				return
			}
			bm = &rc4Wrapper{rc: cipher}
			return
		},
		AlgorithmChaCha20poly1305: func(key []byte) (bm blockMode, err error) {
			cipherAEAD, err := chacha20poly1305.New(key)
			if err != nil {
				return
			}
			bm = newAEADDecryptWrapper(cipherAEAD, defaultBlockSize, modeStream)
			return
		},
		AlgorithmXChaCha20poly1305: func(key []byte) (bm blockMode, err error) {
			cipherAEAD, err := chacha20poly1305.NewX(key)
			if err != nil {
				return
			}
			bm = newAEADDecryptWrapper(cipherAEAD, defaultBlockSize, modeStream)
			return
		},
	}
)

func wrapErr[T any](a T) (b T, err error) { b = a; return }
