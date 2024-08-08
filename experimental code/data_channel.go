package cleisthenes

import (
	"fmt"
	"sync"
)

type DataMessage struct {
	Member Member
	Data   []byte
}

type DataSender interface {
	Send(msg DataMessage)
}

type DataReceiver interface {
	Receive() <-chan DataMessage
}

type DataChannel struct {
	buffer chan DataMessage
}

func NewDataChannel(size int) *DataChannel {
	return &DataChannel{
		buffer: make(chan DataMessage, size),
	}
}

func (c *DataChannel) Send(msg DataMessage) {
	//fmt.Println("test12,RBC阶段结束，将msg放入DataChannel")
	//fmt.Printf("放入的msg=%v\n",msg)
	c.buffer <- msg
}

func (c *DataChannel) Receive() <-chan DataMessage {
	return c.buffer
}

type DataMessageRepository struct {
	lock       sync.RWMutex
	dataMsgMap map[Address]*DataMessage
}

func NewDataMessageRepository() *DataMessageRepository {
	return &DataMessageRepository{
		dataMsgMap: make(map[Address]*DataMessage),
	}
}

func (r *DataMessageRepository) Save(addr Address, dataMsg *DataMessage) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.dataMsgMap[addr] = dataMsg
	return nil
}

func (r *DataMessageRepository) Find(addr Address) (*DataMessage, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	dataMsg, ok := r.dataMsgMap[addr]
	if !ok {
		return nil, fmt.Errorf("dataMsg of addr(%v) is not found", addr)
	}
	return dataMsg, nil
}

func (r *DataMessageRepository) FindAll() []*DataMessage {
	r.lock.Lock()
	defer r.lock.Unlock()

	dataMsgList := make([]*DataMessage, 0)
	for _, dataMsg := range r.dataMsgMap {
		dataMsgList = append(dataMsgList, dataMsg)
	}
	return dataMsgList
}

func (r *DataMessageRepository) FindAllAndClear() []*DataMessage {
	r.lock.Lock()
	defer r.lock.Unlock()

	dataMsgList := make([]*DataMessage, 0)
	for key, dataMsg := range r.dataMsgMap {
		dataMsgList = append(dataMsgList, dataMsg)
		delete(r.dataMsgMap, key)
	}

	// 清空dataMsgMap
	r.dataMsgMap = nil
	r.dataMsgMap = make(map[Address]*DataMessage)

	return dataMsgList
}
