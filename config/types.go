package config

import (
	"hash/crc64"
	"math/rand"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/cipher"
	"github.com/wfusion/gofusion/common/utils/compress"
	"github.com/wfusion/gofusion/common/utils/encode"
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

type cryptoConfigOption struct {
	name string
}

func CryptoConfigName(name string) utils.OptionFunc[cryptoConfigOption] {
	return func(o *cryptoConfigOption) {
		o.name = name
	}
}

type confType string

const (
	confTypeApollo confType = "apollo"
	confTypeKV     confType = "kv"
)

type kvProvider string

const (
	kvTypeConsul    kvProvider = "consul"
	kvTypeEtcd      kvProvider = "etcd"
	kvTypeEtcd3     kvProvider = "etcd3"
	kvTypeFirestore kvProvider = "firestore"
)

type RemoteConf struct {
	Type   confType   `yaml:"type" json:"type" toml:"type"`
	Apollo ApolloConf `yaml:"apollo" json:"apollo" toml:"apollo"`
	KV     KVConf     `yaml:"kv" json:"kv" toml:"kv"`
	// MustStart can be used to control the first synchronization must succeed
	MustStart bool `yaml:"must_start" json:"must_start" toml:"must_start"`
}

type ApolloConf struct {
	AppID   string `yaml:"app_id" json:"app_id" toml:"app_id"`
	Cluster string `yaml:"cluster" json:"cluster" toml:"cluster" default:"default"`
	// Namespace supports multiple namespaces separated by comma, e.g. application.yaml,db.yaml
	Namespaces        string         `yaml:"namespaces" json:"namespaces" toml:"namespaces" default:"application"`
	Endpoint          string         `yaml:"endpoint" json:"endpoint" toml:"endpoint"`
	IsBackupConfig    bool           `yaml:"is_backup_config" json:"is_backup_config" toml:"is_backup_config"`
	BackupConfigPath  string         `yaml:"backup_config_path" json:"backup_config_path" toml:"backup_config_path" default:"./"`
	Secret            string         `yaml:"secret" json:"secret" toml:"secret"`
	Label             string         `yaml:"label" json:"label" toml:"label"`
	SyncServerTimeout utils.Duration `yaml:"sync_server_timeout" json:"sync_server_timeout" toml:"sync_server_timeout" default:"10s"`
}

type KVConf struct {
	EndPointConfigs []KVEndpointConf `yaml:"endpoint_configs" json:"endpoint_configs" toml:"endpoint_configs"`
}

type KVEndpointConf struct {
	Provider      kvProvider `yaml:"provider" json:"provider" toml:"provider"`
	Endpoints     string     `yaml:"endpoints" json:"endpoints" toml:"endpoints"` // splits with comma
	Path          string     `yaml:"path" json:"path" toml:"path"`
	SecretKeyring string     `yaml:"secret_keyring" json:"secret_keyring" toml:"secret_keyring"`
}
