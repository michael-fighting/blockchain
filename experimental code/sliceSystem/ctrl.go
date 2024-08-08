package sliceSystem

import (
	"fmt"
	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/config"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/DE-labtory/cleisthenes/sliceSystem/control"
	"github.com/DE-labtory/cleisthenes/sliceSystem/control/electDelegate"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog"
	"sync"
)

type Ctrl interface {
	//Submit(info cleisthenes.CtrlSubmitInfo) error
	SubmitMainChainDone(epoch cleisthenes.Epoch, txCnt int, bs []byte)
	//SubmitHash(epoch cleisthenes.Epoch, hash string)
	SubmitHashV2(epoch cleisthenes.Epoch, hash string)
	SubmitTxsToMainChain(epoch cleisthenes.Epoch, txs []cleisthenes.Transaction)
	SubmitData(epoch cleisthenes.Epoch, data []byte)
	Advance(epoch cleisthenes.Epoch)
	//Epoch() cleisthenes.Epoch
	Connect(target string) error
	ConnectAll() error
	ConnectionList() []string
	Run()
	GetMainChainAddrs(epoch cleisthenes.Epoch) ([]cleisthenes.Address, error)
	Elect(epoch cleisthenes.Epoch, info *cleisthenes.CtrlInfo) []cleisthenes.Address
	IsDelegateNode(epoch cleisthenes.Epoch) bool
	IsMainChainNode(epoch cleisthenes.Epoch) bool
	Result() <-chan cleisthenes.CtrlResultMessage
	Close()
}

type CtrlNode struct {
	addr               cleisthenes.Address // 自身ctrl的地址 7000
	mainchainAddr      cleisthenes.Address // 自身mainchain的地址 6000
	txQueueManager     cleisthenes.TxQueueManager
	curEpoch           cleisthenes.Epoch
	ctrlRepo           *control.ControlRepository
	ctrlResultSender   cleisthenes.CtrlResultSender
	ctrlResultReceiver cleisthenes.CtrlResultReceiver
	messageEndpoint    cleisthenes.MessageEndpoint
	server             *cleisthenes.GrpcServer
	client             *cleisthenes.GrpcClient
	connPool           *cleisthenes.ConnectionPool
	branchConnPool     *cleisthenes.ConnectionPool

	// 控制节点需要保存很多配置，主要包含以下几个部分：0.所有节点信息；1.自己所在分片的节点信息；2.所有分片的节点信息；3.主链的节点信息
	// 所有节点信息
	// 7000
	memberMap *cleisthenes.MemberMap
	// 自己所在分片的节点信息（控制端口）
	// 7000
	branchCtrlMemberMap *cleisthenes.MemberMap
	// 自己所在分片的节点信息
	// 5000
	branchMemberMap *cleisthenes.MemberMap
	branchN         int
	branchF         int
	// 所有分片的节点信息
	ctrlSlices []cleisthenes.Slice // 7000
	slices     []cleisthenes.Slice // 6000
	// 当前节点所属领导节点
	delegateAddr map[cleisthenes.Epoch]cleisthenes.Address
	// 当前节点所属主链节点
	mainChainAddr map[cleisthenes.Epoch]cleisthenes.Address
	// 所有领导节点信息
	delegateMemberMapMap map[cleisthenes.Epoch]*cleisthenes.MemberMap
	// 所有主链的节点信息
	// 6000
	mainchainMemberMapMap map[cleisthenes.Epoch]*cleisthenes.MemberMap

	reqRepo *cleisthenes.ReqRepo

	n          int // 所有节点的个数
	f          int // 所有分片拜占庭节点个数之和
	mainchainN int // 分片个数，即代表节点个数
	mainchainF int
	gap        int

	mu sync.Mutex

	log zerolog.Logger
}

func NewCtrl() (Ctrl, error) {
	conf := config.Get()
	ctrlIdentity := conf.CtrlIdentity
	local, err := cleisthenes.ToAddress(ctrlIdentity.Address)
	if err != nil {
		return nil, err
	}
	external, err := cleisthenes.ToAddress(ctrlIdentity.ExternalAddress)
	if err != nil {
		return nil, err
	}

	mainchainExternal, err := cleisthenes.ToAddress(conf.MainChainIdentity.ExternalAddress)
	if err != nil {
		return nil, err
	}

	branchMemberMap := cleisthenes.NewMemberMap()
	branchCtrlMemberMap := cleisthenes.NewMemberMap()
	for _, member := range conf.Branch.Members {
		addr, err := cleisthenes.ToAddress(member.Address)
		if err != nil {
			return nil, err
		}
		branchMemberMap.Add(cleisthenes.NewMemberWithAddress(addr))
		// FIXME 当前将branch的端口（5000）转换到control的端口（7000）直接+2000
		branchCtrlMemberMap.Add(cleisthenes.NewMemberWithAddress(cleisthenes.Address{
			Ip:   addr.Ip,
			Port: addr.Port + 2000,
		}))
	}

	n := 0
	f := 0
	var slices []cleisthenes.Slice
	for _, slice := range conf.Slices {
		var sli cleisthenes.Slice
		for _, addrStr := range slice.Slice {
			addr, err := cleisthenes.ToAddress(addrStr)
			if err != nil {
				return nil, err
			}
			sli.Addrs = append(sli.Addrs, addr)
			n++
		}
		sli.N = slice.NetworkSize
		sli.F = slice.Byzantine
		// 读取配置文件里每个分片的拜占庭节点数量配置相加
		f += slice.Byzantine
		slices = append(slices, sli)
	}

	var ctrlSlices []cleisthenes.Slice
	for _, slice := range conf.CtrlSlices {
		var sli cleisthenes.Slice
		for _, addrStr := range slice.Slice {
			addr, err := cleisthenes.ToAddress(addrStr)
			if err != nil {
				return nil, err
			}
			sli.Addrs = append(sli.Addrs, addr)
		}
		sli.N = slice.NetworkSize
		sli.F = slice.Byzantine
		ctrlSlices = append(ctrlSlices, sli)
	}

	memberMap := cleisthenes.NewMemberMap()
	ctrlResultChan := cleisthenes.NewCtrlResultChannel(n)
	connPool := cleisthenes.NewConnectionPool()
	branchConnPool := cleisthenes.NewConnectionPool()

	return &CtrlNode{
		addr:               external,
		mainchainAddr:      mainchainExternal,
		txQueueManager:     nil,
		curEpoch:           0,
		ctrlRepo:           control.NewControlRepository(),
		ctrlResultSender:   ctrlResultChan,
		ctrlResultReceiver: ctrlResultChan,
		//messageEndpoint:    nil,
		server:                cleisthenes.NewServer(local, nil),
		client:                cleisthenes.NewClient(nil),
		connPool:              connPool,
		branchConnPool:        branchConnPool,
		memberMap:             memberMap,
		ctrlSlices:            ctrlSlices,
		branchMemberMap:       branchMemberMap,
		branchCtrlMemberMap:   branchCtrlMemberMap,
		branchN:               conf.Branch.Dumbo1.NetworkSize,
		branchF:               conf.Branch.Dumbo1.Byzantine,
		delegateAddr:          make(map[cleisthenes.Epoch]cleisthenes.Address),
		mainChainAddr:         make(map[cleisthenes.Epoch]cleisthenes.Address),
		delegateMemberMapMap:  make(map[cleisthenes.Epoch]*cleisthenes.MemberMap),
		mainchainMemberMapMap: make(map[cleisthenes.Epoch]*cleisthenes.MemberMap),
		reqRepo:               cleisthenes.NewReqRepo(),
		slices:                slices,
		n:                     n,
		f:                     f,
		mainchainN:            conf.MainChain.Dumbo1.NetworkSize,
		mainchainF:            conf.MainChain.Dumbo1.Byzantine,
		gap:                   (len(slices) + conf.MainChain.Dumbo1.NetworkSize - 1) / conf.MainChain.Dumbo1.NetworkSize,
		log:                   cleisthenes.NewLoggerWithHead("CTRL"),
	}, nil
}

func (n *CtrlNode) findCtrl(epoch cleisthenes.Epoch) (control.Control, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	// 如果epoch是已经close的，直接报错
	if epoch < n.curEpoch {
		return nil, fmt.Errorf("old epoch(%v) has closed, cur Epoch:%v", epoch, n.curEpoch)
	}

	if ctrl := n.ctrlRepo.Find(epoch); ctrl != nil {
		return ctrl, nil
	}

	c, err := control.New(
		n.n, n.f, n.mainchainN, n.mainchainF,
		epoch,
		*cleisthenes.NewMemberWithAddress(n.addr),
		n.memberMap,
		n.branchMemberMap,
		n.branchN,
		n.branchF,
		n.ctrlSlices,
		n.ctrlResultSender,
		n.connPool,
		n.branchConnPool,
	)
	if err != nil {
		return nil, err
	}

	n.log.Debug().Msgf("ctrl repo size:%d", n.ctrlRepo.Size())

	// findCtrl的并发问题，需要判断save的错误
	if err := n.ctrlRepo.Save(epoch, c); err != nil {
		n.log.Warn().Msgf("epoch control already exist, err:%v", err.Error())
		c.Close()
		return n.ctrlRepo.Find(epoch), nil
	}

	return c, nil
}

func (n *CtrlNode) submit(epoch cleisthenes.Epoch, info cleisthenes.CtrlSubmitInfo, broadcaster cleisthenes.Broadcaster) {
	n.log.Debug().Msgf("开始提交ctrl消息, type:%v", info.MsgType)
	ctrl, err := n.findCtrl(epoch)
	if err != nil {
		n.log.Error().Msgf("find ctrl failed, err:%s", err.Error())
		return
	}
	n.log.Debug().Msgf("epoch %v ctrl instance:%p msgType:%v", epoch, ctrl, info.MsgType)
	err = ctrl.HandleInput(info, broadcaster)
	if err != nil {
		n.log.Error().Msgf("handle input failed, err:%s", err.Error())
	}
}

func (n *CtrlNode) SubmitHash(epoch cleisthenes.Epoch, hash string) {
	n.submit(epoch, cleisthenes.CtrlSubmitInfo{
		MsgType: cleisthenes.CtrlInfoType,
		CtrlMemberInfo: cleisthenes.CtrlMemberInfo{
			Hash: hash,
		},
	}, nil)
}

func (n *CtrlNode) SubmitHashV2(epoch cleisthenes.Epoch, hash string) {
	n.submit(epoch, cleisthenes.CtrlSubmitInfo{
		MsgType: cleisthenes.HashType,
		Data:    []byte(hash),
	}, nil)
}

func (n *CtrlNode) SubmitData(epoch cleisthenes.Epoch, data []byte) {
	n.submit(epoch, cleisthenes.CtrlSubmitInfo{
		MsgType: cleisthenes.DataType,
		Data:    data,
	}, nil)
}

func (n *CtrlNode) SubmitMainChainDone(epoch cleisthenes.Epoch, txCnt int, bs []byte) {
	n.submit(epoch, cleisthenes.CtrlSubmitInfo{
		MsgType:             cleisthenes.FlagType,
		MainChainFinishFlag: true,
		MainChainTxCnt:      txCnt,
		Data:                bs,
	}, nil)
}

func (n *CtrlNode) SubmitTxsToMainChain(epoch cleisthenes.Epoch, txs []cleisthenes.Transaction) {
	data, err := sonic.ConfigFastest.Marshal(txs)
	if err != nil {
		n.log.Fatal().Msgf("marshal transactions failed, err:%s", err.Error())
	}
	// FIXME mainChainAddr是主链的端口，转换成ctrl的端口需要+1000
	ip := n.mainChainAddr[epoch].Ip
	port := n.mainChainAddr[epoch].Port
	broadcaster := cleisthenes.NewConnectionPool()
	addr, _ := cleisthenes.ToAddress(fmt.Sprintf("%s:%d", ip, port+1000))
	conn := n.connPool.Get(addr)
	broadcaster.Add(addr, conn)

	n.submit(epoch, cleisthenes.CtrlSubmitInfo{
		MsgType: cleisthenes.TransactionsType,
		Data:    data,
	}, broadcaster)
}

func (n *CtrlNode) Advance(epoch cleisthenes.Epoch) {
	if epoch >= 2 {
		n.closeOldEpoch(epoch - 2)
	}
	n.curEpoch = epoch
}

//func (n *CtrlNode) Epoch() cleisthenes.Epoch {
//	return n.epoch
//}

func (n *CtrlNode) closeOldEpoch(epoch cleisthenes.Epoch) {
	n.log.Debug().Msgf("start to close ctrl old epoch: %v, size:%v", epoch, n.ctrlRepo.Size())
	n.ctrlRepo.DeleteEpochLessEqual(epoch)
	n.log.Debug().Msgf("after close ctrl old epoch: %v, size:%v", epoch, n.ctrlRepo.Size())
}

func (n *CtrlNode) GetMainChainAddrs(epoch cleisthenes.Epoch) ([]cleisthenes.Address, error) {
	mainchainAddrs := make([]cleisthenes.Address, 0)
	if m, ok := n.mainchainMemberMapMap[epoch]; ok {
		for _, member := range m.Members() {
			mainchainAddrs = append(mainchainAddrs, member.Address)
		}
		return mainchainAddrs, nil
	}

	return nil, fmt.Errorf("mainchain addrs not elected for epoch:%v", epoch)
}

// getMainChainAddr 获取分片所属的主链节点地址
func (n *CtrlNode) getMainChainAddr(epoch cleisthenes.Epoch) cleisthenes.Address {
	return n.mainChainAddr[epoch]
}

func (n *CtrlNode) Elect(epoch cleisthenes.Epoch, info *cleisthenes.CtrlInfo) []cleisthenes.Address {
	mainchainAddrs := make([]cleisthenes.Address, 0)
	if m, ok := n.mainchainMemberMapMap[epoch]; ok {
		for _, member := range m.Members() {
			mainchainAddrs = append(mainchainAddrs, member.Address)
		}
		return mainchainAddrs
	}

	// 先选出代表节点
	delegateAddrs := make([]cleisthenes.Address, 0)
	n.delegateMemberMapMap[epoch] = cleisthenes.NewMemberMap()
	for _, slice := range n.slices {
		delegateAddr := electDelegate.ElectDelegate(slice.Addrs, info)
		delegateAddrs = append(delegateAddrs, delegateAddr)
		n.delegateMemberMapMap[epoch].Add(cleisthenes.NewMemberWithAddress(delegateAddr))
		for _, mainChainAddr := range slice.Addrs {
			if mainChainAddr == n.mainchainAddr {
				n.log.Debug().Msgf("当前节点%s的领导节点为%s", n.mainchainAddr.String(), delegateAddr.String())
				n.delegateAddr[epoch] = delegateAddr
			}
		}
	}

	// 再从代表节点中选出主链节点
	n.mainchainMemberMapMap[epoch] = cleisthenes.NewMemberMap()
	for i := 0; i < len(delegateAddrs); i += n.gap {
		start := i
		end := i + n.gap
		if end > len(delegateAddrs) {
			end = len(delegateAddrs)
		}
		mainchainAddr := electDelegate.SelectMainChainNodeFromDelegate(delegateAddrs[start:end], info)
		mainchainAddrs = append(mainchainAddrs, mainchainAddr)
		n.mainchainMemberMapMap[epoch].Add(cleisthenes.NewMemberWithAddress(mainchainAddr))
		for _, delegateAddr := range delegateAddrs[start:end] {
			if delegateAddr == n.delegateAddr[epoch] {
				n.log.Debug().Msgf("当前节点%s的主链节点为%s", n.mainchainAddr.String(), mainchainAddr)
				n.mainChainAddr[epoch] = mainchainAddr
			}
		}
	}

	return mainchainAddrs
}

func (n *CtrlNode) IsDelegateNode(epoch cleisthenes.Epoch) bool {
	return n.delegateMemberMapMap[epoch].Exist(n.mainchainAddr)
}

func (n *CtrlNode) IsMainChainNode(epoch cleisthenes.Epoch) bool {
	return n.mainchainMemberMapMap[epoch].Exist(n.mainchainAddr)
}

func (n *CtrlNode) Run() {
	handler := newHandler(func(msg cleisthenes.Message) {
		lock.Lock()
		defer lock.Unlock()
		addr, err := cleisthenes.ToAddress(msg.Sender)
		if err != nil {
			n.log.Error().Msgf("[handler] failed to parse sender address: addr=%s", addr)
		}
		//接收到其他节点的请求后，会执行这里的handler函数
		if err := n.HandleMessage(msg.Message); err != nil {
			n.log.Error().Msgf("[handler] handle message failed with err: %s", err.Error())
		}
	})

	//n.server.OnConn.connHandler = conn，
	n.server.OnConn(func(conn cleisthenes.Connection) {
		conn.Handle(handler)
		if err := conn.Start(); err != nil {
			n.log.Error().Msgf("conn listen err:%v", err.Error())
			conn.Close()
		}
	})

	go n.server.Listen()
}

func (n *CtrlNode) HandleMessage(msg *pb.Message) error {
	//n.log.Debug().Msgf("收到ctrl消息, epoch:%v, from:%s proposer:%s", msg.Epoch, msg.Sender, msg.Proposer)

	if msg.Epoch > uint64(n.curEpoch) {
		n.reqRepo.Save(cleisthenes.Epoch(msg.Epoch), *msg)
		return nil
	}

	ctrl, err := n.findCtrl(cleisthenes.Epoch(msg.Epoch))
	if err != nil {
		return err
	}

	// 处理之前先收到的消息
	for _, m := range n.reqRepo.FindMsgsByEpochAndDelete(cleisthenes.Epoch(msg.Epoch)) {
		if err := ctrl.HandleMessage(&m); err != nil {
			return err
		}
	}

	return ctrl.HandleMessage(msg)
}

func (n *CtrlNode) ConnectAll() error {
	conf := config.Get()
	for _, slice := range conf.CtrlSlices {
		for _, addrStr := range slice.Slice {
			if err := n.Connect(addrStr); err != nil {
				return err
			}
		}
	}
	return nil
}

func (n *CtrlNode) Connect(target string) error {
	addr, err := cleisthenes.ToAddress(target)
	if err != nil {
		return err
	}
	return n.connect(addr)
}

func (n *CtrlNode) connect(addr cleisthenes.Address) error {
	if n.connPool.Get(addr) != nil {
		return nil
	}
	conn, err := n.client.Dial(cleisthenes.DialOpts{
		Addr:    addr,
		Timeout: cleisthenes.DefaultDialTimeout,
	})
	if err != nil {
		return err
	}
	go func() {
		if err := conn.Start(); err != nil {
			n.log.Error().Msgf("conn dia err:%v", err.Error())
			conn.Close()
		}
	}()

	n.connPool.Add(addr, conn)
	if n.branchCtrlMemberMap.Exist(addr) {
		n.branchConnPool.Add(addr, conn)
	}
	n.memberMap.Add(cleisthenes.NewMemberWithAddress(addr))
	return nil
}

func (n *CtrlNode) ConnectionList() []string {
	result := make([]string, 0)
	for _, conn := range n.connPool.GetAll() {
		result = append(result, conn.Ip().String())
	}
	return result
}

func (n *CtrlNode) Close() {
	n.server.Stop()
	for _, conn := range n.connPool.GetAll() {
		conn.Close()
	}

	n.log.Debug().Msgf("before ctrl repo size:%v", n.ctrlRepo.Size())
	n.ctrlRepo.Clear()
	n.log.Debug().Msgf("after ctrl repo size:%v", n.ctrlRepo.Size())
}

func (n *CtrlNode) Result() <-chan cleisthenes.CtrlResultMessage {
	return n.ctrlResultReceiver.Receive()
}
