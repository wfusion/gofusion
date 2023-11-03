package cipher

import (
	"bytes"
	"github.com/pkg/errors"
)

// PKCS7Pad Apply PKCS#7 padding
func PKCS7Pad(data []byte, blockSize int) []byte {
	if blockSize == 0 {
		return data
	}
	padding := blockSize - len(data)%blockSize
	return append(data, bytes.Repeat([]byte{byte(padding)}, padding)...)
}

// PKCS7Unpad Remove PKCS#7 padding
func PKCS7Unpad(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return data, nil
	}
	padding := int(data[length-1])
	if padding == 0 || padding > len(data) {
		return data, errors.New("invalid pkcs7 padding (padding size > data)")
	}
	for i := 1; i < padding; i++ {
		if data[length-1-i] != byte(padding) {
			return data, errors.New("invalid pkcs7 padding (pad[i] != padding text)")
		}
	}

	return data[:(length - padding)], nil
}
