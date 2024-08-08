package app

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/sliceSystem"
	kitendpoint "github.com/go-kit/kit/endpoint"
	kitlog "github.com/go-kit/kit/log"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	"net/http"
	"reflect"
)

type ErrIllegalArgument struct {
	Reason string
}

func (e ErrIllegalArgument) Error() string {
	return fmt.Sprintf("err illegal argument: %s", e.Reason)
}

type endpoint struct {
	logger      kitlog.Logger
	sliceSystem sliceSystem.SliceSystem
}

type Response struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

func newEndpoint(sliceSystem sliceSystem.SliceSystem, logger kitlog.Logger) *endpoint {
	return &endpoint{
		logger:      logger,
		sliceSystem: sliceSystem,
	}
}

func NewApiHandler(sliceSystem sliceSystem.SliceSystem, logger kitlog.Logger) http.Handler {
	endpoint := newEndpoint(sliceSystem, logger)
	r := mux.NewRouter()

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorLogger(logger),
	}

	r.Methods("GET").Path("/healthz").HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		logger.Log("method", "GET", "endpoint", "healthz")
		//sliceSystem.GetMainChain().Close()
		//sliceSystem.GetCtrl().Close()
		w.Write([]byte("up"))
	})

	r.Methods("GET").Path("/test/historytps").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Access-Control-Allow-Origin", "*")
		writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		writer.Header().Set("Access-Control-Allow-Headers", "DNT,X-CustomHeader,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type")

		resp := &Response{
			Code: 0,
			Data: nil,
			Msg:  "ok",
		}

		defer func() {
			bs, err := json.Marshal(resp)
			if err != nil {
				writer.Write([]byte(`{"code":-1,"msg":"internal error"}`))
				return
			}
			writer.Write(bs)
		}()

		tps := cleisthenes.MainTps
		if sliceSystem.MainChainNetworkSize() == 1 {
			tps = cleisthenes.BranchTps
		}

		if tps.LastOutputTime.IsZero() {
			data := map[string]interface{}{
				"history": make([]map[string]interface{}, 0),
			}
			resp.Data = data
			return
		}

		history := make([]map[string]interface{}, 0)
		for _, tpsInfo := range tps.History {
			history = append(history, map[string]interface{}{
				"txs":   tpsInfo.Total,
				"tps":   float64(tpsInfo.Total) / float64(tpsInfo.OutputTime.Sub(tps.StartTime).Milliseconds()) * 1000.0,
				"time":  float64(tpsInfo.OutputTime.Sub(tps.StartTime).Milliseconds()) / 1000.0,
				"epoch": tpsInfo.Epoch,
			})
		}

		data := map[string]interface{}{
			"history": history,
		}
		resp.Data = data
	})

	r.Methods("GET").Path("/test/querytps").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

		writer.Header().Set("Access-Control-Allow-Origin", "*")
		writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		writer.Header().Set("Access-Control-Allow-Headers", "DNT,X-CustomHeader,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type")

		resp := &Response{
			Code: 0,
			Data: nil,
			Msg:  "ok",
		}

		defer func() {
			bs, err := json.Marshal(resp)
			if err != nil {
				writer.Write([]byte(`{"code":-1,"msg":"internal error"}`))
				return
			}
			writer.Write(bs)
		}()

		tps := cleisthenes.MainTps
		if sliceSystem.MainChainNetworkSize() == 1 {
			tps = cleisthenes.BranchTps
		}

		if tps.LastOutputTime.IsZero() {
			data := map[string]interface{}{
				"txs":   0,
				"tps":   0,
				"time":  0,
				"epoch": 0,
			}
			resp.Data = data
			return
		}

		data := map[string]interface{}{
			"txs":   tps.Total,
			"tps":   float64(tps.Total) / float64(tps.LastOutputTime.Sub(tps.StartTime).Milliseconds()) * 1000.0,
			"time":  float64(tps.LastOutputTime.Sub(tps.StartTime).Milliseconds()) / 1000.0,
			"epoch": tps.Epoch,
		}
		resp.Data = data
	})

	r.Methods("POST").Path("/tx").Handler(kithttp.NewServer(
		endpoint.proposeTx,
		decodeProposeTxRequest,
		encodeResponse,
		opts...,
	))

	r.Methods("POST").Path("/txs").Handler(kithttp.NewServer(
		endpoint.proposeTxs,
		decodeProposeTxsRequest,
		encodeResponse,
		opts...,
	))

	r.Methods("POST").Path("/connections").Handler(kithttp.NewServer(
		endpoint.createConnections,
		decodeCreateConnectionsRequest,
		encodeResponse,
		opts...,
	))

	r.Methods("GET").Path("/connections").Handler(kithttp.NewServer(
		endpoint.getConnections,
		decodeGetConnectionsRequest,
		encodeResponse,
		opts...,
	))

	return r
}

func (e *endpoint) proposeTx(ctx context.Context, request interface{}) (interface{}, error) {
	e.logger.Log("endpoint", "proposeTx")

	f := e.makeProposeTxEndpoint()
	response, err := f(ctx, request)
	if err != nil {
		e.logger.Log("endpoint", "proposeTx", "err", err.Error())
	}
	return response, err
}

func (e *endpoint) proposeTxs(ctx context.Context, request interface{}) (interface{}, error) {
	e.logger.Log("endpoint", "proposeTxs")

	f := e.makeProposeTxsEndpoint()
	response, err := f(ctx, request)
	if err != nil {
		e.logger.Log("endpoint", "proposeTx", "err", err.Error())
	}
	return response, err
}
func validateTx(tx Transaction) bool {
	return true
	buf := bytes.NewBuffer([]byte{})
	buf.WriteString(tx.TxID)
	buf.WriteString(tx.From)
	buf.WriteString(tx.To)
	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, uint64(tx.Amount))
	buf.Write(bs)
	sig, err := base64.StdEncoding.DecodeString(tx.Signature)
	if err != nil {
		log.Fatal().Msgf("decode signature failed, err:%v", err)
	}
	pub, err := base64.StdEncoding.DecodeString(tx.From)
	if err != nil {
		log.Fatal().Msgf("decode from pub failed, err:%v", err)
	}
	return ed25519.Verify(pub, buf.Bytes(), sig)
}

func (e *endpoint) makeProposeTxEndpoint() kitendpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ProposeTxRequest)
		if !validateTx(req.Transaction) {
			return ProposeTxResponse{Code: -1}, fmt.Errorf("verify signature failed, id:%v", req.Transaction.TxID)
		}
		return ProposeTxResponse{}, e.sliceSystem.Submit(req.Transaction)
	}
}

func (e *endpoint) makeProposeTxsEndpoint() kitendpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ProposeTxsRequest)
		txs := make([]cleisthenes.Transaction, 0)
		for _, tx := range req.Transactions {
			if !validateTx(tx) {
				return ProposeTxsResponse{Code: -1}, fmt.Errorf("verify signature failed, id:%v", tx.TxID)
			}
			txs = append(txs, tx)
		}
		return ProposeTxsResponse{}, e.sliceSystem.SubmitTxs(txs)
	}
}

func (e *endpoint) createConnections(ctx context.Context, request interface{}) (interface{}, error) {
	e.logger.Log("endpoint", "createConnections")

	f := e.makeCreateConnectionsEndpoint()
	response, err := f(ctx, request)
	if err != nil {
		e.logger.Log("endpoint", "createConnections", "err", err.Error())
	}
	return response, err
}

func (e *endpoint) makeCreateConnectionsEndpoint() kitendpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		//req := request.(CreateConnectionsRequest)
		//return CreateConnectionsResponse{}, e.sliceSystem.ConnectAll(req.TargetList)
		return CreateConnectionsResponse{}, e.sliceSystem.ConnectAll()
	}
}

func (e *endpoint) getConnections(ctx context.Context, request interface{}) (interface{}, error) {
	e.logger.Log("endpoint", "getConnections")

	f := e.makeGetConnectionsEndpoint()
	response, err := f(ctx, request)
	if err != nil {
		e.logger.Log("endpoint", "getConnections", "err", err.Error())
	}
	return response, err
}

func (e *endpoint) makeGetConnectionsEndpoint() kitendpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return GetConnectionsResponse{
			ConnectionList: e.sliceSystem.ConnectionList(),
		}, nil
	}
}

type ProposeTxRequest struct {
	Transaction Transaction `json:"transaction"`
}

type ProposeTxsRequest struct {
	Transactions []Transaction `json:"transactions"`
}

type ProposeTxResponse struct {
	Code int
}

type ProposeTxsResponse struct {
	Code int
}

type CreateConnectionsRequest struct {
	TargetList []string `json:"targets"`
}

type CreateConnectionsResponse struct{}

type GetConnectionsRequest struct{}

type GetConnectionsResponse struct {
	ConnectionList []string `json:"connections"`
}

func decodeProposeTxRequest(_ context.Context, r *http.Request) (interface{}, error) {
	body := ProposeTxRequest{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, err
	}
	if reflect.DeepEqual(body.Transaction, Transaction{}) {
		return nil, ErrIllegalArgument{"transaction is empty"}
	}
	return body, nil
}

func decodeProposeTxsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	body := ProposeTxsRequest{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, err
	}
	if len(body.Transactions) == 0 {
		return nil, ErrIllegalArgument{"transactions is empty"}
	}
	return body, nil
}

func decodeCreateConnectionsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	body := CreateConnectionsRequest{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func decodeGetConnectionsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return GetConnectionsRequest{}, nil
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if e, ok := response.(errorer); ok && e.error() != nil {
		encodeError(ctx, e.error(), w)
		return nil
	}
	//fmt.Println("test25")
	//fmt.Printf("response=%v", response)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

type errorer interface {
	error() error
}

// encode errors from business-logic
func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	switch err {
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}
