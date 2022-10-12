package main

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
)

//定义一个全局的pool
var pool *redis.Pool

//当启动程序时，就初始化连接池
func init() {
	pool = &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", "localhost:6379")
		},
		MaxIdle:         8,
		MaxActive:       0,
		IdleTimeout:     100,
	}
}

func main(){
	//先从pool取出一个链接
	conn := pool.Get()
	defer conn.Close()

	_, err := conn.Do("Set", "name", "汤姆猫")
	if err != nil{
		fmt.Println("conn.Do err=", err)
		return
	}

	//取出
	r, err := redis.String(conn.Do("Get","name"))
	if err != nil {
		fmt.Println("conn.Do err=", err)
		return
	}
	fmt.Println("r=", r)

	//如果我们我们要从pool取出链接，一定保证连接池是没有关闭
	//pool.Close()
	conn2 := pool.Get()
	_, err = conn2.Do("Set", "name", "汤姆猫123")
	if err != nil{
		fmt.Println("conn2.Do err~~~=", err)
		return
	}

	//取出
	r2, err := redis.String(conn2.Do("Get","name"))
	if err != nil {
		fmt.Println("conn.Do err=", err)
		return
	}
	fmt.Println("r2=", r2)
	fmt.Println("conn2=", conn2)

}