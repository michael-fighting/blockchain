package main

import "fmt"

func main() {
	s := []string{"a","b","c"}
	for _, v :=range s{
		go func(v string) {
			fmt.Println(v)
			fmt.println("hello world")
		}(v)
	}
	select {

	}
}
