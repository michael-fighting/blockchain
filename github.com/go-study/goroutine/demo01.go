package main

//需求：要求统计1-20000的数字中，哪些是素数

//分析思路：
//1）传统的方法：就是使用一个循环，循环的判断各个数是不是素数
//2）使用并发或者并行的方式，将统计素数的任务分配给多个goroutine去完成，这时就会使用到goroutine

//func main(){
//	var flag bool
//	sNumbers := []int{}
//	var j int
//	for i := 2; i * i <= 9000000; i++{
//		flag = true
//		for j = 2; j < i; j ++{
//			if i % j == 0{
//				flag = false
//				break
//			}
//		}
//		if flag {
//			sNumbers = append(sNumbers, i)
//		}
//	}
//	fmt.Println(sNumbers)
//
//}
