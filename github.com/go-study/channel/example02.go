package main

//func putData(intChan chan int) {
//	for i := 1; i <= 8000; i++ {
//		intChan <- i
//	}
//	close(intChan)
//}
//
//func judgePrime(primeChan chan int, exitChan chan bool, intChan chan int) {
//	for {
//		v, ok := <-intChan
//		if !ok {
//			exitChan <- true
//			break
//		}
//		flag := true
//		for i := 2; i < v; i++ {
//			if v%i == 0 {
//				//说明该number不是素数
//				flag = false
//				break
//			}
//		}
//		if flag {
//			//为素数，将数据放入管道中
//			primeChan <- v
//		}
//	}
//	fmt.Println("有一个协程因为取不到数据退出")
//
//	exitChan <- true
//}
//
//func main() {
//	intChan := make(chan int, 8000)
//	primeChan := make(chan int, 8000)
//	exitChan := make(chan bool, 4)
//
//	go putData(intChan)
//	for i := 0; i < 4; i++ {
//		go judgePrime(primeChan, exitChan, intChan)
//
//	}
//	go func(){
//		for i := 0; i < 4; i++ {
//			<-exitChan
//		}
//		close(primeChan)
//	}()
//	for v := range primeChan{
//		fmt.Printf("素数的结果=%v\n,", v)
//
//	}
//}
