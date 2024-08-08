package dumbo1

import (
	"errors"
	"github.com/DE-labtory/cleisthenes"
	"sync"
)

type (
	DecShareRequest struct {
		Sender   cleisthenes.Address
		DecShare cleisthenes.DecryptionShare
	}
)

var ErrNoIdMatchingRequest = errors.New("id is not found.")
var ErrInvalidReqType = errors.New("request is not matching with type.")

func (r DecShareRequest) Recv() {}

type (
	DecShareReqRepsitory struct {
		lock sync.RWMutex
		recv map[cleisthenes.Address]*DecShareRequest
	}
)

func newDecShareReqRepository() *DecShareReqRepsitory {
	return &DecShareReqRepsitory{
		recv: make(map[cleisthenes.Address]*DecShareRequest),
		lock: sync.RWMutex{},
	}
}

func (r *DecShareReqRepsitory) Save(addr cleisthenes.Address, req cleisthenes.Request) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	decShareReq, ok := req.(*DecShareRequest)
	if !ok {
		return ErrInvalidReqType
	}
	r.recv[addr] = decShareReq
	return nil
}
func (r *DecShareReqRepsitory) Find(addr cleisthenes.Address) (cleisthenes.Request, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	_, ok := r.recv[addr]
	if !ok {
		return nil, ErrNoIdMatchingRequest
	}
	return r.recv[addr], nil
}

func (r *DecShareReqRepsitory) FindAll() []cleisthenes.Request {
	r.lock.Lock()
	defer r.lock.Unlock()

	reqList := make([]cleisthenes.Request, 0)
	for _, request := range r.recv {
		reqList = append(reqList, request)
	}
	return reqList
}
