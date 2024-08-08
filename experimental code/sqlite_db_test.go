package cleisthenes

import (
	"github.com/google/uuid"
	"testing"
)

func TestWriteBlock(t *testing.T) {
	block := NewBlock(1, "1f")
	var txs []interface{}
	for i := 0; i < 100; i++ {
		tx := map[string]interface{}{
			"id": uuid.New().String(),
		}
		txs = append(txs, tx)
	}
	block.AddTxs(txs)
	block.Pack()
	bs, _ := block.Serialize()
	WriteBlock(1, bs)
}

func TestWriteIndex(t *testing.T) {
	var txs []interface{}
	for i := 0; i < 100; i++ {
		tx := map[string]interface{}{
			"id": uuid.New().String(),
		}
		txs = append(txs, tx)
	}
	WriteIndex(1, txs)
}
