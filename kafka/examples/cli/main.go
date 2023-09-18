/*
@Date: 2021/11/18 15:24
@Author: max.liu
@File : main
*/

package main

import (
	"context"

	"github.com/common/kafka"
	"github.com/common/logger"
)

var exampleConfig = kafka.Config{
	Addr:         "127.0.0.1:9092,127.0.0.1:9092,127.0.0.1:9092",
	KafkaVersion: "",
	EnableLog:    true,
	LogLevel:     "info",
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli, err := exampleConfig.BuildKafka(ctx)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Info(cli.Address())
	producer, err := cli.NewAsyncProducerClient()
	if err != nil {
		logger.Fatal(err)
	}

	producer.IsRunning()
}
