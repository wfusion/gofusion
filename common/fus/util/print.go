package util

import (
	"encoding/base64"
	"fmt"
	"unicode"
	"unicode/utf8"
)

func PrintOutput(bs []byte) {
	if isPrintable(bs) {
		fmt.Printf("output: %s\n", bs)
	} else {
		fmt.Printf("output(base64): %s\n", base64.StdEncoding.EncodeToString(bs))
	}
}

func isPrintable(bs []byte) (ok bool) {
	for len(bs) > 0 {
		r, size := utf8.DecodeRune(bs)
		if r == utf8.RuneError {
			return
		}
		if !unicode.IsPrint(r) && !unicode.IsSpace(r) {
			return
		}
		bs = bs[size:]
	}

	ok = true
	return
}
