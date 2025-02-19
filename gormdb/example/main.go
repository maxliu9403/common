/*
@Date: 2021/10/29 11:15
@Author: max.liu
@File : main
*/

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/maxliu9403/common/gormdb"
	"github.com/maxliu9403/common/logger"
	"gorm.io/gorm"
)

var testConfig = gormdb.DBConfig{
	WriteDBHost:     "localhost",
	WriteDBPort:     3306,
	WriteDBUser:     "root",
	WriteDBPassword: "root",
	WriteDB:         "gorm",
	ReadDBHostList:  []string{"localhost"},
	ReadDBPort:      3306,
	ReadDBUser:      "root",
	ReadDBPassword:  "root",
	ReadDB:          "gorm",
	Prefix:          "tbl_",
	MaxIdleConns:    10,
	MaxOpenConns:    100,
	LogLevel:        "info",
	Logging:         true,
	ConnMaxLifetime: 3,
}

type User struct {
	gorm.Model
	Name string
}

func main() {
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())

	_, err := testConfig.BuildMySQLClient(ctx)
	if err != nil {
		logger.Fatal(err)
	}

	err = gormdb.GetDB().Migration(&User{})
	if err != nil {
		logger.Fatal(err)
	}

	var user User
	db := gormdb.Cli(ctx)
	if db == nil {
		logger.Fatal(gormdb.ErrClient)
	}

	data := gormdb.Cli(ctx).First(&user)
	if data.Error != nil {
		logger.Fatal(data.Error)
	}

	logger.Info(user.Name)

	<-exit
	cancel()
}
