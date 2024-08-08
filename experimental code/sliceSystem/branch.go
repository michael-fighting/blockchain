package sliceSystem

import (
	"fmt"
	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/config"
	"github.com/DE-labtory/cleisthenes/sliceSystem/branch/dumbo1"
	"github.com/DE-labtory/cleisthenes/tpke"
	"github.com/DE-labtory/cleisthenes/tss"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog"
)

type Branch interface {
	Submit(tx cleisthenes.Transaction) error
	SubmitTxs(txs []cleisthenes.Transaction) error
	Connect(target string) error
	ConnectAll() error
	ConnectionList() []string
	Run()
	Result() <-chan cleisthenes.ResultMessage
	Close()
}

type BranchNode struct {
	addr            cleisthenes.Address
	txQueueManager  cleisthenes.TxQueueManager
	resultReceiver  cleisthenes.ResultReceiver
	messageEndpoint cleisthenes.MessageEndpoint
	server          *cleisthenes.GrpcServer
	client          *cleisthenes.GrpcClient
	connPool        *cleisthenes.ConnectionPool
	memberMap       *cleisthenes.MemberMap
	txValidator     cleisthenes.TxValidator
	log             zerolog.Logger
}

func NewBranch(txValidator cleisthenes.TxValidator) (Branch, error) {
	//获取指定文件的配置
	conf := config.Get()

	identity := conf.BranchIdentity

	local, err := cleisthenes.ToAddress(identity.Address)
	if err != nil {
		return nil, err
	}
	external, err := cleisthenes.ToAddress(identity.ExternalAddress)
	if err != nil {
		return nil, err
	}

	connPool := cleisthenes.NewConnectionPool()

	Dumbo1 := conf.Branch.Dumbo1
	Members := conf.Branch.Members
	Tpke := conf.Tpke

	// 在ConnectAll中会自动将member添加进去
	memberMap := cleisthenes.NewMemberMap()
	//for _, member := range Members {
	//	addr, err := cleisthenes.ToAddress(member.Address)
	//	if err != nil {
	//		return nil, err
	//	}
	//	allMainChainMemberMap.Add(cleisthenes.NewMemberWithAddress(addr))
	//}

	dataChan := cleisthenes.NewDataChannel(Dumbo1.NetworkSize * 100)
	//batchChan := cleisthenes.NewBatchChannel(10)
	batchChan := cleisthenes.NewBatchChannel(Dumbo1.BatchSize)
	binChan := cleisthenes.NewBinaryChannel(Dumbo1.NetworkSize * 100)
	//resultChan := cleisthenes.NewResultChannel(10)
	resultChan := cleisthenes.NewResultChannel(Dumbo1.BatchSize)

	txQueue := cleisthenes.NewTxQueue()

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
	for _, member := range Members {
		addr, err := cleisthenes.ToAddress(member.Address)
		if err != nil {
			return nil, err
		}
		gpkshareMap[addr] = member.GpkShare
		rshareMap[addr] = member.RShare
		xMap[addr] = member.X
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

	var db1 cleisthenes.Dumbo1
	// new dumbo1

	db1 = dumbo1.New(
		Dumbo1.NetworkSize,
		Dumbo1.Byzantine,
		*cleisthenes.NewMemberWithAddress(external),
		memberMap,
		dumbo1.NewDefaultD1ACSFactory(
			Dumbo1.NetworkSize,
			Dumbo1.Byzantine,
			Dumbo1.CommitteeSize,
			*cleisthenes.NewMemberWithAddress(external),
			memberMap,
			// CE委员会使用的门限签名实例
			t,
			tp,
			dataChan,
			dataChan,
			binChan,
			binChan,
			batchChan,
			connPool,
		),
		//&tpke.MockTpke{},
		tp,
		connPool,
		batchChan,
		resultChan,
	)

	return &BranchNode{
		addr: external,
		txQueueManager: cleisthenes.NewDefaultTxQueueManager(
			txQueue,
			db1,
			Dumbo1.BatchSize/Dumbo1.NetworkSize,
			// TODO : contribution size = B / N
			Dumbo1.BatchSize,
			Dumbo1.ProposeInterval,
			txValidator,
		),
		resultReceiver:  resultChan,
		messageEndpoint: db1,
		server:          cleisthenes.NewServer(local, t),
		client:          cleisthenes.NewClient(t),
		connPool:        connPool,
		memberMap:       memberMap,
		log:             cleisthenes.NewLoggerWithHead("BRANCH"),
	}, nil
}

func (n *BranchNode) Run() {
	handler := newHandler(func(msg cleisthenes.Message) {
		lock.Lock()
		defer lock.Unlock()
		addr, err := cleisthenes.ToAddress(msg.Sender)
		if err != nil {
			fmt.Println(fmt.Errorf("[handler] failed to parse sender address: addr=%s", addr))
		}
		//接收到其他节点的请求后，会执行这里的handler函数
		if err := n.messageEndpoint.HandleMessage(msg.Message); err != nil {
			fmt.Println(fmt.Errorf("[handler] handle message failed with err: %s msgId:%v", err.Error(), string(msg.Signature)))
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

func (n *BranchNode) Submit(tx cleisthenes.Transaction) error {
	return n.txQueueManager.AddTransaction(tx)
}

func (n *BranchNode) SubmitTxs(txs []cleisthenes.Transaction) error {
	return n.txQueueManager.AddTransactions(txs)
}

func (n *BranchNode) ConnectAll() error {
	conf := config.Get()
	for _, member := range conf.Branch.Members {
		if err := n.Connect(member.Address); err != nil {
			return err
		}
	}

	return nil
}

func (n *BranchNode) Connect(target string) error {
	addr, err := cleisthenes.ToAddress(target)
	if err != nil {
		return err
	}
	return n.connect(addr)
}

func (n *BranchNode) connect(addr cleisthenes.Address) error {
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
	n.memberMap.Add(cleisthenes.NewMemberWithAddress(addr))
	return nil
}

func (n *BranchNode) ConnectionList() []string {
	result := make([]string, 0)
	for _, conn := range n.connPool.GetAll() {
		result = append(result, conn.Ip().String())
	}
	return result
}

func (n *BranchNode) Close() {
	n.server.Stop()
	for _, conn := range n.connPool.GetAll() {
		conn.Close()
	}
}

func (n *BranchNode) Result() <-chan cleisthenes.ResultMessage {
	return n.resultReceiver.Receive()
}
