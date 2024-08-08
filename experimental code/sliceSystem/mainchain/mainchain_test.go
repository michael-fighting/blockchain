package mainchain

import (
	"fmt"
	"github.com/DE-labtory/cleisthenes"
	"math/rand"
	"testing"
	"time"
)

func TestEpochQueue_Pop(t *testing.T) {
	q := EpochQueue{}
	q.Init()
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 100; i++ {
		r := rand.Intn(1000)
		fmt.Printf("%d ", r)
		q.Add(cleisthenes.Epoch(r))
	}
	fmt.Println()
	fmt.Println("all:", q.All())

	for i := 0; i < 100; i++ {
		e, _ := q.Poll()
		fmt.Printf("%d ", e)
	}
}
