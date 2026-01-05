//go:build !windows
// +build !windows

/*
@Date: 2021/11/10 16:12
@Author: max.liu
@File : shutdown
*/

package apiserver

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/maxliu9403/common/logger"
)

func gracefulStop(signals chan os.Signal) {
	signal.Stop(signals)

	logger.Info("got signal SIGTERM, shutting down...")
	wrapUpListeners.notifyListeners()

	time.Sleep(wrapUpTime)
	shutdownListeners.notifyListeners()

	time.Sleep(delayTimeBeforeForceQuit - wrapUpTime)
	logger.Infof("still alive after %v, going to force kill the process...", delayTimeBeforeForceQuit)
	_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
}

