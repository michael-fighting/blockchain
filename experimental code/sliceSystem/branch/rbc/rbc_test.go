package rbc

import (
	"bytes"
	"fmt"
	"github.com/DE-labtory/cleisthenes/sliceSystem/branch/rbc/merkletree"
	"github.com/klauspost/reedsolomon"
	"testing"
)

func TestRBC_Decode(t *testing.T) {
	f := 0
	n := 10
	numParityShards := 2 * f
	numDataShards := n - numParityShards

	enc, _ := reedsolomon.New(numDataShards, numParityShards)

	data := []byte("data block data block data block data block data block data block")
	shards, _ := shard(enc, data)

	reqs, _ := makeRequest(shards)

	resShards := make([][]byte, numDataShards+numParityShards)
	rootHash := reqs[0].(*ValRequest).RootHash
	for _, req := range reqs {
		if bytes.Equal(rootHash, req.(*ValRequest).RootHash) {
			order := merkletree.OrderOfData(req.(*ValRequest).Indexes)
			//将纠删码分片放在了请求的Data数据里
			resShards[order] = req.(*ValRequest).Data.Bytes()
		}
	}

	if err := enc.Reconstruct(resShards); err != nil {
		fmt.Println("reconstruct failed, err:", err.Error())
	}

	var value []byte
	for _, d := range resShards[:numDataShards] {
		value = append(value, d...)
	}

	fmt.Println("value:", string(value[:len(data)]))
}
