package cleisthenes

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/DE-labtory/iLogger"
)

type TxQueue interface {
	Push(tx Transaction)
	Pushs(txs []Transaction)
	Poll() (Transaction, error)
	Len() int
}

// memQueue defines transaction FIFO memQueue
type MemTxQueue struct {
	txs []Transaction
	sync.RWMutex
}

// EmptyQueErr is for calling peek(), poll() when memQueue is empty.
type emptyQueErr struct{}

func (e *emptyQueErr) Error() string {
	return "memQueue is empty."
}

// indexBoundaryErr is for calling indexOf
type indexBoundaryErr struct {
	queSize int
	want    int
}

func (e *indexBoundaryErr) Error() string {
	return fmt.Sprintf("index is larger than memQueue size.\n"+
		"memQueue size : %d, you want : %d", e.queSize, e.want)
}

func NewTxQueue() *MemTxQueue {
	return &MemTxQueue{
		txs: []Transaction{},
	}
}

// empty checks whether memQueue is empty or not.
func (q *MemTxQueue) empty() bool {
	return len(q.txs) == 0
}

// peek returns first element of memQueue, but not erase it.
func (q *MemTxQueue) peek() (Transaction, error) {
	if q.empty() {
		return nil, &emptyQueErr{}
	}

	return q.txs[0], nil
}

// Poll returns first element of memQueue, and erase it.
func (q *MemTxQueue) Poll() (Transaction, error) {
	q.Lock()
	defer q.Unlock()

	ret, err := q.peek()
	if err != nil {
		return nil, err
	}

	q.txs = q.txs[1:]
	return ret, nil
}

// len returns size of memQueue
func (q *MemTxQueue) Len() int {
	q.Lock()
	defer q.Unlock()
	return len(q.txs)
}

// at returns element of index in memQueue
func (q *MemTxQueue) at(index int) (Transaction, error) {
	if index >= q.Len() {
		return nil, &indexBoundaryErr{
			queSize: q.Len(),
			want:    index,
		}
	}
	return q.txs[index], nil
}

// Push adds transaction to memQueue.
func (q *MemTxQueue) Push(tx Transaction) {
	q.Lock()
	defer q.Unlock()

	q.txs = append(q.txs, tx)
}

func (q *MemTxQueue) Pushs(txs []Transaction) {
	q.Lock()
	defer q.Unlock()

	q.txs = append(q.txs, txs...)
}

type TxValidator func(Transaction) bool

// TxQueueManager manages transaction queue. It receive transaction from client
// and TxQueueManager have its own policy to propose contribution to honeybadger
type TxQueueManager interface {
	AddTransaction(tx Transaction) error
	AddTransactions(txs []Transaction) error
}

type Submitter interface {
	Submit(tx Transaction) error
	SubmitTxs(txs []Transaction) error
}
type ManualSubmitter interface {
	SubmitContribution(txs []Transaction)
}
type ManualTxQueueManager struct {
	txQueue   TxQueue
	submitter ManualSubmitter

	contributionSize int
	stopFlag         int32
	closeChan        chan struct{}
	tryInterval      time.Duration
	txValidator      TxValidator
	onConsensusFlag  int32
	lock             sync.Mutex
}

func NewManualTxQueueManager(
	queue TxQueue,
	submitter ManualSubmitter,
	contributionSize int,
	tryInterval time.Duration,
	validator TxValidator,
) *ManualTxQueueManager {
	m := &ManualTxQueueManager{
		txQueue:          queue,
		submitter:        submitter,
		contributionSize: contributionSize,
		closeChan:        make(chan struct{}),
		tryInterval:      tryInterval,
		txValidator:      validator,
		lock:             sync.Mutex{},
	}

	return m
}

func (m *ManualTxQueueManager) AddTransaction(tx Transaction) error {
	if !m.txValidator(tx) {
		return errors.New(fmt.Sprintf("error invalid transaction: %v", tx))
	}
	m.txQueue.Push(tx)
	return nil
}

func (m *ManualTxQueueManager) AddTransactions(txs []Transaction) error {
	for _, tx := range txs {
		if !m.txValidator(tx) {
			return errors.New(fmt.Sprintf("error invalid transaction: %v", tx))
		}
	}
	m.txQueue.Pushs(txs)
	return nil
}

func (m *ManualTxQueueManager) onConsensus() bool {
	return atomic.LoadInt32(&(m.onConsensusFlag)) == int32(1)
}

func (m *ManualTxQueueManager) startConsensus() bool {
	return atomic.CompareAndSwapInt32(&m.onConsensusFlag, int32(0), int32(1))
}

func (m *ManualTxQueueManager) FinConsensus() {
	atomic.CompareAndSwapInt32(&m.onConsensusFlag, int32(1), int32(0))
}

func (m *ManualTxQueueManager) TryPropose() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.onConsensus() {
		return fmt.Errorf("is on consensus")
	}

	if m.txQueue.Len() < int(m.contributionSize) {
		return fmt.Errorf("txQueue len(%v) is less than contributionSize(%v)", m.txQueue.Len(), m.contributionSize)
	}

	if !m.startConsensus() {
		return fmt.Errorf("start propose failed, reason: already started")
	}

	// 总计时开始
	if BranchTps.StartTime.IsZero() {
		BranchTps.StartTime = time.Now()
		log.Info().Msgf("计时开始%s\n", BranchTps.StartTime.String())
	}
	if MainTps.StartTime.IsZero() {
		MainTps.StartTime = time.Now()
		log.Info().Msgf("计时开始%s\n", MainTps.StartTime.String())
	}

	contribution, err := m.createContribution()
	if err != nil {
		return fmt.Errorf("create contribution failed, err:%s", err.Error())
	}

	m.submitter.SubmitContribution(contribution)
	return nil
}

func (m *ManualTxQueueManager) createContribution() ([]Transaction, error) {
	contribution := make([]Transaction, m.contributionSize)
	var err error
	for i := 0; i < m.contributionSize; i++ {
		contribution[i], err = m.txQueue.Poll()
		if err != nil {
			return nil, err
		}
	}
	return contribution, nil
}

type DefaultTxQueueManager struct {
	txQueue TxQueue
	dumbo1  Dumbo1

	stopFlag int32

	contributionSize int
	batchSize        int

	closeChan chan struct{}

	tryInterval time.Duration

	txValidator TxValidator
}

func NewDefaultTxQueueManager(
	txQueue TxQueue,
	dumbo1 Dumbo1,
	contributionSize int,
	batchSize int,

	// tryInterval is time interval to try create contribution
	// then propose to honeybadger component
	tryInterval time.Duration,

	txVerifier TxValidator,
) *DefaultTxQueueManager {
	m := &DefaultTxQueueManager{
		txQueue:          txQueue,
		dumbo1:           dumbo1,
		contributionSize: contributionSize,
		batchSize:        batchSize,

		closeChan:   make(chan struct{}),
		tryInterval: tryInterval,
		txValidator: txVerifier,
	}

	go m.runContributionProposeRoutine()

	return m
}

func (m *DefaultTxQueueManager) AddTransaction(tx Transaction) error {
	if !m.txValidator(tx) {
		return errors.New(fmt.Sprintf("error invalid transaction: %v", tx))
	}
	m.txQueue.Push(tx)
	return nil
}

func (m *DefaultTxQueueManager) AddTransactions(txs []Transaction) error {
	for _, tx := range txs {
		if !m.txValidator(tx) {
			return errors.New(fmt.Sprintf("error invalid transaction: %v", tx))
		}
	}
	m.txQueue.Pushs(txs)
	return nil
}

func (m *DefaultTxQueueManager) Close() {
	if first := atomic.CompareAndSwapInt32(&m.stopFlag, int32(0), int32(1)); !first {
		return
	}
	m.closeChan <- struct{}{}
	<-m.closeChan
}

func (m *DefaultTxQueueManager) toDie() bool {
	return atomic.LoadInt32(&(m.stopFlag)) == int32(1)
}

// runContributionProposeRoutine tries to propose contribution every its "tryInterval"
// And if honeybadger is on consensus, it waits
func (m *DefaultTxQueueManager) runContributionProposeRoutine() {
	for !m.toDie() {
		if !m.dumbo1.OnConsensus() {
			//iLogger.Infof(nil, "addr: %s, value: %v", out.addr, out.value)
			m.tryPropose()
		}
		iLogger.Debugf(nil, "[DefaultTxQueueManager] try to propose contribution...")
		time.Sleep(m.tryInterval)
	}
	fmt.Println("queuemanager退出!")
}

// tryPropose create contribution and send it to honeybadger only when
// transaction queue size is larger than batch size
func (m *DefaultTxQueueManager) tryPropose() error {
	//fmt.Println("test1\n")
	//fmt.Printf("test1, m.txQueue.Len()=%v,  m.contributionSize=%v\n", m.txQueue.Len(), m.contributionSize)

	if m.txQueue.Len() < int(m.contributionSize) {
		return nil
	}

	// 单次计时开始
	BranchTps.EnterTime = time.Now()
	log.Info().Msgf("开始打包交易处理，进入时间：%s\n", BranchTps.EnterTime.String())
	MainTps.EnterTime = time.Now()
	log.Info().Msgf("开始打包交易处理，进入时间：%s\n", MainTps.EnterTime.String())
	// 总计时开始
	if BranchTps.StartTime.IsZero() {
		BranchTps.StartTime = time.Now()
		log.Info().Msgf("计时开始%s\n", BranchTps.StartTime.String())
	}
	if MainTps.StartTime.IsZero() {
		MainTps.StartTime = time.Now()
		log.Info().Msgf("计时开始%s\n", MainTps.StartTime.String())
	}

	contribution, err := m.createContribution()
	if err != nil {
		return err
	}

	log.Debug().Msgf("txQueueSize:%d", m.txQueue.Len())
	m.dumbo1.HandleContribution(contribution)

	return nil
}

// Create batch polling random transaction in queue
// One caution is that caller of this function should ensure transaction queue
// size is larger than contribution size
func (m *DefaultTxQueueManager) createContribution() (Contribution, error) {
	//batchSize为本地交易池大小
	candidate, err := m.loadCandidateTx(min(m.batchSize, m.txQueue.Len()))
	if err != nil {
		return Contribution{}, err
	}

	return Contribution{
		TxList: m.selectRandomTx(candidate, m.contributionSize),
	}, nil
}

// loadCandidateTx is a function that returns candidate transactions which could be
// included into contribution from the queue
func (m *DefaultTxQueueManager) loadCandidateTx(candidateSize int) ([]Transaction, error) {
	candidate := make([]Transaction, candidateSize)
	var err error
	for i := 0; i < candidateSize; i++ {
		candidate[i], err = m.txQueue.Poll()

		if err != nil {
			return nil, err
		}
	}
	return candidate, nil
}

// selectRandomTx is a function that returns transactions which is randomly selected from input transactions
func (m *DefaultTxQueueManager) selectRandomTx(candidate []Transaction, selectSize int) []Transaction {
	batch := make([]Transaction, selectSize)

	for i := 0; i < selectSize; i++ {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn(len(candidate))

		batch[i] = candidate[idx]
		candidate = append(candidate[:idx], candidate[idx+1:]...)
	}
	//将剩余未选中的交易放到交易队列中
	for _, leftover := range candidate {
		m.txQueue.Push(leftover)
	}
	return batch
}

func min(x int, y int) int {
	if x < y {
		return x
	} else {
		return y
	}
}
