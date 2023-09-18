/*
@Date: 2021/12/17 13:41
@Author: max.liu
@File : config_test
*/

package rediscache

import (
	"context"
	"testing"
)

var redisConf = Config{
	Addr:       "127.0.0.1:6379",
	Password:   "Password",
	DB:         1,
	ServerType: "standalone",
}

func TestNewRedisCli(t *testing.T) {
	err := redisConf.NewRedisCli(context.Background())
	if err != nil {
		t.Fatal(err.Error())
	}

	cli := GetCli()
	if cli == nil {
		t.Fatal("redis cli is nil")
	}

	t.Log(cli.Ping(context.Background()).String())
}
