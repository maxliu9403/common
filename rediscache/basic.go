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
	TryLock(key, uuid string, expiration int) (bool, error)
	TryLockBlocking(key, uuid string, expiration, reTry int, timeout time.Duration) (bool, error)
}
