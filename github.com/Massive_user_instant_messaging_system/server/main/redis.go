package main

import (
	"github.com/garyburd/redigo/redis"
	"time"
)

//定义一个全局的pool
var pool *redis.Pool

func initPool(address string, maxIdle, maxActive int, idleTimeout time.Duration) {
	pool = &redis.Pool{
		Dial: func() (redis.Conn, error) {  //初始化链接的代码，链接哪个ip的 redis
			return redis.Dial("tcp", address)
		},
		MaxIdle:         maxIdle,   //最大空闲连接数
		MaxActive:       maxActive,   //表示与数据库的最大连接数，0表示没有限制
		IdleTimeout:     idleTimeout,  //最大空闲时间
	}

}
