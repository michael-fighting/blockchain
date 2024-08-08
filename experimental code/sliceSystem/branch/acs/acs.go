package acs

import (
	"errors"
	"fmt"
	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/DE-labtory/cleisthenes/sliceSystem/branch/bba"
	"github.com/DE-labtory/cleisthenes/sliceSystem/branch/ce"
	"github.com/DE-labtory/cleisthenes/sliceSystem/branch/irbc"
	"github.com/DE-labtory/cleisthenes/sliceSystem/branch/rbc"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog"
	"sync/atomic"
	"time"
)

type request struct {
	proposer cleisthenes.Member
	sender   cleisthenes.Member
	data     *pb.Message
	err      chan error
}

type ACS struct {
	// number of network nodes
	n int
	// number of byzantine nodes which can tolerate
	f int
	// number of committee
	k int

	epoch cleisthenes.Epoch
	owner cleisthenes.Member

	memberMap *cleisthenes.MemberMap
	output    map[cleisthenes.Member][]byte

	tss cleisthenes.Tss
	tpk cleisthenes.Tpke

	// rbcMap has rbc instances
	rbcRepo *RBCRepository

	// indexRbcRepo has k committee instances
	indexRbcRepo *RBCRepository

	// bbaMap has bba instances
	bbaRepo *BBARepository

	// ce instance
	ce CE

	// 委员会集合
	cmisRepo *CMISRepository

	// 先收到的dataRbc的输出
	incomingDataRbcOutputRepo *cleisthenes.DataMessageRepository

	// 先收到的indexRbc的输出
	incomingIndexRbcOutputRepo *cleisthenes.DataMessageRepository

	// 先收到的binary的输出
	incomingBinaryOutputRepo *cleisthenes.BinaryMessageRepository

	// broadcastResult collects dataRBC instances' result
	broadcastResult BroadcastDataMap
	// indexBroadcastResult collects indexRbc instances' result
	indexBroadcastResult BroadcastDataMap
	// agreementResult collects BBA instances' result
	// each entry have three states: undefined, zero, one
	agreementResult binaryStateMap
	// agreementStarted saves whether BBA instances started
	// binary byzantine agreement
	agreementStarted binaryStateMap

	// indexRbcStarted saves whether indexRbc instances started
	indexRbcStarted binaryStateMap

	// 委员会选举完成标记
	cmisComplete cleisthenes.BinaryState

	dec cleisthenes.BinaryState

	tmp cleisthenes.BinaryState

	reqChan       chan request
	agreementChan chan struct{}
	closeChan     chan struct{}

	stopFlag int32

	cmisReceiver   cleisthenes.CMISReceiver
	dataSender     cleisthenes.DataSender
	dataReceiver   cleisthenes.DataReceiver
	indexReceiver  cleisthenes.DataReceiver
	indexSender    cleisthenes.DataSender
	binaryReceiver cleisthenes.BinaryReceiver
	binarySender   cleisthenes.BinarySender
	batchSender    cleisthenes.BatchSender

	roundReceiver cleisthenes.BinaryReceiver
	//broadCaster   cleisthenes.Broadcaster

	log       zerolog.Logger
	start     time.Time
	irbcStart time.Time
	bbaStart  time.Time
}

// TODO : the coin generator must be injected outside the ACS, BBA should has it's own coin generator.
// TODO : if we consider dynamic network, change MemberMap into pointer
func New(
	n int,
	f int,
	k int,
	epoch cleisthenes.Epoch,
	owner cleisthenes.Member,
	memberMap *cleisthenes.MemberMap,
	tss cleisthenes.Tss,
	tpk cleisthenes.Tpke,
	cmisReceiver cleisthenes.CMISReceiver,
	cmisSender cleisthenes.CMISSender,
	dataReceiver cleisthenes.DataReceiver,
	dataSender cleisthenes.DataSender,
	indexDataReceiver cleisthenes.DataReceiver,
	indexDataSender cleisthenes.DataSender,
	binaryReceiver cleisthenes.BinaryReceiver,
	binarySender cleisthenes.BinarySender,
	batchSender cleisthenes.BatchSender,
	broadCaster cleisthenes.Broadcaster,
) (*ACS, error) {

	acs := &ACS{
		n:                          n,
		f:                          f,
		k:                          k,
		epoch:                      epoch,
		owner:                      owner,
		memberMap:                  memberMap,
		tss:                        tss,
		tpk:                        tpk,
		rbcRepo:                    NewRBCRepository(),
		indexRbcRepo:               NewRBCRepository(),
		bbaRepo:                    NewBBARepository(),
		cmisRepo:                   NewCMISRepository(),
		incomingDataRbcOutputRepo:  cleisthenes.NewDataMessageRepository(),
		incomingIndexRbcOutputRepo: cleisthenes.NewDataMessageRepository(),
		incomingBinaryOutputRepo:   cleisthenes.NewBinaryMessageRepository(),
		broadcastResult:            NewBroadcastDataMap(),
		indexBroadcastResult:       NewBroadcastDataMap(),
		agreementResult:            NewBinaryStateMap(),
		agreementStarted:           NewBinaryStateMap(),
		indexRbcStarted:            NewBinaryStateMap(),
		// TODO : consider Size of reqChan, otherwise this might cause requests to be lost
		//reqChan:        make(chan request, n*n*8),
		reqChan:        make(chan request, n*n*20),
		agreementChan:  make(chan struct{}, n*8),
		closeChan:      make(chan struct{}),
		dataSender:     dataSender,
		dataReceiver:   dataReceiver,
		indexReceiver:  indexDataReceiver,
		indexSender:    indexDataSender,
		binaryReceiver: binaryReceiver,
		binarySender:   binarySender,
		cmisReceiver:   cmisReceiver,
		batchSender:    batchSender,

		log: cleisthenes.NewLoggerWithHead("ACS"),
	}

	// rbc, indexRbc和bba的初始化
	for _, member := range memberMap.Members() {
		r, err := rbc.New(n, f, epoch, owner, member, broadCaster, dataSender, false)
		b := bba.New(n, f, epoch, owner, member, broadCaster, binarySender, tpk, tss)
		if err != nil {
			return nil, err
		}
		acs.rbcRepo.Save(member, r)
		acs.bbaRepo.Save(member, b)

		// indexRbc的初始化
		ir, err := irbc.New(n, f, epoch, owner, member, broadCaster, indexDataSender, true)
		if err != nil {
			return nil, err
		}
		acs.indexRbcRepo.Save(member, ir)

		acs.broadcastResult.Set(member, []byte{})
		acs.indexBroadcastResult.Set(member, []byte{})
		acs.agreementResult.set(member, *cleisthenes.NewBinaryState())
		acs.agreementStarted.set(member, *cleisthenes.NewBinaryState())
		acs.indexRbcStarted.set(member, *cleisthenes.NewBinaryState())
	}

	// ce的初始化
	c, err := ce.New(n, f, k, epoch, tss, owner, owner, broadCaster, memberMap, cmisSender)
	if err != nil {
		return nil, err
	}
	acs.ce = c

	acs.initLog()

	go acs.run()
	return acs, nil
}

func (acs *ACS) initLog() {
	var logPrefix zerolog.HookFunc
	logPrefix = func(e *zerolog.Event, level zerolog.Level, message string) {
		e.Uint64("Epoch", uint64(acs.epoch)).Str("owner", acs.owner.Address.String())
	}
	acs.log = acs.log.Hook(logPrefix)
}

// HandleInput receive encrypted batch from honeybadger
func (acs *ACS) HandleInput(data []byte) error {
	if acs.start.IsZero() {
		acs.start = time.Now()
	}
	acs.log.Debug().Msg("开始处理dataRBC的输入")
	rbc, err := acs.rbcRepo.Find(acs.owner)
	if err != nil {
		return errors.New(fmt.Sprintf("no match rbc instance - address : %s", acs.owner.Address.String()))
	}

	return rbc.HandleInput(data)
}

func (acs *ACS) HandleMessage(sender cleisthenes.Member, msg *pb.Message) error {
	proposerAddr, err := cleisthenes.ToAddress(msg.Proposer)
	if err != nil {
		return err
	}
	proposer, ok := acs.memberMap.Member(proposerAddr)
	if !ok {
		return ErrNoMemberMatchingRequest
	}
	req := request{
		proposer: proposer,
		sender:   sender,
		data:     msg,
		err:      make(chan error),
	}

	//将请求放入acs通道
	acs.reqChan <- req
	return <-req.err
}

// Result return consensused batch to honeybadger component
func (acs *ACS) Result() cleisthenes.Batch {
	return cleisthenes.Batch{}
}

func (acs *ACS) Close() {
	acs.closeChan <- struct{}{}
	<-acs.closeChan

	for _, rbc := range acs.rbcRepo.FindAll() {
		rbc.Close()
	}

	for _, rbc := range acs.indexRbcRepo.FindAll() {
		rbc.Close()
	}

	for _, bba := range acs.bbaRepo.FindAll() {
		bba.Close()
	}

	acs.ce.Close()

	if first := atomic.CompareAndSwapInt32(&acs.stopFlag, int32(0), int32(1)); !first {
		return
	}
	//close(acs.closeChan)
}

func (acs *ACS) muxMessage(proposer, sender cleisthenes.Member, msg *pb.Message) error {
	switch pl := msg.Payload.(type) {
	case *pb.Message_Ce:
		return acs.handleCeMessage(proposer, sender, pl)
	case *pb.Message_Rbc:
		// 处理dataRbc或者indexRbc消息
		return acs.handleRbcMessage(proposer, sender, pl)
	case *pb.Message_Bba:
		return acs.handleBbaMessage(proposer, sender, pl)
	default:
		return cleisthenes.ErrUndefinedRequestType
	}
}

func (acs *ACS) handleCeMessage(proposer, sender cleisthenes.Member, msg *pb.Message_Ce) error {
	ce := acs.ce
	if ce == nil {
		return errors.New(fmt.Sprintf("ce is not init"))
	}
	return ce.HandleMessage(sender, msg)
}

func (acs *ACS) handleRbcMessage(proposer, sender cleisthenes.Member, msg *pb.Message_Rbc) error {
	repo := acs.rbcRepo
	// 如果是indexRbc节点，使用indexRbc仓库
	if msg.Rbc.IsIndexRbc {
		repo = acs.indexRbcRepo
		if acs.irbcStart.IsZero() {
			acs.irbcStart = time.Now()
		}
	}

	rbc, err := repo.Find(proposer)
	if err != nil {
		return errors.New(fmt.Sprintf("no match rbc instance - address : %s\n", proposer.Address.String()))
	}
	return rbc.HandleMessage(sender, msg)
}

func (acs *ACS) handleBbaMessage(proposer, sender cleisthenes.Member, msg *pb.Message_Bba) error {
	if acs.bbaStart.IsZero() {
		acs.bbaStart = time.Now()
	}
	bba, err := acs.bbaRepo.Find(proposer)
	if err != nil {
		return errors.New(fmt.Sprintf("no match bba instance - address : %s", acs.owner.Address.String()))

	}
	return bba.HandleMessage(sender, msg)
}

func (acs *ACS) run() {
	for !acs.toDie() {
		select {
		case <-acs.closeChan:
			acs.closeChan <- struct{}{}
			return
		case req := <-acs.reqChan:
			if acs.start.IsZero() {
				acs.start = time.Now()
			}
			// 处理rbc或者bba消息或者ce消息
			req.err <- acs.muxMessage(req.proposer, req.sender, req.data)
		case <-acs.cmisReceiver.Receive():
		case output := <-acs.dataReceiver.Receive():
			acs.log.Debug().Msgf("dataRBC的一个节点完成, output:%s", output.Member.Address.String())
			acs.processData(output.Member, output.Data)
			acs.tryAgreementStart(output.Member)
			// n-f个ba开始后，剩余的ba置0
			if acs.countStartedBba() == acs.n-acs.f {
				acs.agreementChan <- struct{}{}
			}
			// aba可能没有收到足够的value vi，当收到收到value vi后需要重新触发完成共识
			acs.tryCompleteAgreement()
		case <-acs.indexReceiver.Receive():
		case <-acs.agreementChan:
			acs.log.Info().Msg("向其他委员会bba实例发送0")
			acs.sendZeroToIdleBba()
		case output := <-acs.binaryReceiver.Receive():
			// bba当前轮数共识成功会执行以下步骤
			acs.log.Info().Msgf("一个ba共识成功, output:%s b:%v", output.Member.Address.String(), output.Binary)
			acs.processAgreement(output.Member, output.Binary)
			acs.tryCompleteAgreement()
		}
	}
}

// sendZeroToIdleCMISBba send zero to bba instances which still do
// not receive input value
func (acs *ACS) sendZeroToIdleCMISBba() {
	for _, member := range acs.cmisRepo.Members() {
		b, err := acs.bbaRepo.Find(member)
		if err != nil {
			fmt.Printf("no match bba instance - address : %s\n", member.Address.String())
		}

		if state := acs.agreementStarted.item(member); state.Undefined() && b.Idle() {
			acs.tmp.Set(true)
			if err := b.HandleInput(0, &bba.BvalRequest{Value: cleisthenes.VZero}); err != nil {
				fmt.Printf("error in HandleInput : %s", err.Error())
			}
		}
	}
}

func (acs *ACS) sendZeroToIdleBba() {
	for _, member := range acs.memberMap.Members() {
		b, err := acs.bbaRepo.Find(member)
		if err != nil {
			acs.log.Error().Msgf("no match bba instance - address : %s\n", member.Address.String())
		}

		if state := acs.agreementStarted.item(member); state.Undefined() && b.Idle() {
			acs.tmp.Set(true)
			if err := b.HandleInput(0, &bba.BvalRequest{Value: cleisthenes.VZero}); err != nil {
				acs.log.Error().Msgf("error in HandleInput : %s", err.Error())
			}
		}
	}
}

func (acs *ACS) countStartedBba() int {
	cnt := 0
	for _, state := range acs.agreementStarted.itemMap() {
		if state.Value() {
			cnt++
		}
	}
	return cnt
}

func (acs *ACS) tryCompleteAgreement() {
	//fmt.Println("test35")

	m := make(map[string]bool)
	for key, val := range acs.agreementResult.itemMap() {
		if !val.Undefined() {
			m[key.Address.String()] = val.Value()
		}
	}

	acs.log.Debug().Msgf("acs done agreement count:%d, success count:%d, members:%v",
		acs.countDoneAgreement(), acs.countSuccessDoneAgreement(), m)
	if acs.dec.Value() || acs.countDoneAgreement() != acs.agreementDoneThreshold() {
		return
	}

	// 至此，已经收到k个aba的输出，对index进行取”并集“，并等待”并集“中的所有value vi结果到来后再输出
	agreedNodeList := make([]cleisthenes.Member, 0)
	for member, state := range acs.agreementResult.itemMap() {
		if state.Value() {
			agreedNodeList = append(agreedNodeList, member)
		}
	}

	// 获取并集中的bcResult
	bcResult := make(map[cleisthenes.Member][]byte)
	for _, member := range agreedNodeList {
		if data := acs.broadcastResult.Item(member); data != nil && len(data) != 0 {
			bcResult[member] = data
		}
	}

	// 完成共识，acs发出共识成功的交易到通道
	if len(agreedNodeList) == len(bcResult) && len(bcResult) != 0 {
		acs.agreementSuccess(bcResult)
	} else {
		acs.log.Warn().Msg("unexpected branch")
	}
}

func (acs *ACS) agreementSuccess(result map[cleisthenes.Member][]byte) {
	// dec Set true 要放在靠前位置，batchSender.Send可能会阻塞，导致无法设置dec，后面的acs complete会一直触发
	acs.dec.Set(true)

	members := make([]cleisthenes.Member, 0)
	for member, _ := range result {
		members = append(members, member)
	}
	acs.log.Info().Msgf("[ACS done] final result members:%v", members)

	end := time.Now()
	cleisthenes.MainTps.Acs = end.Sub(acs.start)
	cleisthenes.MainTps.Bba = end.Sub(acs.bbaStart)
	acs.log.Debug().Msgf("bba time:%v", end.Sub(acs.bbaStart))
	acs.log.Debug().Msgf("acs time:%v", end.Sub(acs.start))

	acs.batchSender.Send(cleisthenes.BatchMessage{
		Epoch: acs.epoch,
		Batch: result,
	})
}

func (acs *ACS) tryAgreementStart(sender cleisthenes.Member) {
	b, err := acs.bbaRepo.Find(sender)
	if err != nil {
		fmt.Printf("no match bba instance - address : %s\n", acs.owner.Address.String())
		return
	}

	if ok := acs.agreementStarted.exist(sender); !ok {
		fmt.Printf("no match agreementStarted Item - address : %s\n", sender.Address.String())
		return
	}

	state := acs.agreementStarted.item(sender)
	//if state.Value() {
	//	fmt.Printf("already started bba - address :%s\n", sender.Address)
	//	return
	//}

	end := time.Now()
	cleisthenes.MainTps.Irbc = end.Sub(acs.irbcStart)
	acs.log.Debug().Msgf("irbc time:%v", end.Sub(acs.irbcStart))
	if acs.bbaStart.IsZero() {
		acs.bbaStart = time.Now()
	}

	// estr初始化为1
	if err := b.HandleInput(b.Round(), &bba.BvalRequest{Value: cleisthenes.VOne}); err != nil {
		fmt.Printf("error in HandleInput : %s\n", err.Error())
	}
	state.Set(cleisthenes.One)
	acs.agreementStarted.set(sender, state)
	if acs.tmp.Value() {
		fmt.Printf("acs try start epoch : %d, onwer : %s, proposer : %s\n", acs.epoch, acs.owner.Address.String(), sender.Address.String())
	}
	return
}

func (acs *ACS) processData(sender cleisthenes.Member, data []byte) {
	if ok := acs.broadcastResult.Exist(sender); !ok {
		fmt.Printf("no match broadcastResult Item - address : %s\n", sender.Address.String())
		return
	}

	if data := acs.broadcastResult.Item(sender); len(data) != 0 {
		fmt.Printf("already processed data - address : %s\n", sender.Address.String())
		return
	}

	acs.broadcastResult.Set(sender, data)
}

func (acs *ACS) processIndexData(sender cleisthenes.Member, data []byte) {
	if ok := acs.indexBroadcastResult.Exist(sender); !ok {
		fmt.Printf("no match broadcastResult Item - address : %v\n", sender.Address.String())
		return
	}

	if data := acs.indexBroadcastResult.Item(sender); len(data) != 0 {
		fmt.Printf("already processed data - address : %v\n", sender.Address.String())
		return
	}

	acs.indexBroadcastResult.Set(sender, data)
}

func (acs *ACS) handleDelayedDataMessage() {
	var members []cleisthenes.Member
	for _, dataMsg := range acs.incomingDataRbcOutputRepo.FindAllAndClear() {
		members = append(members, dataMsg.Member)
		acs.dataSender.Send(*dataMsg)
	}
	acs.log.Debug().Msgf("处理先到的data vi消息:%v", members)
}

func (acs *ACS) handleDelayedIndexMessage() {
	var members []cleisthenes.Member
	for _, indexMsg := range acs.incomingIndexRbcOutputRepo.FindAllAndClear() {
		members = append(members, indexMsg.Member)
		acs.indexSender.Send(*indexMsg)
	}
	acs.log.Debug().Msgf("处理先到的index消息:%v", members)
}

func (acs *ACS) handleDelayedBinaryMessage() {
	var members []cleisthenes.Member
	for _, binaryMsg := range acs.incomingBinaryOutputRepo.FindAllAndClear() {
		members = append(members, binaryMsg.Member)
		acs.binarySender.Send(*binaryMsg)
	}
	acs.log.Debug().Msgf("处理先到的binary消息:%v", members)
}

func (acs *ACS) processAgreement(sender cleisthenes.Member, bin cleisthenes.Binary) {
	if ok := acs.agreementResult.exist(sender); !ok {
		acs.log.Error().Msgf("no match agreementResult Item - address : %s\n", sender.Address.String())
		return
	}

	state := acs.agreementResult.item(sender)
	if state.Value() {
		acs.log.Debug().Msgf("already processed agreement - address : %s\n", sender.Address.String())
		return
	}

	state.Set(bin)
	acs.agreementResult.set(sender, state)
}

func (acs *ACS) countSuccessDoneAgreement() int {
	cnt := 0
	for _, state := range acs.agreementResult.itemMap() {
		if state.Value() {
			cnt++
		}
	}
	return cnt
}

func (acs *ACS) countDoneAgreement() int {
	cnt := 0
	for _, state := range acs.agreementResult.itemMap() {
		if !state.Undefined() {
			cnt++
		}
	}
	return cnt
}

func (acs *ACS) indexRbcThreshold() int {
	return acs.n - acs.f
}
func (acs *ACS) agreementThreshold() int {
	return 1
	//return acs.n - acs.f
}

func (acs *ACS) agreementDoneThreshold() int {
	return acs.n
	// 小飞象算法中只要k个委员会成员结果收到就停止
	//return acs.k
}

func (acs *ACS) toDie() bool {
	return atomic.LoadInt32(&(acs.stopFlag)) == int32(1)
}

func (acs *ACS) processCMIS(cmis cleisthenes.CMISMessage) {
	for member := range cmis.CMIS {
		acs.cmisRepo.Save(member)
	}

	acs.cmisComplete.Set(true)
}

func (acs *ACS) tryIndexRBCStart() {
	ir, err := acs.indexRbcRepo.Find(acs.owner)
	if err != nil {
		acs.log.Error().Msg("no match indexRbc instance")
		return
	}

	if ok := acs.indexRbcStarted.exist(acs.owner); !ok {
		acs.log.Error().Msg("no match indexRbcStarted Item")
		return
	}

	state := acs.indexRbcStarted.item(acs.owner)
	if state.Value() {
		return
	}

	// 未收集到n-f个value vi
	if acs.countRecvResult() < acs.indexRbcThreshold() {
		return
	}

	end := time.Now()
	cleisthenes.MainTps.Rbc = end.Sub(acs.start)
	acs.log.Debug().Msgf("rbc time:%v", end.Sub(acs.start))
	if acs.irbcStart.IsZero() {
		acs.irbcStart = time.Now()
	}

	// 将前n-f个vi的索引转换成Index数据data
	addrStrs := make([]cleisthenes.Address, 0)
	i := 0
	for mem, data := range acs.broadcastResult.items {
		if i == acs.indexRbcThreshold() {
			break
		}

		if len(data) == 0 {
			continue
		}

		addrStrs = append(addrStrs, mem.Address)
		i++
	}
	data, err := sonic.ConfigFastest.Marshal(addrStrs)
	if err != nil {
		acs.log.Error().Msgf("marshal addrs failed, err:%s", err.Error())
		return
	}
	acs.log.Debug().Msgf("开始indexRBC阶段, data:%s", string(data))
	ir.HandleInput(data)

	state.Set(cleisthenes.One)
	acs.indexRbcStarted.set(acs.owner, state)
}

func (acs *ACS) indexAllValueReceived(data []byte) bool {
	addrs := make([]cleisthenes.Address, 0)
	err := sonic.ConfigFastest.Unmarshal(data, &addrs)
	if err != nil {
		return false
	}

	acs.log.Debug().Msgf("ada收到的index为addrs:%v", addrs)

	for _, address := range addrs {
		if !acs.broadcastResult.Exist(cleisthenes.Member{Address: address}) {
			acs.log.Debug().Msgf("收到了index:%v的信息中还有%s的value没有收到", addrs, address.String())
			return false
		}
	}

	return true
}

func (acs *ACS) countRecvResult() int {
	count := 0
	for _, data := range acs.broadcastResult.items {
		if len(data) != 0 {
			count++
		}
	}
	return count
}
