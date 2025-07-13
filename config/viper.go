package config

import (
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// safeViper is a wrapper for viper.Viper that provides thread-safe access to the underlying viper instance.
// viper.Viper are not safe for concurrent Get() and Set() operations in its notes.
type safeViper struct {
	*viper.Viper
	lock      sync.RWMutex
	watchOnce sync.Once
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

// Event represents a file system notification.
type Event struct {
	// Path to the file or directory.
	//
	// Paths are relative to the input; for example with Add("dir") the Name
	// will be set to "dir/file" if you create that file, but if you use
	// Add("/path/to/dir") it will be "/path/to/dir/file".
	Name string

	// File operation that triggered the event.
	//
	// This is a bitmask and some systems may send multiple operations at once.
	// Use the Event.Has() method instead of comparing with ==.
	Op Op
}

// Op describes a set of file operations.
type Op uint32

// The operations fsnotify can trigger; see the documentation on [Watcher] for a
// full description, and check them with [Event.Has].
const (
	Create Op = 1 << iota
	Write
	Remove
	Rename
	Chmod
)

func (s *safeViper) OnConfigChange(run func(in Event)) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.watchOnce.Do(func() {
		s.Viper.WatchConfig()
	})
	s.Viper.OnConfigChange(func(in fsnotify.Event) {
		run(Event{Op: Op(in.Op), Name: in.Name})
	})
}

func (s *safeViper) MergeConfigMap(cfg map[string]any) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.Viper.MergeConfigMap(cfg)
}
