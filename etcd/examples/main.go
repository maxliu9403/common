package main

import (
	"context"
	"github.com/maxliu9403/common/etcd"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var testConfig = etcd.Config{
	Endpoints:    "",
	DialTimeout:  0,
	Username:     "Username",
	Password:     "Password",
	CAFilePath:   "Password",
	CertFilePath: "CertFilePath",
	KeyFilePath:  "KeyFilePath",
}

func main() {
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := testConfig.Init(ctx)
	if err != nil {
		log.Fatal(err)
		return
	}
	err = etcd.Default().CreateEtcdV3Client()
	if err != nil {
		log.Fatal(err)
		return
	}
	resp, err := etcd.Cli().Find("key")
	if err != nil {
		log.Fatal(err)
		return
	}
	for _, key := range resp.Kvs {
		log.Println("key is: ", key)
	}

}
