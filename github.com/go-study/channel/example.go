package main

//func WriteData(numChan chan int){
//	for i := 1; i <= 2000; i++{
//		numChan <- i
//		fmt.Printf("将数据%v放入numChan管道中", i)
//	}
//	close(numChan)
//}
//
//func Calculate(resChan chan string, num int,booChan chan bool) {
//	sum := 0
//	for i := 1; i <= num; i++{
//		sum += i
//	}
//	res := fmt.Sprintf("res[%v]=%v", num, sum)
//	resChan <- res
//	if len(resChan) == 2000{
//		close(resChan)
//		booChan <- true
// 	}
//}
//
//func main(){
//	numChan := make(chan int, 2000)
//	resChan := make(chan string, 2000)
//	booChan := make(chan bool, 1)
//	go WriteData(numChan)
//
//	for {
//		v, ok := <- numChan
//		if !ok{  //取不到数据就退出
//			break
//		}
//		go Calculate(resChan, v, booChan)
//	}
//	ok := <- booChan
//	if ok {
//		for v := range resChan{
//			fmt.Println(v)
//		}
//	}
//}
