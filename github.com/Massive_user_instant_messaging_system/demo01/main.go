package demo01

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
	_, err = conn.Do("set", "name", "tomjerry猫猫")
	if err != nil{
		fmt.Println("set err=", err)
		return
	}

	//3.通过go 向redis读取数据string 【key-val]
	r, err := redis.String(conn.Do("get", "name"))
	if err != nil{
		fmt.Println("get err=", err)
		return
	}
	//因为返回r是interface{}
	//因为name对应的值是string,因此我们需要转换
	//nameString := r.(string)

	//通过Golang对Redis操作Hash数据类型
	fmt.Println(r)

}
