package entry

import (
	"gopkg.in/redis.v3"
	"time"
)

type ITokenStorage interface {
	Exist(token string) bool
	Save(token []string)
}

type TokenStorage struct {
	storage        *redis.Client
	master, slave  string
	expiredSeconds int64
}

func NewTokenStorage(expiredSeconds int64, master, slave string) ITokenStorage {

	storage := redis.NewFailoverClient(&redis.FailoverOptions{
		MasterName:    master,
		SentinelAddrs: []string{slave},
		DialTimeout:   30 * time.Second,
		ReadTimeout:   5 * time.Second,
		WriteTimeout:  5 * time.Second,
		PoolSize:      30,
		PoolTimeout:   0,
		IdleTimeout:   60 * time.Second,
		MaxRetries:    3,
	})

	return &TokenStorage{storage: storage,
		master: master, slave: slave, expiredSeconds: expiredSeconds}

}

func (self *TokenStorage) Exist(token string) bool {

	t, _ := self.storage.Exists(token).Result()
	return t

}

func (self *TokenStorage) Save(token []string) {

	p := self.storage.Pipeline()
	for _, t := range token {
		//save invalid token
		p.Set(t, nil, time.Duration(self.expiredSeconds*int64(time.Second)))
	}
	//默认就成成功了吧
	p.Exec()
}
