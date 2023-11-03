package config

import (
	"context"
	"encoding/base64"
	"hash/crc64"
	"math/rand"
	"reflect"

	"github.com/pkg/errors"
	"github.com/spf13/cast"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/cipher"
	"github.com/wfusion/gofusion/common/utils/compress"
	"github.com/wfusion/gofusion/common/utils/encode"
)

const (
	cryptoTagKey = "encrypted"
)

type CryptoConf struct {
	Config *cryptoConf            `yaml:"config" json:"config" toml:"config"`
	Custom map[string]*cryptoConf `yaml:"custom" json:"custom" toml:"custom"`
}

func (c *CryptoConf) ToOptionMap() (result map[string][]utils.OptionExtender) {
	result = make(map[string][]utils.OptionExtender)
	if c.Config != nil {
		result[""] = c.Config.ToOptions()
	}
	for name, cfg := range c.Custom {
		result[name] = cfg.ToOptions()
	}
	return
}

// cryptoConf
//nolint: revive // struct tag too long issue
type cryptoConf struct {
	Key        []byte `yaml:"-" json:"-" toml:"-"`
	KeyBase64  string `yaml:"key_base64" json:"key_base64" toml:"key_base64"`
	ConfuseKey bool   `yaml:"confuse_key" json:"confuse_key" toml:"confuse_key"`

	IV       []byte `yaml:"-" json:"-" toml:"-"`
	IVBase64 string `yaml:"iv_base64" json:"iv_base64" toml:"iv_base64"`

	Algorithm       cipher.Algorithm `yaml:"-" json:"-" toml:"-"`
	AlgorithmString string           `yaml:"algorithm" json:"algorithm" toml:"algorithm"`

	Mode       cipher.Mode `yaml:"-" json:"-" toml:"-"`
	ModeString string      `yaml:"mode" json:"mode" toml:"mode"`

	CompressAlgorithm       compress.Algorithm `yaml:"-" json:"-" toml:"-"`
	CompressAlgorithmString *string            `yaml:"compress_algorithm" json:"compress_algorithm" toml:"compress_algorithm"`

	OutputAlgorithm       encode.Algorithm `yaml:"-" json:"-" toml:"-"`
	OutputAlgorithmString *string          `yaml:"output_algorithm" json:"output_algorithm" toml:"output_algorithm"`
}

func (c *cryptoConf) ToOptions() (opts []utils.OptionExtender) {
	if !c.Algorithm.IsValid() {
		return nil
	}

	if c.ConfuseKey {
		c.Key = c.cryptoConfuseKey(c.Key)
	}

	opts = make([]utils.OptionExtender, 0, 3)
	opts = append(opts, encode.Cipher(c.Algorithm, c.Mode, c.Key, c.IV))
	if c.CompressAlgorithm.IsValid() {
		opts = append(opts, encode.Compress(c.CompressAlgorithm))
	}
	if c.OutputAlgorithm.IsValid() {
		opts = append(opts, encode.Encode(c.OutputAlgorithm))
	}
	return
}

func (c *cryptoConf) cryptoConfuseKey(key []byte) (confused []byte) {
	var (
		k1 = make([]byte, len(key))
		k2 = make([]byte, len(key))
		k3 = make([]byte, len(key))
	)
	rndSeed := int64(crc64.Checksum(key, crc64.MakeTable(crc64.ISO)))
	utils.Must(rand.New(rand.NewSource(cipher.RndSeed ^ compress.RndSeed ^ rndSeed)).Read(k1))
	utils.Must(rand.New(rand.NewSource(cipher.RndSeed ^ encode.RndSeed ^ rndSeed)).Read(k2))
	utils.Must(rand.New(rand.NewSource(compress.RndSeed ^ encode.RndSeed ^ rndSeed)).Read(k3))

	confused = make([]byte, len(key))
	utils.Must(rand.New(rand.NewSource(cipher.RndSeed ^ compress.RndSeed ^ encode.RndSeed)).Read(confused))
	for i := 0; i < len(confused); i++ {
		confused[i] ^= k1[i] ^ k2[i] ^ k3[i]
	}
	return
}

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

type cryptoConfigOption struct {
	name string
}

func CryptoConfigName(name string) utils.OptionFunc[cryptoConfigOption] {
	return func(o *cryptoConfigOption) {
		o.name = name
	}
}

func CryptoEncryptFunc(opts ...utils.OptionExtender) func(src string) (dst string) {
	o := utils.ApplyOptions[InitOption](opts...)
	opt := utils.ApplyOptions[cryptoConfigOption](opts...)
	optsMap := Use(o.AppName).(*registry).cryptoConfig().ToOptionMap()
	opts = optsMap[opt.name]
	return func(src string) (dst string) {
		return utils.Must(encode.From(src).Encode(opts...).ToString())
	}
}

func CryptoDecryptFunc(opts ...utils.OptionExtender) func(src string) (dst string) {
	o := utils.ApplyOptions[InitOption](opts...)
	opt := utils.ApplyOptions[cryptoConfigOption](opts...)
	optsMap := Use(o.AppName).(*registry).cryptoConfig().ToOptionMap()
	for _, opts := range optsMap {
		utils.SliceReverse(opts)
	}
	opts = optsMap[opt.name]
	return func(src string) (dst string) {
		return utils.Must(encode.From(src).Decode(opts...).ToString())
	}
}

func CryptoDecryptByTag(data any, opts ...utils.OptionExtender) {
	o := utils.ApplyOptions[InitOption](opts...)
	optsMap := Use(o.AppName).(*registry).cryptoConfig().ToOptionMap()
	for _, opts := range optsMap {
		utils.SliceReverse(opts)
	}

	supportedFields := utils.NewSet(reflect.Struct, reflect.Array, reflect.Slice, reflect.Map)
	utils.TraverseValue(data, false, func(field reflect.StructField, value reflect.Value) (end, stepIn bool) {
		if !value.IsValid() || !value.CanInterface() || !value.CanSet() {
			return
		}

		vk := value.Kind()
		stepIn = supportedFields.Contains(vk) ||
			(vk == reflect.Ptr && value.Elem().IsValid() && value.Elem().Kind() == reflect.Struct)

		configName, ok := field.Tag.Lookup(cryptoTagKey)
		if !ok {
			return
		}
		opts, ok := optsMap[configName]
		if !ok {
			return
		}
		src := cast.ToString(value.Interface())
		if utils.IsStrBlank(src) {
			return
		}

		dst := utils.Must(encode.From(src).Decode(opts...).ToString())
		value.SetString(dst)
		return
	})
}
