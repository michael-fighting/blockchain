package rbc

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/DE-labtory/cleisthenes/sliceSystem/mainchain/rbc/merkletree"
	"github.com/bytedance/sonic"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/protobuf/ptypes"
	"github.com/klauspost/reedsolomon"
	"github.com/rs/zerolog"
	"sync"
	"sync/atomic"
	"time"
)

type output struct {
	sync.RWMutex
	output []byte
}

func (o *output) set(output []byte) {
	o.Lock()
	defer o.Unlock()
	o.output = output
}

func (o *output) value() []byte {
	o.RLock()
	defer o.RUnlock()
	output := o.output
	return output
}

func (o *output) delete() {
	o.Lock()
	defer o.Unlock()
	o.output = nil
}

type contentLength struct {
	sync.RWMutex
	length uint64
}

func (c *contentLength) set(length uint64) error {
	c.Lock()
	defer c.Unlock()
	if c.length == 0 {
		c.length = length
	}
	return nil
}

func (c *contentLength) value() uint64 {
	c.RLock()
	defer c.RUnlock()
	return c.length
}

type request struct {
	sender cleisthenes.Member
	data   cleisthenes.Request
	err    chan error
}

type RBC struct {
	// number of network nodes
	n int

	// number of byzantine nodes which can tolerate
	f int

	epoch cleisthenes.Epoch
	// owner of rbc instance (node)
	owner cleisthenes.Member

	// proposerId is the ID of proposing node
	proposer cleisthenes.Member

	// Erasure coding using reed-solomon method
	enc reedsolomon.Encoder

	// output of RBC
	output  *output
	rawData *output
	rawMd5  *output

	// length of original data
	contentLength *contentLength

	// number of sharded data and parity
	// data : N - F, parity : F
	numDataShards, numParityShards int

	// Request of other rbcs
	echoReqRepo  cleisthenes.RequestRepository
	readyReqRepo cleisthenes.RequestRepository

	valReceived, echoSent, readySent, done *cleisthenes.BinaryState

	stopFlag int32
	// internal channels to communicate with other components
	closeChan chan struct{}
	reqChan   chan request

	broadcaster cleisthenes.Broadcaster
	dataSender  cleisthenes.DataSender

	// 标识当前rbc是否是indexRbc
	isIndexRbc bool

	log   zerolog.Logger
	start time.Time
}

func New(
	n, f int,
	epoch cleisthenes.Epoch,
	owner, proposer cleisthenes.Member,
	broadcaster cleisthenes.Broadcaster,
	dataSender cleisthenes.DataSender,
	isIndexRbc bool,
) (*RBC, error) {
	numParityShards := 2 * f
	numDataShards := n - numParityShards

	enc, err := reedsolomon.New(numDataShards, numParityShards)
	if err != nil {
		return nil, err
	}

	echoReqRepo := NewEchoReqRepository()
	readyReqRepo := NewReadyReqRepository()
	rbc := &RBC{
		n:               n,
		f:               f,
		epoch:           epoch,
		owner:           owner,
		proposer:        proposer,
		enc:             enc,
		output:          &output{sync.RWMutex{}, nil},
		rawData:         &output{sync.RWMutex{}, nil},
		rawMd5:          &output{RWMutex: sync.RWMutex{}, output: nil},
		contentLength:   &contentLength{sync.RWMutex{}, 0},
		numDataShards:   numDataShards,
		numParityShards: numParityShards,
		echoReqRepo:     echoReqRepo,
		readyReqRepo:    readyReqRepo,
		valReceived:     cleisthenes.NewBinaryState(),
		echoSent:        cleisthenes.NewBinaryState(),
		readySent:       cleisthenes.NewBinaryState(),
		done:            cleisthenes.NewBinaryState(),
		closeChan:       make(chan struct{}),
		reqChan:         make(chan request, n*n),
		//reqChan:     make(chan request, n*n*10),
		broadcaster: broadcaster,
		dataSender:  dataSender,
		isIndexRbc:  isIndexRbc,

		log: cleisthenes.NewLoggerWithHead("MRBC"),
	}

	rbc.initLog()
	go rbc.run()
	return rbc, nil
}

func (rbc *RBC) initLog() {
	var logPrefix zerolog.HookFunc
	logPrefix = func(e *zerolog.Event, level zerolog.Level, message string) {
		e.Uint64("Epoch", uint64(rbc.epoch)).
			Str("owner", rbc.owner.Address.String()).
			Str("proposer", rbc.proposer.Address.String())
	}
	rbc.log = rbc.log.Hook(logPrefix)
}

func (rbc *RBC) distributeMessage(proposer cleisthenes.Member, reqs []cleisthenes.Request) error {
	var typ pb.RBC_Type
	switch reqs[0].(type) {
	case *ValRequest:
		typ = pb.RBC_VAL
	default:
		return errors.New("invalid distributeMessage message type")
	}

	msgList := make([]pb.Message, 0)
	for _, req := range reqs {
		payload, err := sonic.ConfigFastest.Marshal(req)
		if err != nil {
			return err
		}

		msgList = append(msgList, pb.Message{
			Proposer:  proposer.Address.String(),
			Sender:    rbc.owner.Address.String(),
			Timestamp: ptypes.TimestampNow(),
			Epoch:     uint64(rbc.epoch),
			Payload: &pb.Message_Rbc{
				Rbc: &pb.RBC{
					Payload:       payload,
					ContentLength: rbc.contentLength.value(),
					Type:          typ,
					IsIndexRbc:    rbc.isIndexRbc,
				},
			},
		})
	}
	rbc.broadcaster.DistributeMessage(msgList)

	return nil
}

func (rbc *RBC) shareMessage(proposer cleisthenes.Member, req cleisthenes.Request) error {
	//先传入的是RBCType类型
	var typ pb.RBC_Type
	switch req.(type) {
	case *EchoRequest:
		typ = pb.RBC_ECHO
	case *ReadyRequest:
		typ = pb.RBC_READY
	default:
		return errors.New("invalid shareMessage message type")
	}
	payload, err := sonic.ConfigFastest.Marshal(req)
	if err != nil {
		return err
	}
	rbc.broadcaster.ShareMessage(pb.Message{
		Proposer:  proposer.Address.String(),
		Sender:    rbc.owner.Address.String(),
		Timestamp: ptypes.TimestampNow(),
		Epoch:     uint64(rbc.epoch),
		Payload: &pb.Message_Rbc{
			Rbc: &pb.RBC{
				Payload:       payload,
				ContentLength: rbc.contentLength.value(),
				Type:          typ,
				IsIndexRbc:    rbc.isIndexRbc,
			},
		},
	})
	return nil
}

// MakeReqAndBroadcast make requests and broadcast to other nodes
// it is used in ACS
func (rbc *RBC) HandleInput(data []byte) error {
	start := time.Now()

	rbc.log.Debug().Msgf("开始构造request, len(data):%d", len(data))
	reqs, err := makeRequest(data, rbc.n)
	if err != nil {
		return err
	}

	rbc.contentLength.set(uint64(len(data)))

	end := time.Now()
	rbc.log.Debug().Msgf("handle input make request time:%v", end.Sub(start))

	rbc.log.Debug().Msgf("开始分发val信息, connAddrs:%v", rbc.broadcaster.GetAllConnAddr())

	if err := rbc.distributeMessage(rbc.proposer, reqs); err != nil {
		return err
	}

	rbc.log.Debug().Msgf("分发val信息完成, owner:%s proposer:%s len(reqs):%d", rbc.owner.Address.String(), rbc.proposer.Address.String(), len(reqs))

	return nil
}

// HandleMessage will used in ACS
func (rbc *RBC) HandleMessage(sender cleisthenes.Member, msg *pb.Message_Rbc) error {
	start := time.Now()
	rbc.log.Debug().Msgf("收到来自%s的消息，消息类型:%s", sender.Address.String(), msg.Rbc.Type.String())
	req, err := processMessage(msg)
	end := time.Now()
	rbc.log.Debug().Msgf("process message time:%v, msgType:%v", end.Sub(start), msg.Rbc.Type)
	if err != nil {
		return err
	}

	r := request{
		sender: sender,
		data:   req,
		err:    make(chan error),
	}

	rbc.contentLength.set(msg.Rbc.ContentLength)
	if rbc.contentLength.value() != msg.Rbc.ContentLength {
		rbc.log.Error().Msgf("inavlid content length - know as : %d, receive : %d", rbc.contentLength.value(), msg.Rbc.ContentLength)
		return errors.New(fmt.Sprintf("inavlid content length - know as : %d, receive : %d", rbc.contentLength.value(), msg.Rbc.ContentLength))
	}

	if rbc.toDie() {
		rbc.log.Info().Msgf("rbc is closed, should not send req to reqChan")
		close(rbc.reqChan)
		return nil
	}

	rbc.reqChan <- r
	return <-r.err
}

// handleMessage will distinguish input message (from ACS)
func (rbc *RBC) muxRequest(sender cleisthenes.Member, req cleisthenes.Request) error {
	start := time.Now()
	switch r := req.(type) {
	case *ValRequest:
		err := rbc.handleValueRequest(sender, r)
		end := time.Now()
		rbc.log.Debug().Msgf("handle val time:%v", end.Sub(start))
		return err
	case *EchoRequest:
		err := rbc.handleEchoRequest(sender, r)
		end := time.Now()
		rbc.log.Debug().Msgf("handle echo time:%v", end.Sub(start))
		return err
	case *ReadyRequest:
		err := rbc.handleReadyRequest(sender, r)
		end := time.Now()
		rbc.log.Debug().Msgf("handle ready time:%v", end.Sub(start))
		return err
	default:
		return ErrInvalidRBCType
	}
}

func (rbc *RBC) handleValueRequest(sender cleisthenes.Member, req *ValRequest) error {
	if rbc.valReceived.Value() {
		rbc.log.Error().Msgf("already receive req message, sender:%s owner:%s, proposer:%s", sender.Address.String(), rbc.owner.Address.String(), rbc.proposer.Address.String())
		return errors.New("already receive req message")
	}

	if rbc.echoSent.Value() {
		rbc.log.Error().Msgf("already sent echo message, sender:%s owner:%s, proposer:%s", sender.Address.String(), rbc.owner.Address.String(), rbc.proposer.Address.String())
		return errors.New(fmt.Sprintf("already sent echo message - sender id : %s", sender.Address.String()))
	}

	rbc.valReceived.Set(true)
	rbc.echoSent.Set(true)

	rbc.log.Debug().Msgf("收到来自%s的val信息, sender:%s proposer:%s", sender.Address.String(), rbc.proposer.Address.String())
	rbc.log.Debug().Msgf("handle val request, sender:%s proposer:%s", sender.Address.String(), rbc.proposer.Address.String())
	rbc.rawData.set(*req.RawData)
	sum := md5.Sum(rbc.rawData.value())
	rbc.rawMd5.set(sum[:])
	// 使用rootHash字段来保存md5值
	req.RootHash = sum[:]
	req.RawData = nil

	rbc.shareMessage(rbc.proposer, &EchoRequest{*req})
	rbc.log.Debug().Msgf("收到%s的val后发送echo完成, proposer:%s", sender.Address.String(), rbc.proposer.Address.String())

	return nil
}

func (rbc *RBC) handleEchoRequest(sender cleisthenes.Member, req *EchoRequest) error {
	if req, _ := rbc.echoReqRepo.Find(sender.Address); req != nil {
		rbc.log.Warn().Msgf("already received echo request - from : %s", sender.Address.String())
		return errors.New(fmt.Sprintf("already received echo request - from : %s", sender.Address.String()))
	}

	if err := rbc.echoReqRepo.Save(sender.Address, req); err != nil {
		return err
	}

	rbc.log.Debug().Msgf("rbc.countEchos:%d, receivedRawData:%v sender:%s owner:%s, proposer:%s", rbc.countEchos(req.RootHash), rbc.receivedRawData(req.RootHash), sender.Address.String(), rbc.owner.Address.String(), rbc.proposer.Address.String())
	// 当收到n-f个一样的md5值的时候，并且收到了val传递过来的rawdata, 才进行下一步
	if rbc.countEchos(req.RootHash) >= rbc.echoThreshold() && rbc.receivedRawData(req.RootHash) && !rbc.readySent.Value() {
		rbc.readySent.Set(true)
		readyReq := &ReadyRequest{req.RootHash}
		rbc.shareMessage(rbc.proposer, readyReq)
	}

	//正常情况不会进入下面这个循环，会走358行解码RBC成功
	//如果收到了2f+1个ready，但是没有收到足够的echo，会在这里等待收到足够的（n-f）echo信息后对tx进行还原（纠删码）
	if !rbc.done.Value() && rbc.countReadys(req.RootHash) >= rbc.outputThreshold() && rbc.countEchos(req.RootHash) >= rbc.echoThreshold() && rbc.receivedRawData(req.RootHash) {
		value, err := rbc.tryDecodeValue(req.RootHash)
		if err != nil {
			return err
		}
		rbc.decodeSuccess(value)
	}

	return nil
}

func (rbc *RBC) handleReadyRequest(sender cleisthenes.Member, req *ReadyRequest) error {
	if req, _ := rbc.readyReqRepo.Find(sender.Address); req != nil {
		rbc.log.Warn().Msgf("already received ready request - from : %s", sender.Address.String())
		return errors.New(fmt.Sprintf("already received ready request - from : %s", sender.Address.String()))
	}

	if err := rbc.readyReqRepo.Save(sender.Address, req); err != nil {
		return err
	}

	rbc.log.Debug().Msgf("rbc.countReadys:%d, sender:%s owner:%s, proposer:%s", rbc.countReadys(req.RootHash), sender.Address.String(), rbc.owner.Address.String(), rbc.proposer.Address.String())
	if rbc.countReadys(req.RootHash) >= rbc.readyThreshold() && !rbc.readySent.Value() {
		rbc.readySent.Set(true)
		rbc.shareMessage(rbc.proposer, &ReadyRequest{req.RootHash})
	}

	if !rbc.done.Value() && rbc.countReadys(req.RootHash) >= rbc.outputThreshold() && rbc.countEchos(req.RootHash) >= rbc.echoThreshold() && rbc.receivedRawData(req.RootHash) {
		value, err := rbc.tryDecodeValue(req.RootHash)
		if err != nil {
			return err
		}

		rbc.decodeSuccess(value)
	}

	return nil
}

func (rbc *RBC) toDie() bool {
	return atomic.LoadInt32(&(rbc.stopFlag)) == int32(1)
}

func (rbc *RBC) run() {
	for !rbc.toDie() {
		select {
		case <-rbc.closeChan:
			rbc.closeChan <- struct{}{}
			return
		case req := <-rbc.reqChan:
			if rbc.start.IsZero() {
				rbc.start = time.Now()
			}
			//开始进入RBC阶段
			req.err <- rbc.muxRequest(req.sender, req.data)
		}
	}
}

func (rbc *RBC) Close() {
	rbc.closeChan <- struct{}{}
	<-rbc.closeChan

	if first := atomic.CompareAndSwapInt32(&rbc.stopFlag, int32(0), int32(1)); !first {
		return
	}
	//close(rbc.closeChan)
}

func (rbc *RBC) receivedRawData(sum []byte) bool {
	return bytes.Equal(sum, rbc.rawMd5.value())
}

func (rbc *RBC) countEchos(rootHash []byte) int {
	cnt := 0

	reqs := rbc.echoReqRepo.FindAll()
	for _, req := range reqs {
		if bytes.Equal(rootHash, req.(*EchoRequest).RootHash) {
			cnt++
		}
	}

	return cnt
}

func (rbc *RBC) countReadys(rootHash []byte) int {
	cnt := 0

	reqs := rbc.readyReqRepo.FindAll()
	for _, req := range reqs {
		if bytes.Equal(rootHash, req.(*ReadyRequest).RootHash) {
			cnt++
		}
	}

	return cnt
}

// interpolate the given shards
// if try to interpolate not enough ( < N - 2f ) shards then return error
func (rbc *RBC) tryDecodeValue(rootHash []byte) ([]byte, error) {
	if !bytes.Equal(rootHash, rbc.rawMd5.value()) {
		rbc.log.Error().Msgf("rootHash is not found, rootHash:%s, rawMd5:%s", hexutil.Encode(rootHash), hexutil.Encode(rbc.rawMd5.value()))
		return nil, fmt.Errorf("rootHash is not found, rootHash:%s, rawMd5:%s", hexutil.Encode(rootHash), hexutil.Encode(rbc.rawMd5.value()))
	}

	return rbc.rawData.value(), nil
}

func (rbc *RBC) decodeSuccess(decValue []byte) {
	rbc.log.Debug().Msgf("decode success, owner:%s, proposer:%s", rbc.owner.Address.String(), rbc.proposer.Address.String())
	//rbc.output.set(rbc.rawData.value())
	rbc.output.set(decValue)
	rbc.done.Set(true)

	end := time.Now()
	rbc.log.Debug().Msgf("single rbc time:%v", end.Sub(rbc.start))

	//选举模块：在这里加一个判定条件，一旦达到了委员会数量，
	rbc.dataSender.Send(cleisthenes.DataMessage{
		Member: rbc.proposer,
		Data:   rbc.output.value(),
	})
	rbc.output.delete()
	rbc.log.Debug().Msg("[RBC success]")
}

// wait until receive N - f ECHO messages
func (rbc *RBC) echoThreshold() int {
	return rbc.n - rbc.f
}

func (rbc *RBC) readyThreshold() int {
	return rbc.f + 1
}

func (rbc *RBC) outputThreshold() int {
	return 2*rbc.f + 1
}

func makeRequest(data []byte, n int) ([]cleisthenes.Request, error) {

	reqs := make([]cleisthenes.Request, 0)
	rawMsg := json.RawMessage(data)
	for i := 0; i < n; i++ {
		reqs = append(reqs, &ValRequest{
			RawData: &rawMsg,
		})
	}

	return reqs, nil
}

func processMessage(msg *pb.Message_Rbc) (cleisthenes.Request, error) {
	switch msg.Rbc.Type {
	case pb.RBC_VAL:
		return processValueMessage(msg)
	case pb.RBC_ECHO:
		return processEchoMessage(msg)
	case pb.RBC_READY:
		return processReadyMessage(msg)
	default:
		return nil, errors.New("error processing message with invalid type")
	}
}

func processValueMessage(msg *pb.Message_Rbc) (cleisthenes.Request, error) {
	req := &ValRequest{}
	if err := sonic.ConfigFastest.Unmarshal(msg.Rbc.Payload, req); err != nil {
		return nil, err
	}
	return req, nil
}

func processEchoMessage(msg *pb.Message_Rbc) (cleisthenes.Request, error) {
	req := &EchoRequest{}
	if err := sonic.ConfigFastest.Unmarshal(msg.Rbc.Payload, req); err != nil {
		return nil, err
	}
	return req, nil
}

func processReadyMessage(msg *pb.Message_Rbc) (cleisthenes.Request, error) {
	req := &ReadyRequest{}
	if err := sonic.ConfigFastest.Unmarshal(msg.Rbc.Payload, req); err != nil {
		return nil, err
	}
	return req, nil
}

// validate given value message and echo message
func validateMessage(req *ValRequest) bool {
	return merkletree.ValidatePath(req.Data, req.RootHash, req.RootPath, req.Indexes)
}

// make shards using reed-solomon erasure coding
func shard(enc reedsolomon.Encoder, data []byte) ([]merkletree.Data, error) {
	shards, err := enc.Split(data)
	if err != nil {
		return nil, err
	}
	if err := enc.Encode(shards); err != nil {
		return nil, err
	}

	dataList := make([]merkletree.Data, 0)

	for _, shard := range shards {
		dataList = append(dataList, merkletree.NewData(shard))
	}

	return dataList, nil
}
