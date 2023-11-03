package serialize

import (
	"encoding/gob"
	"io"

	"github.com/fxamacker/cbor/v2"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/serialize/json"
)

var (
	marshalFuncMap = map[Algorithm]func(dst io.Writer, src any, opt *marshalOption) (err error){
		AlgorithmGob: func(dst io.Writer, src any, opt *marshalOption) (err error) {
			return gob.NewEncoder(dst).Encode(src)
		},
		AlgorithmJson: func(dst io.Writer, src any, opt *marshalOption) (err error) {
			enc := json.NewEncoder(dst)
			if !opt.jsonEscapeHTML {
				enc.SetEscapeHTML(false)
			}
			enc.SetIndent(opt.jsonIndentPrefix, opt.jsonIndent)
			return enc.Encode(src)
		},
		AlgorithmMsgpack: func(dst io.Writer, src any, opt *marshalOption) (err error) {
			enc := msgpack.NewEncoder(dst)
			enc.UseCompactInts(opt.msgpackUseCompactInts)
			enc.UseCompactFloats(opt.msgpackUseCompactFloats)
			return enc.Encode(src)
		},
		AlgorithmCbor: func(dst io.Writer, src any, opt *marshalOption) (err error) {
			return cbor.NewEncoder(dst).Encode(src)
		},
	}

	unmarshalFuncMap = map[Algorithm]func(dst any, src io.Reader, opt *unmarshalOption) (err error){
		AlgorithmGob: func(dst any, src io.Reader, opt *unmarshalOption) (err error) {
			return gob.NewDecoder(src).Decode(dst)
		},
		AlgorithmJson: func(dst any, src io.Reader, opt *unmarshalOption) (err error) {
			dec := json.NewDecoder(src)
			if opt.jsonNumber {
				dec.UseNumber()
			}
			if opt.disallowUnknownFields {
				dec.DisallowUnknownFields()
			}
			return dec.Decode(dst)
		},
		AlgorithmMsgpack: func(dst any, src io.Reader, opt *unmarshalOption) (err error) {
			dec := msgpack.NewDecoder(src)
			dec.DisallowUnknownFields(opt.disallowUnknownFields)
			return dec.Decode(dst)
		},
		AlgorithmCbor: func(dst any, src io.Reader, opt *unmarshalOption) (err error) {
			return cbor.NewDecoder(src).Decode(dst)
		},
	}
)

type marshalOption struct {
	jsonEscapeHTML               bool
	jsonIndent, jsonIndentPrefix string
	msgpackUseCompactInts        bool
	msgpackUseCompactFloats      bool
}

func MsgpackUseCompactInts(on bool) utils.OptionFunc[marshalOption] {
	return func(o *marshalOption) {
		o.msgpackUseCompactInts = on
	}
}

func MsgpackUseCompactFloats(on bool) utils.OptionFunc[marshalOption] {
	return func(o *marshalOption) {
		o.msgpackUseCompactFloats = on
	}
}

func JsonEscapeHTML(on bool) utils.OptionFunc[marshalOption] {
	return func(o *marshalOption) {
		o.jsonEscapeHTML = on
	}
}

func JsonIndent(prefix, indent string) utils.OptionFunc[marshalOption] {
	return func(o *marshalOption) {
		o.jsonIndentPrefix, o.jsonIndent = prefix, indent
	}
}

type unmarshalOption struct {
	jsonNumber            bool
	disallowUnknownFields bool
}

func JsonNumber() utils.OptionFunc[unmarshalOption] {
	return func(o *unmarshalOption) {
		o.jsonNumber = true
	}
}

func DisallowUnknownFields() utils.OptionFunc[unmarshalOption] {
	return func(o *unmarshalOption) {
		o.disallowUnknownFields = true
	}
}
