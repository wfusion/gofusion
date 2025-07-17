package config

import (
	"context"
	"path/filepath"
	"reflect"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/wfusion/gofusion/common/utils"

	_ "github.com/spf13/viper/remote"
)

func newKVInstance(ctx context.Context, name string, conf *RemoteConf, appName string) (
	instance RemoteConfigurable, err error) {
	vp := viper.New()

	defaultType := false
	configTypeList := make([]string, 0, len(conf.KV.EndPointConfigs))
	for _, cfg := range conf.KV.EndPointConfigs {
		if utils.IsStrBlank(cfg.SecretKeyring) {
			err = vp.AddRemoteProvider(string(cfg.Provider), cfg.Endpoints, cfg.Path)
		} else {
			err = vp.AddSecureRemoteProvider(string(cfg.Provider), cfg.Endpoints, cfg.Path, cfg.SecretKeyring)
		}
		if err != nil {
			return
		}
		if ext := filepath.Ext(cfg.Path); len(ext) > 0 {
			vp.SetConfigType(ext[1:])
			configTypeList = append(configTypeList, ext[1:])
		} else {
			defaultType = true
			vp.SetConfigType("properties")
		}
	}
	if conf.MustStart {
		if err = vp.ReadRemoteConfig(); err != nil {
			return
		}
	}

	sv := &safeViper{Viper: vp, configTypeList: configTypeList}
	kvStoreValue := utils.IndirectValue(reflect.ValueOf(vp)).FieldByName("kvstore")
	if !kvStoreValue.IsValid() {
		panic(errors.New("viper.Viper.kvstore field is invalid"))
	}

	viper.RemoteConfig = &remoteConfigProvider{
		name:        name,
		appName:     appName,
		key:         RemoteDefaultKeyFormat(name),
		listener:    sv,
		defaultType: defaultType,
	}
	if err = vp.WatchRemoteConfigOnChannel(); err != nil {
		return
	}

	instance = sv
	return
}
