package config

import (
	"context"
	"path/filepath"
	"strings"
	"sync"

	"github.com/apolloconfig/agollo/v4"
	"github.com/apolloconfig/agollo/v4/env/config"
	"github.com/apolloconfig/agollo/v4/storage"
	"github.com/spf13/cast"
	"github.com/spf13/viper"

	"github.com/wfusion/gofusion/common/constant"
	"github.com/wfusion/gofusion/common/utils"
)

var (
	apolloClientMap    = make(map[string]map[string]agollo.Client) // app - env - client
	apolloClientLocker sync.RWMutex
	changeTypeMapping  = map[storage.ConfigChangeType]ChangeType{
		storage.ADDED:    ChangeTypeAdded,
		storage.MODIFIED: ChangeTypeModified,
		storage.DELETED:  ChangeTypeDeleted,
	}
)

func KeyFormat(namespace string) string {
	return strings.ReplaceAll(namespace, ".", "~~")
}

func newApolloInstance(ctx context.Context, conf *RemoteConf, appName string) (instance RemoteConfigurable, err error) {
	if utils.IsStrBlank(conf.Apollo.AppID) {
		conf.Apollo.AppID = appName
	}
	if utils.IsStrBlank(conf.Apollo.Namespaces) {
		panic(ErrApolloNameSpacesRequired)
	}
	namespaceSplits := strings.Split(conf.Apollo.Namespaces, constant.Comma)
	namespaces := make([]string, 0, len(namespaceSplits))
	for _, namespace := range namespaceSplits {
		namespaces = append(namespaces, strings.TrimSpace(namespace))
	}

	cfg := &config.AppConfig{
		AppID:             conf.Apollo.AppID,
		Cluster:           conf.Apollo.Cluster,
		NamespaceName:     strings.Join(namespaces, constant.Comma),
		IP:                conf.Apollo.Endpoint,
		IsBackupConfig:    conf.Apollo.IsBackupConfig,
		BackupConfigPath:  conf.Apollo.BackupConfigPath,
		Secret:            conf.Apollo.Secret,
		Label:             conf.Apollo.Label,
		SyncServerTimeout: int(utils.Must(utils.ParseDuration(conf.Apollo.SyncServerTimeout)).Seconds()),
		MustStart:         conf.MustStart,
	}

	cli, err := agollo.StartWithConfig(func() (*config.AppConfig, error) { return cfg, nil })
	if err != nil {
		return
	}
	apolloClientLocker.Lock()
	defer apolloClientLocker.Unlock()
	clusterMap, ok := apolloClientMap[appName]
	if !ok {
		clusterMap = make(map[string]agollo.Client)
		apolloClientMap[appName] = clusterMap
	}
	clusterMap[cfg.Cluster] = cli

	vp := viper.New()
	configTypeList := make([]string, 0, len(namespaces))
	for _, namespace := range namespaces {
		if ext := filepath.Ext(namespace); len(ext) > 0 {
			vp.SetConfigType(ext[1:])
			configTypeList = append(configTypeList, ext[1:])
		}
		if err = parseApolloNamespaceContent(cli, vp, namespace); err != nil {
			return
		}
	}

	instance = &safeViper{Viper: vp, configTypeList: configTypeList}
	cli.AddChangeListener(&apolloListener{instance: instance})

	return
}

func releaseApolloConfig(appName string) {
	apolloClientLocker.Lock()
	defer apolloClientLocker.Unlock()
	for _, client := range apolloClientMap[appName] {
		client.Close()
	}
	delete(apolloClientMap, appName)
}

func parseApolloNamespaceContent(cli agollo.Client, vp *viper.Viper, namespace string) (err error) {
	isTxt := strings.HasSuffix(namespace, ".txt")
	isJson := strings.HasSuffix(namespace, ".json")
	if !isTxt && !isJson {
		cli.GetConfig(namespace).GetCache().Range(func(k, v any) bool {
			key := cast.ToString(k)
			vp.Set(key, v)
			return true
		})
		return
	}

	content := cli.GetConfig(namespace).GetContent()
	content = strings.TrimPrefix(content, "content=")
	if isTxt {
		vp.Set(KeyFormat(namespace), content)
		return
	}

	jsonvp := viper.New()
	jsonvp.SetConfigType("json")
	if err = jsonvp.MergeConfig(strings.NewReader(content)); err != nil {
		return
	}
	if err = vp.MergeConfigMap(jsonvp.AllSettings()); err != nil {
		return
	}
	return
}

type apolloListener struct {
	initOnce sync.Once
	instance RemoteConfigurable
}

func (a *apolloListener) OnChange(changeEvent *storage.ChangeEvent) {
	if changeEvent == nil {
		return
	}

	defer func() {
		evt := &ChangeEvent{Changes: make(map[string]*Change, len(changeEvent.Changes))}
		for k, v := range changeEvent.Changes {
			evt.Changes[k] = &Change{
				OldValue:   v.OldValue,
				NewValue:   v.NewValue,
				ChangeType: changeTypeMapping[v.ChangeType],
			}
		}
		a.instance.pushChangeEvent(evt)
	}()

	namespace := changeEvent.Namespace
	isTxt := strings.HasSuffix(namespace, ".txt")
	isJson := strings.HasSuffix(namespace, ".json")
	if !isTxt && !isJson {
		for key, change := range changeEvent.Changes {
			switch change.ChangeType {
			case storage.ADDED, storage.MODIFIED:
				a.instance.Set(key, change.NewValue)
			case storage.DELETED:
				if a.instance.Get(key) != nil {
					a.instance.Set(key, nil)
				}
			}
		}
		return
	}

	for _, change := range utils.MapValues(changeEvent.Changes) {
		content := strings.TrimPrefix(cast.ToString(change.NewValue), "content=")
		switch change.ChangeType {
		case storage.ADDED, storage.MODIFIED:
			if isTxt {
				a.instance.Set(KeyFormat(namespace), content)
				return
			}
			jsonvp := viper.New()
			jsonvp.SetConfigType("json")
			_ = jsonvp.MergeConfig(strings.NewReader(content))
			_ = a.instance.MergeConfigMap(jsonvp.AllSettings())
		case storage.DELETED:
			if isTxt {
				txtKey := KeyFormat(namespace)
				if a.instance.Get(txtKey) != nil {
					a.instance.Set(txtKey, nil)
				}
				return
			}

			content = strings.TrimPrefix(cast.ToString(change.OldValue), "content=")
			content = strings.TrimSpace(content)

			jsonvp := viper.New()
			jsonvp.SetConfigType("json")
			_ = jsonvp.MergeConfig(strings.NewReader(content))
			for k := range jsonvp.AllSettings() {
				if a.instance.Get(k) != nil {
					a.instance.Set(k, nil)
				}
			}
		}
	}
}

// OnNewestChange provides full config change event before OnChange
func (a *apolloListener) OnNewestChange(event *storage.FullChangeEvent) {
}
