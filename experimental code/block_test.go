package cleisthenes

import (
	"fmt"
	"github.com/google/uuid"
	"testing"
)

func TestCreateGenesis(t *testing.T) {
	b := CreateGenesis()
	fmt.Printf("%#v %#v\n", b.Header, b.Payload)
}

func TestBlock_Pack(t *testing.T) {
	b := CreateGenesis()
	b1 := NewBlock(1, b.Header.Hash)
	txs := make([]interface{}, 0)
	for i := 0; i < 4000000; i++ {
		tx := map[string]interface{}{
			"id":     uuid.New().String(),
			"from":   uuid.New().String(),
			"to":     uuid.New().String(),
			"amount": 1,
		}
		txs = append(txs, tx)
	}
	b1.AddTxs(txs)
	b1.Pack()
	b1.Serialize()
	//fmt.Printf("%#v\n%#v\n", b1.Header, b1.Payload)
}
