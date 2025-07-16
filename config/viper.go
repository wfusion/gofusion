package config

import (
	"sync"
	"time"

	"github.com/spf13/viper"

	"github.com/wfusion/gofusion/common/utils"
)

// safeViper is a wrapper for viper.Viper that provides thread-safe access to the underlying viper instance.
// viper.Viper are not safe for concurrent Get() and Set() operations in its notes.
type safeViper struct {
	*viper.Viper
	lock           sync.RWMutex
	watchOnce      sync.Once
	configTypeList []string

	onConfigChangeList []func(evt *ChangeEvent)
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
	ADDED ChangeType = 1 + iota
	MODIFIED
	DELETED
)
