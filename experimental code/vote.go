package cleisthenes

import "sync"

type Vote = byte

const (
	VZero Vote = 0
	VOne       = 1
	VTwo       = 2
)

func ToBinary(v Vote) Binary {
	if v == VOne {
		return One
	}

	return Zero
}

func ToVote(b Binary) Vote {
	if b == One {
		return VOne
	}

	return VZero
}

type VoteState struct {
	sync.RWMutex
	value     Vote
	undefined bool
}

func (b *VoteState) Set(bin Vote) {
	b.Lock()
	defer b.Unlock()

	b.value = bin
	b.undefined = false
}

func (b *VoteState) Value() Vote {
	b.Lock()
	defer b.Unlock()

	return b.value
}

func (b *VoteState) Undefined() bool {
	b.Lock()
	defer b.Unlock()

	return b.undefined
}

func NewVoteState() *VoteState {
	return &VoteState{
		RWMutex:   sync.RWMutex{},
		undefined: true,
	}
}
