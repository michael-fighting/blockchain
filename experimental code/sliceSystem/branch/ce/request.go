package ce

type (
	ShareRequest struct {
		Share []byte
		ID    string
	}
)

//var ErrNoIdMatchingRequest = errors.New("id is not found.")
//var ErrInvalidReqType = errors.New("request is not matching with type.")

func (r ShareRequest) Recv() {}

//type (
//	ShareReqRepository struct {
//		lock sync.RWMutex
//		recv map[cleisthenes.Address]*ShareRequest
//	}
//)
//
//func NewShareReqRepository() *ShareReqRepository {
//	return &ShareReqRepository{
//		recv: make(map[cleisthenes.Address]*ShareRequest),
//		lock: sync.RWMutex{},
//	}
//}
//
//func (r *ShareReqRepository) Save(addr cleisthenes.Address, req cleisthenes.Request) error {
//	r.lock.Lock()
//	defer r.lock.Unlock()
//
//	shareReq, ok := req.(*ShareRequest)
//	if !ok {
//		return ErrInvalidReqType
//	}
//	r.recv[addr] = shareReq
//	return nil
//}
//
//func (r *ShareReqRepository) Find(addr cleisthenes.Address) (cleisthenes.Request, error) {
//	r.lock.Lock()
//	defer r.lock.Unlock()
//
//	_, ok := r.recv[addr]
//	if !ok {
//		return nil, ErrNoIdMatchingRequest
//	}
//	return r.recv[addr], nil
//}
//
//func (r *ShareReqRepository) FindAll() []cleisthenes.Request {
//	r.lock.Lock()
//	defer r.lock.Unlock()
//	reqList := make([]cleisthenes.Request, 0)
//	for _, request := range r.recv {
//		reqList = append(reqList, request)
//	}
//	return reqList
//}
