package routine

import (
	"strconv"
	"sync"
	"time"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/log"
)

const (
	ErrTimeout        utils.Error = "operation timed out"
	ErrPoolOverload   utils.Error = "too many goroutines blocked on submit or Nonblocking is set"
	ErrDuplicatedName utils.Error = "duplicated goroutine pool name"
)

var (
	// wg 用于简易协程的优雅退出
	wg       sync.WaitGroup
	locker   sync.RWMutex
	routines = map[string]map[string]int{}
)

type Pool interface {
	Submit(task any, opts ...utils.OptionExtender) error
	Running() int
	Free() int
	Waiting() int
	Cap() int
	IsClosed() bool
	Release(opts ...utils.OptionExtender)
	ReleaseTimeout(timeout time.Duration, opts ...utils.OptionExtender) error
}

// Conf routine configure
//nolint: revive // struct tag too long issue
type Conf struct {
	// MaxGoroutineAmount 最大协程数量
	MaxRoutineAmount int `yaml:"max_routine_amount" json:"max_routine_amount" toml:"max_routine_amount" default:"-1"`

	// MaxReleaseTimePerPool 优雅退出时单个 pool 最大等待时间
	MaxReleaseTimePerPool string `yaml:"max_release_time_per_pool" json:"max_release_time_per_pool" toml:"max_release_time_per_pool" default:"30s"`

	// ForceSync will synchronously execute Go, promise function if true
	ForceSync bool `yaml:"force_sync" json:"force_sync" toml:"force_sync" default:"false"`

	// Logger is the customized logger for logging info, if it is not set,
	// default standard logger from log package is used.
	EnabledLogger bool   `yaml:"enable_logger" json:"enable_logger" toml:"enable_logger" default:"false"`
	Logger        string `yaml:"logger" json:"logger" toml:"logger" default:"github.com/wfusion/gofusion/log/customlogger.routineLogger"`
	LogInstance   string `yaml:"log_instance" json:"log_instance" toml:"log_instance" default:"default"`
}

func addRoutine(appName, name string) {
	locker.Lock()
	defer locker.Unlock()

	if _, ok := routines[appName]; !ok {
		routines[appName] = make(map[string]int)
	}
	routines[appName][name]++
}

func delRoutine(appName, name string) {
	locker.Lock()
	defer locker.Unlock()
	if routines == nil || routines[appName] == nil {
		return
	}

	if routines[appName][name]--; routines[appName][name] <= 0 {
		delete(routines, name)
	}
}

func showRoutine(appName string) (r []string) {
	locker.RLock()
	defer locker.RUnlock()
	r = make([]string, 0, len(routines[appName]))
	for n, c := range routines[appName] {
		r = append(r, n+":"+strconv.Itoa(c))
	}
	return
}

type customLogger interface {
	Init(log log.Logable, appName string)
}
