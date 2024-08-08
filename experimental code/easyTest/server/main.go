package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os/exec"
	"time"
)

var (
	TxSize  int
	TxCount int
)

func main() {
	flag.IntVar(&TxSize, "b", 65, "single transaction size")
	flag.IntVar(&TxCount, "n", 409600, "transaction count")
	flag.Parse()

	address := ":8081"
	log.Printf("listen address:%v, transaction size:%v, transaction count:%v\n", address, TxSize, TxCount)
	if err := http.ListenAndServe(address, NewApiHandler()); err != nil {
		log.Printf("listen failed, err:%v\n", err.Error())
	}
}

type Response struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

func NewApiHandler() http.Handler {
	r := mux.NewRouter()

	ctx, cancel := context.WithCancel(context.Background())

	r.Methods("GET").Path("/test/deploy100nodes").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		log.Println("GET deploy100nodes")

		cancel()
		ctx, cancel = context.WithCancel(context.Background())

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

		c := exec.CommandContext(ctx, "cp", "-f", "conf100.json", "conf.json")
		out, err := c.CombinedOutput()
		log.Println("cp", string(out))
		if err != nil {
			resp.Code = 1
			resp.Msg = "cp conf.json failed"
			log.Println("cp failed, err:", err)
			return
		}

		c = exec.CommandContext(ctx, "./genConfig", "allConfig")
		out, err = c.CombinedOutput()
		log.Println("genConfig", string(out))
		if err != nil {
			resp.Code = 2
			resp.Msg = "gen config failed"
			log.Println("gen config failed, err:", err)
			return
		}

		c = exec.CommandContext(ctx, "bash", "k.sh")
		out, err = c.CombinedOutput()
		log.Println("stop nodes", string(out))
		if err != nil {
			resp.Code = 3
			resp.Msg = "stop nodes failed"
			log.Println("stop nodes failed, err:", err)
			return
		}

		c = exec.CommandContext(ctx, "bash", "ssh_copy_config.sh")
		out, err = c.CombinedOutput()
		log.Println("copy config", string(out))
		if err != nil {
			resp.Code = 4
			resp.Msg = "distribute config failed"
			log.Println("distribute config failed, err:", err)
			return
		}

		c = exec.CommandContext(ctx, "bash", "r.sh")
		out, err = c.CombinedOutput()
		log.Println("start nodes", string(out))
		if err != nil {
			resp.Code = 5
			resp.Msg = "start nodes failed"
			log.Println("start nodes failed, err:", err)
			return
		}

		// 等待节点启动完成
		time.Sleep(2 * time.Second)

		c = exec.CommandContext(ctx, "./sendTx", "-c")
		out, err = c.CombinedOutput()
		log.Println("connect nodes", string(out))
		if err != nil {
			resp.Code = 6
			resp.Msg = "connect nodes failed"
			log.Println("connect nodes failed, err:", err)
			return
		}
	})

	r.Methods("GET").Path("/test/deploy5nodes").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		log.Println("GET deploy5nodes")

		cancel()
		ctx, cancel = context.WithCancel(context.Background())

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

		c := exec.CommandContext(ctx, "cp", "-f", "conf5.json", "conf.json")
		out, err := c.CombinedOutput()
		log.Println("cp", string(out))
		if err != nil {
			resp.Code = 1
			resp.Msg = "cp conf.json failed"
			log.Println("cp failed, err:", err)
			return
		}

		c = exec.CommandContext(ctx, "./genConfig", "-dir", "allConfig")
		out, err = c.CombinedOutput()
		log.Println("genConfig", string(out))
		if err != nil {
			resp.Code = 2
			resp.Msg = "gen config failed"
			log.Println("gen config failed, err:", err)
			return
		}

		c = exec.CommandContext(ctx, "bash", "k.sh")
		out, err = c.CombinedOutput()
		log.Println("stop nodes", string(out))
		if err != nil {
			resp.Code = 3
			resp.Msg = "stop nodes failed"
			log.Println("stop nodes failed, err:", err)
			return
		}

		c = exec.CommandContext(ctx, "bash", "ssh_copy_config.sh")
		out, err = c.CombinedOutput()
		log.Println("copy config", string(out))
		if err != nil {
			resp.Code = 4
			resp.Msg = "distribute config failed"
			log.Println("distribute config failed, err:", err)
			return
		}

		c = exec.CommandContext(ctx, "bash", "r.sh")
		out, err = c.CombinedOutput()
		log.Println("start nodes", string(out))
		if err != nil {
			resp.Code = 5
			resp.Msg = "start nodes failed"
			log.Println("start nodes failed, err:", err)
			return
		}

		// 等待节点启动完成
		time.Sleep(2 * time.Second)

		c = exec.CommandContext(ctx, "./sendTx", "-c")
		out, err = c.CombinedOutput()
		log.Println("connect nodes", string(out))
		if err != nil {
			resp.Code = 6
			resp.Msg = "connect nodes failed"
			log.Println("connect nodes failed, err:", err)
			return
		}
	})

	r.Methods("GET").Path("/test/test").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		log.Println("GET test")

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
	})

	r.Methods("GET").Path("/test/sendtxs").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		log.Println("GET sendtxs")

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

		c := exec.CommandContext(ctx, "./sendTx", "-b", fmt.Sprintf("%d", TxSize), "-n", fmt.Sprintf("%d", TxCount), "-t")
		out, err := c.CombinedOutput()
		log.Println("sendTx", string(out))
		if err != nil {
			resp.Code = 1
			resp.Msg = "send txs failed"
			log.Println("send txs failed, err:", err)
			return
		}
	})

	//r.Methods("GET").Path("/test/querytps").HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
	//})

	return r
}
