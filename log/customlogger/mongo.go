package customlogger

import (
	"context"
	"reflect"
	"strings"
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
	mongoFields     = log.Fields{"component": strings.ToLower(config.ComponentMongo)}
)

type mongoLogger struct {
	*event.CommandMonitor

	log                log.Loggable
	enabled            bool
	appName            string
	confName           string
	loggableCommandSet *utils.Set[string]
	traceMonitor       *event.CommandMonitor

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
		requestMap: make(map[int64]struct{ commandString string }),
		loggableCommandSet: utils.NewSet[string](
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

func (m *mongoLogger) Init(log log.Loggable, appName, name string) {
	m.log = log
	m.appName = appName
	m.confName = name
	m.requestMap = make(map[int64]struct{ commandString string })
	m.loggableCommandSet = utils.NewSet[string](
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
	)
}

func (m *mongoLogger) GetMonitor() *event.CommandMonitor {
	return &event.CommandMonitor{
		Started:   m.started,
		Succeeded: m.succeeded,
		Failed:    m.failed,
	}
}

func (m *mongoLogger) SetTraceMonitor(traceMonitor *event.CommandMonitor) {
	if m == nil {
		return
	}
	m.traceMonitor = traceMonitor
}

func (m *mongoLogger) started(ctx context.Context, evt *event.CommandStartedEvent) {
	if !m.isLoggableCommandName(evt.CommandName) {
		return
	}
	m.pushCommandString(evt.RequestID, evt.Command.String())
	if m.traceMonitor != nil {
		m.traceMonitor.Started(ctx, evt)
	}
}

func (m *mongoLogger) succeeded(ctx context.Context, evt *event.CommandSucceededEvent) {
	if !m.isLoggableCommandName(evt.CommandName) {
		return
	}
	m.logger().Info(ctx, "%s succeeded: %s [request[%v] command[%s]]",
		evt.CommandName, evt.Reply, evt.RequestID, m.popCommandString(evt.RequestID),
		m.fields(log.Fields{"latency": int64(evt.Duration) / int64(time.Millisecond)}))
	if m.traceMonitor != nil {
		m.traceMonitor.Succeeded(ctx, evt)
	}
}

func (m *mongoLogger) failed(ctx context.Context, evt *event.CommandFailedEvent) {
	if !m.isLoggableCommandName(evt.CommandName) {
		return
	}

	m.logger().Warn(ctx, "%s failed: %s [request[%v] command[%s]]",
		evt.CommandName, evt.Failure, evt.RequestID, m.popCommandString(evt.RequestID),
		m.fields(log.Fields{"latency": int64(evt.Duration) / int64(time.Millisecond)}))
	if m.traceMonitor != nil {
		m.traceMonitor.Failed(ctx, evt)
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

func (m *mongoLogger) logger() log.Loggable {
	if m.log != nil {
		return m.log
	}
	return log.Use(config.DefaultInstanceKey, log.AppName(m.appName))
}

func (m *mongoLogger) fields(fields log.Fields) log.Fields {
	return utils.MapMerge(fields, mongoFields)
}

func (m *mongoLogger) isLoggableCommandName(commandName string) bool {
	if m.confName == "" {
		return true
	}
	if m.reloadConfig(); !m.enabled {
		return false
	}
	return m.loggableCommandSet.Contains(commandName)
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
	loggableCommandList, ok := logCfg["loggable_commands"].([]string)
	if !ok {
		return
	}

	m.loggableCommandSet = utils.NewSet(loggableCommandList...)
}
