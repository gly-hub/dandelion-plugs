package jwt

import (
	"errors"
	"github.com/gly-hub/go-dandelion/application"
	"github.com/gly-hub/go-dandelion/database/redigo"
	"github.com/gomodule/redigo/redis"
	"github.com/patrickmn/go-cache"
	"strings"
	"time"
)

var (
	Config    *config
	jwt       *Jwt
	storage   *Storage
	ErrUnique = errors.New("unique is empty")
)

type MetaData interface {
	Unique() string
}

type config struct {
	Jwt jwtConfig `json:"jwt" yaml:"jwt"`
}

type jwtConfig struct {
	Model      string `json:"model"`
	Key        string `json:"key" yaml:"key"`
	ExpireTime int64  `json:"expire_time" yaml:"expireTime"`
}

func Plug() *Plugin {
	return &Plugin{}
}

type Plugin struct {
}

func (p *Plugin) Config() interface{} {
	Config = &config{}
	return Config
}

func (p *Plugin) InitPlugin() error {
	if Config.Jwt.Key == "" {
		Config.Jwt.Key = "jwt-key"
	}

	if Config.Jwt.ExpireTime == 0 {
		Config.Jwt.ExpireTime = 60 * 60 * 24 * 7
	}

	if strings.ToLower(Config.Jwt.Model) == "refresh" {
		Config.Jwt.ExpireTime = 60 * 60
	}

	jwt = NewJwt(Config.Jwt.Key, time.Duration(Config.Jwt.ExpireTime)*time.Second)
	storage = InitStorage()
	return nil
}

func Token(metadata MetaData) (token string, err error) {
	token, err = jwt.Token(metadata)
	if err != nil {
		return "", err
	}
	switch strings.ToLower(Config.Jwt.Model) {
	case "unique", "refresh":
		// token全局唯一
		if metadata.Unique() == "" {
			return "", ErrUnique
		}
		// 需要替换掉原有的token
		err = storage.Set(metadata.Unique(), token)
		if err != nil {
			return
		}
	default:
		// 正常模式，使用jwt自身过期时间控制
		// 可同时存在多个token，每个token都有自己的过期时间
		return
	}
	return
}

func Parse(token string, metadata MetaData) error {
	return jwt.Parse(token, metadata)
}

func Check(token string, metadata MetaData) (err error) {
	if err = Parse(token, metadata); err != nil {
		return
	}

	if Config.Jwt.Model == "unique" {
		if sToken, gErr := storage.Get(metadata.Unique()); gErr == nil || sToken != token {
			return errors.New("token is expired")
		}
	}

	if Config.Jwt.Model == "refresh" {
		if sToken, gErr := storage.Get(metadata.Unique()); gErr == nil || sToken != token {
			return errors.New("token is expired")
		}
		// 刷新缓存中的过期时间
		err = storage.Set(metadata.Unique(), token)
		if err != nil {
			return
		}
	}
	return nil
}

func ExpireTime(token string) (expireTime int64, err error) {
	return jwt.ExpireTime(token)
}

func Del(metadata string) error {
	return storage.Del(metadata)
}

type Storage struct {
	s interface{}
}

func (s *Storage) Set(key string, value string) (err error) {
	switch s.s.(type) {
	case *cache.Cache:
		s.s.(*cache.Cache).Set(key, value, time.Duration(Config.Jwt.ExpireTime)*time.Second)
	case *redigo.Client:
		_, err = s.s.(*redigo.Client).Execute(func(c redis.Conn) (res interface{}, err error) {
			return c.Do("set", key, value, "EX", Config.Jwt.ExpireTime)
		})
	}
	return err
}

func (s *Storage) Get(key string) (value string, err error) {
	switch s.s.(type) {
	case *cache.Cache:
		loadValue, ok := s.s.(*cache.Cache).Get(key)
		if !ok {
			return "", errors.New("key not found")
		}
		if loadValue != nil {
			value = loadValue.(string)
		}
	case *redigo.Client:
		value, err = s.s.(*redigo.Client).String(func(c redis.Conn) (res interface{}, err error) {
			return c.Do("get", key)
		})
		if err != nil {
			return "", err
		}
	}
	return value, err
}

func (s *Storage) Del(key string) (err error) {
	switch s.s.(type) {
	case *cache.Cache:
		s.s.(*cache.Cache).Delete(key)
	case *redigo.Client:
		_, err = s.s.(*redigo.Client).Execute(func(c redis.Conn) (res interface{}, err error) {
			return c.Do("del", key)
		})
	}
	return
}

func InitStorage() *Storage {
	r := application.Redis{}
	if db := r.GetRedis(); db != nil {
		return &Storage{s: db}
	}

	return &Storage{s: cache.New(5*time.Minute, 10*time.Minute)}
}
