package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/DE-labtory/cleisthenes/easyTest"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

type Transaction struct {
	TxID      string `json:"id"`
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    int    `json:"amount"`
	Signature string `json:"signature"`
}

type Transactions struct {
	Transactions []Transaction `json:"transactions"`
}

type Targets struct {
	Targets []string `json:"targets"`
}

const startApiPort = 8000
const branchStartGrpcPort = 15000

const mainChainStartGrpcPort = 16000
const maxTxsSize = 10000

func genDelegateTargets(net *easyTest.Net) Targets {
	ts := Targets{}
	port := mainChainStartGrpcPort
	for _, branch := range net.Branches {
		ip := branch.Nodes[0]
		ts.Targets = append(ts.Targets, fmt.Sprintf("%s:%d", ip, port))
		port += 2
	}
	return ts
}

func getTargets(net *easyTest.Net) Targets {
	var targets Targets
	port := branchStartGrpcPort
	for _, branch := range net.Branches {
		for _, node := range branch.Nodes {
			ip := node
			targets.Targets = append(targets.Targets, fmt.Sprintf("%s:%d", ip, port))
			port += 2
		}
	}
	return targets
}

func sendConn(net *easyTest.Net) {
	targets := getTargets(net)
	log.Debug().Msgf("targets:%v", targets)

	bs, err := json.Marshal(targets)
	if err != nil {
		log.Fatal().Msgf("marshal branch ts i, failed, err:%s", err.Error())
	}

	port := startApiPort
	wg := new(sync.WaitGroup)
	for _, branch := range net.Branches {
		for _, node := range branch.Nodes {
			url := fmt.Sprintf("http://%s:%d/connections", node, port)
			wg.Add(1)
			go sendPost(url, bs, wg)
			port++
		}
	}
	wg.Wait()
	log.Info().Msg("create connections success")
}

func sendPost(url string, bs []byte, wg *sync.WaitGroup) {
	defer wg.Done()
	// 发送请求
	resp, err := http.Post(url, "text/json", bytes.NewReader(bs))
	if err != nil {
		log.Fatal().Msgf("post conn failed, err:%s", err.Error())
	}
	defer resp.Body.Close()
	// 获取响应
	bs, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal().Msgf("read conn resp failed, err:%s", err.Error())
	}
	log.Info().Msgf("post resp:%s", string(bs))
}

const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randStr(n int) string {
	str := ""
	for i := 0; i < n; i++ {
		k := rand.Intn(52)
		str += chars[k : k+1]
	}

	return str
}

//
//func genTx(size int) []byte {
//	s := (size - 30) / 2
//	tx := Transaction{
//		TxID:   uuid.NewString(),
//		From:   randStr(s),
//		To:     randStr(s),
//		Amount: rand.Intn(10),
//	}
//	bs, err := json.Marshal(tx)
//	if err != nil {
//		log.Fatal().Msgf("marshal txs failed, err:%s", err.Error())
//	}
//	return bs
//}
//
//func sendTx()

func genTxs(size, n int, passports []easyTest.PassportInfo) []byte {
	// 交易结构为:{"from":"","to":"","amount":0},大小为30个字节，通过控制from和to的大小来控制整体交易大小
	s := (size - 30) / 2
	log.Debug().Msgf("size:%d size(from):%d ", size, s)
	// 构造交易
	txs := Transactions{}
	pn := len(passports)
	if pn == 0 {
		for i := 0; i < n; i++ {
			txs.Transactions = append(txs.Transactions, Transaction{
				TxID:      uuid.NewString(),
				From:      randStr(s),
				To:        randStr(s),
				Amount:    rand.Intn(10),
				Signature: "",
			})
		}
	} else {
		id := uuid.NewString()
		from := passports[rand.Intn(pn)]
		to := passports[rand.Intn(pn)]
		amount := rand.Intn(10)

		buf := bytes.NewBuffer([]byte{})
		buf.WriteString(id)
		buf.WriteString(from.Pub)
		buf.WriteString(to.Pub)
		bs := make([]byte, 8)
		binary.BigEndian.PutUint64(bs, uint64(amount))
		buf.Write(bs)

		privateKey, err := base64.StdEncoding.DecodeString(from.Pri)
		if err != nil {
			log.Fatal().Msgf("decode private key failed, err:%v", err)
		}
		ed25519.Sign(privateKey, buf.Bytes())
		signature := ed25519.Sign(privateKey, buf.Bytes())

		for i := 0; i < n; i++ {
			txs.Transactions = append(txs.Transactions, Transaction{
				TxID:      id,
				From:      from.Pub,
				To:        to.Pub,
				Amount:    amount,
				Signature: base64.StdEncoding.EncodeToString(signature),
			})
		}
	}

	bs, err := json.Marshal(txs)
	if err != nil {
		log.Fatal().Msgf("marshal txs failed, err:%s", err.Error())
	}
	return bs
}

func sendTxs(net *easyTest.Net, singleTxSize, singleNodeTxsSize int, passports []easyTest.PassportInfo) {
	totalR := singleNodeTxsSize / maxTxsSize
	rem := singleNodeTxsSize % maxTxsSize
	maxTxsBody := genTxs(singleTxSize, maxTxsSize, passports)
	remTxsBody := genTxs(singleTxSize, rem, passports)

	log.Debug().Msgf("remTxsBody:%s len:%d", string(remTxsBody), len(string(remTxsBody)))

	for r := 0; r < totalR; r++ {
		port := startApiPort
		wg := new(sync.WaitGroup)
		for _, branch := range net.Branches {
			for _, node := range branch.Nodes {
				url := fmt.Sprintf("http://%s:%d/txs", node, port)
				wg.Add(1)
				go sendPost(url, maxTxsBody, wg)
				port++
			}
		}
		wg.Wait()
	}

	if rem != 0 {
		port := startApiPort
		wg := new(sync.WaitGroup)
		for _, branch := range net.Branches {
			for _, node := range branch.Nodes {
				url := fmt.Sprintf("http://%s:%d/txs", node, port)
				wg.Add(1)
				go sendPost(url, remTxsBody, wg)
				port++
			}
		}
		wg.Wait()
	}

	log.Info().Msg("send txs complete")
}

func init() {
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.StampMicro}
	log.Logger = zerolog.New(output).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func main() {
	singleTxSize := flag.Int("b", 65, "single tx size(byte)")
	singleNodeTxsSize := flag.Int("n", 0, "num of txs send to each node")
	createConnect := flag.Bool("c", false, "create connect for each node")
	sendTxsFlag := flag.Bool("t", false, "send txs to each node")
	confPath := flag.String("conf", "./conf.json", "conf.json path")
	passportPath := flag.String("pp", "./allConfig/passports.yml", "passports.yml path")
	flag.Parse()

	log.Info().Msgf("singleNodeTxsSize: %d", *singleNodeTxsSize)

	myNet, err := easyTest.ParseNetConf(*confPath)
	if err != nil {
		log.Fatal().Msgf("解析配置文件失败, err:%s", err.Error())
	}

	log.Debug().Msgf("myNet:%v", myNet)

	var passportConfig easyTest.PassportConfig
	passports := make([]easyTest.PassportInfo, 0)
	bs, err := os.ReadFile(*passportPath)
	if err != nil {
		log.Warn().Msgf("读取passport配置文件失败, err:%s", err.Error())
	} else {
		err = yaml.Unmarshal(bs, &passportConfig)
		if err != nil {
			log.Fatal().Msgf("解析passport配置文件失败, err:%s", err.Error())
		}
		passports = passportConfig.Passports
	}

	// 分片内建立连接
	if *createConnect {
		sendConn(myNet)
	}

	// 发送交易
	if *sendTxsFlag {
		sendTxs(myNet, *singleTxSize, *singleNodeTxsSize, passports)
	}

	//bs := genTxs(64000)
	//txs := Transactions{}
	//err = json.Unmarshal(bs, &txs)
	//if err != nil {
	//	fmt.Println("err:", err)
	//}
	//fmt.Println(txs)
}
