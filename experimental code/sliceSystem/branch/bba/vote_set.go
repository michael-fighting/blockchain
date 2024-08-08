package bba

import (
	"sync"

	"github.com/DE-labtory/cleisthenes"
)

type voteSet struct {
	lock  sync.RWMutex
	items map[cleisthenes.Vote]bool
}

func newVoteSet() *voteSet {
	return &voteSet{
		lock:  sync.RWMutex{},
		items: make(map[cleisthenes.Vote]bool),
	}
}

func (s *voteSet) exist(vote cleisthenes.Vote) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	_, ok := s.items[vote]
	return ok
}

func (s *voteSet) union(vote cleisthenes.Vote) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.items[vote] = true
}

func (s *voteSet) toList() []cleisthenes.Vote {
	s.lock.Lock()
	defer s.lock.Unlock()

	result := make([]cleisthenes.Vote, 0)
	for vote, exist := range s.items {
		if !exist {
			continue
		}
		result = append(result, vote)
	}
	return result
}
