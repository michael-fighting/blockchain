package bba

import (
	"errors"
	"fmt"
	"github.com/DE-labtory/cleisthenes/test/mock"
	"github.com/bytedance/sonic"
	"github.com/golang/protobuf/ptypes"
	"github.com/rs/zerolog"
	"math/big"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/DE-labtory/cleisthenes/pb"

	"github.com/DE-labtory/cleisthenes"
)

var MAX_ROUND uint64 = 10

type request struct {
	sender cleisthenes.Member
	data   cleisthenes.Request
	round  uint64
	err    chan error
}

type coinResult struct {
	round uint64
	coin  cleisthenes.Coin
}

type round struct {
	lock sync.RWMutex
	val  uint64
}

func newRound() *round {
	return &round{
		lock: sync.RWMutex{},
		val:  0,
	}
}

func (r *round) inc() uint64 {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.val++
	return r.val
}

func (r *round) value() uint64 {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.val
}

type BBA struct {
	sync.RWMutex
	cleisthenes.Tracer

	tpk cleisthenes.Tpke
	tss cleisthenes.Tss

	owner    cleisthenes.Member
	proposer cleisthenes.Member
	// number of network nodes
	n int
	// number of byzantine nodes which can tolerate
	f int

	epoch cleisthenes.Epoch

	stopFlag int32
	// done is flag whether BBA is terminated or not
	done *cleisthenes.BinaryState
	// round is value for current Binary Byzantine Agreement round
	round       *round
	binValueSet []*voteSet
	valsSet     []*binarySet
	// broadcastedBvalSet is set of bval value instance has sent
	broadcastedBvalSet []*voteSet
	// est is estimated value of BBA instance, dec is decision value
	est              []*cleisthenes.BinaryState
	dec              *cleisthenes.BinaryState
	auxBroadcasted   []*cleisthenes.BinaryState
	coinCompleted    []*cleisthenes.BinaryState
	binValueChanSent []*cleisthenes.BinaryState

	bvalRepo         []cleisthenes.RequestRepository
	auxRepo          []cleisthenes.RequestRepository
	mainVoteRepo     []cleisthenes.RequestRepository
	finalVoteRepo    []cleisthenes.RequestRepository
	coinSigShareRepo []cleisthenes.RequestRepository
	incomingReqRepo  incomingRequestRepository

	bvalSent      cleisthenes.BinaryState
	auxSent       cleisthenes.BinaryState
	mainVoteSent  cleisthenes.BinaryState
	finalVoteSent cleisthenes.BinaryState

	reqChan      chan request
	coinChan     chan coinResult
	closeChan    chan struct{}
	binValueChan chan struct {
		round uint64
		vote  cleisthenes.Vote
	}
	tryoutAgreementChan chan uint64
	advanceRoundChan    chan uint64

	broadcaster   cleisthenes.Broadcaster
	coinGenerator cleisthenes.CoinGenerator
	binInputChan  cleisthenes.BinarySender

	log zerolog.Logger
}

func New(
	n, f int,
	epoch cleisthenes.Epoch,
	owner cleisthenes.Member,
	proposer cleisthenes.Member,
	broadcaster cleisthenes.Broadcaster,
	binInputChan cleisthenes.BinarySender,
	tpk cleisthenes.Tpke,
	tss cleisthenes.Tss,
) *BBA {
	instance := &BBA{
		tpk:      tpk,
		tss:      tss,
		owner:    owner,
		proposer: proposer,
		n:        n,
		f:        f,
		epoch:    epoch,
		round:    newRound(),

		//binValueSet:        make([]*binarySet, n+1),
		//broadcastedBvalSet: make([]*binarySet, n+1),
		binValueSet:        make([]*voteSet, MAX_ROUND+2),
		valsSet:            make([]*binarySet, MAX_ROUND+2),
		broadcastedBvalSet: make([]*voteSet, MAX_ROUND+2),

		done: cleisthenes.NewBinaryState(),
		//est:  make([]*cleisthenes.BinaryState, n+1),
		est: make([]*cleisthenes.BinaryState, MAX_ROUND+2),
		dec: cleisthenes.NewBinaryState(),

		//bvalRepo: make([]cleisthenes.RequestRepository, n+1),
		//auxRepo:  make([]cleisthenes.RequestRepository, n+1),
		bvalRepo:         make([]cleisthenes.RequestRepository, MAX_ROUND+2),
		auxRepo:          make([]cleisthenes.RequestRepository, MAX_ROUND+2),
		mainVoteRepo:     make([]cleisthenes.RequestRepository, MAX_ROUND+2),
		finalVoteRepo:    make([]cleisthenes.RequestRepository, MAX_ROUND+2),
		coinSigShareRepo: make([]cleisthenes.RequestRepository, MAX_ROUND+2),

		//auxBroadcasted: make([]*cleisthenes.BinaryState, n+1),
		auxBroadcasted:   make([]*cleisthenes.BinaryState, MAX_ROUND+2),
		coinCompleted:    make([]*cleisthenes.BinaryState, MAX_ROUND+2),
		binValueChanSent: make([]*cleisthenes.BinaryState, MAX_ROUND+2),
		incomingReqRepo:  newDefaultIncomingRequestRepository(),

		closeChan: make(chan struct{}),

		// request channel size as n*8, because each node can have maximum 4 rounds
		// and in each round each node can broadcast at most twice (broadcast BVAL, AUX)
		// so each node should handle 8 requests per node.
		//reqChan: make(chan request, n*8),
		reqChan:  make(chan request, n*20),
		coinChan: make(chan coinResult, n*20),
		// below channel size as n*4, because each node can have maximum 4 rounds
		binValueChan: make(chan struct {
			round uint64
			vote  cleisthenes.Vote
		}, n*4),
		tryoutAgreementChan: make(chan uint64, n*4),
		advanceRoundChan:    make(chan uint64, n*4),

		broadcaster:   broadcaster,
		coinGenerator: mock.NewCoinGenerator(cleisthenes.Coin(cleisthenes.One)),
		binInputChan:  binInputChan,
		Tracer:        cleisthenes.NewMemCacheTracer(),

		log: cleisthenes.NewLoggerWithHead("BBA"),
	}

	//for idx := 0; idx <= n; idx++ {
	//	instance.binValueSet[idx] = newBinarySet()
	//	instance.broadcastedBvalSet[idx] = newBinarySet()
	//	instance.est[idx] = cleisthenes.NewBinaryState()
	//	instance.bvalRepo[idx] = newBvalReqRepository()
	//	instance.auxRepo[idx] = newAuxReqRepository()
	//	instance.auxBroadcasted[idx] = cleisthenes.NewBinaryState()
	//}

	for i := 0; i < 12; i++ {
		instance.binValueSet[i] = newVoteSet()
		instance.valsSet[i] = newBinarySet()
		instance.broadcastedBvalSet[i] = newVoteSet()
		instance.est[i] = cleisthenes.NewBinaryState()
		instance.bvalRepo[i] = newBvalReqRepository()
		instance.auxRepo[i] = newAuxReqRepository()
		instance.mainVoteRepo[i] = newMainVoteReqRepository()
		instance.finalVoteRepo[i] = newFinalVoteReqRepository()
		instance.coinSigShareRepo[i] = newCoinSigShareReqRepository()
		instance.auxBroadcasted[i] = cleisthenes.NewBinaryState()
		instance.coinCompleted[i] = cleisthenes.NewBinaryState()
		instance.binValueChanSent[i] = cleisthenes.NewBinaryState()
	}

	instance.initLog()
	go instance.run()
	return instance
}

func (bba *BBA) initLog() {
	var logPrefix zerolog.HookFunc
	logPrefix = func(e *zerolog.Event, level zerolog.Level, message string) {
		e.Uint64("Epoch", uint64(bba.epoch)).Uint64("Round", bba.Round()).
			Str("owner", bba.owner.Address.String()).
			Str("proposer", bba.proposer.Address.String())
	}
	bba.log = bba.log.Hook(logPrefix)
}

// HandleInput will set the given val as the initial value to be proposed in the
// Agreement
func (bba *BBA) HandleInput(round uint64, bvalRequest *BvalRequest) error {
	//fmt.Printf("test50, round:%d, value:%v\n", round, bvalRequest.Value)
	bba.Log("action", "handleInput")
	bba.log.Debug().Msgf("handle input for round:%d b:%v", round, bvalRequest.Value)
	bba.est[round].Set(cleisthenes.ToBinary(bvalRequest.Value))

	if round == 0 {
		if !bba.bvalSent.Value() {
			// 广播bvalr(estr)
			bba.log.Debug().Msgf("开始广播bval:%v", bvalRequest.Value)
			bba.broadcastedBvalSet[round].union(bvalRequest.Value)
			if err := bba.broadcast(round, bvalRequest); err != nil {
				return err
			}
			//bba.reqChan <- req
		}

		if bvalRequest.Value == cleisthenes.VOne {
			bba.binValueSet[round].union(cleisthenes.VOne)
			bba.binValueChanSent[round].Set(true)
			bba.handleDelayedRequest(round)
			if !bba.auxSent.Value() {
				bba.log.Debug().Msgf("round 0 输入 1 开始广播 aux")
				if err := bba.broadcast(round, &AuxRequest{Value: cleisthenes.VOne}); err != nil {
					return err
				}
			}
			if !bba.mainVoteSent.Value() {
				bba.log.Debug().Msgf("round 0 输入 1 开始广播 mainVote")
				if err := bba.broadcast(round, &MainVoteRequest{Value: cleisthenes.VOne}); err != nil {
					return err
				}
			}
			if !bba.finalVoteSent.Value() {
				bba.log.Debug().Msgf("round 0 输入 1 开始广播 FinalVote")
				if err := bba.broadcast(round, &FinalVoteRequest{Value: cleisthenes.VOne}); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if err := bba.broadcast(round, &BvalRequest{Value: bvalRequest.Value}); err != nil {
		return err
	}

	return nil
}

// HandleMessage will process the given rpc message.
func (bba *BBA) HandleMessage(sender cleisthenes.Member, msg *pb.Message_Bba) error {
	req, round, err := bba.processMessage(msg)
	if err != nil {
		return err
	}

	r := request{
		sender: sender,
		data:   req,
		round:  round,
		err:    make(chan error),
	}

	bba.reqChan <- r
	return nil
}

func (bba *BBA) Result() (cleisthenes.Binary, bool) {
	return bba.dec.Value(), bba.dec.Undefined()
}

func (bba *BBA) Idle() bool {
	return bba.round.value() == 0 && bba.est[0].Undefined()
}

func (bba *BBA) Round() uint64 {
	return bba.round.value()
}

func (bba *BBA) Close() {
	bba.closeChan <- struct{}{}
	<-bba.closeChan
	if first := atomic.CompareAndSwapInt32(&bba.stopFlag, int32(0), int32(1)); !first {
		return
	}
	//close(bba.closeChan)
}

func (bba *BBA) Trace() {
	bba.Tracer.Trace()
}

func (bba *BBA) muxMessage(sender cleisthenes.Member, round uint64, req cleisthenes.Request) error {
	if round < bba.round.value() {
		bba.Log(
			"action", "muxMessage",
			"message", "request from old round, abandon",
			"sender", sender.Address.String(),
			"type", reflect.TypeOf(req).String(),
			"req.round", strconv.FormatUint(round, 10),
			"my.round", strconv.FormatUint(bba.round.value(), 10),
		)
		return nil
	}
	if round > bba.round.value() {
		bba.Log(
			"action", "muxMessage",
			"message", "request from future round, keep it",
			"sender", sender.Address.String(),
			"type", reflect.TypeOf(req).String(),
			"req.round", strconv.FormatUint(round, 10),
			"my.round", strconv.FormatUint(bba.round.value(), 10),
		)
		bba.saveIncomingRequest(round, sender, req)
		return nil
	}

	switch r := req.(type) {
	case *BvalRequest:
		// pre-vote
		return bba.handleBvalRequest(sender, r, round)
	case *AuxRequest:
		// vote
		return bba.handleAuxRequest(sender, r, round)
	case *MainVoteRequest:
		// main-vote
		bba.log.Debug().Msgf("收到mainVote消息, round:%d sender:%v", round, sender.Address.String())
		return bba.handleMainVoteRequest(sender, r, round)
	case *FinalVoteRequest:
		// final-vote
		return bba.handleFinalVoteRequest(sender, r, round)
	//case *CoinSigShareRequest:
	//	return bba.handleCoinSigShare(sender, r, round)
	default:
		return ErrUndefinedRequestType
	}
}

func (bba *BBA) handleBvalRequest(sender cleisthenes.Member, bval *BvalRequest, round uint64) error {
	if err := bba.saveBvalIfNotExist(round, sender, bval); err != nil {
		return err
	}

	count := bba.countBvalByValue(round, bval.Value)
	// 收到2f+1个bvalr后将bval放到bin_valuesr集合中
	/*
		upon receiving pre-voter(v) from 2 f +1 replicas
			bsetr ← bsetr ∪ {v}
	*/
	// FIXME 这里会有并发导致收到超过2f+1个bval才开始判断，所以不能用==
	if count >= bba.binValueSetThreshold() && !bba.binValueSet[round].exist(bval.Value) {
		bba.log.Debug().Msgf("收到了2f+1个bval, 设置binValueSet union b:%v", bval.Value)
		bba.Log("action", "binValueSet", "count", strconv.Itoa(count))
		bba.binValueSet[round].union(bval.Value)
		bba.handleDelayedRequest(round)
		bba.binValueChan <- struct {
			round uint64
			vote  cleisthenes.Vote
		}{round: round, vote: bval.Value}
		return nil
	}
	bba.Log("action", "binValueSet", "count", strconv.Itoa(count))

	// 收到f+1个bvalr后，广播bvalr
	/*
		upon receiving pre-voter(v) from f +1 replicas
			if pre-voter(v) has not been sent, broadcast pre-voter(v)
	*/
	// FIXME 这里会有并发导致收到超过f+1bval才开始判断，所以不能用==
	if count >= bba.bvalBroadcastThreshold() && !bba.broadcastedBvalSet[round].exist(bval.Value) {
		bba.log.Debug().Msgf("收到了f+1个bval，开始广播bval:%v", bval.Value)
		bba.broadcastedBvalSet[round].union(bval.Value)
		bba.Log("action", "broadcastBval", "count", strconv.Itoa(count))
		return bba.broadcast(round, bval)
	}

	return nil
}

func (bba *BBA) fillValsSet(round uint64) {
	auxList := bba.convToAuxList(bba.auxRepo[round].FindAll())
	for _, aux := range auxList {
		bba.valsSet[round].union(cleisthenes.ToBinary(aux.Value))
	}
}

func (bba *BBA) getCoin(round uint64) (cleisthenes.Coin, error) {
	sigShareMap, err := bba.getCoinSigShareMapV2(round)
	if err != nil {
		return false, err
	}

	ss := bba.tss.CombineSig(sigShareMap)
	if !bba.tss.VerifyCombineSig(ss, []byte(strconv.Itoa(int(round)))) {
		return false, fmt.Errorf("multi sig verify failed")
	}

	ss.Mod(ss, big.NewInt(2))
	//return ss.Int64()%2 == 1, nil
	// FIXME 抛硬币有时候会连续10次抛出一样的结果，暂时使用round代替
	return round%2 == 1, nil
}

func (bba *BBA) handleCoinSigShare(sender cleisthenes.Member, sigShare *CoinSigShareRequest, round uint64) error {
	// FIXME 如果超前收到后面的轮数的coinSigShare，在这里由于会将coinCompleted设为true，导致后续自己在后续对应round抛硬币的时候，无法进入到coinChan，会存在问题；
	// FIXME 这里会在后面对应round抛硬币的时候再次进行判断一次
	if bba.coinCompleted[round].Value() {
		return nil
	}

	if err := bba.saveCoinSigShare(round, sender, sigShare); err != nil {
		bba.log.Error().Msgf("保存coinSigShare失败, err:%s", err.Error())
		return err
	}

	count := bba.countCoinSigShare(round)
	bba.log.Debug().Msgf("收到了第%d轮的%d条sigShare, members:%v", round, count, bba.getCoinSigShareMembersString(round))
	if count < bba.coinSigShareThreshold() {
		return nil
	}

	coin, err := bba.getCoin(round)
	if err != nil {
		fmt.Println("getCoin失败:", err)
		return err
	}

	bba.log.Debug().Msgf("收到了第%d轮至少f+1个抛硬币签名share，combine并验证成功，开始共识", round)

	bba.coinChan <- coinResult{
		round: round,
		coin:  coin,
	}

	bba.coinCompleted[round].Set(true)
	return nil
}

func (bba *BBA) auxListString(round uint64) string {
	auxList := bba.convToAuxList(bba.auxRepo[round].FindAll())
	res := []cleisthenes.Vote{}
	for _, aux := range auxList {
		res = append(res, aux.Value)
	}
	return fmt.Sprintf("%v", res)
}

func (bba *BBA) mainVoteListString(round uint64) string {
	res := []cleisthenes.Vote{}
	for _, mainVote := range bba.mainVoteRepo[round].FindAll() {
		res = append(res, mainVote.(*MainVoteRequest).Value)
	}
	return fmt.Sprintf("%v", res)
}

func (bba *BBA) finalVoteListString(round uint64) string {
	res := []cleisthenes.Vote{}
	for _, finalVote := range bba.finalVoteRepo[round].FindAll() {
		res = append(res, finalVote.(*FinalVoteRequest).Value)
	}
	return fmt.Sprintf("%v", res)
}

func (bba *BBA) handleAuxRequest(sender cleisthenes.Member, aux *AuxRequest, round uint64) error {
	if !bba.matchWithCurrentRound(round) {
		bba.saveIncomingRequest(round, sender, aux)
		return nil
	}

	if !bba.binValueSet[round].exist(aux.Value) {
		bba.log.Debug().Msgf("收到了第%d轮%s发过来的aux消息,但不在binValueSet中, curAux:%v, binValueSet:%v",
			round, sender.Address.String(), aux.Value, bba.binValueSet[round].toList())
		bba.saveIncomingRequest(round, sender, aux)
		return nil
	}

	if err := bba.saveAuxIfNotExist(round, sender, aux); err != nil {
		return err
	}
	//count := bba.countAuxByValue(round, aux.Value)
	count := bba.countAux(round)

	bba.Log("action", "handleAux", "from", sender.Address.String(), "count", strconv.Itoa(count))
	bba.log.Debug().Msgf("收到了第%d轮%s发过来的aux消息, curAux:%v auxList:%v", round, sender.Address.String(), aux.Value, bba.auxListString(round))
	// auxr个数 < n-f, 直接返回
	if count < bba.auxBroadcastThreshold() {
		return nil
	}

	/*
		upon receiving n − f voter() such that for each received voter(b), b ∈ bsetr
			if there are n− f voter(v)
				broadcast main-voter(v)
			else broadcast main-voter(∗)

	*/
	// auxr个数 = n-f，尝试共识
	// FIXME 这里不能用==，因为count存在大于的情况，如果用==，就进不了tryoutAgreementChan
	//if count == bba.tryoutAgreementThreshold() {
	if count == bba.auxBroadcastThreshold() {
		statistic := bba.auxStatistics(round)
		for v, cnt := range statistic {
			if cnt == bba.auxBroadcastThreshold() {
				bba.broadcast(round, &MainVoteRequest{
					Value: v,
				})
				return nil
			}
		}
		bba.broadcast(round, &MainVoteRequest{
			Value: cleisthenes.VTwo,
		})
	}

	return nil
}

func (bba *BBA) handleMainVoteRequest(sender cleisthenes.Member, mainVote *MainVoteRequest, round uint64) error {
	if !bba.matchWithCurrentRound(round) {
		bba.saveIncomingRequest(round, sender, mainVote)
		return nil
	}

	/*
		upon receiving n − f main-voter() such that for each main-voter(v): 1) if r = 0, v ∈ bsetr, 2) if r > 0, at least f + 1
		voter(v) have been received; for each main-voter(∗), bsetr = {0,1}
			if there are n− f main-voter(v)
				broadcast final-voter(v)
			else broadcast final-voter(∗)
	*/
	if round == 0 {
		if !bba.binValueSet[round].exist(mainVote.Value) {
			bba.log.Debug().Msgf("收到了第%d轮%s发过来的mainVote消息,但不在binValueSet中, curMainVote:%v, binValueSet:%v",
				round, sender.Address.String(), mainVote.Value, bba.binValueSet[round].toList())
			bba.saveIncomingRequest(round, sender, mainVote)
			return nil
		}
	}

	if mainVote.Value == cleisthenes.VTwo && len(bba.binValueSet[round].toList()) != 2 {
		bba.log.Debug().Msgf("收到了第%d轮%s发过来的mainVote(*)消息,但binValueSet!={0,1}, curMainVote:%v, binValueSet:%v",
			round, sender.Address.String(), mainVote.Value, bba.binValueSet[round].toList())
		bba.saveIncomingRequest(round, sender, mainVote)
		return nil
	}

	if err := bba.saveMainVoteIfNotExist(round, sender, mainVote); err != nil {
		return err
	}
	//count := bba.countAuxByValue(round, aux.Value)
	count := bba.countMainVote(round)

	bba.Log("action", "handleMainVote", "from", sender.Address.String(), "count", strconv.Itoa(count))
	bba.log.Debug().Msgf("收到了第%d轮%s发过来的mainVote消息, curMainVote:%v mainVoteList:%v", round, sender.Address.String(), mainVote.Value, bba.mainVoteListString(round))

	// 个数 < n-f, 直接返回
	if count < bba.n-bba.f {
		return nil
	}

	if round > 0 {
		cnt := bba.countAuxByValue(round, mainVote.Value)
		if cnt < bba.f+1 {
			return nil
		}
	}

	// FIXME 这里不能用==，因为count存在大于的情况，如果用==，就进不了tryoutAgreementChan
	//if count == bba.tryoutAgreementThreshold() {
	if count == bba.n-bba.f {
		statistic := bba.mainVoteStatistics(round)
		for v, cnt := range statistic {
			if cnt == bba.n-bba.f {
				bba.broadcast(round, &FinalVoteRequest{
					Value: v,
				})
				return nil
			}
		}
		bba.broadcast(round, &FinalVoteRequest{
			Value: cleisthenes.VTwo,
		})
	}

	return nil
}

func (bba *BBA) handleFinalVoteRequest(sender cleisthenes.Member, finalVote *FinalVoteRequest, round uint64) error {
	if !bba.matchWithCurrentRound(round) {
		bba.saveIncomingRequest(round, sender, finalVote)
		return nil
	}

	/*
		upon receiving n − f final-voter() such that for each final-voter(v), 1) if r = 0, v ∈ bsetr, 2) if r > 0, at least f + 1
		main-voter(v) have been received; for each final-voter(∗), bsetr = {0,1}
			if there are n− f final-voter(v)
				ivr+1 ← v, decide v
			else if there are only final-voter(v) and final-voter(∗)
				ivr+1 ← v
			else
				if r = 0, ivr+1 ← 1 {coin in the first round is 1}
				else ivr+1 ← Random() {obtain local coin}

	*/
	if round == 0 {
		if !bba.binValueSet[round].exist(finalVote.Value) {
			bba.log.Debug().Msgf("收到了第%d轮%s发过来的finalVote消息,但不在binValueSet中, curFinalVote:%v, binValueSet:%v",
				round, sender.Address.String(), finalVote.Value, bba.binValueSet[round].toList())
			bba.saveIncomingRequest(round, sender, finalVote)
			return nil
		}
	}

	if finalVote.Value == cleisthenes.VTwo && len(bba.binValueSet[round].toList()) != 2 {
		bba.log.Debug().Msgf("收到了第%d轮%s发过来的finalVote(*)消息,但binValueSet!={0,1}, curMainVote:%v, binValueSet:%v",
			round, sender.Address.String(), finalVote.Value, bba.binValueSet[round].toList())
		bba.saveIncomingRequest(round, sender, finalVote)
		return nil
	}

	if err := bba.saveFinalVoteIfNotExist(round, sender, finalVote); err != nil {
		return err
	}
	//count := bba.countAuxByValue(round, aux.Value)
	count := bba.countFinalVote(round)

	bba.Log("action", "handleFinalVote", "from", sender.Address.String(), "count", strconv.Itoa(count))
	bba.log.Debug().Msgf("收到了第%d轮%s发过来的finalVote消息, curFinalVote:%v finalVoteList:%v", round, sender.Address.String(), finalVote.Value, bba.finalVoteListString(round))

	// 个数 < n-f, 直接返回
	if count < bba.n-bba.f {
		return nil
	}

	if round > 0 {
		cnt := bba.countMainVoteByValue(round, finalVote.Value)
		if cnt < bba.f+1 {
			return nil
		}
	}

	if count == bba.n-bba.f {
		coin := cleisthenes.Coin(round%2 == 0)
		bba.tryoutAgreement(round, coin)
	}

	return nil
}

func (bba *BBA) flipCoin(round uint64) {
	if !bba.matchWithCurrentRound(round) {
		return
	}

	if round > MAX_ROUND {
		return
	}

	bba.coinChan <- coinResult{
		round: round,
		coin:  true,
	}
}

func (bba *BBA) tryoutAgreement(round uint64, coin cleisthenes.Coin) {
	if !bba.matchWithCurrentRound(round) {
		return
	}
	bba.Log("action", "tryoutAgreement", "from", bba.owner.Address.String())

	// 轮数大于3直接返回
	if round > MAX_ROUND {
		if !bba.done.Value() {
			bba.log.Warn().Msg("ba超过指定轮数还未共识成功")
		}
		return
	}

	if bba.done.Value() {
		bba.log.Info().Msgf("已经共识成功, dec:%v", bba.dec.Value())
		// FIXME 待优化，这里的ba已经由结果了，感觉可以结束这个ba了，由acs来保存这个ba的结果
		bba.agreementSuccess(round, bba.dec.Value())
		return
	}

	statistic := bba.finalVoteStatistics(round)
	bba.log.Debug().Msgf("tryout agreement, finalVote statistic:%v", statistic)
	if statistic[cleisthenes.VZero] == bba.n-bba.f {
		bba.agreementSuccess(round, cleisthenes.ToBinary(cleisthenes.VZero))
		return
	}

	if statistic[cleisthenes.VOne] == bba.n-bba.f {
		bba.agreementSuccess(round, cleisthenes.ToBinary(cleisthenes.VOne))
		return
	}

	if statistic[cleisthenes.VZero] == 0 {
		bba.agreementFailed(round, cleisthenes.ToBinary(cleisthenes.VOne))
		return
	}

	if statistic[cleisthenes.VOne] == 0 {
		bba.agreementFailed(round, cleisthenes.ToBinary(cleisthenes.VZero))
		return
	}

	if round == 0 {
		bba.agreementFailed(round, cleisthenes.ToBinary(cleisthenes.VOne))
	}

	bba.agreementFailed(round, cleisthenes.Binary(coin))
}

func (bba *BBA) agreementFailed(round uint64, estValue cleisthenes.Binary) {
	if round > MAX_ROUND {
		return
	}
	bba.log.Debug().Msgf("agreement failed, round:%d est:%v", round, estValue)
	bba.est[round+1].Set(estValue)
	bba.advanceRoundChan <- round
}

func (bba *BBA) agreementSuccess(round uint64, decValue cleisthenes.Binary) {
	if round > MAX_ROUND {
		return
	}
	bba.log.Info().Msgf("agreement success, round:%d est:%v", round, decValue)
	bba.est[round+1].Set(decValue)
	bba.dec.Set(decValue)
	if !bba.done.Value() {
		bba.binInputChan.Send(cleisthenes.BinaryMessage{
			Member: bba.proposer,
			Binary: bba.dec.Value(),
		})
	}

	bba.done.Set(true)
	bba.Log(
		"action", "agreementSucess",
		"message", "<agreement finish>",
		"my.round", strconv.FormatUint(bba.round.value(), 10),
	)

	bba.advanceRoundChan <- round
}

func (bba *BBA) advanceRound(round uint64) {
	if round > MAX_ROUND {
		bba.log.Debug().Msgf("ba超过指定轮数, 当前ba轮数:%d", round)
		return
	}
	bba.Log("action", "advanceRound", "from", bba.owner.Address.String())
	bba.round.inc()

	bba.handleDelayedRequest(round + 1)

	bba.HandleInput(round+1, &BvalRequest{
		Value: cleisthenes.ToVote(bba.est[round+1].Value()),
	})
}

func (bba *BBA) handleDelayedRequest(round uint64) {
	delayedReqMap := bba.incomingReqRepo.Find(round)
	for _, ir := range delayedReqMap {
		if ir.round != round {
			continue
		}
		bba.Log(
			"action", "handleDelayedRequest",
			"round", strconv.FormatUint(ir.round, 10),
			"size", strconv.Itoa(len(delayedReqMap)),
			"addr", ir.addr.String(),
			"type", reflect.TypeOf(ir.req).String(),
		)
		r := request{
			sender: cleisthenes.Member{Address: ir.addr},
			data:   ir.req,
			round:  ir.round,
			err:    make(chan error),
		}
		bba.reqChan <- r
	}
	bba.Log("action", "handleDelayedRequest", "message", "done")
}

func (bba *BBA) broadcastAuxOnceForRound(round uint64, vote cleisthenes.Vote) {
	bba.Log("action", "broadcastAux", "from", bba.owner.Address.String(), "round", strconv.FormatUint(bba.round.value(), 10))

	bba.log.Debug().Msgf("binValues不为空，开始广播第%d轮aux:%v, binValues:%v", round, vote, bba.binValueSet[round].toList())

	/*
		wait until bsetr != 0/
			if voter() has not been sent
			broadcast voter(v) where v ∈ bsetr
	*/
	// 广播auxr
	if err := bba.broadcast(round,
		&AuxRequest{
			Value: vote,
		}); err != nil {
		bba.Log("action", "broadcastAux", "binValue length", strconv.Itoa(1), "err", err.Error())
		bba.log.Error().Msgf("broadcastAux error:%v", err.Error())
	}
}

func (bba *BBA) broadcast(round uint64, req cleisthenes.Request) error {
	var typ pb.BBA_Type
	switch req.(type) {
	case *BvalRequest:
		typ = pb.BBA_BVAL
		bba.bvalSent.Set(cleisthenes.One)
	case *AuxRequest:
		typ = pb.BBA_AUX
		bba.auxSent.Set(cleisthenes.One)
	case *MainVoteRequest:
		typ = pb.BBA_MAIN_VOTE
		bba.mainVoteSent.Set(cleisthenes.One)
	case *FinalVoteRequest:
		typ = pb.BBA_FINAL_VOTE
		bba.finalVoteSent.Set(cleisthenes.One)
	case *CoinSigShareRequest:
		typ = pb.BBA_COIN_SIG_SHARE
	default:
		return errors.New("invalid broadcast message type")
	}
	payload, err := sonic.ConfigFastest.Marshal(req)
	if err != nil {
		return err
	}

	bbaMsg := &pb.Message_Bba{
		Bba: &pb.BBA{
			Round:   round,
			Type:    typ,
			Payload: payload,
		},
	}
	broadcastMsg := pb.Message{
		Proposer:  bba.proposer.Address.String(),
		Sender:    bba.owner.Address.String(),
		Timestamp: ptypes.TimestampNow(),
		Epoch:     uint64(bba.epoch),
		Payload:   bbaMsg,
	}

	bba.HandleMessage(bba.owner, bbaMsg)
	bba.broadcaster.ShareMessage(broadcastMsg)
	return nil
}

func (bba *BBA) toDie() bool {
	return atomic.LoadInt32(&(bba.stopFlag)) == int32(1)
}

func (bba *BBA) run() {
	for !bba.toDie() {
		select {
		case <-bba.closeChan:
			bba.closeChan <- struct{}{}
			return
		case req := <-bba.reqChan:
			// 处理bvalr和auxr信息
			// 收到f+1个bvalr后，广播bvalr
			// 收到2f+1个bvalr后将bvalr放到bin_valuesr集合中
			bba.muxMessage(req.sender, req.round, req.data)
		case s := <-bba.binValueChan:
			// bin_valuesr集合不为空，广播auxr，执行以下步骤
			if bba.matchWithCurrentRound(s.round) {
				bba.broadcastAuxOnceForRound(s.round, s.vote)
				//bba.handleDelayedAuxRequest(r)
			}
		case r := <-bba.coinChan:
			if bba.matchWithCurrentRound(r.round) {
				bba.tryoutAgreement(r.round, r.coin)
			}
		case r := <-bba.tryoutAgreementChan:
			// 收到n-f个auxr时，会尝试共识（抛硬币），执行以下步骤
			if bba.matchWithCurrentRound(r) {
				bba.flipCoin(r)
			}
		case r := <-bba.advanceRoundChan:
			// 共识成功和失败轮数都会+1。（共识成功还会通过通道让acs尝试完成共识tryCompleteAgreement）
			if bba.matchWithCurrentRound(r) {
				bba.advanceRound(r)
			}
		}
	}
}

func (bba *BBA) saveBvalIfNotExist(round uint64, sender cleisthenes.Member, data *BvalRequest) error {
	//_, err := bba.bvalRepo.Find(sender.Address)
	//if err != nil && !IsErrNoResult(err) {
	//	return err
	//}
	//if r != nil {
	//	return nil
	//}
	return bba.bvalRepo[round].Save(sender.Address, data)
}

func (bba *BBA) saveAuxIfNotExist(round uint64, sender cleisthenes.Member, data *AuxRequest) error {
	//_, err := bba.auxRepo.Find(sender.Address)
	//if err != nil && !IsErrNoResult(err) {
	//	return err
	//}
	//if r != nil {
	//	return nil
	//}
	return bba.auxRepo[round].Save(sender.Address, data)
}

func (bba *BBA) saveMainVoteIfNotExist(round uint64, sender cleisthenes.Member, data *MainVoteRequest) error {
	return bba.mainVoteRepo[round].Save(sender.Address, data)
}

func (bba *BBA) saveFinalVoteIfNotExist(round uint64, sender cleisthenes.Member, data *FinalVoteRequest) error {
	return bba.finalVoteRepo[round].Save(sender.Address, data)
}

func (bba *BBA) saveCoinSigShare(round uint64, sender cleisthenes.Member, data *CoinSigShareRequest) error {
	return bba.coinSigShareRepo[round].Save(sender.Address, data)
}

func (bba *BBA) countBvalByValue(round uint64, val cleisthenes.Vote) int {
	bvalList := bba.convToBvalList(bba.bvalRepo[round].FindAll())
	count := 0
	for _, bval := range bvalList {
		if bval.Value == val {
			count++
		}
	}
	return count
}

// for tpke current is no use
func (bba *BBA) getCoinSigShareMap(round uint64) (map[cleisthenes.Address]cleisthenes.SignatureShare, error) {
	res := map[cleisthenes.Address]cleisthenes.SignatureShare{}
	sigShares := bba.coinSigShareRepo[round].FindAll()
	for _, req := range sigShares {
		r, ok := req.(*CoinSigShareRequest)
		if !ok {
			return nil, fmt.Errorf("NOT CoinSigShareRequest")
		}
		sigShare := cleisthenes.SignatureShare{}
		for i := 0; i < 96; i++ {
			sigShare[i] = r.Value[i]
		}
		res[r.Addr] = sigShare
	}
	return res, nil
}

// for tss
func (bba *BBA) getCoinSigShareMapV2(round uint64) (map[cleisthenes.Address][]byte, error) {
	res := map[cleisthenes.Address][]byte{}
	sigShares := bba.coinSigShareRepo[round].FindAll()
	for _, req := range sigShares {
		r, ok := req.(*CoinSigShareRequest)
		if !ok {
			return nil, fmt.Errorf("NOT CoinSigShareRequest")
		}
		res[r.Addr] = r.Value
	}
	return res, nil
}

func (bba *BBA) countCoinSigShare(round uint64) int {
	return len(bba.coinSigShareRepo[round].FindAll())
}

func (bba *BBA) getCoinSigShareMembersString(round uint64) string {
	m, err := bba.getCoinSigShareMapV2(round)
	if err != nil {
		bba.log.Error().Msgf("get coin sig share map failed, err:%s", err.Error())
		return "get coin sig share map failed, err:" + err.Error()
	}
	addrs := []string{}
	for addr, _ := range m {
		addrs = append(addrs, addr.String())
	}

	return fmt.Sprintf("%v", addrs)
}

func (bba *BBA) countAux(round uint64) int {
	auxList := bba.convToAuxList(bba.auxRepo[round].FindAll())
	return len(auxList)
}

func (bba *BBA) countMainVote(round uint64) int {
	return len(bba.mainVoteRepo[round].FindAll())
}

func (bba *BBA) countFinalVote(round uint64) int {
	return len(bba.finalVoteRepo[round].FindAll())
}

func (bba *BBA) auxStatistics(round uint64) map[cleisthenes.Vote]int {
	statistic := map[cleisthenes.Vote]int{}
	for _, v := range bba.convToAuxList(bba.auxRepo[round].FindAll()) {
		statistic[v.Value]++
	}
	return statistic
}

func (bba *BBA) mainVoteStatistics(round uint64) map[cleisthenes.Vote]int {
	statistic := map[cleisthenes.Vote]int{}
	for _, v := range bba.mainVoteRepo[round].FindAll() {
		statistic[v.(*MainVoteRequest).Value]++
	}
	return statistic
}

func (bba *BBA) finalVoteStatistics(round uint64) map[cleisthenes.Vote]int {
	statistic := map[cleisthenes.Vote]int{}
	for _, v := range bba.finalVoteRepo[round].FindAll() {
		statistic[v.(*FinalVoteRequest).Value]++
	}
	return statistic
}

func (bba *BBA) countAuxByValue(round uint64, val cleisthenes.Vote) int {
	auxList := bba.convToAuxList(bba.auxRepo[round].FindAll())
	count := 0
	for _, aux := range auxList {
		if aux.Value == val {
			count++
		}
	}
	return count
}

func (bba *BBA) countMainVoteByValue(round uint64, val cleisthenes.Vote) int {
	cnt := 0
	for _, req := range bba.mainVoteRepo[round].FindAll() {
		mainVote := req.(*MainVoteRequest).Value
		if mainVote == val {
			cnt++
		}
	}
	return cnt
}

func (bba *BBA) convToBvalList(reqList []cleisthenes.Request) []*BvalRequest {
	result := make([]*BvalRequest, 0)
	for _, req := range reqList {
		result = append(result, req.(*BvalRequest))
	}
	return result
}

func (bba *BBA) convToAuxList(reqList []cleisthenes.Request) []*AuxRequest {
	result := make([]*AuxRequest, 0)
	for _, req := range reqList {
		result = append(result, req.(*AuxRequest))
	}
	return result
}

func (bba *BBA) bvalBroadcastThreshold() int {
	return bba.f + 1
}

func (bba *BBA) coinSigShareThreshold() int {
	return bba.f + 1
}

func (bba *BBA) binValueSetThreshold() int {
	return 2*bba.f + 1
}

func (bba *BBA) auxBroadcastThreshold() int {
	return bba.n - bba.f
}
func (bba *BBA) tryoutAgreementThreshold() int {
	return bba.n - bba.f
}

func (bba *BBA) processMessage(msg *pb.Message_Bba) (cleisthenes.Request, uint64, error) {
	switch msg.Bba.Type {
	case pb.BBA_BVAL:
		return bba.processBvalMessage(msg)
	case pb.BBA_AUX:
		return bba.processAuxMessage(msg)
	case pb.BBA_MAIN_VOTE:
		return bba.processMainVoteMessage(msg)
	case pb.BBA_FINAL_VOTE:
		return bba.processFinalVoteMessage(msg)
	case pb.BBA_COIN_SIG_SHARE:
		return bba.processCoinSigShareMessage(msg)
	default:
		return nil, 0, errors.New("error processing message with invalid type")
	}
}

func (bba *BBA) processBvalMessage(msg *pb.Message_Bba) (cleisthenes.Request, uint64, error) {
	req := &BvalRequest{}
	if err := sonic.ConfigFastest.Unmarshal(msg.Bba.Payload, req); err != nil {
		return nil, 0, err
	}
	return req, msg.Bba.Round, nil
}

func (bba *BBA) processAuxMessage(msg *pb.Message_Bba) (cleisthenes.Request, uint64, error) {
	req := &AuxRequest{}
	if err := sonic.ConfigFastest.Unmarshal(msg.Bba.Payload, req); err != nil {
		return nil, 0, err
	}
	return req, msg.Bba.Round, nil
}

func (bba *BBA) processMainVoteMessage(msg *pb.Message_Bba) (cleisthenes.Request, uint64, error) {
	req := &MainVoteRequest{}
	if err := sonic.ConfigFastest.Unmarshal(msg.Bba.Payload, req); err != nil {
		return nil, 0, err
	}
	return req, msg.Bba.Round, nil
}

func (bba *BBA) processFinalVoteMessage(msg *pb.Message_Bba) (cleisthenes.Request, uint64, error) {
	req := &FinalVoteRequest{}
	if err := sonic.ConfigFastest.Unmarshal(msg.Bba.Payload, req); err != nil {
		return nil, 0, err
	}
	return req, msg.Bba.Round, nil
}

func (bba *BBA) processCoinSigShareMessage(msg *pb.Message_Bba) (cleisthenes.Request, uint64, error) {
	req := &CoinSigShareRequest{}
	if err := sonic.ConfigFastest.Unmarshal(msg.Bba.Payload, req); err != nil {
		return nil, 0, err
	}
	return req, msg.Bba.Round, nil
}

func (bba *BBA) saveIncomingRequest(round uint64, sender cleisthenes.Member, req cleisthenes.Request) {
	bba.incomingReqRepo.Save(round, sender.Address, req)
}

func (bba *BBA) matchWithCurrentRound(round uint64) bool {
	if round != bba.round.value() {
		return false
	}
	return true
}
