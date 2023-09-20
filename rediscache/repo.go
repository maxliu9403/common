/*
@Date: 2021/12/17 14:21
@Author: max.liu
@File : repo
*/

package rediscache

import (
	"context"
	"fmt"
	"github.com/maxliu9403/common/logger"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisCrud struct {
	Ctx             context.Context
	Rdb             *redis.Client
	cancelRenewFunc map[string]context.CancelFunc // 控制解锁后释放续约
}

func NewCRUD(ctx context.Context, cli *redis.Client) BasicCrud {
	return &RedisCrud{Ctx: ctx, Rdb: cli}
}

func (c *RedisCrud) Get(key string) (val string, err error) {
	if c.Rdb == nil {
		return "", fmt.Errorf("redis client is not initialized yet")
	}

	val, err = c.Rdb.Get(c.Ctx, key).Result()
	return
}

func (c *RedisCrud) Set(key string, value interface{}, timeOut time.Duration) (err error) {
	if c.Rdb == nil {
		return fmt.Errorf("redis client is not initialized yet")
	}
	return c.Rdb.Set(c.Ctx, key, value, timeOut).Err()
}

func (c *RedisCrud) keepAlive(lockCtx context.Context, key string, expiration time.Duration) {
	// 1/2 过期时间时开始续租
	ticker := time.NewTicker(expiration / 2)
	defer ticker.Stop()
	fmt.Println(22999)
	for {
		select {
		// 监听续租被释放
		case <-lockCtx.Done():
			fmt.Println("Done")
			return
		case <-c.Ctx.Done():
			return
		case <-ticker.C:
			// 获取过期剩余时间
			//ttl, err := c.Rdb.TTL(c.Ctx, key).Result()
			//fmt.Println(ttl.Seconds(), key, "ttl.Seconds()")
			//if err != nil || ttl <= 0 {
			//	logger.Errorf("failed to renew redis distributed lock : %v, key:%v", err, key)
			//	// 锁已经过期或出现错误，停止续租
			//	return
			//}
			// 为其增加更多的生存时间，延长一倍的过期时间，因为最小过期时间需要1秒
			_, err := c.Rdb.Expire(c.Ctx, key, expiration).Result()
			if err != nil {
				// 处理错误，例如记录日志
				logger.Errorf("failed to renew redis distributed lock : %v, key:%v", err.Error(), key)
				return
			}
		}
	}
}

// TryLock 非阻塞
// expiration 最小单位1秒
func (c *RedisCrud) TryLock(key, uuid string, expiration int) (bool, error) {
	if c.Rdb == nil {
		return false, fmt.Errorf("redis client is not initialized yet")
	}
	c.cancelRenewFunc = map[string]context.CancelFunc{}
	duration := time.Duration(expiration) * time.Second
	// 使用SETNX命令尝试获取锁
	ok, err := c.Rdb.SetNX(c.Ctx, key, uuid, duration).Result()
	if err != nil {
		return false, err
	}
	// 获取到锁，启动续约
	if ok {
		uniqueKey := key + uuid
		lockCtx, lockKeyCancel := context.WithCancel(c.Ctx)
		c.cancelRenewFunc[uniqueKey] = lockKeyCancel
		fmt.Println(c.cancelRenewFunc, "cancelRenewFunc")
		go c.keepAlive(lockCtx, key, duration)
	}
	return ok, nil
}

// TryLockBlocking 阻塞
func (c *RedisCrud) TryLockBlocking(key, uuid string, expiration int, timeout time.Duration) (bool, error) {
	if c.Rdb == nil {
		return false, fmt.Errorf("redis client is not initialized yet")
	}
	c.cancelRenewFunc = map[string]context.CancelFunc{}
	timeoutCtx, cancel := context.WithTimeout(c.Ctx, timeout)
	defer cancel()
	for {
		select {
		case <-c.Ctx.Done():
			return false, fmt.Errorf("failed to acquire redis lock ctx done %v: %v", timeout, c.Ctx.Err())
		case <-timeoutCtx.Done():
			return false, fmt.Errorf("failed to acquire redis lock within %v: %v", timeout, timeoutCtx.Err())
		default:
			ok, err := c.TryLock(key, uuid, expiration)
			if err != nil {
				return false, err
			}
			return ok, nil
		}
	}
}

func (c *RedisCrud) UnLock(key, uuid string) error {
	if c.Rdb == nil {
		return fmt.Errorf("redis client is not initialized yet")
	}

	// 使用 Lua 脚本来原子地检查锁的值并释放它，确保是锁的持有者来释放锁
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	_, err := c.Rdb.Eval(c.Ctx, script, []string{key}, uuid).Result()
	if err != nil {
		return err
	}
	uniqueKey := key + uuid
	fmt.Println(uniqueKey)
	// 释放续约
	if cancelFunc, exists := c.cancelRenewFunc[uniqueKey]; exists {
		fmt.Println(1189898)
		cancelFunc()
		delete(c.cancelRenewFunc, uniqueKey)
	}
	return err
}
