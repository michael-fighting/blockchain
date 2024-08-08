package cleisthenes

import (
	"github.com/DE-labtory/cleisthenes/pb"
	"sync"
)

type Request interface {
	Recv()
}

type RequestRepository interface {
	Save(addr Address, req Request) error
	Find(addr Address) (Request, error)
	FindAll() []Request
}

type ReqRepo struct {
	lock   sync.Mutex
	reqMap map[Epoch][]pb.Message
}

func NewReqRepo() *ReqRepo {
	return &ReqRepo{
		lock:   sync.Mutex{},
		reqMap: make(map[Epoch][]pb.Message),
	}
}

func (r *ReqRepo) Save(epoch Epoch, req pb.Message) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.reqMap[epoch] == nil {
		r.reqMap[epoch] = make([]pb.Message, 0)
	}
	r.reqMap[epoch] = append(r.reqMap[epoch], req)
}

func (r *ReqRepo) Find(epoch Epoch) []pb.Message {
	r.lock.Lock()
	defer r.lock.Unlock()

	msgs := make([]pb.Message, 0)
	for _, msg := range r.reqMap[epoch] {
		msgs = append(msgs, msg)
	}

	return msgs
}

func (r *ReqRepo) FindMsgsByEpochAndDelete(epoch Epoch) []pb.Message {
	r.lock.Lock()
	defer r.lock.Unlock()

	msgs := make([]pb.Message, 0)
	for _, msg := range r.reqMap[epoch] {
		msgs = append(msgs, msg)
	}
	delete(r.reqMap, epoch)
	r.reqMap[epoch] = make([]pb.Message, 0)

	return msgs
}

func (r *ReqRepo) Size() int {
	r.lock.Lock()
	defer r.lock.Unlock()

	return len(r.reqMap)
}
