package utils

import (
	"bytes"
	"encoding/gob"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/printer"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"

	"github.com/wfusion/gofusion/common/utils/clone"
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

type MarshalType string

const (
	MarshalTypeJson   MarshalType = "json"
	MarshalTypeYaml   MarshalType = "yaml"
	MarshalTypeYml    MarshalType = "yml"
	MarshalTypeToml   MarshalType = "toml"
	MarshalTypeHCL    MarshalType = "hcl"
	MarshalTypeTFVars MarshalType = "tfvars"
)

func Marshal(s any, tag MarshalType) (d []byte, err error) {
	bs, cb := BytesBufferPool.Get(nil)
	defer cb()

	switch tag {
	case MarshalTypeJson:
		err = json.NewEncoder(bs).Encode(s)
	case MarshalTypeYaml, MarshalTypeYml:
		err = yaml.NewEncoder(bs).Encode(s)
	case MarshalTypeToml:
		err = toml.NewEncoder(bs).Encode(s)
	case MarshalTypeHCL, MarshalTypeTFVars:
		err = marshalHCL(bs, d)
	default:
		err = gob.NewDecoder(bs).Decode(s)
	}
	if err != nil {
		return
	}
	d = clone.SliceComparable(bs.Bytes())
	return
}

func Unmarshal(s, d any, tag MarshalType) (err error) {
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
		case MarshalTypeJson:
			return json.NewDecoder(bs).Decode(d)
		case MarshalTypeYaml, MarshalTypeYml:
			return yaml.NewDecoder(bs).Decode(d)
		case MarshalTypeToml:
			_, err = toml.NewDecoder(bs).Decode(d)
			return
		case MarshalTypeHCL, MarshalTypeTFVars:
			return hcl.Unmarshal(bs.Bytes(), d)
		default:
			err = gob.NewDecoder(bs).Decode(d)
			return
		}
	default:
		cfg := &mapstructure.DecoderConfig{
			Result:           d,
			WeaklyTypedInput: true,
			DecodeHook: mapstructure.OrComposeDecodeHookFunc(
				mapstructure.StringToTimeDurationHookFunc(),
				mapstructure.StringToSliceHookFunc(","),
				mapstructure.StringToIPHookFunc(),
				mapstructure.StringToIPNetHookFunc(),
				mapstructure.StringToTimeHookFunc("2006-01-02 15:04:05"),       // time.DateTime format
				mapstructure.StringToTimeHookFunc("2006-01-02 15:04:05Z07:00"), // time.DateTime with timezone
				mapstructure.StringToTimeHookFunc(time.RFC1123),
				mapstructure.StringToTimeHookFunc(time.RFC3339),
			),
		}
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

// marshalHCL fork from github.com/spf13/viper@v1.16.0/internal/encoding/hcl/codec.go
func marshalHCL(buf *bytes.Buffer, v any) (err error) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}

	// TODO: use printer.Format? Is the trailing newline an issue?

	ast, err := hcl.Parse(string(b))
	if err != nil {
		return
	}

	err = printer.Fprint(buf, ast.Node)
	if err != nil {
		return
	}

	return
}
