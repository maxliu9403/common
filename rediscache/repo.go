/*
@Date: 2021/12/17 14:21
@Author: max.liu
@File : repo
*/

package rediscache

import (
	"context"
	"fmt"
	"time"

	"github.com/maxliu9403/common/logger"

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
	for {
		select {
		// 监听续租被释放
		case <-lockCtx.Done():
			return
		case <-c.Ctx.Done():
			return
		case <-ticker.C:
			// 获取过期剩余时间，已过期的停止续租
			ttl, err := c.Rdb.TTL(c.Ctx, key).Result()
			if err != nil || ttl <= 0 {
				return
			}

			// 为其增加更多的生存时间，延长一倍的过期时间，最小过期时间需要1秒
			_, err = c.Rdb.Expire(c.Ctx, key, expiration).Result()
			if err != nil {
				// 处理错误，例如记录日志
				logger.Errorf("failed to renew redis distributed lock : %v, key:%v", err.Error(), key)
				return
			}
		}
	}
}

// TryLock 非阻塞
// key 锁在redis中的名称
// uuid 锁的值必须唯一
// expiration 锁过期时间，最小单位1秒
func (c *RedisCrud) TryLock(key, uuid string, expiration int) (bool, error) {
	if expiration == 0 {
		expiration = 1
	}

	if c.Rdb == nil {
		return false, fmt.Errorf("redis client is not initialized yet")
	}

	c.cancelRenewFunc = map[string]context.CancelFunc{}
	duration := time.Duration(expiration) * time.Second
	ok, err := c.Rdb.SetNX(c.Ctx, key, uuid, duration).Result()
	if err != nil {
		return false, err
	}
	// 获取到锁，启动续约
	if ok {
		uniqueKey := key + uuid
		lockCtx, lockKeyCancel := context.WithCancel(c.Ctx)
		c.cancelRenewFunc[uniqueKey] = lockKeyCancel
		go c.keepAlive(lockCtx, key, duration)
	}
	return ok, nil
}

// TryLockBlocking 阻塞
// key 锁在redis中的名称
// uuid 锁的值必须唯一
// expiration 锁过期时间，最小单位1秒
// reTry 在超时时间范围内的重试次数，必须大于0，例如超时时间为9秒，重试3次，那么每隔3s重试一次
// timeout 阻塞等待时间
func (c *RedisCrud) TryLockBlocking(key, uuid string, expiration, reTry int, timeout time.Duration) (bool, error) {
	if expiration == 0 || reTry == 0 || timeout.Seconds() == 0 {
		return false, fmt.Errorf("failed to acquire redis lock, expiration or reTry or timeout must be greater than 0 ")
	}

	if c.Rdb == nil {
		return false, fmt.Errorf("redis client is not initialized yet")
	}

	c.cancelRenewFunc = map[string]context.CancelFunc{}
	timeoutCtx, cancel := context.WithTimeout(c.Ctx, timeout)
	defer cancel()
	// 加锁失败，不断尝试，
	ticker := time.NewTicker(timeout / time.Duration(reTry))
	defer ticker.Stop()
	for {
		select {
		case <-c.Ctx.Done():
			return false, fmt.Errorf("failed to acquire redis lock, ctx done %v: %v", timeout, c.Ctx.Err())
		case <-timeoutCtx.Done():
			return false, fmt.Errorf("failed to acquire redis lock within %v: %v", timeout, timeoutCtx.Err())
		case <-ticker.C:
			ok, err := c.TryLock(key, uuid, expiration)
			// 获得锁||报错时返回
			if ok || err != nil {
				return ok, err
			}
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
	// 释放续约
	if cancelFunc, exists := c.cancelRenewFunc[uniqueKey]; exists {
		cancelFunc()
		delete(c.cancelRenewFunc, uniqueKey)
	}
	return err
}
