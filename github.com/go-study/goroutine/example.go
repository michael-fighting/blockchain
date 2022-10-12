package main

import (
	"fmt"
	"sync"
	"time"
)

//计算1-200各个数的阶乘，并且把各个数的阶乘放入到map中。最后显示出来。

var myMap = make(map[int]int, 10)
//lock是一个全局的互斥锁
var lock sync.Mutex

//test函数用于计算n!,将这个结果放入到myMap
func test(n int){
	res := 1
	for i := 1; i <= n; i++{
		res *= i
	}
	lock.Lock()
	//将res放入到myMap
	myMap[n] = res
	lock.Unlock()
}

func main(){
	//开启多个协程
	for i := 1; i <= 20; i++{
		go test(i)
	}
	time.Sleep(time.Second*10)

	//这里我们输出结果，变量这个结果
	lock.Lock()
	for i, v := range myMap{
		fmt.Printf("map[%d]=%d\n", i, v)
	}
	lock.Unlock()
}
