package sliceSystem

import (
	"fmt"
	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/config"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/DE-labtory/cleisthenes/sliceSystem/mainchain"
	"github.com/DE-labtory/cleisthenes/tpke"
	"github.com/DE-labtory/cleisthenes/tss"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog"
)

type MainChain interface {
	SubmitTxs(epoch cleisthenes.Epoch, txs []cleisthenes.Transaction)
	SetMembers(epoch cleisthenes.Epoch, addrs []cleisthenes.Address) error
	Connect(target string) error
	ConnectAll() error
	ConnectionList() []string
	Run()
	Result() <-chan cleisthenes.ResultMessage
	Close()
	CloseOldEpoch(epoch cleisthenes.Epoch)
}

type MainChainNode struct {
	epoch cleisthenes.Epoch
	addr  cleisthenes.Address
	//txQueueManager cleisthenes.TxQueueManager
	resultReceiver cleisthenes.ResultReceiver
	//messageEndpoint cleisthenes.MainChainMessageEndpoint
	server      *cleisthenes.GrpcServer
	client      *cleisthenes.GrpcClient
	allConnPool *cleisthenes.ConnectionPool
	//connPool     *cleisthenes.ConnectionPool
	allMemberMap *cleisthenes.MemberMap
	//memberMap    *cleisthenes.MemberMap
	reqRepo      *cleisthenes.ReqRepo
	conbRepo     *cleisthenes.ContributionRepo
	dumbo1Repo   *mainchain.Dumbo1Repository
	txValidator  cleisthenes.TxValidator
	canHandleMsg int32
	log          zerolog.Logger
}

func NewMainChain(txValidator cleisthenes.TxValidator) (MainChain, error) {
	//获取指定文件的配置
	conf := config.Get()
	identity := conf.MainChainIdentity
	local, err := cleisthenes.ToAddress(identity.Address)
	if err != nil {
		return nil, err
	}
	external, err := cleisthenes.ToAddress(identity.ExternalAddress)
	if err != nil {
		return nil, err
	}

	allConnPool := cleisthenes.NewConnectionPool()

	Dumbo1 := conf.MainChain.Dumbo1
	Members := conf.MainChain.Members
	Tpke := conf.MTpke

	// memberMap要在主链每一轮结束后才能确定
	allMemberMap := cleisthenes.NewMemberMap()

	//resultChan := cleisthenes.NewResultChannel(10)
	resultChan := cleisthenes.NewResultChannel(Dumbo1.BatchSize)

	tpSecKey := cleisthenes.SecretKey{}
	tpSecBytes := hexutil.MustDecode(Tpke.SecretKeyShare)

	for i := 0; i < 32; i++ {
		tpSecKey[i] = tpSecBytes[i]
	}

	tp, err := tpke.NewDefaultTpke(Dumbo1.Byzantine,
		tpSecKey,
		cleisthenes.PublicKey(hexutil.MustDecode(Tpke.MasterPublicKey)),
	)
	if err != nil {
		return nil, err
	}

	gpkshareMap := make(map[cleisthenes.Address]string)
	rshareMap := make(map[cleisthenes.Address]string)
	xMap := make(map[cleisthenes.Address]string)
	// gpkshareMap,rshareMap,xMap保存
	gap := (len(conf.Slices) + len(conf.MainChain.Members) - 1) / len(conf.MainChain.Members)
	for index, slice := range conf.Slices {
		member := Members[index/gap]
		for _, addrStr := range slice.Slice {
			addr, err := cleisthenes.ToAddress(addrStr)
			if err != nil {
				return nil, err
			}
			gpkshareMap[addr] = member.GpkShare
			rshareMap[addr] = member.RShare
			xMap[addr] = member.X
		}
	}

	t, err := tss.NewDefaultTss(
		Dumbo1.NetworkSize,
		Dumbo1.Byzantine,
		identity.Gpk,
		identity.Rpk,
		identity.GpkShare,
		identity.GskShare,
		identity.RShare,
		gpkshareMap,
		rshareMap,
		xMap,
	)
	if err != nil {
		return nil, err
	}

	dumbo1Repo := mainchain.NewDumbo1Repository(
		Dumbo1.NetworkSize,
		Dumbo1.Byzantine,
		Dumbo1.CommitteeSize,
		Dumbo1.BatchSize,
		cleisthenes.NewMemberWithAddress(external),
		t,
		tp,
		resultChan,
	)

	node := &MainChainNode{
		epoch:          0,
		addr:           external,
		resultReceiver: resultChan,
		server:         cleisthenes.NewServer(local, t),
		client:         cleisthenes.NewClient(t),
		allConnPool:    allConnPool,
		allMemberMap:   allMemberMap,
		reqRepo:        cleisthenes.NewReqRepo(),
		conbRepo:       cleisthenes.NewContributionRepo(),
		dumbo1Repo:     dumbo1Repo,
		log:            cleisthenes.NewLoggerWithHead("MCHAIN"),
	}

	node.initLog()

	return node, nil
}

func (n *MainChainNode) initLog() {
	var logPrefix zerolog.HookFunc
	logPrefix = func(e *zerolog.Event, level zerolog.Level, message string) {
		e.Uint64("Epoch", uint64(n.epoch)).
			Str("owner", n.addr.String())
	}
	n.log = n.log.Hook(logPrefix)
}

func (n *MainChainNode) Run() {
	handler := newHandler(func(msg cleisthenes.Message) {
		lock.Lock()
		defer lock.Unlock()
		addr, err := cleisthenes.ToAddress(msg.Sender)
		if err != nil {
			n.log.Error().Msgf("[handler] failed to parse sender address: addr=%s", addr)
		}
		//接收到其他节点的请求后，会执行这里的handler函数
		if err = n.HandleMessage(msg.Message); err != nil {
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

func (n *MainChainNode) HandleMessage(msg *pb.Message) error {
	switch pl := msg.Payload.(type) {
	case *pb.Message_Rbc:
		switch pl.Rbc.Type {
		case pb.RBC_VAL:
			{
				n.log.Debug().Msgf("node recv rbc/irbc val message from sender:%v proposer:%v epoch:%v", msg.Sender, msg.Proposer, msg.Epoch)
			}
		}
	}

	epoch := cleisthenes.Epoch(msg.Epoch)

	if epoch+10 <= n.epoch {
		n.log.Warn().Msgf("收到了10个epoch之前的消息, msg epoch:%v mainchain epoch:%v", epoch, n.epoch)
		return nil
	}

	// FIXME 这里的req repo可能存在泄露，没有删除
	n.log.Debug().Msgf("req repo size:%d", n.reqRepo.Size())

	if dumbo1 := n.dumbo1Repo.Find(epoch); dumbo1 != nil {
		// 之前存储的消息清空
		for _, m := range n.reqRepo.FindMsgsByEpochAndDelete(epoch) {
			if err := dumbo1.HandleMessage(&m); err != nil {
				return err
			}
		}
		return dumbo1.HandleMessage(msg)
	}

	// 未初始化的dumbo1，先保存请求
	n.reqRepo.Save(epoch, *msg)
	return nil
}

func (n *MainChainNode) SubmitTxs(epoch cleisthenes.Epoch, txs []cleisthenes.Transaction) {
	if epoch+10 <= n.epoch {
		n.log.Warn().Msgf("收到了10个epoch之前的contribution, contribution epoch:%v mainchain epoch:%v", epoch, n.epoch)
		return
	}

	contribution := cleisthenes.Contribution{TxList: txs}
	if dumbo1 := n.dumbo1Repo.Find(epoch); dumbo1 != nil {
		// FIXME 什么时候删掉老的dumbo1
		if epoch > 10 {
			n.log.Debug().Msgf("start to delete epoch dumbo1, epoch:%d dumbo1RepoSize:%d", epoch-10, n.dumbo1Repo.Size())
			n.dumbo1Repo.DeleteEpochLessEqual(epoch - 10)
		}
		// FIXME 这里主链的提交大小没有和batchSize联系起来
		// 清空之前存的contribution
		for _, contrib := range n.conbRepo.FindContributionsByEpochAndDelete(epoch) {
			dumbo1.HandleContribution(*contrib)
		}
		dumbo1.HandleContribution(contribution)
		return
	}

	n.log.Warn().Msgf("recv not initial dumbo1 contribution, recv epoch:%v, cur epoch:%v", epoch, n.epoch)
	// 未初始化的dumbo1，先保存contribution
	n.conbRepo.Save(epoch, &contribution)
}

func (n *MainChainNode) SetMembers(epoch cleisthenes.Epoch, addrs []cleisthenes.Address) error {
	n.epoch = epoch

	n.log.Debug().Msgf("dumbo1 repo size:%d", n.dumbo1Repo.Size())

	memberMap := cleisthenes.NewMemberMap()
	connPool := cleisthenes.NewConnectionPool()
	for _, addr := range addrs {
		connPool.Add(addr, n.allConnPool.Get(addr))
		memberMap.Add(cleisthenes.NewMemberWithAddress(addr))
	}
	n.log.Debug().Msgf("connPool:%v", connPool.GetAll())

	// 创建新的dumbo1
	dumbo1 := n.dumbo1Repo.Create(epoch, memberMap, connPool)
	if err := n.dumbo1Repo.Save(epoch, dumbo1); err != nil {
		n.log.Fatal().Msgf("duplicate created dumbo, err:%s", err.Error())
	}
	for _, msg := range n.reqRepo.FindMsgsByEpochAndDelete(epoch) {
		if err := dumbo1.HandleMessage(&msg); err != nil {
			fmt.Println(fmt.Errorf("[handler] handle message failed with err: %s", err.Error()))
		}
	}
	for _, contribution := range n.conbRepo.FindContributionsByEpochAndDelete(epoch) {
		dumbo1.HandleContribution(*contribution)
	}

	return nil
}

func (n *MainChainNode) ConnectAll() error {
	conf := config.Get()
	for _, slice := range conf.Slices {
		for _, addrStr := range slice.Slice {
			if err := n.Connect(addrStr); err != nil {
				return err
			}
		}
	}

	return nil
}

func (n *MainChainNode) Connect(target string) error {
	addr, err := cleisthenes.ToAddress(target)
	if err != nil {
		return err
	}
	return n.connect(addr)
}

func (n *MainChainNode) connect(addr cleisthenes.Address) error {
	if n.allConnPool.Get(addr) != nil {
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

	n.allConnPool.Add(addr, conn)
	n.allMemberMap.Add(cleisthenes.NewMemberWithAddress(addr))
	return nil
}

func (n *MainChainNode) ConnectionList() []string {
	result := make([]string, 0)
	for _, conn := range n.allConnPool.GetAll() {
		result = append(result, conn.Ip().String())
	}
	return result
}

func (n *MainChainNode) Close() {
	n.server.Stop()
	for _, conn := range n.allConnPool.GetAll() {
		conn.Close()
	}

	n.log.Debug().Msgf("before dumbo1 repo size:%v", n.dumbo1Repo.Size())
	n.log.Debug().Msgf("req repo size:%v", n.reqRepo.Size())
	n.log.Debug().Msgf("conb repo size:%v", n.conbRepo.Size())
	n.dumbo1Repo.DeleteEpochLessEqual(n.epoch + 10)
	n.log.Debug().Msgf("after dumbo1 repo size:%v", n.dumbo1Repo.Size())
}

func (n *MainChainNode) CloseOldEpoch(epoch cleisthenes.Epoch) {
	n.dumbo1Repo.DeleteEpochLessEqual(epoch)
}

func (n *MainChainNode) Result() <-chan cleisthenes.ResultMessage {
	return n.resultReceiver.Receive()
}
