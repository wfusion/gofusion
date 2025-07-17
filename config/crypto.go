package config

import (
	"context"
	"encoding/base64"
	"reflect"

	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"github.com/spf13/pflag"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/cipher"
	"github.com/wfusion/gofusion/common/utils/compress"
	"github.com/wfusion/gofusion/common/utils/encode"
)

const (
	cryptoTagKey = "encrypted"
)

var (
	cryptoFlagString string
)

func CryptoConstruct(ctx context.Context, c CryptoConf, _ ...utils.OptionExtender) func() {
	if c.Config != nil {
		checkCryptoConf("", c.Config)
	}
	for name, cfg := range c.Custom {
		if cfg != nil {
			checkCryptoConf(name, cfg)
		}
	}

	return func() {

	}
}

func checkCryptoConf(name string, c *cryptoConf) {
	// cipher
	if c.Algorithm = cipher.ParseAlgorithm(c.AlgorithmString); !c.Algorithm.IsValid() {
		panic(errors.Errorf("unknown config %s algorithm: %s", name, c.AlgorithmString))
	}
	c.Mode = cipher.ParseMode(c.ModeString)
	if !c.Mode.IsValid() {
		panic(errors.Errorf("unknown config %s mode: %s", name, c.ModeString))
	}
	if utils.IsStrBlank(c.KeyBase64) {
		panic(errors.Errorf("%s not found crypto key", name))
	}
	c.Key = utils.Must(base64.StdEncoding.DecodeString(c.KeyBase64))
	if c.Mode.NeedIV() && utils.IsStrBlank(c.IVBase64) {
		panic(errors.Errorf("%s not found crypto iv", name))
	}
	c.IV = utils.Must(base64.StdEncoding.DecodeString(c.IVBase64))

	// compress
	if utils.IsStrPtrNotBlank(c.CompressAlgorithmString) {
		c.CompressAlgorithm = compress.ParseAlgorithm(*c.CompressAlgorithmString)
		if !c.CompressAlgorithm.IsValid() {
			panic(errors.Errorf("unknown config %s compress algorithm: %s", name, *c.CompressAlgorithmString))
		}
	}

	// output
	if utils.IsStrPtrNotBlank(c.OutputAlgorithmString) {
		c.OutputAlgorithm = encode.ParseAlgorithm(*c.OutputAlgorithmString)
		if !c.OutputAlgorithm.IsValid() {
			panic(errors.Errorf("unknown config %s output algorithm: %s", name, *c.OutputAlgorithmString))
		}
	}
}

func CryptoEncryptFunc[T ~[]byte | ~string](opts ...utils.OptionExtender) func(src T) (dst T) {
	o := utils.ApplyOptions[InitOption](opts...)
	opt := utils.ApplyOptions[cryptoConfigOption](opts...)
	optsMap := Use(o.AppName).(*registry).cryptoConfig().ToOptionMap()
	opts = optsMap[opt.name]
	return func(src T) (dst T) {
		return T(utils.Must(encode.From(src).Encode(opts...).ToBytes()))
	}
}

func CryptoDecryptFunc[T ~[]byte | ~string](opts ...utils.OptionExtender) func(src T) (dst T) {
	o := utils.ApplyOptions[InitOption](opts...)
	opt := utils.ApplyOptions[cryptoConfigOption](opts...)
	optsMap := Use(o.AppName).(*registry).cryptoConfig().ToOptionMap()
	for _, copt := range optsMap {
		utils.SliceReverse(copt)
	}
	opts = optsMap[opt.name]
	return func(src T) (dst T) {
		return T(utils.Must(encode.From(src).Decode(opts...).ToBytes()))
	}
}

type cryptoOption struct {
	tag string
}

func CryptoTag(tag string) utils.OptionFunc[cryptoOption] {
	return func(o *cryptoOption) {
		o.tag = tag
	}
}

func CryptoEncryptByTag(data any, opts ...utils.OptionExtender) {
	o := utils.ApplyOptions[InitOption](opts...)
	co := utils.ApplyOptions[cryptoOption](opts...)
	tag := cryptoTagKey
	if co.tag != "" {
		tag = co.tag
	}

	optsMap := Use(o.AppName).(*registry).cryptoConfig().ToOptionMap()
	supportedFields := utils.NewSet(reflect.Struct, reflect.Array, reflect.Slice, reflect.Map)
	utils.TraverseValue(data, false, func(field reflect.StructField, value reflect.Value) (end, stepIn bool) {
		if !value.IsValid() || !value.CanInterface() || !value.CanSet() {
			return
		}

		vk := value.Kind()
		stepIn = supportedFields.Contains(vk) ||
			(vk == reflect.Ptr && value.Elem().IsValid() && value.Elem().Kind() == reflect.Struct)

		configName, ok := field.Tag.Lookup(tag)
		if !ok {
			return
		}
		encOpts, ok := optsMap[configName]
		if !ok {
			return
		}
		src := cast.ToString(value.Interface())
		if utils.IsStrBlank(src) {
			return
		}

		dst := utils.Must(encode.From(src).Encode(encOpts...).ToString())
		value.SetString(dst)
		return
	})
}

func CryptoDecryptByTag(data any, opts ...utils.OptionExtender) {
	o := utils.ApplyOptions[InitOption](opts...)
	co := utils.ApplyOptions[cryptoOption](opts...)
	tag := cryptoTagKey
	if co.tag != "" {
		tag = co.tag
	}

	optsMap := Use(o.AppName).(*registry).cryptoConfig().ToOptionMap()
	for _, copt := range optsMap {
		utils.SliceReverse(copt)
	}

	supportedFields := utils.NewSet(reflect.Struct, reflect.Array, reflect.Slice, reflect.Map)
	utils.TraverseValue(data, false, func(field reflect.StructField, value reflect.Value) (end, stepIn bool) {
		if !value.IsValid() || !value.CanInterface() || !value.CanSet() {
			return
		}

		vk := value.Kind()
		stepIn = supportedFields.Contains(vk) ||
			(vk == reflect.Ptr && value.Elem().IsValid() && value.Elem().Kind() == reflect.Struct)

		configName, ok := field.Tag.Lookup(tag)
		if !ok {
			return
		}
		decOpts, ok := optsMap[configName]
		if !ok {
			return
		}
		src := cast.ToString(value.Interface())
		if utils.IsStrBlank(src) {
			return
		}

		dst := utils.Must(encode.From(src).Decode(decOpts...).ToString())
		value.SetString(dst)
		return
	})
}

func init() {
	pflag.StringVarP(&cryptoFlagString, "crypto", "", "", "json string for crypto config")
}
