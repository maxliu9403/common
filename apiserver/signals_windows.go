//go:build windows
// +build windows

/*
@Date: 2021/11/10 16:04
@Author: max.liu
@File : signals
*/

package apiserver

import (
	"os"
	"os/signal"

	"github.com/maxliu9403/common/logger"
)

const timeFormat = "0102150405"

var done = make(chan struct{})

func init() {
	go func() {
		// Windows 不支持 SIGUSR1 和 SIGTERM，只监听 os.Interrupt (Ctrl+C)
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt)

		for {
			v := <-signals
			switch v {
			case os.Interrupt:
				select {
				case <-done:
					// already closed
				default:
					close(done)
				}

				gracefulStop(signals)
			default:
				logger.Errorf("got unregistered signal: %+v", v)
			}
		}
	}()
}

