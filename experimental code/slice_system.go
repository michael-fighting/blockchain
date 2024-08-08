package cleisthenes

import "sync"

type Slice struct {
	N     int
	F     int
	Addrs []Address
}

type FlagRespository struct {
	lock    sync.Mutex
	flagMap map[Epoch]struct{}
}

func NewFlagRepository() *FlagRespository {
	return &FlagRespository{
		lock:    sync.Mutex{},
		flagMap: make(map[Epoch]struct{}),
	}
}

func (r *FlagRespository) Save(epoch Epoch) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.flagMap[epoch] = struct{}{}
}

func (r *FlagRespository) Find(epoch Epoch) bool {
	r.lock.Lock()
	defer r.lock.Unlock()

	_, ok := r.flagMap[epoch]
	return ok
}

func (r *FlagRespository) Delete(epoch Epoch) {
	r.lock.Lock()
	defer r.lock.Unlock()

	delete(r.flagMap, epoch)
}
