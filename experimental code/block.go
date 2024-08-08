package cleisthenes

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"github.com/bytedance/sonic"
	"time"
)

type Block struct {
	Header  *BlockHeader
	Payload *BlockPayload
}

type BlockHeader struct {
	Height    uint64
	PrevHash  string
	Root      string
	TimeStamp uint64
	Hash      string
}

type BlockPayload struct {
	Txs []interface{}
}

func NewBlock(height uint64, prevHash string) *Block {
	return &Block{
		Header: &BlockHeader{
			Height:   height,
			PrevHash: prevHash,
		},
		Payload: &BlockPayload{},
	}
}

func (b *Block) AddTxs(txs []interface{}) {
	b.Payload.Txs = append(b.Payload.Txs, txs...)
	// TODO 默克尔树
}

func (b *Block) calculateHash() {
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, b.Header.Height)
	bs := bytes.NewBuffer(heightBytes)

	prevHash, err := hex.DecodeString(b.Header.PrevHash)
	if err != nil {
		log.Fatal().Msgf("calculate hash failed, decode prev hash err:%s", err.Error())
	}
	bs.Write(prevHash)

	root, err := hex.DecodeString(b.Header.Root)
	if err != nil {
		log.Fatal().Msgf("calculate hash failed, decode root err:%s", err.Error())
	}
	bs.Write(root)

	timeStampBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timeStampBytes, b.Header.TimeStamp)
	bs.Write(timeStampBytes)

	hash := md5.Sum(bs.Bytes())
	b.Header.Hash = hex.EncodeToString(hash[:])
}

func (b *Block) Pack() {
	// 默克尔树构建
	//datas := make([]merkletree.Data, len(b.Payload.Txs))
	//for _, tx := range b.Payload.Txs {
	//	bs, _ := sonic.ConfigFastest.Marshal(tx)
	//	data := merkletree.NewData(bs)
	//	datas = append(datas, data)
	//}
	//tree, err := merkletree.New(datas)
	//if err != nil {
	//	log.Fatal().Msgf("build merkle tree failed, err:%s", err)
	//}
	//root := tree.MerkleRoot()
	//rootHex := hex.EncodeToString(root)

	rootHex := hex.EncodeToString([]byte{0x00})

	// 区块时间戳
	now := time.Now()

	// 区块头编写
	b.Header.Root = rootHex
	b.Header.TimeStamp = uint64(now.UnixMilli())

	b.calculateHash()
}

func (b *Block) Serialize() ([]byte, error) {
	bs, err := sonic.ConfigFastest.Marshal(b)
	if err != nil {
		return nil, err
	}
	return bs, nil
}

func DeSerialize(data []byte) (*Block, error) {
	var b Block
	err := sonic.ConfigFastest.Unmarshal(data, &b)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func CreateGenesis() *Block {
	b := &Block{
		Header: &BlockHeader{
			Height:   0,
			Hash:     "",
			PrevHash: "00000000000000000000000000000000",
			Root:     "00000000000000000000000000000000",
		},
		Payload: &BlockPayload{
			Txs: nil,
		},
	}

	b.calculateHash()

	return b
}
