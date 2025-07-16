package config

import (
	"context"
	"sync"
	"time"

	"github.com/wfusion/gofusion/common/utils"
)

const (
	ErrUnsupportedConfigType    utils.Error = "unsupported config type"
	ErrDuplicatedConfigName     utils.Error = "duplicated config name"
	ErrApolloNameSpacesRequired utils.Error = "apollo namespaces required"
	ErrUnsupportedKVType        utils.Error = "unsupported kv type"
)

type RemoteConfigurable interface {
	Set(key string, value any)
	Get(key string) any
	GetString(key string) string
	GetBool(key string) bool
	GetInt(key string) int
	GetInt64(key string) int64
	GetFloat64(key string) float64
	GetStringSlice(key string) []string
	GetTime(key string) time.Time
	Unmarshal(rawVal any) error                // mapstructure decode
	UnmarshalKey(key string, rawVal any) error // mapstructure decode
	GetDuration(key string) time.Duration
	GetAllSettings() map[string]any
	OnConfigChange(run func(evt *ChangeEvent))
	MergeConfigMap(cfg map[string]any) (err error)

	getConfigType() (tag string)
	pushChangeEvent(evt *ChangeEvent)
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

var (
	appRemoteConfigs      = make(map[string]map[string]RemoteConfigurable)
	appRemoteConfigLocker sync.RWMutex
)

type RemoteConf struct {
	Type   confType   `yaml:"type" json:"type" toml:"type"`
	Apollo ApolloConf `yaml:"apollo" json:"apollo" toml:"apollo"`
	KV     KVConf     `yaml:"kv" json:"kv" toml:"kv"`
}

type ApolloConf struct {
	AppID   string `yaml:"app_id" json:"app_id" toml:"app_id"`
	Cluster string `yaml:"cluster" json:"cluster" toml:"cluster" default:"default"`
	// Namespace supports multiple namespaces separated by comma, e.g. application.yaml,db.yaml
	Namespaces        string `yaml:"namespaces" json:"namespaces" toml:"namespaces" default:"application.yaml"`
	Endpoint          string `yaml:"endpoint" json:"endpoint" toml:"endpoint"`
	IsBackupConfig    bool   `yaml:"is_backup_config" json:"is_backup_config" toml:"is_backup_config" default:"true"`
	BackupConfigPath  string `yaml:"backup_config_path" json:"backup_config_path" toml:"backup_config_path" default:"./"`
	Secret            string `yaml:"secret" json:"secret" toml:"secret"`
	Label             string `yaml:"label" json:"label" toml:"label"`
	SyncServerTimeout string `yaml:"sync_server_timeout" json:"sync_server_timeout" toml:"sync_server_timeout" default:"10s"`
	// MustStart can be used to control the first synchronization must succeed
	MustStart bool `yaml:"must_start" json:"must_start" toml:"must_start" default:"true"`
}

type KVConf struct {
	Endpoints KVEndpointConfList `yaml:"endpoints" json:"endpoints" toml:"endpoints"`
	MustStart bool               `yaml:"must_start" json:"must_start" toml:"must_start" default:"true"`
}

type KVEndpointConf struct {
	Provider  kvProvider `yaml:"provider" json:"provider" toml:"provider"`
	Endpoints string     `yaml:"endpoints" json:"endpoints" toml:"endpoints"` // splits with comma
	Path      string     `yaml:"path" json:"path" toml:"path"`
	Secret    string     `yaml:"secret" json:"secret" toml:"secret"`
	MustStart bool       `yaml:"must_start" json:"must_start" toml:"must_start" default:"true"`
}

type KVEndpointConfList []*KVEndpointConf

func RemoteConstruct(ctx context.Context, confs map[string]*RemoteConf, opts ...utils.OptionExtender) func() {
	opt := utils.ApplyOptions[InitOption](opts...)
	for name, cfg := range confs {
		addConfigInstance(ctx, name, cfg, opt)
	}

	return func() {
		appRemoteConfigLocker.Lock()
		defer appRemoteConfigLocker.Unlock()

		//pid := syscall.Getpid()
		for appName := range appRemoteConfigs {
			releaseApolloConfig(appName)
			delete(appRemoteConfigs, appName)
			//log.Printf("%v [Gofusion] %s %s close error: %s", pid, appName, ComponentRemoteConfig, err)
		}
	}
}

func Remote(name string, opts ...utils.OptionExtender) RemoteConfigurable {
	opt := utils.ApplyOptions[InitOption](opts...)

	appRemoteConfigLocker.RLock()
	defer appRemoteConfigLocker.RUnlock()
	if appRemoteConfigs == nil {
		return nil
	}
	if appRemoteConfigs[opt.AppName] == nil {
		return nil
	}
	return appRemoteConfigs[opt.AppName][name]
}

func addConfigInstance(ctx context.Context, name string, conf *RemoteConf, opt *InitOption) {
	var instance RemoteConfigurable
	switch conf.Type {
	case confTypeApollo:
		instance = utils.Must(newApolloInstance(ctx, &conf.Apollo, opt.AppName))
	case confTypeKV:
		instance = utils.Must(newKVInstance(ctx, name, &conf.KV, opt.AppName))
	default:
		panic(ErrUnsupportedConfigType)
	}

	appRemoteConfigLocker.Lock()
	defer appRemoteConfigLocker.Unlock()

	if appRemoteConfigs == nil {
		appRemoteConfigs = make(map[string]map[string]RemoteConfigurable)
	}
	if appRemoteConfigs[opt.AppName] == nil {
		appRemoteConfigs[opt.AppName] = make(map[string]RemoteConfigurable)
	}

	if _, ok := appRemoteConfigs[opt.AppName][name]; ok {
		panic(ErrDuplicatedConfigName)
	}

	appRemoteConfigs[opt.AppName][name] = instance
}
