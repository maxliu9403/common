/*
@Date: 2021/12/17 14:17
@Author: max.liu
@File : basic
*/

package rediscache

import (
	"time"
)

type BasicCrud interface {
	Set(key string, value interface{}, timeOut time.Duration) (err error)
	Get(key string) (val string, err error)
	UnLock(key, uuid string) error
	TryAcquireLock(key, uuid string, expiration time.Duration) (bool, error)
	TryAcquireLockBlocking(key, uuid string, expiration time.Duration, timeout time.Duration) (bool, error)
}
