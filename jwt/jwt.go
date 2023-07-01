package jwt

import (
	jwtgo "github.com/dgrijalva/jwt-go"
	jsoniter "github.com/json-iterator/go"
	"time"
)

type Jwt struct {
	key    string
	expire time.Duration
}

func NewJwt(key string, expire time.Duration) *Jwt {
	return &Jwt{
		key:    key,
		expire: expire,
	}
}

func (j *Jwt) Token(metadata interface{}) (string, error) {
	var (
		metaStr, token string
		claims         jwtgo.MapClaims
		err            error
	)
	if metaStr, err = jsoniter.MarshalToString(metadata); err != nil {
		return "", err
	}

	if err = jsoniter.UnmarshalFromString(metaStr, &claims); err != nil {
		return "", err
	}

	claims["exp"] = time.Now().Add(j.expire * time.Minute).Unix()
	tokenClaims := jwtgo.NewWithClaims(jwtgo.SigningMethodHS256, claims)
	// 调用加密方法，发挥Token字符串
	token, err = tokenClaims.SignedString([]byte(j.key))
	if err != nil {
		return "", err
	}
	return token, nil
}

func (j *Jwt) Parse(token string, metadata interface{}) error {
	var (
		claim   *jwtgo.Token
		err     error
		metaStr string
	)
	claim, err = jwtgo.Parse(token, func(token *jwtgo.Token) (interface{}, error) {
		return []byte(j.key), nil
	})
	if err != nil {
		return err
	}

	if metaStr, err = jsoniter.MarshalToString(claim.Claims.(jwtgo.MapClaims)); err != nil {
		return err
	}

	if err = jsoniter.UnmarshalFromString(metaStr, metadata); err != nil {
		return err
	}

	return nil
}

func (j *Jwt) ExpireTime(token string) (expireTime int64, err error) {
	var metaData = make(map[string]interface{})
	if err = j.Parse(token, &metaData); err != nil {
		return
	}
	expireTime = int64(metaData["exp"].(float64))
	return
}
