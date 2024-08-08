package acs

import (
	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
	"sync"
)

type CE interface {
	HandleInput(epoch cleisthenes.Epoch) error
	HandleMessage(sender cleisthenes.Member, msg *pb.Message_Ce) error
	Close()
}

type CMISRepository struct {
	lock    sync.RWMutex
	cmisSet map[cleisthenes.Member]struct{}
}

func NewCMISRepository() *CMISRepository {
	return &CMISRepository{
		lock:    sync.RWMutex{},
		cmisSet: make(map[cleisthenes.Member]struct{}),
	}
}

func (c *CMISRepository) Save(mem cleisthenes.Member) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.cmisSet[mem] = struct{}{}
}

func (c *CMISRepository) Exist(mem cleisthenes.Member) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	_, ok := c.cmisSet[mem]
	return ok
}

func (c *CMISRepository) Size() int {
	c.lock.Lock()
	defer c.lock.Unlock()
	return len(c.cmisSet)
}

func (c *CMISRepository) Members() []cleisthenes.Member {
	c.lock.Lock()
	defer c.lock.Unlock()

	members := make([]cleisthenes.Member, 0)
	for member, _ := range c.cmisSet {
		members = append(members, member)
	}
	return members
}
