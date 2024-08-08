package irbc

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/DE-labtory/cleisthenes/sliceSystem/branch/rbc/merkletree"
	"github.com/bytedance/sonic"
	"github.com/golang/protobuf/ptypes"
	"github.com/klauspost/reedsolomon"
	"github.com/rs/zerolog"
	"sync"
	"sync/atomic"
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
	output *output

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

	log zerolog.Logger
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

		log: cleisthenes.NewLoggerWithHead("IRBC"),
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
	// TODO 在这里不需要shard了，以方便在echo阶段直接获取到data（即md5值）
	shards, err := shard(rbc.enc, data)
	if err != nil {
		return err
	}

	reqs, err := makeRequest(shards)
	if err != nil {
		return err
	}

	rbc.contentLength.set(uint64(len(data)))

	if err := rbc.distributeMessage(rbc.proposer, reqs); err != nil {
		return err
	}

	return nil
}

// HandleMessage will used in ACS
func (rbc *RBC) HandleMessage(sender cleisthenes.Member, msg *pb.Message_Rbc) error {
	req, err := processMessage(msg)
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
		return errors.New(fmt.Sprintf("inavlid content length - know as : %d, receive : %d", rbc.contentLength.value(), msg.Rbc.ContentLength))
	}

	rbc.reqChan <- r
	return <-r.err
}

// handleMessage will distinguish input message (from ACS)
func (rbc *RBC) muxRequest(sender cleisthenes.Member, req cleisthenes.Request) error {
	switch r := req.(type) {
	case *ValRequest:
		return rbc.handleValueRequest(sender, r)
	case *EchoRequest:
		return rbc.handleEchoRequest(sender, r)
	case *ReadyRequest:
		return rbc.handleReadyRequest(sender, r)
	default:
		return ErrInvalidRBCType
	}
}

func (rbc *RBC) handleValueRequest(sender cleisthenes.Member, req *ValRequest) error {
	if rbc.valReceived.Value() {
		return errors.New("already receive req message")
	}

	if rbc.echoSent.Value() {
		return errors.New(fmt.Sprintf("already sent echo message - sender id : %s", sender.Address.String()))
	}

	// 默克尔树校验
	if !validateMessage(req) {
		return errors.New("invalid VALUE request")
	}

	rbc.valReceived.Set(true)
	rbc.echoSent.Set(true)
	rbc.shareMessage(rbc.proposer, &EchoRequest{*req})

	return nil
}

func (rbc *RBC) handleEchoRequest(sender cleisthenes.Member, req *EchoRequest) error {
	if req, _ := rbc.echoReqRepo.Find(sender.Address); req != nil {
		return errors.New(fmt.Sprintf("already received echo request - from : %s", sender.Address.String()))
	}

	if !validateMessage(&req.ValRequest) {
		return errors.New(fmt.Sprintf("invalid ECHO request - from : %s", sender.Address.String()))
	}

	if err := rbc.echoReqRepo.Save(sender.Address, req); err != nil {
		return err
	}

	// TODO 这里需要改成当收到n-f个一样的md5值的时候才进行下一步？
	if rbc.countEchos(req.RootHash) >= rbc.echoThreshold() && !rbc.readySent.Value() {
		rbc.log.Debug().Msgf("recv n-f echo, start to share ready message, rootHash:%s", hex.EncodeToString(req.RootHash))
		rbc.readySent.Set(true)
		readyReq := &ReadyRequest{req.RootHash}
		rbc.shareMessage(rbc.proposer, readyReq)
	}

	//正常情况不会进入下面这个循环，会走358行解码RBC成功
	//如果收到了2f+1个ready，但是没有收到足够的echo，会在这里等待收到足够的（n-f）echo信息后对tx进行还原（纠删码）
	if !rbc.done.Value() && rbc.countReadys(req.RootHash) >= rbc.outputThreshold() && rbc.countEchos(req.RootHash) >= rbc.echoThreshold() {
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

	if rbc.countReadys(req.RootHash) >= rbc.readyThreshold() && !rbc.readySent.Value() {
		rbc.log.Debug().Msgf("recv f+1 ready, start to share ready message, rootHash:%s", hex.EncodeToString(req.RootHash))
		rbc.readySent.Set(true)
		rbc.shareMessage(rbc.proposer, &ReadyRequest{req.RootHash})
	}

	if !rbc.done.Value() && rbc.countReadys(req.RootHash) >= rbc.outputThreshold() && rbc.countEchos(req.RootHash) >= rbc.echoThreshold() {
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
			//开始进入RBC阶段
			req.err <- rbc.muxRequest(req.sender, req.data)
		}
	}
}

func (rbc *RBC) Close() {
	rbc.closeChan <- struct{}{}
	<-rbc.closeChan
	close(rbc.reqChan)
	if first := atomic.CompareAndSwapInt32(&rbc.stopFlag, int32(0), int32(1)); !first {
		return
	}
	//close(rbc.closeChan)
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
	//fmt.Println("test76")

	reqs := rbc.echoReqRepo.FindAll()

	if len(reqs) < rbc.numDataShards {
		return nil, errors.New(fmt.Sprintf("not enough shards - minimum : %d, got : %d ", rbc.numDataShards, len(reqs)))
	}

	// To indicate missing data, you should set the shard to nil before calling Reconstruct
	shards := make([][]byte, rbc.numDataShards+rbc.numParityShards)
	for _, req := range reqs {
		if bytes.Equal(rootHash, req.(*EchoRequest).RootHash) {
			order := merkletree.OrderOfData(req.(*EchoRequest).Indexes)
			//将纠删码分片放在了请求的Data数据里
			shards[order] = req.(*EchoRequest).Data.Bytes()
		}
	}

	if err := rbc.enc.Reconstruct(shards); err != nil {
		return nil, err
	}

	// TODO : check interpolated data's merkle root hash and request's merkle root hash

	var value []byte
	for _, data := range shards[:rbc.numDataShards] {
		value = append(value, data...)
	}

	return value[:rbc.contentLength.value()], nil
}

func (rbc *RBC) decodeSuccess(decValue []byte) {
	rbc.output.set(decValue)
	rbc.done.Set(true)
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

func makeRequest(shards []merkletree.Data) ([]cleisthenes.Request, error) {
	tree, err := merkletree.New(shards)
	if err != nil {
		return nil, err
	}

	reqs := make([]cleisthenes.Request, 0)
	rootHash := tree.MerkleRoot()
	for _, shard := range shards {
		paths, indexes, err := tree.MerklePath(shard)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, &ValRequest{
			RootHash: rootHash,
			Data:     shard,
			RootPath: paths,
			Indexes:  indexes,
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
