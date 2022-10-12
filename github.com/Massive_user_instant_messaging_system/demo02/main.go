package main

import (
"fmt"
"github.com/garyburd/redigo/redis"
)

func main(){
	//通过go 向redis写入数据和读取数据
	//1.链接到redis
	conn, err := redis.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		fmt.Println("redis.Dial err=", err)
		return
	}
	defer conn.Close()  //关闭

	//2. 通过go 向redis写入数据string 【key-val]
	_, err = conn.Do("Hset", "user01", "name", "john")
	if err != nil{
		fmt.Println("hset err=", err)
		return
	}

	_, err = conn.Do("Hset", "user01", "age", "18")
	if err != nil{
		fmt.Println("hset err=", err)
		return
	}

	//3.通过go 向redis读取数据string 【key-val]
	r1, err := redis.String(conn.Do("Hget", "user01", "name"))
	if err != nil{
		fmt.Println("hget err=", err)
		return
	}

	r2, err := redis.String(conn.Do("Hget", "user01", "age"))
	if err != nil{
		fmt.Println("hget err=", err)
		return
	}
	//因为返回r是interface{}
	//因为name对应的值是string,因此我们需要转换
	//nameString := r.(string)

	//通过Golang对Redis操作Hash数据类型
	fmt.Printf("r1=%v, r2=%v\n", r1, r2)

}

