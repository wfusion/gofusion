package utils

import (
	"encoding/gob"
	"reflect"

	"github.com/BurntSushi/toml"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"

	"github.com/wfusion/gofusion/common/utils/serialize/json"
)

// JsonStringify for k8s ConfigMap, Secret
type JsonStringify[T any] struct {
	Value T
}

func (j *JsonStringify[T]) MarshalJSON() (output []byte, err error) {
	var buf []byte
	if buf, err = json.Marshal(j.Value); err != nil {
		return
	}
	if output, err = json.Marshal(buf); err != nil {
		return
	}
	return
}

func (j *JsonStringify[T]) UnmarshalJSON(input []byte) (err error) {
	var buf string
	if err = json.Unmarshal(input, &buf); err != nil {
		return
	}
	if err = json.Unmarshal(UnsafeStringToBytes(buf), &j.Value); err != nil {
		return
	}
	return
}

func MustJsonMarshal(s any) []byte             { return Must(json.Marshal(s)) }
func MustJsonMarshalString(s any) string       { return string(MustJsonMarshal(s)) }
func MustJsonUnmarshal[T any](s []byte) (t *T) { MustSuccess(json.Unmarshal(s, &t)); return }

type unmarshalType string

const (
	UnmarshalTypeJson unmarshalType = "json"
	UnmarshalTypeYaml unmarshalType = "yaml"
	UnmarshalTypeToml unmarshalType = "toml"
)

// An InvalidUnmarshalError describes an invalid argument passed to Unmarshal.
// (The argument to Unmarshal must be a non-nil pointer.)
type InvalidUnmarshalError struct {
	Type reflect.Type
}

func (e *InvalidUnmarshalError) Error() string {
	if e.Type == nil {
		return "common/utils: Unmarshal(nil)"
	}

	if e.Type.Kind() != reflect.Pointer {
		return "common/utils: Unmarshal(non-pointer " + e.Type.String() + ")"
	}
	return "common/utils: Unmarshal(nil " + e.Type.String() + ")"
}

func Unmarshal(s, d any, tag unmarshalType) (err error) {
	switch s.(type) {
	case string, []byte:
		bs, cb := BytesBufferPool.Get(nil)
		defer cb()
		if ss, ok := s.(string); ok {
			bs.WriteString(ss)
		} else {
			bs.Write(s.([]byte))
		}

		switch tag {
		case UnmarshalTypeJson:
			return json.NewDecoder(bs).Decode(d)
		case UnmarshalTypeYaml:
			err = yaml.NewDecoder(bs).Decode(d)
			return
		case UnmarshalTypeToml:
			_, err = toml.NewDecoder(bs).Decode(d)
			return
		default:
			err = gob.NewDecoder(bs).Decode(d)
			return
		}
	default:
		cfg := &mapstructure.DecoderConfig{Result: d}
		if tag != "" {
			cfg.TagName = string(tag)
		}

		var dec *mapstructure.Decoder
		if dec, err = mapstructure.NewDecoder(cfg); err != nil {
			return
		}
		err = dec.Decode(s)
		return
	}
}
