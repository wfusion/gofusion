package serialize

import (
	"fmt"
	"io"

	"github.com/wfusion/gofusion/common/utils"
)

func MarshalFunc(algo Algorithm, opts ...utils.OptionExtender) func(src any) ([]byte, error) {
	fn, ok := marshalFuncMap[algo]
	if !ok {
		panic(fmt.Errorf("unknown serialize algorithm type %+v", algo))
	}
	opt := utils.ApplyOptions[marshalOption](opts...)
	return func(src any) (dst []byte, err error) {
		bs, cb := utils.BytesBufferPool.Get(nil)
		defer cb()

		if err = fn(bs, src, opt); err != nil {
			return
		}

		dst = make([]byte, bs.Len())
		copy(dst, bs.Bytes())
		return
	}
}

func MarshalStreamFunc(algo Algorithm, opts ...utils.OptionExtender) func(dst io.Writer, src any) error {
	fn, ok := marshalFuncMap[algo]
	if !ok {
		panic(fmt.Errorf("unknown serialize algorithm type %+v", algo))
	}
	opt := utils.ApplyOptions[marshalOption](opts...)
	return func(dst io.Writer, src any) error {
		return fn(dst, src, opt)
	}
}
