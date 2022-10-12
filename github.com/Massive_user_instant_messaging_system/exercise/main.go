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
	//通过终端输入Monster信息
	var name, age, skill string
	fmt.Println("请依次输入Monster的name、age、skill信息")
	fmt.Scanln(&name, &age, &skill)
	_, err = conn.Do("Hmset", "user01", "name", name, "age", age, "skill", skill)
	if err != nil{
		fmt.Println("hset err=", err)
		return
	}

	//3.通过go 向redis读取数据string 【key-val]
	r, err := redis.Strings(conn.Do("Hmget", "user01", "name", "age", "skill"))
	if err != nil{
		fmt.Println("hget err=", err)
		return
	}

	//因为返回r是interface{}
	//因为name对应的值是string,因此我们需要转换
	//nameString := r.(string)

	//通过Golang对Redis操作Hash数据类型
	//fmt.Printf("r=%v\n", r)

	for i, v := range r{
		fmt.Printf("r[%d]=%s\n", i, v)
	}

}

