package main

import (
	apiserver "Dapp/api_server"
	db2 "Dapp/db"
	"Dapp/processor"
	serverconfig "Dapp/server_config"
	"flag"
	"fmt"
	"os"
)

func main() {
	var defaultConfigPath string
	//读取配置文件
	flag.StringVar(&defaultConfigPath, "i", "./config_files/config.yml", "config file")
	flag.Parse()
	configErr := serverconfig.ReadConfigFile(defaultConfigPath) //配置
	if configErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "load cfg err %s \n", configErr.Error())
		return
	}
	// 初始化数据库todo
	db2.InitDB()

	// 加载sdk，client模块

	voteProcessor := processor.InitProcessor()

	//使用协程订阅项目发布合约事件和投票事件
	//var ginContext *gin.Context
	//go apiserver.IssueSubject(voteProcessor, ginContext)
	//go apiserver.Vote(voteProcessor, ginContext)

	httpSrv := apiserver.NewApiServer(voteProcessor)
	httpSrv.Listen()
}
