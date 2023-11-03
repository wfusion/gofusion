package serialize

import (
	"fmt"
	"io"
	"reflect"

	"github.com/wfusion/gofusion/common/utils"
)

func UnmarshalFunc[T any](algo Algorithm, opts ...utils.OptionExtender) func(src []byte) (T, error) {
	fn, ok := unmarshalFuncMap[algo]
	if !ok {
		panic(fmt.Errorf("unknown serialize algorithm type %+v", algo))
	}
	opt := utils.ApplyOptions[unmarshalOption](opts...)
	return func(src []byte) (dst T, err error) {
		bs, cb := utils.BytesBufferPool.Get(nil)
		defer cb()

		bs.Write(src)
		err = fn(&dst, bs, opt)
		return
	}
}

func UnmarshalFuncByType(algo Algorithm, dst any, opts ...utils.OptionExtender) func([]byte) (any, error) {
	fn, ok := unmarshalFuncMap[algo]
	if !ok {
		panic(fmt.Errorf("unknown serialize algorithm type %+v", algo))
	}
	dstType, ok := dst.(reflect.Type)
	if !ok {
		dstType = reflect.TypeOf(dst)
	}
	opt := utils.ApplyOptions[unmarshalOption](opts...)
	return func(src []byte) (dst any, err error) {
		dst = reflect.New(dstType).Interface()

		bs, cb := utils.BytesBufferPool.Get(nil)
		defer cb()

		bs.Write(src)
		err = fn(dst, bs, opt)
		return
	}
}

func UnmarshalStreamFunc[T any](algo Algorithm, opts ...utils.OptionExtender) func(io.Reader) (T, error) {
	fn, ok := unmarshalFuncMap[algo]
	if !ok {
		panic(fmt.Errorf("unknown serialize algorithm type %+v", algo))
	}
	opt := utils.ApplyOptions[unmarshalOption](opts...)
	return func(src io.Reader) (dst T, err error) {
		err = fn(&dst, src, opt)
		return
	}
}

func UnmarshalStreamFuncByType(algo Algorithm, dst any, opts ...utils.OptionExtender) func(io.Reader) (any, error) {
	fn, ok := unmarshalFuncMap[algo]
	if !ok {
		panic(fmt.Errorf("unknown serialize algorithm type %+v", algo))
	}
	dstType, ok := dst.(reflect.Type)
	if !ok {
		dstType = reflect.TypeOf(dst)
	}
	opt := utils.ApplyOptions[unmarshalOption](opts...)
	return func(src io.Reader) (dst any, err error) {
		d := reflect.New(dstType).Interface()
		if err = fn(d, src, opt); err != nil {
			return
		}
		dst = reflect.Indirect(reflect.ValueOf(d)).Interface()
		return
	}
}
