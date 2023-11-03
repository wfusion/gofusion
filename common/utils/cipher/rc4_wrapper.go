package cipher

import "crypto/rc4"

type rc4Wrapper struct {
	rc *rc4.Cipher
}

func (r *rc4Wrapper) CipherMode() Mode     { return modeStream }
func (r *rc4Wrapper) PlainBlockSize() int  { return 0 }
func (r *rc4Wrapper) CipherBlockSize() int { return 0 }
func (r *rc4Wrapper) CryptBlocks(dst, src, buf []byte) (n int, err error) {
	if len(src) == 0 {
		return
	}
	r.rc.XORKeyStream(dst, src)
	n = len(src)
	return
}
