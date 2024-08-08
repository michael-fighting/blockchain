package irbc

import (
	"errors"
	"github.com/DE-labtory/cleisthenes/sliceSystem/branch/rbc/merkletree"
	"sync"

	"github.com/DE-labtory/cleisthenes"
)

type (
	ValRequest struct {
		RootHash []byte
		Data     merkletree.Data
		RootPath merkletree.RootPath
		Indexes  []int64
	}

	EchoRequest struct {
		ValRequest
	}

	ReadyRequest struct {
		RootHash []byte
	}
)

var ErrNoIdMatchingRequest = errors.New("id is not found.")
var ErrInvalidReqType = errors.New("request is not matching with type.")

// It means it is abstracted as Request interface (cleisthenes/request.go)
func (r ValRequest) Recv()   {}
func (r EchoRequest) Recv()  {}
func (r ReadyRequest) Recv() {}

// Received request
type (
	ValReqRepository struct {
		lock sync.RWMutex
		recv map[cleisthenes.Address]*ValRequest
	}

	EchoReqRepository struct {
		lock sync.RWMutex
		recv map[cleisthenes.Address]*EchoRequest
	}

	ReadyReqRepository struct {
		lock sync.RWMutex
		recv map[cleisthenes.Address]*ReadyRequest
	}
)

func NewEchoReqRepository() *EchoReqRepository {

	return &EchoReqRepository{
		recv: make(map[cleisthenes.Address]*EchoRequest),
		lock: sync.RWMutex{},
	}
}

func (r *EchoReqRepository) Save(addr cleisthenes.Address, req cleisthenes.Request) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	echoReq, ok := req.(*EchoRequest)
	if !ok {
		return ErrInvalidReqType
	}
	r.recv[addr] = echoReq
	return nil
}

func (r *EchoReqRepository) Find(addr cleisthenes.Address) (cleisthenes.Request, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	_, ok := r.recv[addr]
	if !ok {
		return nil, ErrNoIdMatchingRequest
	}
	return r.recv[addr], nil

}

func (r *EchoReqRepository) FindAll() []cleisthenes.Request {
	r.lock.Lock()
	defer r.lock.Unlock()

	reqList := make([]cleisthenes.Request, 0)
	for _, request := range r.recv {
		reqList = append(reqList, request)
	}
	return reqList
}

func NewValReqRepository() *ValReqRepository {
	return &ValReqRepository{
		recv: make(map[cleisthenes.Address]*ValRequest),
		lock: sync.RWMutex{},
	}
}

func (r *ValReqRepository) Save(addr cleisthenes.Address, req cleisthenes.Request) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	valReq, ok := req.(*ValRequest)
	if !ok {
		return ErrInvalidReqType
	}
	r.recv[addr] = valReq
	return nil
}

func (r *ValReqRepository) Find(addr cleisthenes.Address) (cleisthenes.Request, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	_, ok := r.recv[addr]
	if !ok {
		return nil, ErrNoIdMatchingRequest
	}
	return r.recv[addr], nil

}

func (r *ValReqRepository) FindAll() []cleisthenes.Request {
	r.lock.Lock()
	defer r.lock.Unlock()
	reqList := make([]cleisthenes.Request, 0)
	for _, request := range r.recv {
		reqList = append(reqList, request)
	}
	return reqList
}

func NewReadyReqRepository() *ReadyReqRepository {
	return &ReadyReqRepository{
		recv: make(map[cleisthenes.Address]*ReadyRequest),
		lock: sync.RWMutex{},
	}
}

func (r *ReadyReqRepository) Save(addr cleisthenes.Address, req cleisthenes.Request) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	readyReq, ok := req.(*ReadyRequest)
	if !ok {
		return ErrInvalidReqType
	}
	r.recv[addr] = readyReq
	return nil
}

func (r *ReadyReqRepository) Find(addr cleisthenes.Address) (cleisthenes.Request, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	_, ok := r.recv[addr]
	if !ok {
		return nil, ErrNoIdMatchingRequest
	}
	return r.recv[addr], nil

}

func (r *ReadyReqRepository) FindAll() []cleisthenes.Request {
	r.lock.Lock()
	defer r.lock.Unlock()

	reqList := make([]cleisthenes.Request, 0)
	for _, request := range r.recv {
		reqList = append(reqList, request)
	}
	return reqList
}
