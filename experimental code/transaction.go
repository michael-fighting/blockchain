package cleisthenes

import "sync"

type Transaction interface{}

type Data interface{}

type AbstractTx interface{}

// TODO: should be removed, after ACS merged
// TODO: ACS should use BatchMessage instead
type Batch struct {
	// TxList is a transaction set of batch which is polled from queue
	txList []Transaction
}

// TxList is a function returns the transaction list on batch
func (batch *Batch) TxList() []Transaction {
	return batch.txList
}

type Contribution struct {
	TxList []Transaction
}

type ContributionRepo struct {
	lock   sync.Mutex
	conMap map[Epoch][]*Contribution
}

func NewContributionRepo() *ContributionRepo {
	return &ContributionRepo{
		lock:   sync.Mutex{},
		conMap: make(map[Epoch][]*Contribution),
	}
}

func (r *ContributionRepo) Save(epoch Epoch, contribution *Contribution) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.conMap[epoch] = append(r.conMap[epoch], contribution)
}

func (r *ContributionRepo) FindContributionsByEpochAndDelete(epoch Epoch) []*Contribution {
	r.lock.Lock()
	defer r.lock.Unlock()

	res := r.conMap[epoch]
	delete(r.conMap, epoch)
	return res
}

func (r *ContributionRepo) Size() int {
	r.lock.Lock()
	defer r.lock.Unlock()

	cnt := 0
	for epoch, _ := range r.conMap {
		epoch++
		cnt++
	}
	return cnt
}
