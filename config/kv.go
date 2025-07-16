package config

import (
	"context"
	"path/filepath"
	"reflect"

	"github.com/spf13/viper"

	"github.com/wfusion/gofusion/common/utils"

	_ "github.com/spf13/viper/remote"
)

func newKVInstance(ctx context.Context, name string, cfg *KVConf, appName string) (instance RemoteConfigurable, err error) {
	vp := viper.New()

	configTypeList := make([]string, 0, len(cfg.Endpoints))
	for _, conf := range cfg.Endpoints {
		if utils.IsStrBlank(conf.Secret) {
			err = vp.AddRemoteProvider(string(conf.Provider), conf.Endpoints, conf.Path)
		} else {
			err = vp.AddSecureRemoteProvider(string(conf.Provider), conf.Endpoints, conf.Path, conf.Secret)
		}
		if err != nil {
			return
		}
		if ext := filepath.Ext(conf.Path); len(ext) > 0 {
			vp.SetConfigType(ext[1:])
			configTypeList = append(configTypeList, ext[1:])
		}
	}
	if cfg.MustStart {
		if err = vp.ReadRemoteConfig(); err != nil {
			return
		}
	}

	instance = &safeViper{Viper: vp, configTypeList: configTypeList}
	v := reflect.ValueOf(vp)
	m := v.MethodByName("unmarshalReader")
	kvstore := utils.IndirectValue(v).FieldByName("kvstore")

	viper.RemoteConfig = &remoteConfigProvider{
		key:                   KeyFormat(name),
		listener:              instance,
		viperValue:            v,
		unmarshalReaderMethod: m,
		kvStoreValue:          kvstore,
	}
	if err = vp.WatchRemoteConfigOnChannel(); err != nil {
		return
	}

	return
}
