package config

import (
	"bytes"
	"io"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/sagikazarmark/crypt/config"
	"github.com/spf13/viper"
	"github.com/wfusion/gofusion/common/utils"
)

type ChangeEvent struct {
	Changes map[string]*Change
}

type Change struct {
	OldValue   interface{}
	NewValue   interface{}
	ChangeType ChangeType
}

type ChangeType int

const (
	ChangeTypeAdded ChangeType = 1 + iota
	ChangeTypeModified
	ChangeTypeDeleted
)

// safeViper is a wrapper for viper.Viper that provides thread-safe access to the underlying viper instance.
// viper.Viper are not safe for concurrent Get() and Set() operations in its notes.
type safeViper struct {
	*viper.Viper
	lock           sync.RWMutex
	watchOnce      sync.Once
	configTypeList []string

	onConfigChangeList   []func(evt *ChangeEvent)
	remoteConfigProvider *remoteConfigProvider
}

func (s *safeViper) Set(key string, value any) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.Viper.Set(key, value)
}

func (s *safeViper) Get(key string) any {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.Viper.Get(key)
}

func (s *safeViper) GetString(key string) string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.Viper.GetString(key)
}

func (s *safeViper) GetBool(key string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.Viper.GetBool(key)
}

func (s *safeViper) GetInt(key string) int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.Viper.GetInt(key)
}

func (s *safeViper) GetInt64(key string) int64 {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.Viper.GetInt64(key)
}

func (s *safeViper) GetFloat64(key string) float64 {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.Viper.GetFloat64(key)
}

func (s *safeViper) GetStringSlice(key string) []string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.Viper.GetStringSlice(key)
}

func (s *safeViper) GetTime(key string) time.Time {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.Viper.GetTime(key)
}

func (s *safeViper) GetDuration(key string) time.Duration {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.Viper.GetDuration(key)
}

func (s *safeViper) GetAllSettings() map[string]any {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.Viper.AllSettings()
}

func (s *safeViper) Unmarshal(rawVal any) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.Viper.Unmarshal(rawVal)
}

func (s *safeViper) UnmarshalKey(key string, rawVal any) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.Viper.UnmarshalKey(key, rawVal)
}

func (s *safeViper) getConfigType() (tag string) {
	for _, configType := range s.configTypeList {
		return configType
	}
	return
}

func (s *safeViper) OnConfigChange(run func(*ChangeEvent)) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.onConfigChangeList = append(s.onConfigChangeList, run)
}

func (s *safeViper) MergeConfigMap(cfg map[string]any) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.Viper.MergeConfigMap(cfg)
}

func (s *safeViper) pushChangeEvent(evt *ChangeEvent) {
	s.lock.Lock()
	defer s.lock.Unlock()
	for _, run := range s.onConfigChangeList {
		go func(callback func(*ChangeEvent)) {
			_, _ = utils.Catch(func() { callback(evt) })
		}(run)
	}
}

// remoteConfigProvider
// fork from github.com/spf13/viper@v1.16.0/remote/remote.go
// with https://github.com/sagikazarmark/crypt@v1.12.0
type remoteConfigProvider struct {
	key                   string
	listener              RemoteConfigurable
	viperValue            reflect.Value
	kvStoreValue          reflect.Value
	unmarshalReaderMethod reflect.Value
}

func (r *remoteConfigProvider) Get(rp viper.RemoteProvider) (io.Reader, error) {
	cm, err := getConfigManager(rp)
	if err != nil {
		return nil, err
	}
	b, err := cm.Get(rp.Path())
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

func (r *remoteConfigProvider) Watch(rp viper.RemoteProvider) (io.Reader, error) {
	cm, err := getConfigManager(rp)
	if err != nil {
		return nil, err
	}
	resp, err := cm.Get(rp.Path())
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(resp), nil
}

func (r *remoteConfigProvider) WatchChannel(rp viper.RemoteProvider) (<-chan *viper.RemoteResponse, chan bool) {
	cm, err := getConfigManager(rp)
	if err != nil {
		return nil, nil
	}
	quit := make(chan bool)
	quitwc := make(chan bool)
	viperResponsCh := make(chan *viper.RemoteResponse)
	cryptoResponseCh := cm.Watch(rp.Path(), quit)
	// need this function to convert the Channel response form crypt.Response to viper.Response
	go func(cr <-chan *config.Response, vr chan<- *viper.RemoteResponse, quitwc <-chan bool, quit chan<- bool) {
		for {
			select {
			case <-quitwc:
				quit <- true
				return
			case resp := <-cr:
				changes := &viper.RemoteResponse{
					Error: resp.Error,
					Value: resp.Value,
				}
				// Call this function in advance, so that when the notification is triggered,
				// the latest changes can be obtained.
				reader := bytes.NewReader(changes.Value)
				r.unmarshalReaderMethod.Call([]reflect.Value{reflect.ValueOf(reader), r.kvStoreValue})
				r.listener.pushChangeEvent(&ChangeEvent{
					Changes: map[string]*Change{
						r.key: {
							OldValue:   nil,
							NewValue:   changes.Value,
							ChangeType: ChangeTypeModified,
						},
					}},
				)
			}
		}
	}(cryptoResponseCh, viperResponsCh, quitwc, quit)

	return viperResponsCh, quitwc
}

func getConfigManager(rp viper.RemoteProvider) (config.ConfigManager, error) {
	var cm config.ConfigManager
	var err error

	endpoints := strings.Split(rp.Endpoint(), ",")
	if rp.SecretKeyring() != "" {
		var kr *os.File
		kr, err = os.Open(rp.SecretKeyring())
		if err != nil {
			return nil, err
		}
		defer utils.CloseAnyway(kr)
		switch kvProvider(rp.Provider()) {
		case kvTypeEtcd:
			cm, err = config.NewEtcdConfigManager(endpoints, kr)
		case kvTypeEtcd3:
			cm, err = config.NewEtcdV3ConfigManager(endpoints, kr)
		case kvTypeFirestore:
			cm, err = config.NewFirestoreConfigManager(endpoints, kr)
		case kvTypeConsul:
			cm, err = config.NewConsulConfigManager(endpoints, kr)
		default:
			err = ErrUnsupportedKVType
		}
	} else {
		switch kvProvider(rp.Provider()) {
		case kvTypeEtcd:
			cm, err = config.NewStandardEtcdConfigManager(endpoints)
		case kvTypeEtcd3:
			cm, err = config.NewStandardEtcdV3ConfigManager(endpoints)
		case kvTypeFirestore:
			cm, err = config.NewStandardFirestoreConfigManager(endpoints)
		case kvTypeConsul:
			cm, err = config.NewStandardConsulConfigManager(endpoints)
		default:
			err = ErrUnsupportedKVType
		}
	}
	if err != nil {
		return nil, err
	}
	return cm, nil
}

func init() {
	viper.RemoteConfig = &remoteConfigProvider{}
}
