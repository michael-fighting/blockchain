package cleisthenes

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"github.com/bytedance/sonic"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"sort"
	"sync"
)

// Batch is result of ACS set of contributions of
// at least n-f number of nodes.
// BatchMessage is used between ACS component and Honeybadger component.
//
// After ACS done its own task for its epoch send BatchMessage to
// Honeybadger, then Honeybadger decrypt batch message.
type BatchMessage struct {
	Epoch Epoch
	Batch map[Member][]byte
}

type BatchSender interface {
	Send(msg BatchMessage)
}

type BatchReceiver interface {
	Receive() <-chan BatchMessage
}

type BatchChannel struct {
	buffer chan BatchMessage
}

func NewBatchChannel(size int) *BatchChannel {
	return &BatchChannel{
		buffer: make(chan BatchMessage, size),
	}
}

func (c *BatchChannel) Send(msg BatchMessage) {
	c.buffer <- msg
}

// 谁调用receive的
func (c *BatchChannel) Receive() <-chan BatchMessage {
	return c.buffer
}

// ResultMessage is result of Honeybadger. When Honeybadger receive
// BatchMessage from ACS, it decrypt BatchMessage and use it to
// ResultMessage.Batch field.
//
// Honeybadger knows what epoch of ACS done its task. and Honeybadger
// use that information of epoch and decrypted batch to create ResultMessage
// then send it back to application
type ResultMessage struct {
	Epoch Epoch
	Batch map[string][]AbstractTx
	Hash  string
}

type ResultSender interface {
	Send(msg ResultMessage)
}

type ResultReceiver interface {
	Receive() <-chan ResultMessage
}

type ResultChannel struct {
	buffer chan ResultMessage
}

func NewResultChannel(size int) *ResultChannel {
	return &ResultChannel{
		buffer: make(chan ResultMessage, size),
	}
}

func (c *ResultChannel) Send(msg ResultMessage) {
	//fmt.Println("test40")
	c.buffer <- msg
}

func (c *ResultChannel) Receive() <-chan ResultMessage {
	//fmt.Println("test41")
	return c.buffer
}

type ResultMessageQueue []ResultMessage

func (q ResultMessageQueue) Len() int {
	return len(q)
}

func (q ResultMessageQueue) Less(i, j int) bool {
	return q[i].Epoch < q[j].Epoch
}

func (q ResultMessageQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}

//func (q *ResultMessageQueue) Push(x any) {
//	*q = append(*q, x.(ResultMessage))
//}
//
//func (q *ResultMessageQueue) Pop() any {
//	item := (*q)[q.Len()-1]
//	*q = (*q)[:q.Len()-1]
//	return item
//}
//
//func (q *ResultMessageQueue) Init() {
//	heap.Init(q)
//}
//
//func (q *ResultMessageQueue) Add(message ResultMessage) {
//	heap.Push(q, message)
//}
//
//func (q *ResultMessageQueue) Sort() ResultMessageQueue {
//	sort.Sort(q)
//	return *q
//}

type ResultMessageRepository struct {
	lock         sync.Mutex
	resultMsgMap map[Epoch]ResultMessageQueue
}

func NewResultMessageRepository() *ResultMessageRepository {
	return &ResultMessageRepository{
		lock:         sync.Mutex{},
		resultMsgMap: make(map[Epoch]ResultMessageQueue),
	}
}

//var fakeEpoch int = -1

func ResultMessageHash(message ResultMessage) string {
	//fakeEpoch++
	hashMap := make(map[string]string)
	keys := make([]string, 0)

	for addrStr, txs := range message.Batch {
		keys = append(keys, addrStr)
		// FIXME 这里不能使用sonic.ConfigFast 和 ConfigDefault，解析出来的bs会与其他节点解析出来的不一致
		bs, err := sonic.ConfigStd.Marshal(txs)
		if err != nil {
			log.Fatal().Msgf("marshal txs failed, err:%s", err.Error())
		}
		h := md5.Sum(bs)
		hashMap[addrStr] = hexutil.Encode(h[:])
		//fmt.Printf("fakeEpoch:%d, addr:%s, txs:%v bs:%v hash:%s\n", fakeEpoch, addrStr, txs, bs, hashMap[addrStr])
	}

	buff := bytes.NewBuffer([]byte{})
	sort.Strings(keys)
	for _, key := range keys {
		buff.WriteString(hashMap[key])
	}
	h := md5.Sum(buff.Bytes())
	return hexutil.Encode(h[:])
}

func ResultMessageQueueHash(messageQueue ResultMessageQueue) string {
	buff := bytes.NewBuffer([]byte{})
	//messageQueue.Sort()
	sort.Sort(messageQueue)
	for _, msg := range messageQueue {
		buff.WriteString(msg.Hash)
	}
	h := md5.Sum(buff.Bytes())
	return hexutil.Encode(h[:])
}

func (r *ResultMessageRepository) Save(epoch Epoch, message ResultMessage) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if _, ok := r.resultMsgMap[epoch]; !ok {
		r.resultMsgMap[epoch] = ResultMessageQueue{}
		//r.resultMsgMap[epoch].Init()
	}
	//r.resultMsgMap[epoch].Add(message)
	r.resultMsgMap[epoch] = append(r.resultMsgMap[epoch], message)
}

func (r *ResultMessageRepository) Find(epoch Epoch) (ResultMessageQueue, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	resultMsgQueue, ok := r.resultMsgMap[epoch]
	if !ok {
		return ResultMessageQueue{}, fmt.Errorf("resultMsgQueue of epoch(%v) not found", epoch)
	}

	return resultMsgQueue, nil
}

func (r *ResultMessageRepository) FindAndDelete(epoch Epoch) (ResultMessageQueue, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	resultMsgQueue, ok := r.resultMsgMap[epoch]
	if !ok {
		return ResultMessageQueue{}, fmt.Errorf("resultMsgQueue of epoch(%v) not found", epoch)
	}

	r.resultMsgMap[epoch] = nil
	delete(r.resultMsgMap, epoch)

	return resultMsgQueue, nil
}

func (r *ResultMessageRepository) Size() int {
	r.lock.Lock()
	defer r.lock.Unlock()

	return len(r.resultMsgMap)
}

// 当MessageType为Flag时，MainChainFinishFlag 有意义；
// 当MessageType为Ctrl时，CtrlInfo 有意义
// 当MessageType为Data时，Data 有意义
// 当MessageType为TransactionsType时，Data有意义
const (
	FlagType = iota
	CtrlInfoType
	DataType
	TransactionsType
	HashType
)

type MessageType int

type CtrlResultMessage struct {
	Epoch           Epoch
	MsgType         MessageType
	MainChainFinish bool
	MainChainTxCnt  int    // 统计用
	Data            []byte // 主链交易集合
	CtrlInfo        CtrlInfo
}

type CtrlInfo struct {
	CtrlMemberMap map[Address]CtrlSubmitInfo
}

type CtrlMemberInfo struct {
	Random int
	Hash   string
	TxSig  []byte
}

type CtrlSubmitInfo struct {
	MsgType             MessageType
	MainChainFinishFlag bool
	MainChainTxCnt      int    // 统计用
	Data                []byte // 主链交易集合
	CtrlMemberInfo      CtrlMemberInfo
}

type CtrlResultSender interface {
	Send(msg CtrlResultMessage)
}

type CtrlResultReceiver interface {
	Receive() <-chan CtrlResultMessage
}

type CtrlResultChannel struct {
	buffer chan CtrlResultMessage
}

func NewCtrlResultChannel(size int) *CtrlResultChannel {
	return &CtrlResultChannel{
		buffer: make(chan CtrlResultMessage, size),
	}
}

func (c *CtrlResultChannel) Send(msg CtrlResultMessage) {
	//fmt.Println("test40")
	c.buffer <- msg
}

func (c *CtrlResultChannel) Receive() <-chan CtrlResultMessage {
	//fmt.Println("test41")
	return c.buffer
}
