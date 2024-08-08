package sliceSystem

import (
	"fmt"
	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/config"
	"github.com/bytedance/sonic"
	"github.com/rs/zerolog"
	"os"
	"path"
	"runtime"
	"sort"
	"sync"
)

var lock sync.Mutex

type SliceSystem interface {
	GetBranch() Branch
	GetMainChain() MainChain
	MainChainNetworkSize() int
	GetCtrl() Ctrl
	Submit(tx cleisthenes.Transaction) error
	SubmitTxs(txs []cleisthenes.Transaction) error
	ConnectAll() error
	ConnectionList() []string
}

type DefaultManualSubmitter struct {
	submitter cleisthenes.Submitter
}

func NewDefaultManualSubmitter(submitter cleisthenes.Submitter) cleisthenes.ManualSubmitter {
	return &DefaultManualSubmitter{submitter: submitter}
}

func (s *DefaultManualSubmitter) SubmitContribution(txs []cleisthenes.Transaction) {
	s.submitter.SubmitTxs(txs)
}

type DefaultSliceSystem struct {
	epoch                          cleisthenes.Epoch
	branch                         Branch
	mainchain                      MainChain
	ctrl                           Ctrl
	addr                           string
	branchTxQueueManager           *cleisthenes.ManualTxQueueManager
	resultMessageRepo              *cleisthenes.ResultMessageRepository
	mainChainNotSubmitHashFlagRepo *cleisthenes.FlagRespository
	mainChainNotSubmitTxsFlagRepo  *cleisthenes.FlagRespository
	mainChainElectedFlagRepo       *cleisthenes.FlagRespository
	// 从链每隔g轮向主链提交前g轮的结果 g = 主链batchSize/主链network
	g int

	mainChainNetworkSize int

	lastBlockHash string

	enableAdvance cleisthenes.BinaryState

	stateLock sync.Mutex
	log       zerolog.Logger
}

type handler struct {
	handleFunc func(msg cleisthenes.Message)
}

func newHandler(handleFunc func(cleisthenes.Message)) *handler {
	return &handler{
		handleFunc: handleFunc,
	}
}

func (h *handler) ServeRequest(msg cleisthenes.Message) {
	h.handleFunc(msg)
}

func New(validator cleisthenes.TxValidator) (SliceSystem, error) {
	branch, err := NewBranch(validator)
	if err != nil {
		return nil, err
	}

	mainchain, err := NewMainChain(validator)
	if err != nil {
		return nil, err
	}

	ctrl, err := NewCtrl()

	// 从链和主链节点后台运行并监听端口
	go func() {
		branch.Run()
		mainchain.Run()
		ctrl.Run()
	}()

	conf := config.Get()
	//g := conf.MainChain.Dumbo1.BatchSize / conf.MainChain.Dumbo1.NetworkSize
	g := conf.MainChain.Dumbo1.BatchSize / len(conf.Slices)
	contributionSize := conf.Branch.Dumbo1.BatchSize / conf.Branch.Dumbo1.NetworkSize * g

	s := &DefaultSliceSystem{
		epoch:     0,
		branch:    branch,
		mainchain: mainchain,
		ctrl:      ctrl,
		addr:      conf.MainChainIdentity.Address,
		branchTxQueueManager: cleisthenes.NewManualTxQueueManager(
			cleisthenes.NewTxQueue(),
			NewDefaultManualSubmitter(branch),
			contributionSize,
			0,
			validator,
		),
		resultMessageRepo:              cleisthenes.NewResultMessageRepository(),
		mainChainNotSubmitHashFlagRepo: cleisthenes.NewFlagRepository(),
		mainChainNotSubmitTxsFlagRepo:  cleisthenes.NewFlagRepository(),
		mainChainElectedFlagRepo:       cleisthenes.NewFlagRepository(),
		g:                              g,
		mainChainNetworkSize:           conf.MainChain.Dumbo1.NetworkSize,
		stateLock:                      sync.Mutex{},
		log:                            cleisthenes.NewLoggerWithHead("SLICE"),
	}

	s.mainChainNotSubmitHashFlagRepo.Save(0)

	s.initLog()

	s.log.Debug().Msgf("system g:%v, contributionSize:%v", s.g, contributionSize)

	// 创建创世区块
	b := cleisthenes.CreateGenesis()
	bs, err := b.Serialize()
	if err != nil {
		s.log.Fatal().Msgf("create genesis block failed, err:%s", err.Error())
	}
	cleisthenes.WriteBlock(0, bs)
	s.lastBlockHash = b.Header.Hash

	s.run()

	return s, nil
}

func (s *DefaultSliceSystem) initLog() {
	var logPrefix zerolog.HookFunc
	logPrefix = func(e *zerolog.Event, level zerolog.Level, message string) {
		e.Uint64("Epoch", uint64(s.epoch)).Int("g", s.g)
	}
	s.log = s.log.Hook(logPrefix)
}

func (s *DefaultSliceSystem) GetBranch() Branch {
	return s.branch
}

func (s *DefaultSliceSystem) GetMainChain() MainChain {
	return s.mainchain
}

func (s *DefaultSliceSystem) GetCtrl() Ctrl {
	return s.ctrl
}

func (s *DefaultSliceSystem) onlyRunBranch() bool {
	return s.mainChainNetworkSize == 1
}

func (s *DefaultSliceSystem) MainChainNetworkSize() int {
	return s.mainChainNetworkSize
}

func (s *DefaultSliceSystem) Submit(tx cleisthenes.Transaction) error {
	// FIXME sliceSystem要有自己的queueManager提交交易方便对交易进行管理
	if s.onlyRunBranch() {
		return s.branch.Submit(tx)
	} else {
		s.branchTxQueueManager.AddTransaction(tx)
		s.branchTxQueueManager.TryPropose()
	}
	return nil
}

func (s *DefaultSliceSystem) SubmitTxs(txs []cleisthenes.Transaction) error {
	if s.onlyRunBranch() {
		return s.branch.SubmitTxs(txs)
	} else {
		s.branchTxQueueManager.AddTransactions(txs)
		s.branchTxQueueManager.TryPropose()
	}
	return nil
}

func (s *DefaultSliceSystem) ConnectAll() error {
	// connect branch
	err := s.branch.ConnectAll()
	if err != nil {
		s.log.Fatal().Msgf("branch connect all failed, err:%s", err.Error())
	}
	s.log.Debug().Msgf("branch connected")

	// connect ctrl slices
	err = s.ctrl.ConnectAll()
	if err != nil {
		s.log.Fatal().Msgf("ctrl connect all failed, err:%s", err.Error())
	}
	s.log.Debug().Msg("ctrl slices connected")

	// connect mainchain
	err = s.mainchain.ConnectAll()
	if err != nil {
		s.log.Fatal().Msgf("mainchain connect all failed, err:%s", err.Error())
	}
	s.log.Debug().Msgf("mainchain connected")

	return nil
}

func (s *DefaultSliceSystem) ConnectionList() []string {
	return s.ctrl.ConnectionList()
}

func (s *DefaultSliceSystem) Close() {
	s.ctrl.Close()
	s.branch.Close()
	s.mainchain.Close()
}

func (s *DefaultSliceSystem) submitHash(epoch cleisthenes.Epoch) {
	s.log.Debug().Msgf("start to find %d epoch and submit hash", epoch)
	branchMsgQueue, _ := s.resultMessageRepo.Find(epoch)
	if branchMsgQueue.Len() < s.g {
		// FIXME 如果是拜占庭节点，中间可能会有某些branch轮数不会出结果
		s.log.Error().Msgf("submit hash,第%d轮branch结果还未出来, branch msg len:%d", epoch, branchMsgQueue.Len())
	}

	hash := cleisthenes.ResultMessageQueueHash(branchMsgQueue)
	//s.ctrl.SubmitHash(hash)
	s.ctrl.SubmitHashV2(epoch, hash)
}

func (s *DefaultSliceSystem) branchDone(result cleisthenes.ResultMessage, block *cleisthenes.Block) {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()

	runtime.GC()

	if s.onlyRunBranch() {
		//data, err := sonic.ConfigFastest.Marshal(result.Batch)
		//if err != nil {
		//	s.log.Fatal().Msgf("marshal result batch to bytes failed, err:%s", err.Error())
		//}
		//result.Batch = nil
		//s.writeToFile(result.Epoch, data)

		bs, err := block.Serialize()
		if err != nil {
			s.log.Fatal().Msgf("serialize block failed, err:%s", err.Error())
		}
		result.Batch = nil
		cleisthenes.WriteBlock(block.Header.Height, bs)
		return
	}

	s.resultMessageRepo.Save(cleisthenes.Epoch(int(result.Epoch)/s.g), result)
	s.log.Debug().Msgf("save %d branch result, result message repo size: %d", int(result.Epoch)/s.g, s.resultMessageRepo.Size())

	// 第一次选举触发
	if int(result.Epoch)%s.g == s.g-1 {
		mainChainEpoch := cleisthenes.Epoch(int(result.Epoch) / s.g)
		if s.mainChainNotSubmitHashFlagRepo.Find(mainChainEpoch) {
			s.log.Debug().Msgf("SUBMITHASH 第%d轮结果刚出，可以提交hash", mainChainEpoch)
			s.mainChainNotSubmitHashFlagRepo.Delete(mainChainEpoch)
			s.submitHash(mainChainEpoch)
		}

		if s.mainChainNotSubmitTxsFlagRepo.Find(mainChainEpoch) {
			s.log.Debug().Msgf("SUBMITTXS 第%d轮结果刚出，可以提交txs", mainChainEpoch)
			s.mainChainNotSubmitTxsFlagRepo.Delete(mainChainEpoch)
			s.submitTxsToMainChain(mainChainEpoch)
		}
	}
}

func (s *DefaultSliceSystem) trySubmitHash(epoch cleisthenes.Epoch) {
	branchMsgQueue, _ := s.resultMessageRepo.Find(epoch)
	if branchMsgQueue.Len() == s.g {
		s.log.Debug().Msgf("SUBMITHASH 第%d轮结果已出，可以提交hash", epoch)
		s.submitHash(epoch)
	} else {
		s.mainChainNotSubmitHashFlagRepo.Save(epoch)
	}
}

func (s *DefaultSliceSystem) mainChainDone(epoch cleisthenes.Epoch, cnt int, block *cleisthenes.Block) {
	go func() {
		block.Pack()
		bs, err := block.Serialize()
		if err != nil {
			s.log.Fatal().Msgf("serialize block failed, err:%s", err.Error())
		}
		cleisthenes.WriteBlock(block.Header.Height, bs)
	}()
	block.Pack()
	bs, err := block.Serialize()
	if err != nil {
		s.log.Fatal().Msgf("serialize block failed, err:%s", err.Error())
	}
	s.ctrl.SubmitMainChainDone(epoch, cnt, bs)

	//// 主链结束后需要等收到其他主链结束后的消息才能进入到下一轮
	//if !s.enableAdvance.Value() {
	//	s.enableAdvance.Set(true)
	//	return
	//}
	//
	//s.handleMainChainDone(epoch, cnt)
}

func (s *DefaultSliceSystem) writeToFile(epoch cleisthenes.Epoch, data []byte) {
	p := path.Join("output", fmt.Sprintf("%s-%v.txt", s.addr, epoch))
	file, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0766)
	if err != nil {
		s.log.Fatal().Msgf("create file %s failed, err:%s", p, err.Error())
	}
	defer file.Close()
	_, err = file.Write(data)
	if err != nil {
		s.log.Fatal().Msgf("write file %s failed, err:%s", p, err.Error())
	}
	if err = file.Sync(); err != nil {
		s.log.Fatal().Msgf("sync file %s failed, err:%s", p, err.Error())
	}
}

func (s *DefaultSliceSystem) handleMainChainDoneMsg(result *cleisthenes.CtrlResultMessage) {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	beforeAlloc := mem.Alloc
	s.log.Debug().Msgf("mem statics before Alloc: %v bytes, result.Epoch:%v", mem.Alloc, result.Epoch)
	s.log.Debug().Msgf("mem statics before TotalAlloc: %v bytes, result.Epoch:%v", mem.TotalAlloc, result.Epoch)
	s.log.Debug().Msgf("mem statics before Sys: %v bytes, result.Epoch:%v", mem.Sys, result.Epoch)
	s.log.Debug().Msgf("mem statics before NumGC: %v, result.Epoch:%v", mem.NumGC, result.Epoch)
	runtime.GC()
	runtime.ReadMemStats(&mem)
	afterAlloc := mem.Alloc
	s.log.Debug().Msgf("mem statics after Alloc: %v bytes, result.Epoch:%v", mem.Alloc, result.Epoch)
	s.log.Debug().Msgf("mem statics after TotalAlloc: %v bytes, result.Epoch:%v", mem.TotalAlloc, result.Epoch)
	s.log.Debug().Msgf("mem statics after Sys: %v bytes, result.Epoch:%v", mem.Sys, result.Epoch)
	s.log.Debug().Msgf("mem statics after NumGC: %v, result.Epoch:%v", mem.NumGC, result.Epoch)
	s.log.Debug().Msgf("mem statics release alloc:%v bytes, result.Epoch:%v", beforeAlloc-afterAlloc, result.Epoch)
	// TODO 同步交易

	// TODO 当前节点处理落后，直接同步并重新提交交易
	//if result.Epoch > s.ctrl.Epoch() {
	//	s.log.Warn().Msgf("main chain done epoch:%d, local epoch:%d", result.Epoch, s.ctrl.Epoch())
	//	// 主链结束，同步epoch
	//	// ctrl 同步epoch
	//	cnt := 0
	//	from := s.ctrl.Epoch()
	//	to := result.Epoch
	//	for result.Epoch > s.ctrl.Epoch() {
	//		cnt++
	//		s.ctrl.Advance()
	//	}
	//	s.log.Debug().Msgf("ctrl added %d epoch, from:%d to:%d", cnt, from, to)
	//	//return
	//}

	s.log.Debug().Msgf("recv mainchain done msg, result message size:%d result Epoch:%v", s.resultMessageRepo.Size(), result.Epoch)
	// TODO 可能收到了mainchain done消息，但是还未获取到选举结果，此时直接同步（包括交易和epoch的同步）
	if !s.mainChainElectedFlagRepo.Find(result.Epoch) {
		s.log.Warn().Msgf("elect not found but recv mainchain done, mainchain epoch:%d", result.Epoch)
		// TODO 交易的同步
		//cleisthenes.MainTps.CompleteCount <- cleisthenes.EpochResult{
		//	Epoch: result.Epoch,
		//	Count: result.MainChainTxCnt,
		//}
		// TODO 主链之前的epoch全部close，从链之前的epoch close
		//return
	}

	//// 如果是当前节点是主链，需要等主链结束，主链结束，异步共识的交易才会同步过来
	//if !s.enableAdvance.Value() && s.ctrl.IsMainChainNode(result.Epoch) {
	//	s.enableAdvance.Set(true)
	//	return
	//}

	// 如果不是主链节点，收到主链结束消息后可以马上进入到下一轮
	s.handleMainChainDone(result.Epoch, result.MainChainTxCnt)
}

func (s *DefaultSliceSystem) handleMainChainDone(epoch cleisthenes.Epoch, cnt int) {
	/*
		FIXME:
			关闭历史轮次的主链实例需要注意：有时候主链节点的BBA结果没有达成，其他主链节点已经关闭了该主链，导致
			BBA结果没有达成的主链一直卡在BBA阶段无法结束
	*/

	s.mainchain.CloseOldEpoch(epoch)

	s.log.Debug().Msgf("主链(epoch:%v)结束，开始分发第(%v)轮随机数", epoch, epoch+1)
	// FIXME 在advance的时候可能同步的data尚未收到
	// ctrl epoch + 1
	s.ctrl.Advance(epoch + 1)
	runtime.GC()
	s.trySubmitHash(epoch + 1)

	s.branchTxQueueManager.FinConsensus()

	cleisthenes.MainTps.CompleteCount <- cleisthenes.EpochResult{
		Epoch: epoch,
		Count: cnt,
	}

	s.log.Debug().Msgf("recv mainchain done msg, try propose branch, result Epoch:%v", epoch)
	// FIXME 可能有些节点运行比其他节点慢，还没提交，当前轮就结束了，此时提交会提前进入到下一轮
	if err := s.branchTxQueueManager.TryPropose(); err != nil {
		s.log.Warn().Msgf("propose branch failed, result.Epoch:%v, err:%v", epoch, err.Error())
	}

	// advance后将状态回置
	s.enableAdvance.Set(false)
}

func (s *DefaultSliceSystem) submitTxsToMainChain(epoch cleisthenes.Epoch) {
	branchMsgQueue, _ := s.resultMessageRepo.FindAndDelete(epoch)
	if branchMsgQueue.Len() != s.g {
		s.log.Fatal().Msgf("handle elect msg,branch result is not complete for epoch:%d, len:%d", epoch, branchMsgQueue.Len())
	}

	s.log.Debug().Msgf("find and delete %d epoch, result message repo size: %d", epoch, s.resultMessageRepo.Size())

	if s.ctrl.IsMainChainNode(epoch) {
		var txs []cleisthenes.Transaction
		for _, branchResult := range branchMsgQueue {
			txs = append(txs, cleisthenes.Transaction(branchResult.Batch))
		}
		s.log.Debug().Msgf("开始向主链节点提交交易, 数量:%d, Epoch:%d", len(txs), epoch)
		s.mainchain.SubmitTxs(epoch, txs)
	} else if s.ctrl.IsDelegateNode(epoch) {
		var txs []cleisthenes.Transaction
		for _, branchResult := range branchMsgQueue {
			txs = append(txs, cleisthenes.Transaction(branchResult.Batch))
		}
		s.log.Debug().Msgf("代表节点开始向主链节点提交交易, 数量:%d, Epoch:%d", len(txs), epoch)
		// FIXME epoch是否存在和ctrl的Epoch不一致的情况？
		s.ctrl.SubmitTxsToMainChain(epoch, txs)
	} else {
		s.log.Debug().Msgf("当前节点非主链节点，无需提交")
	}
}

func (s *DefaultSliceSystem) handleElectMsg(result *cleisthenes.CtrlResultMessage) {
	s.stateLock.Lock()
	defer s.stateLock.Unlock()

	mainChainAddrs := s.ctrl.Elect(result.Epoch, &result.CtrlInfo)
	s.log.Debug().Msgf("mainChainAddrs, epoch:%d, mainChainAddrs:%v", result.Epoch, mainChainAddrs)
	if s.ctrl.IsMainChainNode(result.Epoch) {
		s.log.Debug().Msgf("当前节点为主链节点，setMembers result.Epoch:%v", result.Epoch)
		s.mainchain.SetMembers(result.Epoch, mainChainAddrs)
	}
	s.mainChainElectedFlagRepo.Save(result.Epoch)

	// 有可能选举结果出来了，但是本地的交易结果还没出来
	branchMsgQueue, _ := s.resultMessageRepo.Find(result.Epoch)
	if branchMsgQueue.Len() == s.g {
		s.submitTxsToMainChain(result.Epoch)
	} else {
		s.mainChainNotSubmitTxsFlagRepo.Save(result.Epoch)
	}
}

func (s *DefaultSliceSystem) run() {
	go func() {
		for {
			select {
			case result := <-s.branch.Result():
				s.log.Info().Msgf("[BRANCH DONE]epoch : %d, batch tx count : %d\n", result.Epoch, len(result.Batch))

				block := cleisthenes.NewBlock(uint64(result.Epoch)+1, s.lastBlockHash)

				cnt := 0

				// 需要对txs按照节点进行排序
				var keys []string
				for key, _ := range result.Batch {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				for _, key := range keys {
					txs := result.Batch[key]
					cnt += len(txs)
					var ts []interface{}
					for _, absTx := range txs {
						ts = append(ts, absTx.(interface{}))
					}
					block.AddTxs(ts)
					//cleisthenes.WriteIndex(block.Header.Height, block.Payload.Txs)
				}

				block.Pack()
				s.lastBlockHash = block.Header.Hash

				// 将从链结果保存起来，后续提交到主链中，等待后续主链选出了代表节点，再提交
				s.branchDone(result, block)

				cleisthenes.BranchTps.CompleteCount <- cleisthenes.EpochResult{
					Epoch: result.Epoch,
					Count: cnt,
				}

			case result := <-s.mainchain.Result():
				s.log.Info().Msgf("[MAINCHAIN DONE]epoch : %d, batch tx count : %d\n", result.Epoch, len(result.Batch))

				block := cleisthenes.NewBlock(uint64(result.Epoch)+1, s.lastBlockHash)

				cnt := 0

				var mainAddrs []string
				for addr, _ := range result.Batch {
					mainAddrs = append(mainAddrs, addr)
				}
				sort.Strings(mainAddrs)
				for _, mainAddr := range mainAddrs {
					mainTxs := result.Batch[mainAddr]
					for _, branchBatch := range mainTxs {
						branchMap := branchBatch.(map[string]interface{})
						var branchAddrs []string
						for addr, _ := range branchMap {
							branchAddrs = append(branchAddrs, addr)
						}
						sort.Strings(branchAddrs)
						for _, branchAddr := range branchAddrs {
							txs := branchMap[branchAddr].([]interface{})
							cnt += len(txs)
							block.AddTxs(txs)
							//cleisthenes.WriteIndex(uint64(result.Epoch), txs)
						}
					}
				}
				//for _, branchTxs := range result.Batch {
				//	for _, absTx := range branchTxs {
				//		//jsonparser.ArrayEach(tx.([]byte), func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
				//		//	cnt += 1
				//		//})
				//		for _, bAbsTxs := range absTx.([]interface{}) {
				//			for _, bAbsTx := range bAbsTxs.([]interface{}) {
				//				for _, txs := range bAbsTx.(map[string]interface{}) {
				//					ts := txs.([]interface{})
				//					cnt += len(ts)
				//					block.AddTxs(ts)
				//					cleisthenes.WriteIndex(uint64(result.Epoch), ts)
				//				}
				//			}
				//		}
				//	}
				//}

				//block.Pack()
				//s.lastBlockHash = block.Header.Hash

				s.mainChainDone(result.Epoch, cnt, block)

				//cleisthenes.MainTps.CompleteCount <- cleisthenes.EpochResult{
				//	Epoch: result.Epoch,
				//	Count: cnt,
				//}

			case result := <-s.ctrl.Result():
				//  如果收到其他节点的随机数信息，对该信息对应的epoch进行计算得到新的主链代表节点，并向主链提交交易，如果epoch的主链代表节点已经选出，则不操作
				if result.MsgType == cleisthenes.CtrlInfoType {
					//if result.Epoch != s.ctrl.Epoch() {
					//	s.log.Debug().Msgf("recv result Epoch(%d) != cur ctrl Epoch(%d)", result.Epoch, s.ctrl.Epoch())
					//}
					s.log.Info().Msgf("[RANDOM DONE] epoch:%d, result:%v", result.Epoch, result.CtrlInfo)
					s.handleElectMsg(&result)
				}

				//  如果收到主链的结束信息，提交随机数进行选举（一般来说从链会每隔s.g轮提交一次）
				if result.MsgType == cleisthenes.FlagType && result.MainChainFinish {
					s.handleMainChainDoneMsg(&result)
				}

				if result.MsgType == cleisthenes.TransactionsType {
					s.log.Debug().Msgf("收到了transactions消息，result.Epoch:%v", result.Epoch)
					var txs []cleisthenes.Transaction
					err := sonic.ConfigFastest.Unmarshal(result.Data, &txs)
					if err != nil {
						s.log.Fatal().Msgf("unmarshal transactions failed, err:%s", err.Error())
					}
					s.mainchain.SubmitTxs(result.Epoch, txs)
				}

				if result.MsgType == cleisthenes.DataType {
					go s.writeToFile(result.Epoch, result.Data)
				}
			}
		}
	}()
}
