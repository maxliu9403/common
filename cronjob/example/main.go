package main

import (
	"github.com/common/cronjob"
	"github.com/common/logger"
	"time"
)

type (
	ExampleTask struct {
	}
)

func (e *ExampleTask) Run() {
	logger.Infof("doing")
}

func Task() *ExampleTask {
	return &ExampleTask{}
}

func main() {
	job := cronjob.CronJobs
	_, err := job.AddJob("*/1 * * * *", Task())
	if err != nil {
		logger.Error(err.Error())
		return
	}
	job.Start()
	time.Sleep(time.Second * 120)
}