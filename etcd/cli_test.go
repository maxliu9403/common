package etcd

import (
	"context"
	"log"
	"sync"
	"testing"
	"time"
)

const (
	TestPrefix = "/template/diagram/"
)

func initEtcd() (cancelFunc context.CancelFunc, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	etcdConfig := &Config{
		Endpoints: "127.0.0.1:2379",
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
				log.Fatal(err.Error())
				return
			}
			log.Println(resp.Kvs)
		}()
	}

	wg.Wait()
}

func TestPut(t *testing.T) {
	cancel, err := initEtcd()
	if err != nil {
		log.Fatal(err)
	}
	defer cancel()

	resp, err := Cli().Put(TestPrefix, "value")
	if err != nil {
		log.Fatal(err)
	}
	log.Println(resp)
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

	lockKey := TestPrefix
	resp, err := Cli().Lock(lockKey, 10, time.Second*20)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(resp)
}

func TestUnLock(t *testing.T) {
	cancel, err := initEtcd()
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()

	lockKey := TestPrefix
	resp, err := Cli().Lock(lockKey, 2, time.Second*1)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("resp:%v", resp)
	log.Println("doing")

	// 超出ttl，会进行续租
	time.Sleep(time.Second * 3)

	err = Cli().Unlock(resp.ID)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(resp)
}
