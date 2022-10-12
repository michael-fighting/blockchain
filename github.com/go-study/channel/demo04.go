package main

type Cat struct{
	Name string
}
type Person struct {
	Name string
	Age int
	Address string
}

// 演示管道的使用
//func main() {
//	////1.创建一个可以存放3个int类型的管道
//	//var intChan chan int
//	//intChan = make(chan int, 3)
//	//
//	////2.看看intChan是什么
//	//fmt.Printf("intChan 的值=%v intChan本身的地址=%p\n", intChan, &intChan) //intChan 的值=0xc000110000 intChan本身的地址=0xc000006028
//	//
//	////3.向管道写入数据
//	//intChan <- 3
//	//num := 211
//	//intChan <- num
//	////注意向管道存入的数据长度不能超过3
//	//intChan <- 6
//	////intChan <- 12 //fatal error: all goroutines are asleep - deadlock!
//	//
//	////4.看看管道的长度和cap（容量）
//	//fmt.Printf("channel len=%v cap=%v \n", len(intChan), cap(intChan))
//	//
//	////5.从管道中取数据
//	//var num2 int
//	//num2 = <- intChan
//	//fmt.Println("num2=", num2)
//	//
//	////6.在没有使用协程的情况下，如果我们的管道数据已经全部取出，再取就会报deadlock
//	//num3 := <- intChan
//	//num4 := <- intChan
//	//num5 := <- intChan  //fatal error: all goroutines are asleep - deadlock!
//	//fmt.Println(num3, num4, num5)
//
//	//var allChan chan interface{}
//	//allChan = make(chan interface{}, 10)
//	//
//	//cat1 := Cat{Name :"tom"}
//	//allChan <- cat1
//	//cat2 := <- allChan
//	//fmt.Printf("%T, %v\n", cat2, cat2)
//	//fmt.Println(cat2.(Cat).Name)
//	var struChan chan Person
//	struChan = make(chan Person, 10)
//	for i := 0; i < 10; i++{
//		add := rand.Int()
//		struct1 := Person{
//			Name:    "",
//			Age:     add,
//			Address: "",
//		}
//		struChan <- struct1
//	}
//	close(struChan)
//	for v := range struChan{
//		fmt.Printf("%v,%v,%v \n", v.Name, v.Age, v.Address)
//	}
//}

//func main() {
//	intChan := make(chan int, 3)
//	intChan <- 100
//	intChan <- 200
//	intChan <- 300
//	close(intChan) //这时不能够再写入数到channel
//	//intChan <- 400  //panic: send on closed channel
//
//	//遍历管道
//	intChan2 := make(chan int, 100)
//	for i := 0; i < 100; i++{
//		intChan2 <- i*2
//	}
//
//	//遍历管道不能使用普通的for循环
//
//	//遍历时，一定要关闭管道，否则出现dead lock
//	close(intChan2)
//	for v := range intChan2{
//		fmt.Println("v=", v)
//	}
//
//}