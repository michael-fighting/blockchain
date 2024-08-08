package rscode

import (
	"fmt"
	"testing"
)

func TestMockEncoder_Reconstruct(t *testing.T) {
	enc := NewMockEncoder(4, 6)
	shards := make([][]byte, 10)
	for i := 0; i < 10; i++ {
		shards[i] = []byte("hello")
	}
	enc.Reconstruct(shards)
	for i := 0; i < 10; i++ {
		fmt.Println(i, string(shards[i]))
	}
}

func TestMockEncoder_Split(t *testing.T) {
	enc := NewMockEncoder(4, 6)
	dst, _ := enc.Split([]byte("hello"))
	for i := 0; i < len(dst); i++ {
		fmt.Println(i, string(dst[i]))
	}
}
