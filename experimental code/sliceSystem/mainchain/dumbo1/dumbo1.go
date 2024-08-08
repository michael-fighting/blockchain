package dumbo1

import (
	"errors"
	"fmt"
	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/DE-labtory/iLogger"
	"github.com/bytedance/sonic"
	"github.com/golang/protobuf/ptypes"
	"github.com/rs/zerolog"
	"sync"
	"sync/atomic"
	"time"
)

type Epoch struct {
	lock  sync.RWMutex
	value cleisthenes.Epoch
}

func NewEpoch(value cleisthenes.Epoch) *Epoch {
	return &Epoch{
		lock:  sync.RWMutex{},
		value: value,
	}
}

func (e *Epoch) up() {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.value++
}

func (e *Epoch) val() cleisthenes.Epoch {
	e.lock.Lock()
	defer e.lock.Unlock()
	value := e.value
	return value
}

type contributionBuffer struct {
	lock  sync.RWMutex
	value []cleisthenes.Contribution
}

func newContributionBuffer() *contributionBuffer {
	return &contributionBuffer{
		lock:  sync.RWMutex{},
		value: make([]cleisthenes.Contribution, 0),
	}
}

func (cb *contributionBuffer) add(buffer cleisthenes.Contribution) {
	cb.lock.Lock()
	defer cb.lock.Unlock()
	cb.value = append(cb.value, buffer)
}

func (cb *contributionBuffer) one() cleisthenes.Contribution {
	cb.lock.Lock()
	defer cb.lock.Unlock()
	buffer := cb.value[0]
	cb.value = append(cb.value[:0], cb.value[1:]...)
	return buffer
}

func (cb *contributionBuffer) empty() bool {
	cb.lock.Lock()
	defer cb.lock.Unlock()
	if len(cb.value) != 0 {
		return false
	}
	return true
}

func (cb *contributionBuffer) size() int {
	cb.lock.Lock()
	defer cb.lock.Unlock()
	return len(cb.value)
}

func (cb *contributionBuffer) clear() {
	cb.lock.Lock()
	defer cb.lock.Unlock()
	cb.value = make([]cleisthenes.Contribution, 0)
}

const initialEpoch = 0

type request struct {
	proposer cleisthenes.Member
	sender   cleisthenes.Member
	data     *pb.Message
	err      chan error
}

type BatchResult struct {
	decShareReqRepositorys map[cleisthenes.Member]*DecShareReqRepsitory
	encryptTxMap           map[cleisthenes.Member][]byte
	unCompleteTxMap        map[cleisthenes.Member]*cleisthenes.BinaryState
	needCompleteCount      int
	decryptedBatch         map[string][]cleisthenes.AbstractTx
	finished               *cleisthenes.BinaryState

	start time.Time
}

type DecryptReq struct {
	Epoch    cleisthenes.Epoch
	Proposer cleisthenes.Member
}

type DecryptedTx struct {
	Epoch cleisthenes.Epoch
	Addr  string
	Txs   []cleisthenes.AbstractTx
}

type Dumbo1 struct {
	lock          sync.RWMutex
	acsRepository *d1acsRepository

	n, f int

	memberMap     *cleisthenes.MemberMap
	txQueue       cleisthenes.TxQueue
	broadcaster   cleisthenes.Broadcaster
	resultSender  cleisthenes.ResultSender
	batchReceiver cleisthenes.BatchReceiver
	acsFactory    D1ACSFactory

	tpk cleisthenes.Tpke

	epoch *Epoch
	done  *cleisthenes.BinaryState
	owner cleisthenes.Member

	batchResultMap map[cleisthenes.Epoch]*BatchResult

	contributionBuffer *contributionBuffer
	contributionChan   chan struct{}
	closeChan          chan struct{}
	decryptChan        chan DecryptReq
	reqChan            chan request
	batchChan          chan DecryptedTx

	stopFlag        int32
	onConsensusFlag int32

	decryptLock sync.RWMutex

	reqChanLock sync.RWMutex

	start    time.Time
	acsStart time.Time

	log zerolog.Logger
}

func New(
	epoch cleisthenes.Epoch,
	n, f int,
	owner cleisthenes.Member,
	memberMap *cleisthenes.MemberMap,
	acsFactory D1ACSFactory,
	tpk cleisthenes.Tpke,
	broadcaster cleisthenes.Broadcaster,
	batchReceiver cleisthenes.BatchReceiver,
	resultSender cleisthenes.ResultSender,
) *Dumbo1 {
	db1 := &Dumbo1{
		n:             n,
		f:             f,
		lock:          sync.RWMutex{},
		acsRepository: newD1ACSRepository(),
		owner:         owner,
		memberMap:     memberMap,
		txQueue:       cleisthenes.NewTxQueue(),
		acsFactory:    acsFactory,
		broadcaster:   broadcaster,
		batchReceiver: batchReceiver,
		resultSender:  resultSender,

		tpk: tpk,

		epoch: NewEpoch(epoch),
		done:  cleisthenes.NewBinaryState(),

		contributionBuffer: newContributionBuffer(),
		contributionChan:   make(chan struct{}, 4),
		closeChan:          make(chan struct{}),
		decryptChan:        make(chan DecryptReq, n*10),
		batchChan:          make(chan DecryptedTx, n*10),
		//FIXME reqChan的大小暂时设置
		reqChan: make(chan request, n*n*10),

		batchResultMap: make(map[cleisthenes.Epoch]*BatchResult),
		log:            cleisthenes.NewLoggerWithHead("MDUMBO1"),
	}

	db1.initLog()

	go db1.run()

	return db1
}

func (db1 *Dumbo1) initLog() {
	var logPrefix zerolog.HookFunc
	logPrefix = func(e *zerolog.Event, level zerolog.Level, message string) {
		e.Uint64("Epoch", uint64(db1.epoch.val())).
			Str("owner", db1.owner.Address.String()).
			Bool("OnConsensus", db1.OnConsensus()).
			Str("memberMap", fmt.Sprintf("%v", db1.memberMap.Members()))
	}
	db1.log = db1.log.Hook(logPrefix)
}

func (db1 *Dumbo1) getBatchResult(epoch cleisthenes.Epoch) *BatchResult {
	batchResult, ok := db1.batchResultMap[epoch]
	if ok {
		return batchResult
	}

	db1.batchResultMap[epoch] = &BatchResult{
		decShareReqRepositorys: make(map[cleisthenes.Member]*DecShareReqRepsitory),
		encryptTxMap:           make(map[cleisthenes.Member][]byte),
		unCompleteTxMap:        make(map[cleisthenes.Member]*cleisthenes.BinaryState),
		decryptedBatch:         make(map[string][]cleisthenes.AbstractTx, 0),
		finished:               cleisthenes.NewBinaryState(),
	}

	for _, member := range db1.memberMap.Members() {
		db1.batchResultMap[epoch].decShareReqRepositorys[member] = newDecShareReqRepository()
		db1.batchResultMap[epoch].unCompleteTxMap[member] = cleisthenes.NewBinaryState()
	}

	return db1.batchResultMap[epoch]
}

func (db1 *Dumbo1) clearBatchResult(epoch cleisthenes.Epoch) {
	db1.batchResultMap[epoch] = nil
	delete(db1.batchResultMap, epoch)
}

func (db1 *Dumbo1) clearAllBatchResult() {
	for epoch, _ := range db1.batchResultMap {
		db1.batchResultMap[epoch] = nil
		delete(db1.batchResultMap, epoch)
	}
}

func (db1 *Dumbo1) HandleContribution(contribution cleisthenes.Contribution) {
	//将本地打包的交易batch放入contributionBuffer
	db1.contributionBuffer.add(contribution)
	db1.log.Debug().Msgf("开始尝试提交contribution到acs, bufSize:%d queueSize:%d", db1.contributionBuffer.size(), db1.txQueue.Len())
	if !db1.OnConsensus() {
		db1.contributionChan <- struct{}{}
	}
}

func (db1 *Dumbo1) HandleMessage(msg *pb.Message) error {
	//db1.log.Debug().Msgf("msg.Epoch:%v msg.Proposer:%v msg.Sender:%v", msg.Epoch, msg.Proposer, msg.Sender)
	addr, err := cleisthenes.ToAddress(msg.Proposer)
	if err != nil {
		return err
	}
	proposer, ok := db1.memberMap.Member(addr)
	if !ok {
		return errors.New(fmt.Sprintf("member not exist in member map: %s", addr.String()))
	}

	addr, err = cleisthenes.ToAddress(msg.Sender)
	if err != nil {
		return err
	}
	sender, ok := db1.memberMap.Member(addr)
	if !ok {
		return errors.New(fmt.Sprintf("member not exist in member map: %s", addr.String()))
	}
	req := request{
		proposer: proposer,
		sender:   sender,
		data:     msg,
		err:      make(chan error),
	}

	db1.reqChanLock.Lock()
	if db1.toDie() { // 如果dumbo1被关闭，根本不会执行到这里来
		db1.log.Debug().Msgf("db1 is closed, should not send req to reqChan")
		//close(db1.reqChan)
		db1.reqChanLock.Unlock()
		return nil
	}
	db1.reqChan <- req
	db1.reqChanLock.Unlock()
	return <-req.err
}

func (db1 *Dumbo1) handleAcsMessage(msg *pb.Message) error {
	a, err := db1.getACS(cleisthenes.Epoch(msg.Epoch))
	if err != nil {
		return err
	}

	addr, err := cleisthenes.ToAddress(msg.Sender)
	if err != nil {
		return err
	}

	member, ok := db1.memberMap.Member(addr)
	if !ok {
		return errors.New(fmt.Sprintf("member not exist in member map: %s", addr.String()))
	}

	return a.HandleMessage(member, msg)
}

func (db1 *Dumbo1) countDecShares(epoch cleisthenes.Epoch, proposer cleisthenes.Member) int {
	batchResult := db1.getBatchResult(epoch)
	return len(batchResult.decShareReqRepositorys[proposer].FindAll())
}

func (db1 *Dumbo1) decShareThreshold() int {
	return db1.f + 1
}

// handleDsRequest 处理到来的decShare消息
func (db1 *Dumbo1) handleDsRequest(epoch cleisthenes.Epoch, proposer, sender cleisthenes.Member, msg *pb.Message_Ds) error {
	// FIXME 正常情况一轮epoch开始后，历史epoch轮的已经结束，并且历史epoch轮的结果肯定已经出来，对于历史epoch，应该直接废弃掉
	if epoch < db1.epoch.val() {
		db1.log.Warn().Msgf("收到历史epoch:%d ds请求", epoch)
		return nil
	}

	decShare := cleisthenes.DecryptionShare{}
	for i := 0; i < 96; i++ {
		decShare[i] = msg.Ds.Payload[i]
	}

	req := &DecShareRequest{
		Sender:   sender.Address,
		DecShare: decShare,
	}

	batchResult := db1.getBatchResult(epoch)

	if err := batchResult.decShareReqRepositorys[proposer].Save(sender.Address, req); err != nil {
		return err
	}

	// FIXME 防重入，后续的f+2, f+3仍然会进入
	if db1.countDecShares(epoch, proposer) >= db1.decShareThreshold() {
		db1.decryptChan <- DecryptReq{
			Epoch:    epoch,
			Proposer: proposer,
		}
	}

	return nil
}

func (db1 *Dumbo1) muxRequest(proposer, sender cleisthenes.Member, msg *pb.Message) error {
	switch pl := msg.Payload.(type) {
	case *pb.Message_Ds:
		return db1.handleDsRequest(cleisthenes.Epoch(msg.Epoch), proposer, sender, pl)
	default:
		return db1.handleAcsMessage(msg)
	}
}

func (db1 *Dumbo1) propose(contribution cleisthenes.Contribution) error {
	a, err := db1.getACS(db1.epoch.val())
	if err != nil {
		return err
	}

	start := time.Now()
	//对数据进行门限加密，data为密文
	bytes, err := sonic.ConfigFastest.Marshal(contribution.TxList)
	if err != nil {
		return err
	}
	var data []byte
	if true {
		data = bytes
	} else {
		data, err = db1.tpk.Encrypt(bytes)
		if err != nil {
			return err
		}
	}
	end := time.Now()
	db1.log.Debug().Msgf("tpke enc time:%v", end.Sub(start))

	db1.acsStart = time.Now()
	return a.HandleInput(data)
}

// getACS returns D1ACS instance anyway. if D1ACS exist in repository for epoch
// then return it. otherwise create and save new D1ACS instance then return it
func (db1 *Dumbo1) getACS(epoch cleisthenes.Epoch) (D1ACS, error) {
	if db1.epoch.val() > epoch+10 {
		return nil, errors.New("too old epoch")
	}

	a, ok := db1.acsRepository.find(epoch)
	if ok {
		return a, nil
	}

	if epoch < db1.epoch.val() {
		db1.log.Warn().Msgf("acs for %d epoch has closed", epoch)
		return nil, fmt.Errorf("acs for %d epoch has closed", epoch)
	}

	a, err := db1.acsFactory.Create(epoch)
	if err != nil {
		return nil, err
	}
	if err := db1.acsRepository.save(epoch, a); err != nil {
		a, _ := db1.acsRepository.find(epoch)
		return a, nil
	}
	return a, nil
}

func (db1 *Dumbo1) run() {
	for !db1.toDie() {
		select {
		case <-db1.closeChan:
			db1.closeChan <- struct{}{}
			return
		// 打包进行propose
		case <-db1.contributionChan:
			if !db1.contributionBuffer.empty() && !db1.done.Value() && db1.startConsensus() {
				db1.start = time.Now()
				db1.log.Debug().Msg("开始提案")
				//从contributionBuffer拿出第一个contribution出来进行共识
				if err := db1.propose(db1.contributionBuffer.one()); err != nil {
					fmt.Printf("error in propose : %s\n", err.Error())
				}
			}
		// 本轮epoch共识的结果
		case batchMessage := <-db1.batchReceiver.Receive():
			end := time.Now()
			db1.log.Debug().Msgf("acs batch time:%v", end.Sub(db1.acsStart))

			if true {
				db1.finish(batchMessage)
				break
			}

			// 共识完成处理，后续进行解密
			if err := db1.handleBatchMessage(batchMessage); err != nil {
				iLogger.Debugf(nil, "error in handleBatchMessage : %s", err.Error())
			}
			// 如果decShare先到，batchMessage后到，当batchMessage到了后才能解密
			for proposer, _ := range batchMessage.Batch {
				db1.tryDecrypt(batchMessage.Epoch, proposer)
			}
		case req := <-db1.reqChan:
			// FIXME 这里可以不用管道，直接将acs的消息传递给acs处理，是否会快一点？
			req.err <- db1.muxRequest(req.proposer, req.sender, req.data)
		case decryptReq := <-db1.decryptChan:
			err := db1.tryDecrypt(decryptReq.Epoch, decryptReq.Proposer)
			if err != nil {
				db1.log.Warn().Msgf("decrypt batch message failed, err:%s", err.Error())
			}
		case decryptedTx := <-db1.batchChan:
			db1.collectTx(decryptedTx.Epoch, decryptedTx.Addr, decryptedTx.Txs)
			db1.tryFinish(decryptedTx.Epoch)
		}
	}
}

func (db1 *Dumbo1) collectTx(epoch cleisthenes.Epoch, addr string, txs []cleisthenes.AbstractTx) {
	batchResult := db1.getBatchResult(epoch)
	batchResult.decryptedBatch[addr] = txs
	batchResult.needCompleteCount--
}

func (db1 *Dumbo1) finish(batchMsg cleisthenes.BatchMessage) {
	decryptedBatch := make(map[string][]cleisthenes.AbstractTx, 0)
	for proposer, encryptedTx := range batchMsg.Batch {
		transactions := make([]cleisthenes.Transaction, 0)
		err := sonic.ConfigFastest.Unmarshal(encryptedTx, &transactions)
		if err != nil {
			db1.log.Fatal().Msgf("unmarshal tx list failed, err:%v", err)
			return
		}

		txs := make([]cleisthenes.AbstractTx, 0)
		for _, tx := range transactions {
			txs = append(txs, tx)
		}

		decryptedBatch[proposer.Address.String()] = txs
	}

	db1.resultSender.Send(cleisthenes.ResultMessage{
		Epoch: batchMsg.Epoch,
		Batch: decryptedBatch,
	})
	//db1.Close()
}

func (db1 *Dumbo1) tryFinish(epoch cleisthenes.Epoch) {
	db1.log.Debug().Msg("try finish")
	batchResult := db1.getBatchResult(epoch)
	if batchResult.finished.Value() {
		return
	}
	for k, v := range batchResult.unCompleteTxMap {
		if v.Value() {
			db1.log.Info().Str("proposer", k.Address.String()).Msg("对应的proposer暂未完成")
			return
		}
	}
	if batchResult.needCompleteCount != 0 {
		return
	}

	batchResult.finished.Set(true)
	db1.resultSender.Send(cleisthenes.ResultMessage{
		Epoch: epoch,
		Batch: batchResult.decryptedBatch,
	})

	if epoch != db1.epoch.val() {
		db1.log.Warn().Msgf("unexpected epoch:%d", epoch)
	}

	end := time.Now()
	db1.log.Debug().Msgf("all tpke decrypted, time:%v", end.Sub(batchResult.start))
	db1.log.Debug().Msgf("dumbo1 time:%v", end.Sub(db1.start))

	//db1.AdvanceEpoch()
	//db1.Close()
}

// 门限解密
func (db1 *Dumbo1) tryDecrypt(epoch cleisthenes.Epoch, proposer cleisthenes.Member) error {
	start := time.Now()
	batchResult := db1.getBatchResult(epoch)
	if !batchResult.unCompleteTxMap[proposer].Value() {
		//fmt.Println("epoch:", epoch, "proposer:", proposer.Address.String(), "already completed")
		return nil
	}

	db1.log.Debug().Str("proposer", proposer.Address.String()).Msg("try decrypt")

	if db1.countDecShares(epoch, proposer) < db1.decShareThreshold() {
		db1.log.Debug().Str("proposer", proposer.Address.String()).Msgf("dec share is not enough, count:%d", db1.countDecShares(epoch, proposer))
		return nil
	}

	if len(batchResult.encryptTxMap[proposer]) == 0 {
		db1.log.Info().Str("proposer", proposer.Address.String()).Msg("encrypt tx is empty!")
		return nil
	}

	t1 := time.Now()
	db1.log.Debug().Msgf("t1 time:%v", t1.Sub(start))

	db1.log.Debug().Str("proposer", proposer.Address.String()).Msg("start to decrypt")
	// TODO 待优化decryptLock
	db1.decryptLock.Lock()
	db1.tpk.ClearDecShare()
	for _, req := range batchResult.decShareReqRepositorys[proposer].FindAll() {
		r, _ := req.(*DecShareRequest)
		db1.tpk.AcceptDecShare(r.Sender, r.DecShare)
	}

	dec, err := db1.tpk.Decrypt(batchResult.encryptTxMap[proposer])
	if err != nil {
		db1.log.Error().Str("proposer", proposer.Address.String()).Msg("解密失败")
		db1.decryptLock.Unlock()
		return err
	}
	db1.tpk.ClearDecShare()
	db1.decryptLock.Unlock()

	t2 := time.Now()
	db1.log.Debug().Msgf("t2 time:%v", t2.Sub(t1))

	//var contribution cleisthenes.Contribution
	//err = json.Unmarshal(dec, &contribution.TxList)
	//if err != nil {
	//	db1.log.Error().Str("proposer", proposer.Address.String()).Msgf("unmarshal txList failed, err:%s", err)
	//	return err
	//}

	//decryptedBatch := make([]cleisthenes.AbstractTx, 0)
	//for _, tx := range contribution.TxList {
	//	abstractTx := cleisthenes.AbstractTx(map[string]interface{}{proposer.Address.String(): tx})
	//	decryptedBatch = append(decryptedBatch, abstractTx)
	//}

	txs := make([]cleisthenes.AbstractTx, 0)
	transactions := make([]cleisthenes.Transaction, 0)
	err = sonic.ConfigFastest.Unmarshal(dec, &transactions)
	if err != nil {
		db1.log.Fatal().Msgf("unmarshal contribution.TxList failed, err:%s", err.Error())
	}
	for _, tx := range transactions {
		txs = append(txs, tx)
	}
	transactions = nil
	//jsonparser.ArrayEach(dec, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
	//	tx = append(tx, value)
	//})

	t3 := time.Now()
	db1.log.Debug().Msgf("t3 time:%v", t3.Sub(t2))

	db1.log.Debug().Str("proposer", proposer.Address.String()).Msg("tx decrypted")

	batchResult.unCompleteTxMap[proposer].Set(false)

	end := time.Now()
	db1.log.Debug().Msgf("t4 time:%v,", end.Sub(t3))
	db1.log.Debug().Msgf("tpke dec time:%v", end.Sub(start))

	db1.batchChan <- DecryptedTx{
		Epoch: epoch,
		Addr:  proposer.Address.String(),
		Txs:   txs,
	}
	return nil
}

func (db1 *Dumbo1) handleBatchMessage(batchMessage cleisthenes.BatchMessage) error {
	start := time.Now()
	batchResult := db1.getBatchResult(batchMessage.Epoch)
	if batchResult.start.IsZero() {
		batchResult.start = time.Now()
	}

	batchResult.needCompleteCount = 0
	for proposer, encryptedTx := range batchMessage.Batch {
		batchResult.encryptTxMap[proposer] = encryptedTx
		// 计算共识结果为1的节点的tx的decShare，并将decShare广播
		batchResult.unCompleteTxMap[proposer].Set(true)
		batchResult.needCompleteCount++

		decShare := db1.tpk.DecShare(encryptedTx)
		db1.broadcaster.ShareMessage(pb.Message{
			Proposer:  proposer.Address.String(),
			Sender:    db1.owner.Address.String(),
			Timestamp: ptypes.TimestampNow(),
			Epoch:     uint64(batchMessage.Epoch),
			Payload: &pb.Message_Ds{
				Ds: &pb.DecShare{
					Payload: decShare[:],
				},
			},
		})
	}
	end := time.Now()
	db1.log.Debug().Msgf("share dec share time:%v", end.Sub(start))

	return nil
}

func (db1 *Dumbo1) AdvanceEpoch() {
	//db1.epoch.up()
	db1.closeOldEpoch(db1.epoch.val())
	db1.finConsensus()
}

func (db1 *Dumbo1) closeOldEpoch(epoch cleisthenes.Epoch) {
	db1.log.Debug().Msgf("start to close old epoch:%v", epoch)
	acs, ok := db1.acsRepository.find(epoch)
	if !ok {
		return
	}

	db1.log.Info().Msg("start to close old epoch")
	acs.Close()
	db1.acsRepository.delete(epoch)
	db1.contributionBuffer.clear()
	db1.clearBatchResult(epoch)
}

func (db1 *Dumbo1) OnConsensus() bool {
	return atomic.LoadInt32(&(db1.onConsensusFlag)) == int32(1)
}

func (db1 *Dumbo1) startConsensus() bool {
	db1.log.Debug().Msg("start consensus")
	return atomic.CompareAndSwapInt32(&db1.onConsensusFlag, int32(0), int32(1))
}

func (db1 *Dumbo1) finConsensus() {
	db1.log.Debug().Msg("fin consensus")
	atomic.CompareAndSwapInt32(&db1.onConsensusFlag, int32(1), int32(0))
	db1.contributionChan <- struct{}{}
}

func (db1 *Dumbo1) Close() {
	//db1.acsRepository.delete(db1.epoch.val())
	//db1.clearBatchResult(db1.epoch.val())

	//db1.log.Debug().Msgf("start to close acs, e:%d", db1.epoch.val())
	//db1.acsRepository.CloseAndDeleteAll()
	//db1.log.Debug().Msgf("fin to close acs, e:%d", db1.epoch.val())
	//db1.contributionBuffer.clear()
	//db1.clearAllBatchResult()
	//db1.log.Debug().Msgf("fin to close batch result, e:%d", db1.epoch.val())

	// 已经close掉了不再重复close
	if db1.toDie() {
		db1.log.Debug().Msgf("dumbo1 already closed")
		return
	}

	db1.log.Debug().Msgf("start to close dumbo1")

	db1.closeChan <- struct{}{}
	<-db1.closeChan

	db1.reqChanLock.Lock()

	if first := atomic.CompareAndSwapInt32(&db1.stopFlag, int32(0), int32(1)); !first {
		db1.reqChanLock.Unlock()
		return
	}

	// 将剩余的reqChan里的消息直接返回nil
	db1.log.Debug().Msgf("remain %v req", len(db1.reqChan))
	// FIXME 不应该在消费者线程中关闭reqChan
	close(db1.reqChan)
	db1.reqChanLock.Unlock()

	for msg := range db1.reqChan {
		msg.err <- nil
	}

	db1.AdvanceEpoch()
}

func (db1 *Dumbo1) toDie() bool {
	return atomic.LoadInt32(&(db1.stopFlag)) == int32(1)
}
