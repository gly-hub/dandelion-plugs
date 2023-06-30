package captcha

import (
	"errors"
	"github.com/gly-hub/go-dandelion/application"
	"github.com/gly-hub/go-dandelion/database/redigo"
	"github.com/gomodule/redigo/redis"
	"github.com/patrickmn/go-cache"
	"time"
)

type Storage struct {
	s interface{}
}

func (s *Storage) Set(key string, value string) (err error) {
	switch s.s.(type) {
	case *cache.Cache:
		s.s.(*cache.Cache).Set(key, value, 5*time.Minute)
	case *redigo.Client:
		_, err = s.s.(*redigo.Client).Execute(func(c redis.Conn) (res interface{}, err error) {
			return c.Do("set", key, value, "EX", 60*5)
		})
	}
	return err
}

func (s *Storage) Get(key string, clear bool) (value string, err error) {
	switch s.s.(type) {
	case *cache.Cache:
		loadValue, ok := s.s.(*cache.Cache).Get(key)
		if !ok {
			return "", errors.New("key not found")
		}
		if loadValue != nil {
			value = loadValue.(string)
		}
		if clear {
			s.s.(*cache.Cache).Delete(key)
		}
	case *redigo.Client:
		value, err = s.s.(*redigo.Client).String(func(c redis.Conn) (res interface{}, err error) {
			return c.Do("get", key)
		})
		if err != nil {
			return "", err
		}
		if clear {
			_, _ = s.s.(*redigo.Client).Execute(func(c redis.Conn) (res interface{}, err error) {
				return c.Do("del", key)
			})
		}
	}
	return value, err
}

func InitStorage() *Storage {
	r := application.Redis{}
	if db := r.GetRedis(); db != nil {
		return &Storage{s: db}
	}

	return &Storage{s: cache.New(5*time.Minute, 10*time.Minute)}
}
