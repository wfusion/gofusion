package encode

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/wfusion/gofusion/common/fus/debug"
)

const (
	tooLargeSize = 512 * 1024 * 1024 // 512m
)

func In(input string) (r io.Reader, isStream bool, err error) {
	if path.Ext(input) == "" {
		r = bytes.NewBufferString(input)
		return
	}

	filename := path.Clean(input)
	st, err := os.Stat(filename)

	// it is not a file
	if err != nil {
		r = bytes.NewBufferString(input)
		err = nil
		return
	}

	// read all content from small file
	if st.Size() <= tooLargeSize {
		var bs []byte
		bs, err = ioutil.ReadFile(filename)
		if err != nil {
			return
		}
		r = bytes.NewBuffer(bs)
		return
	}

	debug.Printf("begin streaming encoding\n")

	// with stream
	isStream = true
	r, err = os.Open(filename)
	return
}
