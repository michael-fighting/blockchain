package control

import (
	"fmt"
	"github.com/DE-labtory/cleisthenes"
	"sync"
)

type ControlRepository struct {
	lock       sync.Mutex
	controlMap map[cleisthenes.Epoch]Control
}

func NewControlRepository() *ControlRepository {
	return &ControlRepository{
		lock:       sync.Mutex{},
		controlMap: make(map[cleisthenes.Epoch]Control),
	}
}

func (r *ControlRepository) Find(epoch cleisthenes.Epoch) Control {
	r.lock.Lock()
	defer r.lock.Unlock()

	control, ok := r.controlMap[epoch]
	if !ok {
		return nil
	}

	return control
}

func (r *ControlRepository) Save(epoch cleisthenes.Epoch, control Control) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	_, ok := r.controlMap[epoch]
	if ok {
		return fmt.Errorf("control instance already exist with epoch [%d]", epoch)
	}
	r.controlMap[epoch] = control
	return nil
}

func (r *ControlRepository) Delete(epoch cleisthenes.Epoch) {
	r.lock.Lock()
	defer r.lock.Unlock()

	delete(r.controlMap, epoch)
}

func (r *ControlRepository) DeleteEpochLessEqual(epoch cleisthenes.Epoch) {
	r.lock.Lock()
	defer r.lock.Unlock()

	for e, control := range r.controlMap {
		if e <= epoch {
			control.Close()
			delete(r.controlMap, e)
		}
	}
}

func (r *ControlRepository) Clear() {
	r.lock.Lock()
	defer r.lock.Unlock()

	for epoch, control := range r.controlMap {
		control.Close()
		delete(r.controlMap, epoch)
	}
}

func (r *ControlRepository) Size() int {
	r.lock.Lock()
	defer r.lock.Unlock()

	return len(r.controlMap)
}

type HashRepository struct {
	lock    sync.Mutex
	hashMap map[cleisthenes.Address]string
}

func NewHashRepository() *HashRepository {
	return &HashRepository{
		lock:    sync.Mutex{},
		hashMap: make(map[cleisthenes.Address]string),
	}
}

func (r *HashRepository) Save(addr cleisthenes.Address, hash string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.hashMap[addr] = hash
}

func (r *HashRepository) FindAllByAddrs(addrs []cleisthenes.Address) []string {
	r.lock.Lock()
	defer r.lock.Unlock()

	hashArr := make([]string, 0)
	for _, addr := range addrs {
		hashArr = append(hashArr, r.hashMap[addr])
	}
	return hashArr
}
