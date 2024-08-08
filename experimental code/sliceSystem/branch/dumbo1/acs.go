package dumbo1

import (
	"errors"
	"fmt"
	"github.com/DE-labtory/cleisthenes/sliceSystem/branch/acs"
	"sync"

	"github.com/DE-labtory/cleisthenes/pb"

	"github.com/DE-labtory/cleisthenes"
)

type D1ACS interface {
	HandleInput(data []byte) error
	HandleMessage(sender cleisthenes.Member, msg *pb.Message) error
	Close()
}

type d1acsRepository struct {
	lock  sync.RWMutex
	items map[cleisthenes.Epoch]D1ACS
}

func newD1ACSRepository() *d1acsRepository {
	return &d1acsRepository{
		lock:  sync.RWMutex{},
		items: make(map[cleisthenes.Epoch]D1ACS),
	}
}

func (r *d1acsRepository) save(epoch cleisthenes.Epoch, instance D1ACS) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	_, ok := r.items[epoch]
	if ok {
		return errors.New(fmt.Sprintf("acs instance already exist with epoch [%d]", epoch))
	}
	r.items[epoch] = instance
	return nil
}

func (r *d1acsRepository) find(epoch cleisthenes.Epoch) (D1ACS, bool) {
	r.lock.Lock()
	defer r.lock.Unlock()
	result, ok := r.items[epoch]
	return result, ok
}

func (r *d1acsRepository) delete(epoch cleisthenes.Epoch) {
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.items, epoch)
}

// D1ACSFactory helps create D1ACS instance easily. To create D1ACS, we need lots of DI
// And for the ease of creating D1ACS, D1ACSFactory have components which is need to
// create D1ACS
type D1ACSFactory interface {
	Create(epoch cleisthenes.Epoch) (D1ACS, error)
}

type DefaultD1ACSFactory struct {
	n           int
	f           int
	k           int
	acsOwner    cleisthenes.Member
	batchSender cleisthenes.BatchSender
	memberMap   *cleisthenes.MemberMap
	tss         cleisthenes.Tss
	// FIXME 暂时使用tpke实现抛硬币
	tpk            cleisthenes.Tpke
	dataReceiver   cleisthenes.DataReceiver
	dataSender     cleisthenes.DataSender
	binaryReceiver cleisthenes.BinaryReceiver
	binarySender   cleisthenes.BinarySender
	broadcaster    cleisthenes.Broadcaster
}

func NewDefaultD1ACSFactory(
	n int,
	f int,
	k int,
	acsOwner cleisthenes.Member,
	memberMap *cleisthenes.MemberMap,
	tss cleisthenes.Tss,
	tpk cleisthenes.Tpke,
	// rbc给bba发送结果的通道
	dataReceiver cleisthenes.DataReceiver,
	dataSender cleisthenes.DataSender,
	// bba给acs发送结果的通道
	binaryReceiver cleisthenes.BinaryReceiver,
	binarySender cleisthenes.BinarySender,
	// acs给hb发送结果的通道
	batchSender cleisthenes.BatchSender,
	// 广播实例
	broadcaster cleisthenes.Broadcaster,
) *DefaultD1ACSFactory {
	return &DefaultD1ACSFactory{
		n:              n,
		f:              f,
		k:              k,
		acsOwner:       acsOwner,
		memberMap:      memberMap,
		tss:            tss,
		tpk:            tpk,
		dataReceiver:   dataReceiver,
		dataSender:     dataSender,
		binaryReceiver: binaryReceiver,
		binarySender:   binarySender,
		batchSender:    batchSender,
		broadcaster:    broadcaster,
	}
}

func (f *DefaultD1ACSFactory) Create(epoch cleisthenes.Epoch) (D1ACS, error) {
	dataChan := cleisthenes.NewDataChannel(f.n)
	binaryChan := cleisthenes.NewBinaryChannel(f.n)
	cmisChan := cleisthenes.NewCMISChannel()
	indexDataChan := cleisthenes.NewDataChannel(f.k)

	return acs.New(
		f.n,
		f.f,
		f.k,
		epoch,
		f.acsOwner,
		f.memberMap,
		f.tss,
		f.tpk,
		cmisChan,
		cmisChan,
		dataChan,
		dataChan,
		indexDataChan,
		indexDataChan,
		binaryChan,
		binaryChan,
		f.batchSender,
		f.broadcaster,
	)
}
