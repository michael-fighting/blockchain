package main

import "fmt"

func main(){
	//使用select可以解决从管道取数据的阻塞问题

	//定义10个数据的管道
	intChan := make(chan int, 10)
	for i := 0; i < 10; i++{
		intChan <- i
	}

	//定义5个数据的管道
	stringChan := make(chan string, 5)
	for i := 0; i < 5; i++{
		stringChan <- "hello" + fmt.Sprintf("%d", i)
	}

	//传统的方法在遍历管道时，如果不关闭会阻塞而导致deadlock

	//问题，在实际开发中，不好确定什么时候关闭该管道
	//使用select方式解决
	//label:
	for {
		select {
		//注意：这里，如果intChan一直没有关闭，不会一直阻塞而deadlock
		//会自动到下一个case匹配
		case v := <- intChan:
			fmt.Printf("从intChan读取的数据%d\n", v)
		case v := <- stringChan:
			fmt.Printf("从stringChan读取的数据%s\n", v)
		default:
			fmt.Printf("都取不到了，程序员可以加入自己逻辑\n")
			return
			//break
		}
	}
}
