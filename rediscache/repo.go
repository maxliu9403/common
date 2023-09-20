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

	"github.com/go-redis/redis/v8"
)

type RedisCrud struct {
	Ctx context.Context
	Rdb *redis.Client
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

//func (c *RedisCrud) KeepAlive(key string, interval time.Duration, extension time.Duration) {
//	ticker := time.NewTicker(interval)
//	defer ticker.Stop()
//
//	for {
//		select {
//		case <-ticker.C:
//			// 检查锁的剩余生存时间
//			ttl, err := c.Rdb.TTL(c.Ctx, key).Result()
//			if err != nil || ttl <= 0 {
//				// 锁已经过期或出现错误，停止续租
//				return
//			}
//			if ttl < interval {
//				// 如果锁的生存时间低于阈值，为其增加更多的生存时间
//				_, err := c.Rdb.Expire(c.Ctx, key, ttl+extension).Result()
//				if err != nil {
//					// 处理错误，例如记录日志
//				}
//			}
//		case <-c.Ctx.Done():
//			// 上下文被取消或超时，停止续租
//			return
//		}
//	}
//}

// TryAcquireLock 非阻塞
func (c *RedisCrud) TryAcquireLock(key, uuid string, expiration time.Duration) (bool, error) {
	if c.Rdb == nil {
		return false, fmt.Errorf("redis client is not initialized yet")
	}

	// 使用SETNX命令尝试获取锁
	ok, err := c.Rdb.SetNX(c.Ctx, key, uuid, expiration).Result()
	if err != nil {
		return false, err
	}

	//go c.KeepAlive(key)
	return ok, nil
}

// TryAcquireLockBlocking 阻塞
func (c *RedisCrud) TryAcquireLockBlocking(key, uuid string, expiration time.Duration, timeout time.Duration) (bool, error) {
	if c.Rdb == nil {
		return false, fmt.Errorf("redis client is not initialized yet")
	}
	timeoutCtx, cancel := context.WithTimeout(c.Ctx, timeout)
	defer cancel()
	for {
		select {
		case <-c.Ctx.Done():
			return false, fmt.Errorf("failed to acquire redis lock ctx done %v: %v", timeout, c.Ctx.Err())
		case <-timeoutCtx.Done():
			return false, fmt.Errorf("failed to acquire redis lock within %v: %v", timeout, timeoutCtx.Err())
		default:
			ok, err := c.TryAcquireLock(key, uuid, expiration)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
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
	return err
}
