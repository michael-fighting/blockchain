package main

import (
	"fmt"
	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/config"
	"github.com/DE-labtory/cleisthenes/core"
	"github.com/DE-labtory/cleisthenes/example/app"
	"github.com/DE-labtory/iLogger"
	kitlog "github.com/go-kit/kit/log"
	"github.com/rs/zerolog"
	"net/http"
	"runtime"
	"strconv"

	_ "net/http/pprof"
	"os"
	"path/filepath"
)

var (
	log zerolog.Logger
)

func init() {
	log = cleisthenes.NewLogger()
}

func main() {
	runtime.GOMAXPROCS(2)
	//var maxMemSize int64 = 4 * 1024 * 1024 * 1024
	//host := flag.String("address", "127.0.0.2", "Application address")
	//port := flag.Int("port", 8000, "Application port")
	//configPath := flag.String("config", "", "User defined config path")
	//flag.Parse()
	fmt.Println(os.Args)
	// api监听端口，人机交互输入的接口
	host := os.Args[1]
	port := os.Args[2]
	// node 节点配置
	configPath := os.Args[3]

	// 开启pprof，监听请求
	go func() {
		iport, _ := strconv.Atoi(port)
		ip := fmt.Sprintf("%s:%d", host, iport+1000)
		if err := http.ListenAndServe(ip, nil); err != nil {
			fmt.Printf("start pprof failed on %s\n", ip)
			os.Exit(1)
		}
	}()

	//初始化日志文件路径
	os.RemoveAll("./log")
	absPath, _ := filepath.Abs("./log/cleisthenes" + port + ".log")
	defer os.RemoveAll("./log")
	err := iLogger.EnableFileLogger(true, absPath)

	address := fmt.Sprintf("%s:%s", host, port)

	kitLogger := kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stderr))
	kitLogger = kitlog.With(kitLogger, "ts", kitlog.DefaultTimestampUTC) //DefaultTimestampUTC返回现在的世界时
	httpLogger := kitlog.With(kitLogger, "component", "http")

	//将defaultConfig的默认配置文件写到tempConfig/node/config.yml
	config.Init(configPath)

	txValidator := func(tx cleisthenes.Transaction) bool {
		// custom transaction validation logic，交易验证逻辑
		return true
	}

	// new sliceSystem, 处理branch和mainchain逻辑
	sliceSystem, err := core.NewSliceSystem(txValidator)
	if err != nil {
		panic(fmt.Sprintf("Cleisthenes instantiate failed with err: %s", err))
	}

	// tps
	cleisthenes.BranchTps.ListenResult("[BRANCH_TPS]")
	cleisthenes.MainTps.ListenResult("[MAINCHAIN_TPS]")

	//// branch node节点的输出， 结果展示
	//runBranchNodeAndListenResult(sliceSystem.GetBranch(), sliceSystem.GetMainChain())
	//
	//// 代表节点的输出，结果展示
	//if sliceSystem.GetMainChain() != nil {
	//	runDelegateNodeAndListenResult(sliceSystem.GetMainChain())
	//}

	// api的配置
	log.Info().Msgf("http server started: %v", address)
	if err := http.ListenAndServe(address, app.NewApiHandler(sliceSystem, httpLogger)); err != nil {
		log.Fatal().Msgf("http server closed:%s", err.Error())
	}

}
