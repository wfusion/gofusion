package customlogger

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/event"

	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/config"
	"github.com/wfusion/gofusion/log"
)

var (
	// MongoLoggerType FIXME: should not be deleted to avoid compiler optimized
	MongoLoggerType = reflect.TypeOf(mongoLogger{})
)

type mongoLogger struct {
	*event.CommandMonitor

	log               log.Logable
	enabled           bool
	appName           string
	confName          string
	logableCommandSet *utils.Set[string]

	mutex sync.RWMutex
	// requestMap The key is the monotonically atomic incrementing RequestID from the Mongo command event,
	// used to associate the command with the duration and only print one line of log.
	requestMap map[int64]struct {
		commandString string
	}
}

func DefaultMongoLogger() (logger *mongoLogger) {
	logger = &mongoLogger{
		enabled:    true,
		requestMap: map[int64]struct{ commandString string }{},
		logableCommandSet: utils.NewSet[string](
			"ping",
			"insert",
			"find",
			"update",
			"delete",
			"aggregate",
			"distinct",
			"count",
			"findAndModify",
			"getMore",
			"killCursors",
			"create",
			"drop",
			"listDatabases",
			"dropDatabase",
			"createIndexes",
			"listIndexes",
			"dropIndexes",
			"listCollections",
		),
	}
	logger.CommandMonitor = &event.CommandMonitor{
		Started:   logger.started,
		Succeeded: logger.succeeded,
		Failed:    logger.failed,
	}
	return
}

func (m *mongoLogger) Init(log log.Logable, appName, name string) {
	m.log = log
	m.appName = appName
	m.confName = name
}

func (m *mongoLogger) GetMonitor() *event.CommandMonitor {
	return &event.CommandMonitor{
		Started:   m.started,
		Succeeded: m.succeeded,
		Failed:    m.failed,
	}
}

func (m *mongoLogger) started(ctx context.Context, evt *event.CommandStartedEvent) {
	if !m.isLoggableCommandName(evt.CommandName) {
		return
	}
	m.pushCommandString(evt.RequestID, evt.Command.String())
}

func (m *mongoLogger) succeeded(ctx context.Context, evt *event.CommandSucceededEvent) {
	if !m.isLoggableCommandName(evt.CommandName) {
		return
	}
	if m.log != nil {
		m.log.Info(ctx, "[mongodb] %s succeeded [request[%v] command[%s]]",
			evt.CommandName, evt.RequestID, m.popCommandString(evt.RequestID),
			log.Fields{"latency": int64(evt.Duration) / int64(time.Millisecond)})
	} else {
		log.Info(ctx, "[mongodb] %s succeeded [request[%v] command[%s]]",
			evt.CommandName, evt.RequestID, m.popCommandString(evt.RequestID),
			log.Fields{"latency": int64(evt.Duration) / int64(time.Millisecond)})
	}
}

func (m *mongoLogger) failed(ctx context.Context, evt *event.CommandFailedEvent) {
	if !m.isLoggableCommandName(evt.CommandName) {
		return
	}
	if m.log != nil {
		m.log.Warn(ctx, "[mongodb] %s failed [request[%v] command[%s]]",
			evt.CommandName, evt.RequestID, m.popCommandString(evt.RequestID),
			log.Fields{"latency": int64(evt.Duration) / int64(time.Millisecond)})
	} else {
		log.Warn(ctx, "[mongodb] %s failed [request[%v] command[%s]]",
			evt.CommandName, evt.RequestID, m.popCommandString(evt.RequestID),
			log.Fields{"latency": int64(evt.Duration) / int64(time.Millisecond)})
	}
}

func (m *mongoLogger) pushCommandString(requestID int64, commandString string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.requestMap[requestID] = struct{ commandString string }{commandString: commandString}
}

func (m *mongoLogger) popCommandString(requestID int64) string {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	reqCtx, ok := m.requestMap[requestID]
	if ok {
		delete(m.requestMap, requestID)
		return reqCtx.commandString
	}
	return ""
}

func (m *mongoLogger) isLoggableCommandName(commandName string) bool {
	m.reloadConfig()
	if !m.enabled {
		return false
	}
	return m.logableCommandSet.Contains(commandName)
}

func (m *mongoLogger) reloadConfig() {
	var cfgs map[string]map[string]any
	_ = config.Use(m.appName).LoadComponentConfig(config.ComponentMongo, &cfgs)
	if len(cfgs) == 0 {
		return
	}

	cfg, ok := cfgs[m.confName]
	if !ok {
		return
	}
	m.enabled = cast.ToBool(cfg["enable_logger"])
	logConfigObj, ok1 := cfg["logger_config"]
	logCfg, ok2 := logConfigObj.(map[string]any)
	if !ok1 || !ok2 {
		return
	}
	logableCommandList, ok := logCfg["logable_commands"].([]string)
	if !ok {
		return
	}

	m.logableCommandSet = utils.NewSet(logableCommandList...)
}
