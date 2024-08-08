package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"
)

type Net struct {
	MainChain MChain
	Branches  []MBranch
}

type Setting struct {
	Byzantine     int
	BatchSize     int
	CommitteeSize int
	NetworkSize   int
}

type MChain struct {
	Setting
}

type IP string
type MBranch struct {
	Setting
	Nodes []IP
}

type Identity struct {
	Address         string
	ExternalAddress string
	GskShare        string
	GpkShare        string
	RShare          string
	Gpk             string
	Rpk             string
}

type ShortIdentity struct {
	Address         string
	ExternalAddress string
}

type Dumbo1 struct {
	NetworkSize     int
	Byzantine       int
	BatchSize       int
	CommitteeSize   int
	ProposeInterval time.Duration
}

type Member struct {
	Address  string
	GpkShare string
	RShare   string
	X        string
}

type MemberInfo struct {
	Index    int
	GpkShare string
	RShare   string
	X        string
}

type Tpke struct {
	SecretKeyShare  string
	MasterPublicKey string
	Threshold       int
}

type Branch struct {
	Dumbo1  Dumbo1
	Members []Member
}

type Slice struct {
	NetworkSize int
	Byzantine   int
	Slice       []string
}

type MainChain struct {
	Dumbo1  Dumbo1
	Members []MemberInfo
}

type Config struct {
	BranchIdentity    Identity
	MainChainIdentity Identity
	CtrlIdentity      ShortIdentity
	Branch            Branch
	Slices            []Slice
	CtrlSlices        []Slice
	MainChain         MainChain
	Tpke              Tpke
	MTpke             Tpke
}

func writeConf(confDir, nodeName string, conf Config) {
	yamlConf, err := yaml.Marshal(conf)
	if err != nil {
		log.Fatal().Msgf("序列化成yaml格式错误, err:%s", err.Error())
	}

	nodeDir := path.Join(confDir, nodeName)
	if err := os.Mkdir(nodeDir, 0755); err != nil {
		log.Fatal().Msgf("创建node文件夹失败, err:%s", err.Error())
	}

	cPath := path.Join(nodeDir, "config.yml")
	cFile, err := os.OpenFile(cPath, os.O_CREATE|os.O_RDWR, 0766)
	if err != nil {
		log.Fatal().Msgf("创建config文件失败, err:%s", err.Error())
	}
	_, err = cFile.Write(yamlConf)
	if err != nil {
		cFile.Close()
		log.Fatal().Msgf("写文件失败, err:%s", err.Error())
	}
	cFile.Close()
}
func main() {
	confDir := flag.String("dir", "config", "config dir path")
	addr := flag.String("addr", "127.0.0.1:8090", "gen config server address")
	allConf := flag.Bool("all", false, "get all conf")
	confPath := flag.String("conf", "./conf.json", "conf path")

	flag.Parse()

	if err := os.RemoveAll(*confDir); err != nil {
		log.Fatal().Msgf("删除文件夹失败, err:%s", err.Error())
	}

	if err := os.Mkdir(*confDir, 0755); err != nil {
		log.Fatal().Msgf("创建文件夹失败, err:%s", err.Error())
	}

	if *allConf {
		genConfigSgx(*addr, *confPath)
		getAllConf(*addr, *confDir)
	} else {
		getLocalConf(*addr, *confDir)
	}

}

func genConfigSgx(addr, confPath string) error {
	url := fmt.Sprintf("http://%s/setConf", addr)
	bs, err := ioutil.ReadFile(confPath)
	if err != nil {
		return err
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(bs))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var res map[string]interface{}
	bs, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(bs, &res)
	fmt.Println(string(bs))
	return err
}

func getAllConf(addr, confDir string) {
	url := fmt.Sprintf("http://%s/getAllConf", addr)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal().Msgf("获取配置请求失败, err:%s", err.Error())
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal().Msgf("read body failed, err:%s", err.Error())
	}

	res := map[string]interface{}{}
	err = json.Unmarshal(body, &res)
	if err != nil {
		log.Fatal().Msgf("unmarshal body failed, err:%s", err.Error())
	}

	if res["data"] == nil {
		fmt.Println("no config")
		return
	}

	fmt.Println(res["data"])

	confMap := res["data"].(map[string]interface{})
	for nodeIp, nodeData := range confMap {
		fmt.Println("get", nodeIp, "config")
		data := nodeData.(map[string]interface{})
		for nodeName, confInterface := range data {
			fmt.Println("get", nodeName, "config")
			bytes, _ := json.Marshal(confInterface)
			var conf Config
			json.Unmarshal(bytes, &conf)
			writeConf(confDir, nodeName, conf)
		}
	}
}

func getLocalConf(addr, confDir string) {
	url := fmt.Sprintf("http://%s/getConf", addr)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal().Msgf("获取配置请求失败, err:%s", err.Error())
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal().Msgf("read body failed, err:%s", err.Error())
	}

	res := map[string]interface{}{}
	err = json.Unmarshal(body, &res)
	if err != nil {
		log.Fatal().Msgf("unmarshal body failed, err:%s", err.Error())
	}

	if res["data"] == nil {
		fmt.Println("no config")
		return
	}

	confMap := res["data"].(map[string]interface{})
	for nodeName, confInterface := range confMap {
		fmt.Println("get", nodeName, "config")
		bytes, _ := json.Marshal(confInterface)
		var conf Config
		json.Unmarshal(bytes, &conf)
		writeConf(confDir, nodeName, conf)
	}
}
