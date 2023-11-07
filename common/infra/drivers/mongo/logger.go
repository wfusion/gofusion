package mongo

import (
	"go.mongodb.org/mongo-driver/event"

	"github.com/wfusion/gofusion/common/utils"
)

func WithMonitor(monitor *event.CommandMonitor) utils.OptionFunc[newOption] {
	return func(o *newOption) {
		o.monitor = monitor
	}
}

func WithPoolMonitor(monitor *event.PoolMonitor) utils.OptionFunc[newOption] {
	return func(o *newOption) {
		o.poolMonitor = monitor
	}
}
