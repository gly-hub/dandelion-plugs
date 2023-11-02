package redislock

import (
	"context"
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/team-dandelion/go-dandelion/application"
	"github.com/team-dandelion/go-dandelion/database/redigo"
	"github.com/team-dandelion/go-dandelion/logger"
	"time"
)

// DistributeLockRedis 基于redis的分布式可重入锁，自动续租
type DistributeLockRedis struct {
	key       string             // 锁的key
	value     interface{}        // 锁设置的值
	expire    int64              // 锁超时时间
	status    bool               // 上锁成功标识
	cancelFun context.CancelFunc // 用于取消自动续租携程
	redis     *redigo.Client     // redis句柄
}

const (
	minExpire = 2 //最小过期时间,单位秒
)

// NewDistributeLockRedis 创建锁实例
func NewDistributeLockRedis(key string, expire int64, value interface{}) (*DistributeLockRedis, error) {
	if expire < minExpire {
		return nil, fmt.Errorf("最小过期时间为%d秒", minExpire)
	}

	r := application.Redis{}
	if db := r.GetRedis(); db == nil {
		return nil, errors.New("redis is nil")
	}

	l := &DistributeLockRedis{
		key:    key,
		expire: expire,
		redis:  r.GetRedis(),
		value:  value,
	}
	err := l.tryLock()
	return l, err
}

// tryLock 上锁
func (dl *DistributeLockRedis) tryLock() (err error) {
	if err = dl.lock(); err != nil {
		return err
	}
	ctx, cancelFun := context.WithCancel(context.Background())
	dl.cancelFun = cancelFun
	dl.startWatchDog(ctx) // 创建守护协程，自动对锁进行续期
	dl.status = true
	return nil
}

// competition 竞争锁
func (dl *DistributeLockRedis) lock() error {
	result, err := dl.redis.Bool(func(c redis.Conn) (res interface{}, err error) {
		return c.Do("SETNX", genKey(dl.key), dl.value, "EX", dl.expire)
	})
	if err != nil {
		return err
	}
	if !result {
		return fmt.Errorf("已经被锁住")
	}
	return nil
}

// startWatchDog guard 创建守护协程，自动续期
func (dl *DistributeLockRedis) startWatchDog(ctx context.Context) {
	task := func() {
		for {
			select {
			// Unlock通知结束
			case <-ctx.Done():
				return
			default:
				// 否则只要开始了，就自动重入（续租锁）
				if dl.status {
					if _, err := dl.redis.Execute(func(c redis.Conn) (res interface{}, err error) {
						return c.Do("EXPIRE", genKey(dl.key), dl.expire)
					}); err != nil {
						logger.Warn(ctx, fmt.Sprintf("startWatchDog: %v", err))
						return
					}
					// 续租时间为 expire/2 秒
					time.Sleep(time.Duration(dl.expire/2) * time.Second)
				}
			}
		}
	}
	go task()
}

// Unlock 释放锁
func (dl *DistributeLockRedis) Unlock() (err error) {
	// 这个重入锁必须取消，放在第一个地方执行
	if dl.cancelFun != nil {
		dl.cancelFun() // 释放成功，取消重入锁
	}
	var res int64
	if dl.status {
		if res, err = dl.redis.Int64(func(c redis.Conn) (res interface{}, err error) {
			return c.Do("DEL", genKey(dl.key))
		}); err != nil {
			logger.Warn(context.Background(), fmt.Sprintf("DistributeLockRedis Unlock err: %v", err))
			return fmt.Errorf("释放锁失败")
		}

		if res == 1 {
			dl.status = false
			return nil
		}
	}
	return fmt.Errorf("释放锁失败")
}

// genKey 获取key前缀
func genKey(key string) string {
	return "lock:" + key
}
