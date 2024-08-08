package mainchain

import (
	"container/heap"
	"fmt"
	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/sliceSystem/mainchain/dumbo1"
	"sync"
)

type EpochQueue []cleisthenes.Epoch

func (q EpochQueue) Len() int {
	return len(q)
}

func (q EpochQueue) Less(i, j int) bool {
	return (q)[i] < (q)[j]
}

func (q EpochQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}

func (q *EpochQueue) Push(x any) {
	*q = append(*q, x.(cleisthenes.Epoch))
}

func (q *EpochQueue) Pop() any {
	epoch := (*q)[q.Len()-1]
	*q = (*q)[:q.Len()-1]
	return epoch
}

func (q *EpochQueue) Add(epoch cleisthenes.Epoch) {
	heap.Push(q, epoch)
}

func (q *EpochQueue) Poll() (cleisthenes.Epoch, error) {
	if q.Len() <= 0 {
		return cleisthenes.Epoch(0), fmt.Errorf("epoch queue is empty")
	}

	x := heap.Pop(q)
	return x.(cleisthenes.Epoch), nil
}

func (q *EpochQueue) All() EpochQueue {
	return *q
}

func (q *EpochQueue) Init() {
	heap.Init(q)
}

type Dumbo1Repository struct {
	n         int
	f         int
	k         int
	batchSize int

	owner      *cleisthenes.Member
	tss        cleisthenes.Tss
	tpk        cleisthenes.Tpke
	resultChan cleisthenes.ResultSender

	lock      sync.Mutex
	dumbo1Map map[cleisthenes.Epoch]cleisthenes.Dumbo1
	//epochQueue *EpochQueue
}

func NewDumbo1Repository(
	n, f, k, batchSize int,
	owner *cleisthenes.Member,
	tss cleisthenes.Tss,
	tpk cleisthenes.Tpke,
	resultChan cleisthenes.ResultSender,
) *Dumbo1Repository {
	//epochQueue := &EpochQueue{}
	//epochQueue.Init()

	return &Dumbo1Repository{
		n:          n,
		f:          f,
		k:          k,
		batchSize:  batchSize,
		owner:      owner,
		tss:        tss,
		tpk:        tpk,
		resultChan: resultChan,
		lock:       sync.Mutex{},
		dumbo1Map:  make(map[cleisthenes.Epoch]cleisthenes.Dumbo1),
		//epochQueue: epochQueue,
	}
}

func (r *Dumbo1Repository) Find(epoch cleisthenes.Epoch) cleisthenes.Dumbo1 {
	r.lock.Lock()
	defer r.lock.Unlock()

	dumbo1, ok := r.dumbo1Map[epoch]
	if !ok {
		return nil
	}

	return dumbo1
}

func (r *Dumbo1Repository) Create(
	epoch cleisthenes.Epoch,
	memberMap *cleisthenes.MemberMap,
	broadcaster cleisthenes.Broadcaster,
) cleisthenes.Dumbo1 {
	dataChan := cleisthenes.NewDataChannel(r.f * 100)
	batchChan := cleisthenes.NewBatchChannel(r.batchSize)
	binChan := cleisthenes.NewBinaryChannel(r.n * 100)

	return dumbo1.New(
		epoch,
		r.n,
		r.f,
		*r.owner,
		memberMap,
		dumbo1.NewDefaultD1ACSFactory(
			r.n,
			r.f,
			r.k,
			*r.owner,
			memberMap,
			// CE委员会使用的门限签名实例
			r.tss,
			r.tpk,
			dataChan,
			dataChan,
			binChan,
			binChan,
			batchChan,
			broadcaster,
		),
		//&tpke.MockTpke{},
		nil,
		broadcaster,
		batchChan,
		r.resultChan,
	)
}

func (r *Dumbo1Repository) Save(epoch cleisthenes.Epoch, dumbo1 cleisthenes.Dumbo1) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	_, ok := r.dumbo1Map[epoch]
	if ok {
		return fmt.Errorf("dumbo1 instance already exist with epoch [%d]", epoch)
	}
	r.dumbo1Map[epoch] = dumbo1
	//r.epochQueue.Add(epoch)
	return nil
}

func (r *Dumbo1Repository) DeleteEpochLessEqual(epoch cleisthenes.Epoch) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if epoch < 0 {
		return
	}

	for e, _ := range r.dumbo1Map {
		if e <= epoch {
			r.dumbo1Map[e].Close()
			r.dumbo1Map[e] = nil
			delete(r.dumbo1Map, e)
		}
	}
}

func (r *Dumbo1Repository) Size() int {
	r.lock.Lock()
	defer r.lock.Unlock()

	return len(r.dumbo1Map)
}
