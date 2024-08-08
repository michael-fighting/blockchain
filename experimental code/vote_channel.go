package cleisthenes

import (
	"fmt"
	"sync"
)

type VoteMessage struct {
	Member Member
	Vote   Vote
}

type VoteSender interface {
	Send(msg VoteMessage)
}

type VoteReceiver interface {
	Receive() <-chan VoteMessage
}

type VoteChannel struct {
	buffer chan VoteMessage
}

func NewVoteChannel(size int) *VoteChannel {
	return &VoteChannel{
		buffer: make(chan VoteMessage, size),
	}
}

func (c *VoteChannel) Send(msg VoteMessage) {
	c.buffer <- msg
}

func (c *VoteChannel) Receive() <-chan VoteMessage {
	return c.buffer
}

type VoteMessageRepository struct {
	lock       sync.RWMutex
	voteMsgMap map[Address]*VoteMessage
}

func NewVoteMessageRepository() *VoteMessageRepository {
	return &VoteMessageRepository{
		voteMsgMap: make(map[Address]*VoteMessage),
	}
}

func (r *VoteMessageRepository) Save(addr Address, voteMsg *VoteMessage) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.voteMsgMap[addr] = voteMsg
	return nil
}

func (r *VoteMessageRepository) Find(addr Address) (*VoteMessage, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	voteMsg, ok := r.voteMsgMap[addr]
	if !ok {
		return nil, fmt.Errorf("voteMsg of addr(%v) is not found", addr)
	}
	return voteMsg, nil
}

func (r *VoteMessageRepository) FindAll() []*VoteMessage {
	r.lock.Lock()
	defer r.lock.Unlock()

	voteMsgList := make([]*VoteMessage, 0)
	for _, voteMsg := range r.voteMsgMap {
		voteMsgList = append(voteMsgList, voteMsg)
	}
	return voteMsgList
}

func (r *VoteMessageRepository) FindAllAndClear() []*VoteMessage {
	r.lock.Lock()
	defer r.lock.Unlock()

	voteMsgList := make([]*VoteMessage, 0)
	for key, voteMsg := range r.voteMsgMap {
		voteMsgList = append(voteMsgList, voteMsg)
		delete(r.voteMsgMap, key)
	}

	// 清空dataMsgMap
	r.voteMsgMap = nil
	r.voteMsgMap = make(map[Address]*VoteMessage)

	return voteMsgList
}
