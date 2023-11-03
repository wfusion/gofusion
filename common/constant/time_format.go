package constant

import (
	"time"
)

const (
	StdDateLayout      = "2006-01-02"
	StdTimeLayout      = "2006-01-02 15:04:05"
	StdTimeWithZLayout = "2006-01-02T15:04:05Z"
	StdTimeMSLayout    = "2006-01-02 15:04:05.999999"
)

const (
	DefaultTimezone = "Asia/Shanghai"
)

func DefaultLocation() *time.Location {
	loc, err := time.LoadLocation(DefaultTimezone)
	if err != nil {
		panic(err)
	}
	return loc
}
