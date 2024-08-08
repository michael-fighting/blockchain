package cleisthenes

import (
	"fmt"
	"sync"
)

type BinaryMessage struct {
	Member Member
	Binary Binary
}

type BinarySender interface {
	Send(msg BinaryMessage)
}

type BinaryReceiver interface {
	Receive() <-chan BinaryMessage
}

type BinaryChannel struct {
	buffer chan BinaryMessage
}

func NewBinaryChannel(size int) *BinaryChannel {
	return &BinaryChannel{
		buffer: make(chan BinaryMessage, size),
	}
}

func (c *BinaryChannel) Send(msg BinaryMessage) {
	c.buffer <- msg
}

func (c *BinaryChannel) Receive() <-chan BinaryMessage {
	return c.buffer
}

type BinaryMessageRepository struct {
	lock         sync.RWMutex
	binaryMsgMap map[Address]*BinaryMessage
}

func NewBinaryMessageRepository() *BinaryMessageRepository {
	return &BinaryMessageRepository{
		binaryMsgMap: make(map[Address]*BinaryMessage),
	}
}

func (r *BinaryMessageRepository) Save(addr Address, binaryMsg *BinaryMessage) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.binaryMsgMap[addr] = binaryMsg
	return nil
}

func (r *BinaryMessageRepository) Find(addr Address) (*BinaryMessage, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	binaryMsg, ok := r.binaryMsgMap[addr]
	if !ok {
		return nil, fmt.Errorf("binaryMsg of addr(%v) is not found", addr)
	}
	return binaryMsg, nil
}

func (r *BinaryMessageRepository) FindAll() []*BinaryMessage {
	r.lock.Lock()
	defer r.lock.Unlock()

	binaryMsgList := make([]*BinaryMessage, 0)
	for _, binaryMsg := range r.binaryMsgMap {
		binaryMsgList = append(binaryMsgList, binaryMsg)
	}
	return binaryMsgList
}

func (r *BinaryMessageRepository) FindAllAndClear() []*BinaryMessage {
	r.lock.Lock()
	defer r.lock.Unlock()

	binaryMsgList := make([]*BinaryMessage, 0)
	for key, binaryMsg := range r.binaryMsgMap {
		binaryMsgList = append(binaryMsgList, binaryMsg)
		delete(r.binaryMsgMap, key)
	}

	// 清空dataMsgMap
	r.binaryMsgMap = nil
	r.binaryMsgMap = make(map[Address]*BinaryMessage)

	return binaryMsgList
}
