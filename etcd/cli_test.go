package etcd

import (
	"context"
	"github.com/maxliu9403/common/logger"
	"sync"
	"testing"
)

const (
	TestPrefix = "/template/diagram/"
)

func initEtcd() (cancelFunc context.CancelFunc, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	etcdConfig := &Config{
		Endpoints:    "",
		DialTimeout:  0,
		Username:     "Username",
		Password:     "Password",
		CAFilePath:   "Password",
		CertFilePath: "CertFilePath",
		KeyFilePath:  "KeyFilePath",
	}
	err = etcdConfig.Init(ctx)
	if err != nil {
		return cancel, err
	}
	err = Default().CreateEtcdV3Client()
	if err != nil {
		return cancel, err
	}
	return cancel, err
}

func TestCreateEtcdV3Client(t *testing.T) {
	cancel, err := initEtcd()
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()

	if len(Cli().cli.Endpoints()) > 0 {
		t.Log(Cli().cli.Cluster.MemberList(Cli().ctx))
	} else {
		t.Fatal("create client failed")
	}
}

func TestFind(t *testing.T) {
	cancel, err := initEtcd()
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()

	resp, err := Cli().Find(TestPrefix)
	if err != nil {
		t.Fatal(err)
	}

	for _, key := range resp.Kvs {
		t.Log("key is: ", key)
	}

}

func TestGet(t *testing.T) {
	cancel, err := initEtcd()
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()

	keyList := []string{
		TestPrefix,
	}

	wg := sync.WaitGroup{}
	wg.Add(len(keyList))

	for _, key := range keyList {
		k := key
		go func() {
			defer wg.Done()
			resp, err := Cli().Get(k, 0)
			if err != nil {
				logger.Errorf(err.Error())
				return
			}
			logger.Debug(resp.Kvs)
		}()
	}

	wg.Wait()
}

func TestDelete(t *testing.T) {
	cancel, err := initEtcd()
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()

	if err = Cli().Delete(TestPrefix); err != nil {
		t.Fatal(err)
	}
}

func TestLock(t *testing.T) {
	cancel, err := initEtcd()
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()

	lockKey := "lockKey"
	taskPrefix := "taskPrefix"
	taskKey := taskPrefix + "taskID"
	resp, err := Cli().Lock(lockKey, taskKey)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(resp)
}
