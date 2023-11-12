package context

import "errors"

const (
	KeyLangs        = "base:langs"
	KeyUserID       = "base:user_id"
	KeyTraceID      = "base:trace_id"
	KeyLoggable     = "base:loggable"
	KeyLogFields    = "base:log_fields"
	KeyGormDB       = "base:gorm_db"
	KeyDALOption    = "base:dal_option"
	KeyCronTaskID   = "base:cron_task_id"
	KeyCronTaskName = "base:cron_task_name"
)

var (
	ErrUnknownInstantiationMethod = errors.New("unknown instantiation method")
)
