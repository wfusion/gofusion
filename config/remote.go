package config

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/wfusion/gofusion/common/di"
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

var (
	appRemoteConfigs      = make(map[string]map[string]RemoteConfigurable)
	appRemoteConfigLocker sync.RWMutex
)

func RemoteConstruct(ctx context.Context, confs map[string]*RemoteConf, opts ...utils.OptionExtender) func() {
	opt := utils.ApplyOptions[InitOption](opts...)
	for name, cfg := range confs {
		addRemoteConfigInstance(ctx, name, cfg, opt)
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

// RemoteDefaultKeyFormat format default key for RemoteConfigurable.GetAllSettings
// for apollo type it is namespace
// for kv type it is config name
func RemoteDefaultKeyFormat(nameOrNamespace string) string {
	return strings.ReplaceAll(nameOrNamespace, ".", "~~")
}

func addRemoteConfigInstance(ctx context.Context, name string, conf *RemoteConf, opt *InitOption) {
	var instance RemoteConfigurable
	switch conf.Type {
	case confTypeApollo:
		instance = utils.Must(newApolloInstance(ctx, conf, opt.AppName))
	case confTypeKV:
		instance = utils.Must(newKVInstance(ctx, name, conf, opt.AppName))
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

	if opt.DI != nil {
		opt.DI.MustProvide(func() RemoteConfigurable { return Remote(name, AppName(opt.AppName)) }, di.Name(name))
	}
	if opt.App != nil {
		opt.App.MustProvide(
			func() RemoteConfigurable { return Remote(name, AppName(opt.AppName)) },
			di.Name(name),
		)
	}

	// TODO: metric remote configuration center latency
}
