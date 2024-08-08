package ce

import (
	"errors"
	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/bytedance/sonic"
	"github.com/golang/protobuf/ptypes"
	"github.com/rs/zerolog"
	"math/rand"
	"sort"
	"strconv"
	"sync/atomic"
)

type request struct {
	sender cleisthenes.Member
	data   cleisthenes.Request
	err    chan error
}
type CE struct {
	// number of network nodes
	n int
	// number of byzantine nodes which can tolerate
	f int
	// size of cmis
	k     int
	epoch cleisthenes.Epoch
	// proposerId is the ID of proposing node
	proposer cleisthenes.Member
	// owner of ce instance (node)
	owner cleisthenes.Member
	// 已经验证的share消息的发送者的集合
	segema map[cleisthenes.Member]struct{}
	// 门限签名算法相关
	tss cleisthenes.Tss

	cmisSent, done *cleisthenes.BinaryState

	stopFlag int32

	closeChan chan struct{}
	reqChan   chan request

	broadcaster cleisthenes.Broadcaster

	memberMap  *cleisthenes.MemberMap
	cmisSender cleisthenes.CMISSender

	log zerolog.Logger
}

func New(
	n, f, k int,
	epoch cleisthenes.Epoch,
	tss cleisthenes.Tss,
	owner, proposer cleisthenes.Member,
	broadcaster cleisthenes.Broadcaster,
	memberMap *cleisthenes.MemberMap,
	cmisSender cleisthenes.CMISSender,
) (*CE, error) {
	ce := &CE{
		n:           n,
		f:           f,
		k:           k,
		epoch:       epoch,
		tss:         tss,
		owner:       owner,
		proposer:    proposer,
		cmisSent:    cleisthenes.NewBinaryState(),
		done:        cleisthenes.NewBinaryState(),
		closeChan:   make(chan struct{}),
		reqChan:     make(chan request, n*n),
		segema:      make(map[cleisthenes.Member]struct{}),
		broadcaster: broadcaster,
		memberMap:   memberMap,
		cmisSender:  cmisSender,
		log:         cleisthenes.NewLoggerWithHead("CE"),
	}

	ce.initLog()
	go ce.run()
	return ce, nil
}

func (ce *CE) initLog() {
	var logPrefix zerolog.HookFunc
	logPrefix = func(e *zerolog.Event, level zerolog.Level, message string) {
		e.Uint64("Epoch", uint64(ce.epoch)).
			Str("owner", ce.owner.Address.String())
	}
	ce.log = ce.log.Hook(logPrefix)
}

func (ce *CE) shareMessage(proposer cleisthenes.Member, req cleisthenes.Request) error {
	var typ pb.CE_Type
	switch req.(type) {
	case *ShareRequest:
		typ = pb.CE_SHARE
	default:
		return errors.New("invalid shareMessage message type")
	}
	payload, err := sonic.ConfigFastest.Marshal(req)
	if err != nil {
		return err
	}
	ce.broadcaster.ShareMessage(pb.Message{
		Proposer:  proposer.Address.String(),
		Sender:    ce.owner.Address.String(),
		Timestamp: ptypes.TimestampNow(),
		Epoch:     uint64(ce.epoch),
		Payload: &pb.Message_Ce{
			Ce: &pb.CE{
				Payload: payload,
				Type:    typ,
			},
		},
	})
	return nil
}

func (ce *CE) distributeMessage(proposer cleisthenes.Member, reqs []cleisthenes.Request) error {
	var typ pb.CE_Type
	switch reqs[0].(type) {
	case *ShareRequest:
		typ = pb.CE_SHARE
	default:
		return errors.New("invalid ce distribute message type")
	}

	msgList := make([]pb.Message, 0)
	for _, req := range reqs {
		payload, err := sonic.ConfigFastest.Marshal(req)
		if err != nil {
			return err
		}

		msgList = append(msgList, pb.Message{
			Proposer:  proposer.Address.String(),
			Sender:    ce.owner.Address.String(),
			Timestamp: ptypes.TimestampNow(),
			Epoch:     uint64(ce.epoch),
			Payload: &pb.Message_Ce{
				Ce: &pb.CE{
					Payload: payload,
					Type:    typ,
				},
			},
		})
	}
	ce.broadcaster.DistributeMessage(msgList)

	return nil
}

// 当一轮迭代开始时，CE举算法开始，将epoch作为CE的输入，进行门限签名，并将签名分发给其他节点
func (ce *CE) HandleInput(epoch cleisthenes.Epoch) error {
	if ce.cmisSent.Value() {
		return nil
	}

	id := strconv.FormatUint(uint64(epoch), 10)
	share := ce.cshare(id)

	req, err := makeRequest(id, share)
	if err != nil {
		return err
	}

	if err := ce.shareMessage(ce.proposer, req); err != nil {
		return err
	}
	return nil
}

func (ce *CE) HandleMessage(sender cleisthenes.Member, msg *pb.Message_Ce) error {
	req, err := processMessage(msg)
	if err != nil {
		return err
	}
	r := request{
		sender: sender,
		data:   req,
		err:    make(chan error),
	}

	ce.reqChan <- r
	return <-r.err
}

// 根据id计算出share
func (ce *CE) cshare(id string) []byte {
	return ce.tss.Sign([]byte(id))
}

// 根据id和share构造请求
func makeRequest(id string, share []byte) (cleisthenes.Request, error) {
	return &ShareRequest{
		ID:    id,
		Share: share,
	}, nil
}

func processMessage(msg *pb.Message_Ce) (cleisthenes.Request, error) {
	switch msg.Ce.Type {
	case pb.CE_SHARE:
		return processShareMessage(msg)
	default:
		return nil, errors.New("error processing message with invalid type")
	}
}

func processShareMessage(msg *pb.Message_Ce) (cleisthenes.Request, error) {
	req := &ShareRequest{}
	if err := sonic.ConfigFastest.Unmarshal(msg.Ce.Payload, req); err != nil {
		return nil, err
	}
	return req, nil
}

func (ce *CE) Close() {
	ce.closeChan <- struct{}{}
	<-ce.closeChan
	if first := atomic.CompareAndSwapInt32(&ce.stopFlag, int32(0), int32(1)); !first {
		return
	}
}

func (ce *CE) toDie() bool {
	return atomic.LoadInt32(&(ce.stopFlag)) == int32(1)
}

func (ce *CE) muxRequest(sender cleisthenes.Member, req cleisthenes.Request) error {
	switch r := req.(type) {
	case *ShareRequest:
		return ce.handleShareRequest(sender, r)
	default:
		return ErrInvalidCEType
	}
}

func (ce *CE) segemaThreshold() int {
	return ce.f + 1
}

// 收到其他节点的share消息的处理
func (ce *CE) handleShareRequest(sender cleisthenes.Member, req *ShareRequest) error {
	// cmis已经产生并发送
	if ce.cmisSent.Value() {
		return nil
	}

	if ce.tss.Verify(sender.Address, req.Share, []byte(req.ID)) {
		ce.log.Debug().Msgf("来自%s的签名验证成功", sender.Address.String())
		ce.segema[sender] = struct{}{}
	} else {
		ce.log.Warn().Msgf("来自%s的签名验证失败", sender.Address.String())
		return nil
	}

	// segema集合已经有f+1个已经验证的share消息，并且没有发送cmisSent消息，才发送cmis消息给acs
	if len(ce.segema) == ce.segemaThreshold() && !ce.cmisSent.Value() {
		ce.cmisSent.Set(true)
		cmis, err := ce.ctoss(req.ID)
		if err != nil {
			return err
		}
		ce.cmisSender.Send(cleisthenes.CMISMessage{
			CMIS: cmis,
		})
		return nil
	}

	return nil
}

// 使用伪随机数发生器实现ctoss
func (ce *CE) ctoss(id string) (map[cleisthenes.Member]struct{}, error) {
	// 使用共同的轮数作为种子
	rand.Seed(int64(ce.epoch))
	perm := rand.Perm(ce.n)
	members := ce.memberMap.Members()
	// FIXME 这里使用端口号作为排序的依据，如果是分布式网络，各节点之间应该按照什么规则排序呢（公网ip+端口？）？
	sort.Slice(members, func(i, j int) bool {
		return members[i].Address.Port < members[j].Address.Port
	})
	cmis := make(map[cleisthenes.Member]struct{})
	for _, idx := range perm[:ce.k] {
		cmis[members[idx]] = struct{}{}
	}
	return cmis, nil
}

func (ce *CE) run() {
	for !ce.toDie() {
		select {
		case <-ce.closeChan:
			ce.closeChan <- struct{}{}
			return
		case req := <-ce.reqChan:
			req.err <- ce.muxRequest(req.sender, req.data)
		}
	}
}
