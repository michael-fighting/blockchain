package cleisthenes

import (
	"context"
	"google.golang.org/grpc/keepalive"
	"net"
	"time"

	"github.com/DE-labtory/iLogger"
	"github.com/google/uuid"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"

	"github.com/DE-labtory/cleisthenes/pb"
)

type ConnHandler func(conn Connection)
type ErrHandler func(err error)

type GrpcServer struct {
	tss         Tss
	connHandler ConnHandler
	errHandler  ErrHandler
	addr        Address
	lis         net.Listener
}

func NewServer(addr Address, t Tss) *GrpcServer {
	return &GrpcServer{
		addr: addr,
		tss:  t,
	}
}

// MessageStream handle request to connection from remote peer. First,
// configure ip of remote peer, then based on that info create connection
// after that handler of server process handles that connection.
func (s GrpcServer) MessageStream(streamServer pb.StreamService_MessageStreamServer) error {
	ip := extractRemoteAddr(streamServer)

	_, cancel := context.WithCancel(context.Background())
	streamWrapper := NewServerStreamWrapper(streamServer, cancel)

	conn, err := NewConnection(Address{Ip: ip}, uuid.New().String(), streamWrapper, s.tss)
	if err != nil {
		return err
	}
	if s.connHandler != nil {
		s.connHandler(conn)
	}
	return nil
}

// extractRemoteAddr returns address of attached peer
func extractRemoteAddr(stream pb.StreamService_MessageStreamServer) string {
	p, ok := peer.FromContext(stream.Context())
	if !ok {
		return ""
	}
	if p.Addr == nil {
		return ""
	}
	return p.Addr.String()
}

func (s *GrpcServer) OnConn(handler ConnHandler) {
	if handler == nil {
		return
	}
	s.connHandler = handler
}

func (s *GrpcServer) OnErr(handler ErrHandler) {
	if handler == nil {
		return
	}
	s.errHandler = handler
}

func (s *GrpcServer) Listen() {
	lis, err := net.Listen("tcp", s.addr.String())
	if err != nil {
		iLogger.Fatalf(nil, "listen error: %s", err.Error())
	}
	defer lis.Close()

	g := grpc.NewServer(
		//grpc.InitialWindowSize(1024*1024*1024),
		//grpc.InitialConnWindowSize(1024*1024*1024),
		grpc.MaxSendMsgSize(2147483647),
		grpc.MaxRecvMsgSize(2147483647),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             1 * time.Second,
			PermitWithoutStream: false,
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Timeout: 500 * time.Second,
		}),
	)
	defer g.Stop()

	pb.RegisterStreamServiceServer(g, s)
	reflection.Register(g)
	s.lis = lis

	log.Info().Msgf("listen ... on: [%s]", s.addr.String())

	if err := g.Serve(lis); err != nil {
		log.Error().Msgf("listen error: [%s]", err.Error())
		iLogger.Errorf(nil, "listen error: [%s]", err.Error())
		g.Stop()
		lis.Close()
	}
}

func (s *GrpcServer) Stop() {
	if s.lis != nil {
		s.lis.Close()
	}
}

const (
	DefaultDialTimeout = 3 * time.Second
)

type DialOpts struct {
	// Ip is target address which grpc client is going to dial
	Addr Address

	// Duration for which to block while established a new connection
	Timeout time.Duration
}

type GrpcClient struct {
	tss Tss
}

func NewClient(t Tss) *GrpcClient {
	return &GrpcClient{
		tss: t,
	}
}

func (c GrpcClient) Dial(opts DialOpts) (Connection, error) {
	dialContext, _ := context.WithTimeout(context.Background(), opts.Timeout)
	gconn, err := grpc.DialContext(dialContext, opts.Addr.String(),
		append([]grpc.DialOption{},
			grpc.WithInsecure(),
			//grpc.WithInitialWindowSize(1024*1024*1024),
			//grpc.WithInitialConnWindowSize(1024*1024*1024),
			grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(2147483647)),
			grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(2147483647)),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{
				Timeout:             500 * time.Second,
				PermitWithoutStream: false,
			}),
		)...,
	)
	if err != nil {
		return nil, err
	}
	streamWrapper, err := NewClientStreamWrapper(gconn)
	if err != nil {
		return nil, err
	}
	conn, err := NewConnection(opts.Addr, uuid.New().String(), streamWrapper, c.tss)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
