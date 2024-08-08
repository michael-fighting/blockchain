package cleisthenes

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"google.golang.org/protobuf/encoding/prototext"
	"sync"
	"sync/atomic"

	"github.com/DE-labtory/cleisthenes/pb"
)

type ConnId = string

// message used in HBBFT
type innerMessage struct {
	Message   *pb.Message
	OnErr     func(error)
	OnSuccess func(interface{})
}

// message used with other nodes
type Message struct {
	*pb.Message
	Conn Connection
}

// request handler
type Handler interface {
	ServeRequest(msg Message)
}

type Connection interface {
	Send(msg pb.Message, successCallBack func(interface{}), errCallBack func(error))
	Ip() Address
	Id() ConnId
	Close()
	Start() error
	Handle(handler Handler)
}

type GrpcConnection struct {
	id            ConnId
	ip            Address
	streamWrapper StreamWrapper
	stopFlag      int32
	handler       Handler
	outChan       chan *innerMessage
	readChan      chan *pb.Message
	stopChan      chan struct{}
	tss           Tss
	sync.RWMutex
}

func NewConnection(ip Address, id ConnId, streamWrapper StreamWrapper, t Tss) (Connection, error) {
	if streamWrapper == nil {
		return nil, errors.New("fail to create connection ! : streamWrapper is nil")
	}
	return &GrpcConnection{
		id:            id,
		ip:            ip,
		streamWrapper: streamWrapper,
		outChan:       make(chan *innerMessage, 10000),
		readChan:      make(chan *pb.Message, 10000),
		stopChan:      make(chan struct{}, 1),
		tss:           t,
	}, nil
}

func (conn *GrpcConnection) Send(msg pb.Message, successCallBack func(interface{}), errCallBack func(error)) {
	conn.Lock()
	defer conn.Unlock()

	m := &innerMessage{
		Message:   &msg,
		OnErr:     errCallBack,
		OnSuccess: successCallBack,
	}

	switch pl := msg.Payload.(type) {
	case *pb.Message_Rbc:
		switch pl.Rbc.Type {
		case pb.RBC_VAL:
			{
				log.Debug().Msgf("send to outChan rbc/irbc val message from sender:%v proposer:%v epoch:%v, to:%v",
					msg.Sender, msg.Proposer, msg.Epoch, conn.Ip().String())
			}
		}
	}

	log.Debug().Msgf("out chan len:%d", len(conn.outChan))

	conn.outChan <- m
}

func (conn *GrpcConnection) Ip() Address {
	return conn.ip
}

func (conn *GrpcConnection) Id() ConnId {
	return conn.id
}

func (conn *GrpcConnection) Close() {
	log.Warn().Msgf("conn start to close, conn:%v", conn.Ip().String())

	if conn.isDie() {
		return
	}

	isFirst := atomic.CompareAndSwapInt32(&conn.stopFlag, int32(0), int32(1))
	if !isFirst {
		return
	}

	conn.stopChan <- struct{}{}
	conn.Lock()
	defer conn.Unlock()

	conn.streamWrapper.Close()
}

func (conn *GrpcConnection) Start() error {
	errChan := make(chan error, 1)

	go conn.readStream(errChan)
	go conn.writeStream()

	for !conn.isDie() {
		select {
		case stop := <-conn.stopChan:
			conn.stopChan <- stop
			return nil
		case err := <-errChan:
			log.Error().Msgf("errChan has err:%v", err.Error())
			return err
		case message := <-conn.readChan:
			msg := message
			log.Debug().Msgf("read stream message, msgId:%v", string(msg.Signature))
			switch pl := msg.Payload.(type) {
			case *pb.Message_Rbc:
				log.Debug().Msgf("stream recv %v irbc:%v message from sender:%v proposer:%v epoch:%v, from:%v msgId:%v",
					pl.Rbc.Type, pl.Rbc.IsIndexRbc, msg.Sender, msg.Proposer, msg.Epoch, conn.Ip().String(), string(msg.Signature))
			}
			if conn.verify(msg) {
				if conn.handler != nil {
					m := Message{Message: msg, Conn: conn}
					//接收到其他节点的值后，会执行conn里面的handler函数
					// FIXME 协程太多如何关闭？
					//go conn.handler.ServeRequest(m)
					conn.handler.ServeRequest(m)
				}
			}
		}
	}

	return nil
}

func (conn *GrpcConnection) Handle(handler Handler) {
	conn.handler = handler
}

// TODO : implements me w/ test case
func (conn *GrpcConnection) verify(envelope *pb.Message) bool {
	return true
	if conn.tss != nil {
		md5sum := md5.Sum(marshal(envelope))
		addr, err := ToAddress(envelope.Sender)
		if err != nil {
			log.Fatal().Msgf("to address failed, sender:%v err:%v", envelope.Sender, err)
		}
		res := conn.tss.Verify(addr, envelope.Signature, md5sum[:])
		log.Debug().Msgf("verify result:%v sender:%v md5:%v sig:%v", res, addr.String(), hex.EncodeToString(md5sum[:]), hex.EncodeToString(envelope.Signature))
		return res
	}
	return true
}

func (conn *GrpcConnection) isDie() bool {
	return atomic.LoadInt32(&(conn.stopFlag)) == int32(1)
}

func marshal(m *pb.Message) []byte {
	bs := bytes.NewBuffer([]byte{})
	bs.WriteString(m.Sender)
	bs.WriteString(m.Proposer)
	es := make([]byte, 8)
	binary.BigEndian.PutUint64(es, m.Epoch)
	bs.Write(es)
	ts, _ := prototext.Marshal(m.Timestamp)
	bs.Write(ts)

	switch pl := m.Payload.(type) {
	case *pb.Message_Rbc:
		ps, _ := prototext.Marshal(pl.Rbc)
		bs.Write(ps)
	case *pb.Message_Ce:
		ps, _ := prototext.Marshal(pl.Ce)
		bs.Write(ps)
	case *pb.Message_Bba:
		ps, _ := prototext.Marshal(pl.Bba)
		bs.Write(ps)
	case *pb.Message_Ds:
		ps, _ := prototext.Marshal(pl.Ds)
		bs.Write(ps)
	}

	return bs.Bytes()
}

func (conn *GrpcConnection) writeStream() {
	defer func() {
		log.Warn().Msgf("write stream return out, conn:%v", conn.Ip().String())
	}()
	for !conn.isDie() {
		select {
		case m := <-conn.outChan:
			if conn.tss != nil {
				//md5Sum := md5.Sum(marshal(m.Message))
				//m.Message.Signature = conn.tss.Sign(md5Sum[:])
				//log.Debug().Msgf("message md5:%v sig:%v", hex.EncodeToString(md5Sum[:]), hex.EncodeToString(m.Message.Signature))
			}
			//使用streamWrapper包装消息体并发送出去
			msg := m.Message
			switch pl := msg.Payload.(type) {
			case *pb.Message_Rbc:
				log.Debug().Msgf("stream send %v irbc:%v message from sender:%v proposer:%v epoch:%v, to:%v msgId:%v",
					pl.Rbc.Type, pl.Rbc.IsIndexRbc, msg.Sender, msg.Proposer, msg.Epoch, conn.Ip().String(), string(m.Message.Signature))
			}
			err := conn.streamWrapper.Send(m.Message)
			if err != nil {
				log.Error().Msgf("stream wrapper send message failed, err:%s", err.Error())
				if m.OnErr != nil {
					go m.OnErr(err)
				}
			} else {
				if m.OnSuccess != nil {
					go m.OnSuccess("")
				}
			}

			switch pl := msg.Payload.(type) {
			case *pb.Message_Rbc:
				log.Debug().Msgf("stream send complete %v irbc:%v message from sender:%v proposer:%v epoch:%v, to:%v msgId:%v",
					pl.Rbc.Type, pl.Rbc.IsIndexRbc, msg.Sender, msg.Proposer, msg.Epoch, conn.Ip().String(), string(m.Message.Signature))
			}
		case stop := <-conn.stopChan:
			conn.stopChan <- stop
			return
		}
	}
}

func (conn *GrpcConnection) readStream(errChan chan error) {
	defer func() {
		recover()
	}()

	for !conn.isDie() {
		envelope, err := conn.streamWrapper.Recv()
		if conn.isDie() {
			return
		}
		if err != nil {
			log.Error().Msgf("read stream err:%v", err.Error())
			errChan <- err
			return
		}
		log.Debug().Msgf("read chan len:%d, msgId:%v Sender:%v Proposer:%v Epoch:%v", len(conn.readChan), string(envelope.Signature),
			envelope.GetSender(), envelope.GetProposer(), envelope.Epoch)
		//从读通道获取其他节点发送过来的消息
		conn.readChan <- envelope
	}
}

type Broadcaster interface {
	// ShareMessage is a function that broadcast Single message to all nodes
	ShareMessage(msg pb.Message)

	// DistributeMessage is a function that broadcast multiple different messages one by one to all nodes
	DistributeMessage(msgList []pb.Message)

	GetAllConnAddr() []string
}

type ConnectionPool struct {
	lock    sync.RWMutex
	connMap map[Address]Connection
}

func NewConnectionPool() *ConnectionPool {
	return &ConnectionPool{
		lock:    sync.RWMutex{},
		connMap: make(map[Address]Connection),
	}
}

func (p *ConnectionPool) GetAll() []Connection {
	p.lock.RLock()
	defer p.lock.RUnlock()

	connList := make([]Connection, 0)
	for _, conn := range p.connMap {
		connList = append(connList, conn)
	}
	return connList
}

func (p *ConnectionPool) Get(addr Address) Connection {
	p.lock.RLock()
	defer p.lock.RUnlock()

	conn, ok := p.connMap[addr]
	if !ok {
		return nil
	}
	return conn
}

func (p *ConnectionPool) ShareMessage(msg pb.Message) {
	for _, conn := range p.GetAll() {
		switch pl := msg.Payload.(type) {
		case *pb.Message_Rbc:
			log.Debug().Msgf("share send %v irbc:%v message from sender:%v proposer:%v epoch:%v, to:%v",
				pl.Rbc.Type, pl.Rbc.IsIndexRbc, msg.Sender, msg.Proposer, msg.Epoch, conn.Ip().String())
		}

		go conn.Send(msg, nil, func(err error) {
			log.Error().Msgf("share message failed, err:%s", err.Error())
		})
	}
}

// DistributeMessage 用来将val的每个分片分发给其他节点，一般是每个节点各一块
func (p *ConnectionPool) DistributeMessage(msgList []pb.Message) {
	for i, conn := range p.GetAll() {
		//if i == len(msgList) {
		//	break
		//}
		msg := msgList[i]
		switch pl := msg.Payload.(type) {
		case *pb.Message_Rbc:
			log.Debug().Msgf("distribute send %v irbc:%v message from sender:%v proposer:%v epoch:%v, to:%v",
				pl.Rbc.Type, pl.Rbc.IsIndexRbc, msg.Sender, msg.Proposer, msg.Epoch, conn.Ip().String())
		}

		go conn.Send(msgList[i], nil, func(err error) {
			log.Error().Msgf("distribute message failed, err:%s", err.Error())
		})
	}
}

func (p *ConnectionPool) GetAllConnAddr() []string {
	p.lock.Lock()
	defer p.lock.Unlock()

	res := make([]string, 0)
	for addr, _ := range p.connMap {
		res = append(res, addr.String())
	}
	return res
}

func (p *ConnectionPool) Add(addr Address, conn Connection) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.connMap[addr] = conn
}

func (p *ConnectionPool) Remove(addr Address) {
	p.lock.Lock()
	defer p.lock.Unlock()

	delete(p.connMap, addr)
}

func (p *ConnectionPool) Clear() {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.connMap = nil
	p.connMap = make(map[Address]Connection)
}

type MessageEndpoint interface {
	HandleMessage(*pb.Message) error
}

type MainChainMessageEndpoint interface {
	HandleMessage(*pb.Message) error
	// 为了方便对那些非主链节点能正常递增主链轮次
	AdvanceEpoch()
}
