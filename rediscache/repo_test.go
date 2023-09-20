/*
@Date: 2021/12/17 14:55
@Author: max.liu
@File : repo_test
*/

package rediscache

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"log"
	"sync"
	"testing"
	"time"
)

type checkData struct {
	Name string `json:"name"`
	Age  int64  `json:"age"`
}

func (c checkData) MarshalBinary() ([]byte, error) {
	return json.Marshal(c)
}

func (c checkData) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &c)
}

var testConfig = Config{
	Addr:       "127.0.0.1:6380",
	Password:   "123456",
	ServerType: "standalone",
}

func TestGet(t *testing.T) {
	ctx := context.Background()
	redisConf = testConfig
	err := redisConf.NewRedisCli(ctx)
	if err != nil {
		t.Fatal(err.Error())
	}

	cli := GetCli()
	curd := NewCRUD(ctx, cli)

	err = curd.Set("test", checkData{
		Name: "hello",
		Age:  10,
	}, 10*time.Second)
	if err != nil {
		t.Fatalf("set failed: %s", err.Error())
	}

	val, err := curd.Get("test")
	if err != nil {
		t.Fatal(err.Error())
	}

	t.Log(val)
}

func TestLock(t *testing.T) {
	ctx := context.Background()
	redisConf = testConfig
	err := redisConf.NewRedisCli(ctx)
	if err != nil {
		t.Fatal(err.Error())
	}

	cli := GetCli()
	curd := NewCRUD(ctx, cli)
	ok, err := curd.TryLock("lock", "uuid", 1)
	log.Print(ok)
	if err != nil {
		t.Fatalf("lock failed: %s", err.Error())
	}
}

func TestUnLock(t *testing.T) {
	ctx := context.Background()
	redisConf = testConfig
	err := redisConf.NewRedisCli(ctx)
	if err != nil {
		t.Fatal(err.Error())
	}

	cli := GetCli()
	curd := NewCRUD(ctx, cli)
	err = curd.UnLock("lock", "uuid")
	if err != nil {
		t.Fatalf("lock failed: %s", err.Error())
	}
}

// 非阻塞
func TestLockConcurrency(t *testing.T) {
	ctx := context.Background()
	redisConf = testConfig
	err := redisConf.NewRedisCli(ctx)
	if err != nil {
		t.Fatal(err.Error())
	}

	cli := GetCli()
	curd := NewCRUD(ctx, cli)

	const numGoroutines = 10
	const lockKey = "test-lock"

	var (
		counter int // 通过维护和检查 counter，可以确保锁确实提供了互斥访问
		wg      sync.WaitGroup
	)
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// 尝试获取锁
			ok, err := curd.TryLock(lockKey, fmt.Sprintf("%d", id), 3)
			if err != nil {
				log.Fatalf(" goroutine %d 获取锁失败: %v", id, err)
			}
			log.Printf("goroutine %d 获取锁", id)
			if ok {
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
				log.Printf("goroutine %d 释放锁\n-------------------", id)
				err = curd.UnLock(lockKey, fmt.Sprintf("%d", id))
				if err != nil {
					log.Fatalf("goroutine %d 释放锁失败: %v", id, err)
				}
			}

		}(i)
	}

	wg.Wait()
	// 断言共享状态没有被并发访问
	assert.Equal(t, 0, counter, "共享状态被同时访问")
}

// 阻塞
func TestLockConcurrencyBlocking(t *testing.T) {
	ctx := context.Background()
	redisConf = testConfig
	err := redisConf.NewRedisCli(ctx)
	if err != nil {
		t.Fatal(err.Error())
	}

	cli := GetCli()
	curd := NewCRUD(ctx, cli)

	const numGoroutines = 10
	const lockKey = "test-lock"

	var (
		counter int // 通过维护和检查 counter，可以确保锁确实提供了互斥访问
		wg      sync.WaitGroup
	)
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// 尝试获取锁
			ok, err := curd.TryLockBlocking(lockKey, fmt.Sprintf("%d", id), 1, time.Second*5)
			if err != nil {
				log.Printf(" goroutine %d 获取锁失败: %v", id, err)
			}
			if ok {
				log.Printf("goroutine %d 获取锁", id)
				// 更新共享状态
				counter++
				if counter > 1 {
					log.Fatal("多个goroutine同时访问临界区")
				}
				// doing
				time.Sleep(500 * time.Millisecond)
				log.Printf("goroutine %d doing....", id)
				counter--
				// 释放锁
				log.Printf("goroutine %d 释放锁\n-------------------", id)
				err = curd.UnLock(lockKey, fmt.Sprintf("%d", id))
				if err != nil {
					log.Fatalf("goroutine %d 释放锁失败: %v", id, err)
				}
			}

		}(i)
	}

	wg.Wait()
	// 断言共享状态没有被并发访问
	assert.Equal(t, 0, counter, "共享状态被同时访问")
}
