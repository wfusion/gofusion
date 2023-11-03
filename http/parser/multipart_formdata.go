package parser

import (
	"io"
	"io/ioutil"
	"mime/multipart"
	"reflect"

	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/serialize/json"
)

const (
	keyBoundary = "boundary"
)

var (
	byteSliceType = reflect.TypeOf(([]byte)(nil))
)

type MultipartFormDataParser struct {
	boundary string
}

func (m *MultipartFormDataParser) PreParse(args map[string]string) error {
	boundary, ok := args[keyBoundary]
	if !ok {
		return malformedRequest("missing boundary in multipart/form-data")
	}
	m.boundary = boundary
	return nil
}

func (m *MultipartFormDataParser) Parse(src io.Reader, dst reflect.Value) (err error) {
	for dst.Kind() == reflect.Ptr {
		dst.Set(reflect.New(dst.Type().Elem()))
		dst = dst.Elem()
	}

	var (
		part   *multipart.Part
		body   []byte
		dt     = reflect.TypeOf(dst.Interface())
		reader = multipart.NewReader(src, m.boundary)
	)
	defer func() {
		if part != nil {
			utils.CloseAnyway(part)
		}
	}()

	for {
		part, err = reader.NextPart()
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			break
		}

		k := part.FormName()
		fNum := m.lookupFieldByTag(dt, k, "json")
		if fNum == -1 {
			continue
		}
		fv := dst.Field(fNum)
		if !fv.IsValid() || !fv.CanSet() {
			continue
		}

		// transform
		if body, err = ioutil.ReadAll(part); err != nil {
			break
		} else if err = m.transformField(fv, body); err != nil {
			break
		} else if err = part.Close(); err != nil {
			break
		}
	}

	return
}

func (m *MultipartFormDataParser) lookupFieldByTag(t reflect.Type, key, tag string) (fNum int) {
	n := t.NumField()
	for i := 0; i < n; i++ {
		f := t.Field(i)
		if v, ok := f.Tag.Lookup(tag); ok && v == key {
			return i
		}
	}
	return -1
}

var (
	castReflectTypeMap = map[reflect.Kind]func(reflect.Value, []byte) error{
		reflect.Bool:    func(f reflect.Value, b []byte) (e error) { v, e := cast.ToBoolE(b); f.SetBool(v); return },
		reflect.String:  func(f reflect.Value, b []byte) (e error) { v, e := cast.ToStringE(b); f.SetString(v); return },
		reflect.Int:     func(f reflect.Value, b []byte) (e error) { v, e := cast.ToInt64E(b); f.SetInt(v); return },
		reflect.Int8:    func(f reflect.Value, b []byte) (e error) { v, e := cast.ToInt64E(b); f.SetInt(v); return },
		reflect.Int16:   func(f reflect.Value, b []byte) (e error) { v, e := cast.ToInt64E(b); f.SetInt(v); return },
		reflect.Int32:   func(f reflect.Value, b []byte) (e error) { v, e := cast.ToInt64E(b); f.SetInt(v); return },
		reflect.Int64:   func(f reflect.Value, b []byte) (e error) { v, e := cast.ToInt64E(b); f.SetInt(v); return },
		reflect.Uint:    func(f reflect.Value, b []byte) (e error) { v, e := cast.ToUint64E(b); f.SetUint(v); return },
		reflect.Uint8:   func(f reflect.Value, b []byte) (e error) { v, e := cast.ToUint64E(b); f.SetUint(v); return },
		reflect.Uint16:  func(f reflect.Value, b []byte) (e error) { v, e := cast.ToUint64E(b); f.SetUint(v); return },
		reflect.Uint32:  func(f reflect.Value, b []byte) (e error) { v, e := cast.ToUint64E(b); f.SetUint(v); return },
		reflect.Uint64:  func(f reflect.Value, b []byte) (e error) { v, e := cast.ToUint64E(b); f.SetUint(v); return },
		reflect.Float32: func(f reflect.Value, b []byte) (e error) { v, e := cast.ToFloat64E(b); f.SetFloat(v); return },
		reflect.Float64: func(f reflect.Value, b []byte) (e error) { v, e := cast.ToFloat64E(b); f.SetFloat(v); return },
	}
)

func (m *MultipartFormDataParser) transformField(f reflect.Value, param []byte) (err error) {
	ft := f.Type()
	if ft.Kind() == reflect.Ptr {
		ft = ft.Elem()
		if f.IsNil() {
			f.Set(reflect.New(ft))
		}
		f = f.Elem()
	}

	if byteSliceType.ConvertibleTo(ft) {
		f.Set(reflect.ValueOf(param).Convert(ft))
		return
	}

	caster, ok := castReflectTypeMap[ft.Kind()]
	if !ok {
		return json.Unmarshal(param, f.Addr().Interface())
	}

	return caster(f, param)
}
