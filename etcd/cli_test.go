package etcd

import (
	"context"
	"github.com/stretchr/testify/assert"
	"log"
	"sync"
	"testing"
	"time"
)

const (
	TestPrefix = "/template"
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
	resp, err := Cli().TryLockBlocking(lockKey, 10, time.Second*20)
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
	resp, err := Cli().TryLockBlocking(lockKey, 2, time.Second*1)
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

// 阻塞
func TestLockConcurrencyBlocking(t *testing.T) {
	cancel, err := initEtcd()
	if err != nil {
		log.Fatal(err)
	}
	defer cancel()

	const numGoroutines = 10
	const lockKey = "test-lock"

	var (
		counter int // 通过维护和检查 counter，可以确保锁确实提供了互斥访问
		wg      sync.WaitGroup
	)
	cli := Cli()
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// 尝试获取锁
			resp, err := cli.TryLockBlocking(lockKey, 1, time.Second*2)
			if err != nil {
				log.Fatalf(" goroutine %d 获取锁失败: %v", id, err)
			}
			log.Printf("goroutine %d 获取锁，leaseId %v ", id, resp.ID)
			// 更新共享状态
			counter++
			if counter > 1 {
				log.Fatal("多个goroutine同时访问临界区")
			}
			// doing
			time.Sleep(100 * time.Millisecond)
			log.Printf("goroutine %d doing....", id)
			counter--

			// 释放锁
			log.Printf("goroutine %d 释放锁，leaseId %v \n-------------------", id, resp.ID)
			err = cli.Unlock(resp.ID)
			if err != nil {
				log.Fatalf("goroutine %d 释放锁失败: %v", id, err)
			}
		}(i)
	}

	wg.Wait()
	cli.Close()

	// 断言共享状态没有被并发访问
	assert.Equal(t, 0, counter, "共享状态被同时访问")
}

// 非阻塞
func TestLockConcurrency(t *testing.T) {
	cancel, err := initEtcd()
	if err != nil {
		log.Fatal(err)
	}
	defer cancel()

	const numGoroutines = 10
	const lockKey = "test-lock"

	var (
		counter int // 通过维护和检查 counter，可以确保锁确实提供了互斥访问
		wg      sync.WaitGroup
	)
	cli := Cli()
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// 尝试获取锁
			resp, err := cli.TryLock(lockKey, 1)
			if err != nil {
				log.Printf(" goroutine %d 获取锁失败: %v", id, err)
				return
			}
			log.Printf("goroutine %d 获取锁，leaseId %v ", id, resp.ID)
			// 更新共享状态
			counter++
			if counter > 1 {
				log.Fatal("多个goroutine同时访问临界区")
			}
			// doing
			time.Sleep(100 * time.Millisecond)
			log.Printf("goroutine %d doing....", id)
			counter--

			// 释放锁
			log.Printf("goroutine %d 释放锁，leaseId %v \n-------------------", id, resp.ID)
			err = cli.Unlock(resp.ID)
			if err != nil {
				log.Fatalf("goroutine %d 释放锁失败: %v", id, err)
			}
		}(i)
	}

	wg.Wait()
	cli.Close()

	// 断言共享状态没有被并发访问
	assert.Equal(t, 0, counter, "共享状态被同时访问")
}
