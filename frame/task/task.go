package task

import (
	"reflect"

	"github.com/robfig/cron"
	"wataru.com/gogo/logger"
)

type Task interface {
	Process()
}

var c = cron.New()

func NewTask(cron string, t Task) {
	c.AddFunc("0/10 * * * * *", func() {
		defer func() {
			logger.Info("==== Task [%s] finished ====", reflect.TypeOf(t))
		}()
		logger.Info("==== Task [%s] start ====", reflect.TypeOf(t))
		t.Process()
	})
}

func StartTaskSchedule() {
	c.Start()
}
