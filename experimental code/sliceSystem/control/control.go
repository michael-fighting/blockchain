package control

import (
	"errors"
	"fmt"
	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/DE-labtory/cleisthenes/sliceSystem/mainchain/acs"
	"github.com/buger/jsonparser"
	"github.com/bytedance/sonic"
	"github.com/golang/protobuf/ptypes"
	"github.com/rs/zerolog"
	"sync"
	"sync/atomic"
)

type request struct {
	proposer cleisthenes.Member
	sender   cleisthenes.Member
	data     *pb.Message
	err      chan error
}
type Control interface {
	HandleInput(info cleisthenes.CtrlSubmitInfo, broadcaster cleisthenes.Broadcaster) error
	HandleMessage(message *pb.Message) error
	Close()
}

type DefaultControl struct {
	n               int
	f               int
	mainchainN      int // mainchainN 表示有多少个分片，即代表节点的个数
	mainchainF      int // 主链拜占庭节点数量
	epoch           cleisthenes.Epoch
	owner           cleisthenes.Member
	memberMap       *cleisthenes.MemberMap
	branchMemberMap *cleisthenes.MemberMap
	branchN         int
	branchF         int
	ctrlSlices      []cleisthenes.Slice // 7000

	// FIXME RBC应该是个很独立的模块，可以单独脱离acs存在，后续需要将其移除acs文件夹
	rbcRepo          *acs.RBCRepository
	dataReceiver     cleisthenes.DataReceiver
	ctrlResultSender cleisthenes.CtrlResultSender

	broadcaster         cleisthenes.Broadcaster
	branchBroadcaster   cleisthenes.Broadcaster
	rbcBroadcastResult  acs.BroadcastDataMap
	flagBroadcastResult acs.BroadcastDataMap
	hashBroadcastResult acs.BroadcastDataMap
	hashRepo            *HashRepository

	finishFlagSent *cleisthenes.BinaryState
	hashResultSend *cleisthenes.BinaryState

	finishFlagLock sync.Mutex

	reqChan   chan request
	closeChan chan struct{}
	stopFlag  int32
	log       zerolog.Logger
}

func New(
	n, f, mainchainN, mainchainF int,
	epoch cleisthenes.Epoch,
	owner cleisthenes.Member,
	memberMap *cleisthenes.MemberMap,
	branchMemberMap *cleisthenes.MemberMap,
	branchN, branchF int,
	ctrlSlices []cleisthenes.Slice,
	ctrlResultSender cleisthenes.CtrlResultSender,
	broadcaster cleisthenes.Broadcaster,
	branchBroadcaster cleisthenes.Broadcaster,
) (Control, error) {
	// 用于rbc通信通道
	dataChan := cleisthenes.NewDataChannel(n)

	ctrl := &DefaultControl{
		n:                   n,
		f:                   f,
		mainchainN:          mainchainN,
		mainchainF:          mainchainF,
		epoch:               epoch,
		owner:               owner,
		memberMap:           memberMap,
		branchMemberMap:     branchMemberMap,
		branchN:             branchN,
		branchF:             branchF,
		ctrlSlices:          ctrlSlices,
		rbcRepo:             acs.NewRBCRepository(),
		dataReceiver:        dataChan,
		ctrlResultSender:    ctrlResultSender, // 将结果发送给ctrl
		broadcaster:         broadcaster,
		branchBroadcaster:   branchBroadcaster,
		rbcBroadcastResult:  acs.NewBroadcastDataMap(),
		flagBroadcastResult: acs.NewBroadcastDataMap(),
		hashBroadcastResult: acs.NewBroadcastDataMap(),
		hashRepo:            NewHashRepository(),
		finishFlagSent:      cleisthenes.NewBinaryState(),
		hashResultSend:      cleisthenes.NewBinaryState(),
		finishFlagLock:      sync.Mutex{},
		reqChan:             make(chan request, n*n*8),
		closeChan:           make(chan struct{}),
		log:                 cleisthenes.NewLoggerWithHead("CTRL"),
	}

	// rbc初始化
	for _, member := range memberMap.Members() {
		// 暂时不需要rbc
		//r, err := rbc.New(n, f, epoch, owner, member, broadcaster, dataChan, false)
		//if err != nil {
		//	return nil, err
		//}
		//ctrl.rbcRepo.Save(member, r)

		ctrl.rbcBroadcastResult.Set(member, []byte{})
		ctrl.flagBroadcastResult.Set(member, []byte{})
		ctrl.hashBroadcastResult.Set(member, []byte{})
	}

	ctrl.initLog()

	go ctrl.run()

	return ctrl, nil
}

func (c *DefaultControl) initLog() {
	var logPrefix zerolog.HookFunc
	logPrefix = func(e *zerolog.Event, level zerolog.Level, message string) {
		e.Uint64("Epoch", uint64(c.epoch)).Str("owner", c.owner.Address.String())
	}
	c.log = c.log.Hook(logPrefix)
}

func (c *DefaultControl) muxMessage(proposer, sender cleisthenes.Member, msg *pb.Message) error {
	switch pl := msg.Payload.(type) {
	case *pb.Message_Rbc:
		c.log.Debug().Msgf("收到rbc消息: sender:%s, proposer:%s", sender.Address.String(), proposer.Address.String())
		return c.handleRbcMessage(proposer, sender, pl)
	case *pb.Message_Ce:
		c.log.Debug().Msgf("收到普通广播消息,sender:%s", sender.Address.String())
		typ := c.getNormalBroadcastType(pl.Ce.Payload)
		if typ == cleisthenes.HashType {
			c.log.Debug().Msgf("收到hashType消息,sender:%s", sender.Address.String())
			return c.handleHashMessage(sender, pl.Ce.Payload)
		}
		if typ == cleisthenes.FlagType {
			c.log.Debug().Msgf("收到flagType消息,sender:%s", sender.Address.String())
			c.processFlagData(sender, pl.Ce.Payload)
			return c.handleFlagMessage(proposer, sender, pl)
		}
		if typ == cleisthenes.DataType {
			c.log.Debug().Msgf("收到DataType消息,sender:%s", sender.Address.String())
			return c.handleDataMessage(proposer, sender, pl)
		}
		if typ == cleisthenes.TransactionsType {
			c.log.Debug().Msgf("收到TransactionsType消息,sender:%s", sender.Address.String())
			return c.handleTransactionsMessage(proposer, sender, pl)
		}
		return cleisthenes.ErrUndefinedRequestType
	default:
		return cleisthenes.ErrUndefinedRequestType
	}
}

func (c *DefaultControl) handleRbcMessage(proposer, sender cleisthenes.Member, msg *pb.Message_Rbc) error {
	rbc, err := c.rbcRepo.Find(proposer)
	if err != nil {
		return errors.New(fmt.Sprintf("no match rbc instance - address : %s\n", proposer.Address.String()))
	}
	return rbc.HandleMessage(sender, msg)
}

func (c *DefaultControl) normalBroadcastThreshold() int {
	return c.mainchainN - c.mainchainF
}

func (c *DefaultControl) handleFlagMessage(proposer cleisthenes.Member, sender cleisthenes.Member, pl *pb.Message_Ce) error {
	c.finishFlagLock.Lock()
	defer c.finishFlagLock.Unlock()

	if c.finishFlagSent.Value() {
		return nil
	}

	// 未收集到m个主链代表节点的结束标记
	cnt := c.countCompleteFlag()
	c.log.Debug().Msgf("结束标记个数为%d, 主链代表节点个数为%d", cnt, c.mainchainN)
	if cnt < c.normalBroadcastThreshold() {
		return nil
	}

	var ctrlMemberInfo cleisthenes.CtrlSubmitInfo
	err := sonic.ConfigFastest.Unmarshal(pl.Ce.Payload, &ctrlMemberInfo)
	if err != nil {
		c.log.Fatal().Msgf("解析data错误, err:%s", err.Error())
	}

	c.log.Debug().Msgf("准备发送主链结束消息, flag:%v", c.finishFlagSent.Value())
	c.finishFlagSent.Set(true)
	c.log.Debug().Msgf("开始发送主链结束消息, flag:%v", c.finishFlagSent.Value())
	c.ctrlResultSender.Send(cleisthenes.CtrlResultMessage{
		Epoch:           c.epoch,
		MsgType:         cleisthenes.FlagType,
		MainChainFinish: ctrlMemberInfo.MainChainFinishFlag,
		MainChainTxCnt:  ctrlMemberInfo.MainChainTxCnt,
	})

	return nil
}

func (c *DefaultControl) handleDataMessage(proposer cleisthenes.Member, sender cleisthenes.Member, pl *pb.Message_Ce) error {
	var ctrlMemberInfo cleisthenes.CtrlSubmitInfo
	err := sonic.ConfigFastest.Unmarshal(pl.Ce.Payload, &ctrlMemberInfo)
	if err != nil {
		c.log.Fatal().Msgf("解析data错误, err:%s", err.Error())
	}

	c.ctrlResultSender.Send(cleisthenes.CtrlResultMessage{
		Epoch:   c.epoch,
		MsgType: cleisthenes.DataType,
		Data:    ctrlMemberInfo.Data,
	})

	return nil
}

func (c *DefaultControl) handleTransactionsMessage(proposer cleisthenes.Member, sender cleisthenes.Member, pl *pb.Message_Ce) error {
	var ctrlMemberInfo cleisthenes.CtrlSubmitInfo
	err := sonic.ConfigFastest.Unmarshal(pl.Ce.Payload, &ctrlMemberInfo)
	if err != nil {
		c.log.Fatal().Msgf("解析data错误, err:%s", err.Error())
	}

	c.ctrlResultSender.Send(cleisthenes.CtrlResultMessage{
		Epoch:   c.epoch,
		MsgType: cleisthenes.TransactionsType,
		Data:    ctrlMemberInfo.Data,
	})

	return nil
}

func (c *DefaultControl) HandleInput(info cleisthenes.CtrlSubmitInfo, broadcaster cleisthenes.Broadcaster) error {
	c.log.Debug().Msgf("开始处理输入 info, type:%v", info.MsgType)

	if info.MsgType == cleisthenes.CtrlInfoType {
		rbc, err := c.rbcRepo.Find(c.owner)
		if err != nil {
			return errors.New(fmt.Sprintf("no match rbc instance - address : %s", c.owner.Address.String()))
		}

		bs, err := sonic.ConfigFastest.Marshal(info)
		if err != nil {
			return err
		}
		return rbc.HandleInput(bs)
	}

	if info.MsgType == cleisthenes.HashType {
		return c.normalBroadcast(info, c.broadcaster)
	}

	if info.MsgType == cleisthenes.FlagType {
		c.log.Debug().Msgf("start to broadcast flag, broadcaster:%v", c.broadcaster.GetAllConnAddr())
		return c.normalBroadcast(info, c.broadcaster)
	}

	if info.MsgType == cleisthenes.TransactionsType {
		return c.normalBroadcast(info, broadcaster)
	}

	if info.MsgType == cleisthenes.DataType {
		return c.normalBroadcast(info, c.branchBroadcaster)
	}

	return nil
}
func (c *DefaultControl) HandleMessage(msg *pb.Message) error {
	proposerAddr, err := cleisthenes.ToAddress(msg.Proposer)
	if err != nil {
		return err
	}
	proposer, ok := c.memberMap.Member(proposerAddr)
	if !ok {
		return ErrNoMemberMatchingRequest
	}

	senderAddr, err := cleisthenes.ToAddress(msg.Sender)
	if err != nil {
		return err
	}
	sender, ok := c.memberMap.Member(senderAddr)

	req := request{
		proposer: proposer,
		sender:   sender,
		data:     msg,
		err:      make(chan error),
	}

	if c.toDie() {
		return nil
	}

	c.reqChan <- req
	return <-req.err
}

func (c *DefaultControl) run() {
	for !c.toDie() {
		select {
		case <-c.closeChan:
			c.closeChan <- struct{}{}
			return
		case req := <-c.reqChan:
			// 处理rbc或者bba消息或者ce消息
			req.err <- c.muxMessage(req.proposer, req.sender, req.data)
		case output := <-c.dataReceiver.Receive():
			// 收到了rbc的输出
			c.log.Debug().Msgf("收到了一个rbc的输出,member:%s", output.Member.Address.String())
			c.processRbcData(output.Member, output.Data)
			c.tryComplete()
		}
	}
}

func (c *DefaultControl) normalBroadcast(info cleisthenes.CtrlSubmitInfo, broadcaster cleisthenes.Broadcaster) error {
	// FIXME 暂时使用CE类型消息发送普通广播消息
	payload, err := sonic.ConfigFastest.Marshal(info)
	if err != nil {
		return err
	}
	broadcaster.ShareMessage(pb.Message{
		Proposer:  c.owner.Address.String(),
		Sender:    c.owner.Address.String(),
		Timestamp: ptypes.TimestampNow(),
		Epoch:     uint64(c.epoch),
		Payload: &pb.Message_Ce{
			Ce: &pb.CE{
				Payload: payload,
				Type:    pb.CE_SHARE,
			},
		},
	})
	return nil
}

func (c *DefaultControl) countCompleteRbc() int {
	cnt := 0
	for _, data := range c.rbcBroadcastResult.ItemMap() {
		if len(data) != 0 {
			cnt++
		}
	}
	return cnt
}

func (c *DefaultControl) countCompleteHash() (int, map[cleisthenes.Address]string) {
	res := make(map[cleisthenes.Address]string)
	completeCnt := 0
	for _, slice := range c.ctrlSlices {
		hashs := c.hashRepo.FindAllByAddrs(slice.Addrs)
		hashMap := make(map[string]int)
		for _, hash := range hashs {
			if hash != "" {
				hashMap[hash]++
			}
		}
		for hash, cnt := range hashMap {
			if cnt >= slice.F+1 {
				res[slice.Addrs[0]] = hash
				completeCnt++
				break
			}
		}
	}

	return completeCnt, res
}

func (c *DefaultControl) countCompleteFlag() int {
	cnt := 0
	// FIXME 这里应该计算交易数据Hash相同的标记的个数
	for _, data := range c.flagBroadcastResult.ItemMap() {
		if len(data) != 0 {
			cnt++
		}
	}
	return cnt
}

func (c *DefaultControl) tryComplete() {
	if c.hashResultSend.Value() {
		return
	}

	count, hashMap := c.countCompleteHash()
	c.log.Debug().Msgf("complete hash num:%d, hash map:%v", count, hashMap)
	// 收集到m个分片的
	if count == len(c.ctrlSlices) {
		if c.hashResultSend.Value() {
			return
		}
		c.hashResultSend.Set(true)
		ctrlMemberMap := make(map[cleisthenes.Address]cleisthenes.CtrlSubmitInfo, 0)
		for addr, hash := range hashMap {
			ctrlMemberMap[addr] = cleisthenes.CtrlSubmitInfo{
				MsgType: cleisthenes.CtrlInfoType,
				CtrlMemberInfo: cleisthenes.CtrlMemberInfo{
					Hash: hash,
				},
			}
		}
		c.ctrlResultSender.Send(cleisthenes.CtrlResultMessage{
			Epoch:   c.epoch,
			MsgType: cleisthenes.CtrlInfoType,
			CtrlInfo: cleisthenes.CtrlInfo{
				CtrlMemberMap: ctrlMemberMap,
			},
		})
	}
}

//func (c *DefaultControl) tryComplete() {
//	count := c.countCompleteRbc()
//	c.log.Debug().Msgf("complete rbc num:%d", count)
//	// FIXME 收集到了n个就输出，如果有作恶节点呢？
//	if count == c.n {
//		ctrlMemberMap := make(map[cleisthenes.Address]cleisthenes.CtrlSubmitInfo, 0)
//		for member, data := range c.rbcBroadcastResult.ItemMap() {
//			if len(data) == 0 {
//				continue
//			}
//			var ctrlMemberInfo cleisthenes.CtrlSubmitInfo
//			err := sonic.ConfigFastest.Unmarshal(data, &ctrlMemberInfo)
//			if err != nil {
//				c.log.Fatal().Msgf("解析data错误, err:%s", err.Error())
//			}
//			ctrlMemberMap[member.Address] = ctrlMemberInfo
//		}
//
//		c.log.Info().Msg("ctrl result complete")
//		c.ctrlResultSender.Send(cleisthenes.CtrlResultMessage{
//			Epoch:   c.epoch,
//			MsgType: cleisthenes.CtrlInfoType,
//			CtrlInfo: cleisthenes.CtrlInfo{
//				CtrlMemberMap: ctrlMemberMap,
//			},
//		})
//	}
//}

func (c *DefaultControl) processRbcData(sender cleisthenes.Member, data []byte) {
	if ok := c.rbcBroadcastResult.Exist(sender); !ok {
		c.log.Error().Msgf("no match rbcBroadcastResult Item - address : %s\n", sender.Address.String())
		return
	}

	if d := c.rbcBroadcastResult.Item(sender); len(d) != 0 {
		c.log.Error().Msgf("already processed rbc data - address : %s\n", sender.Address.String())
		return
	}

	c.rbcBroadcastResult.Set(sender, data)

	var ctrlSubmitInfo cleisthenes.CtrlSubmitInfo
	err := sonic.ConfigFastest.Unmarshal(data, &ctrlSubmitInfo)
	if err != nil {
		c.log.Error().Msgf("unmarshal data failed, err:%s", err.Error())
	}
	c.hashRepo.Save(sender.Address, ctrlSubmitInfo.CtrlMemberInfo.Hash)
}

func (c *DefaultControl) getNormalBroadcastType(data []byte) cleisthenes.MessageType {
	value, _, _, err := jsonparser.Get(data, "MsgType")
	if err != nil {
		c.log.Fatal().Msgf("get msg type from normal broadcast failed, err:%s", err.Error())
	}
	msgType, err := jsonparser.ParseInt(value)
	if err != nil {
		c.log.Fatal().Msgf("parse msg type from normal broadcast failed, err:%s", err.Error())
	}
	c.log.Debug().Msgf("normal broadcast value:%v msgType:%v", value, msgType)
	return cleisthenes.MessageType(msgType)
}

func (c *DefaultControl) handleHashMessage(sender cleisthenes.Member, data []byte) error {
	if c.hashResultSend.Value() {
		return nil
	}

	if ok := c.hashBroadcastResult.Exist(sender); !ok {
		c.log.Error().Msgf("no match hashBroadcastResult item - address : %s\n", sender.Address.String())
		return nil
	}

	if d := c.hashBroadcastResult.Item(sender); len(d) != 0 {
		c.log.Error().Msgf("already process hash data - address : %s\n", sender.Address.String())
		return nil
	}

	var ctrlMemberInfo cleisthenes.CtrlSubmitInfo
	err := sonic.ConfigFastest.Unmarshal(data, &ctrlMemberInfo)
	if err != nil {
		c.log.Fatal().Msgf("解析data错误, err:%s", err.Error())
		return err
	}

	c.log.Debug().Msgf("保存来自%s的hash数据:%v", sender.Address.String(), string(ctrlMemberInfo.Data))
	c.hashBroadcastResult.Set(sender, ctrlMemberInfo.Data)

	c.hashRepo.Save(sender.Address, string(ctrlMemberInfo.Data))

	c.tryComplete()

	return nil
}

func (c *DefaultControl) processFlagData(sender cleisthenes.Member, data []byte) {
	if ok := c.flagBroadcastResult.Exist(sender); !ok {
		fmt.Printf("no match flagBroadcastResult Item - address : %s\n", sender.Address.String())
		return
	}

	if data := c.flagBroadcastResult.Item(sender); len(data) != 0 {
		fmt.Printf("already processed flag data - address : %s\n", sender.Address.String())
		return
	}

	// TODO 需要对sender的合法性进行保证，必须要是当前epoch的领导节点

	c.log.Debug().Msgf("保存来自%s的flag数据, len(data):%d", sender.Address.String(), len(data))
	c.flagBroadcastResult.Set(sender, data)

	//resStr := fmt.Sprintf("%p 保存来自%s的flag数据, len(data):%d ", &c.flagBroadcastResult, sender.Address.String(), len(data))
	//for member, item := range c.flagBroadcastResult.ItemMap() {
	//	resStr += fmt.Sprintf("%s:%d ", member.Address.String(), len(item))
	//}
	//resStr += "]"
	//c.log.Debug().Msgf("flag res:%s", resStr)
}

func (c *DefaultControl) toDie() bool {
	return atomic.LoadInt32(&(c.stopFlag)) == int32(1)
}
func (c *DefaultControl) Close() {
	c.log.Debug().Msg("call close")

	c.closeChan <- struct{}{}
	<-c.closeChan

	for _, rbc := range c.rbcRepo.FindAll() {
		rbc.Close()
	}

	if first := atomic.CompareAndSwapInt32(&c.stopFlag, int32(0), int32(1)); !first {
		return
	}

	close(c.reqChan)
	for msg := range c.reqChan {
		msg.err <- nil
	}
}
