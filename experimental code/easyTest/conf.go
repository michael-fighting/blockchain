package easyTest

import (
	"fmt"
	"github.com/bytedance/sonic"
	"io/ioutil"
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

func ParseNetConf(filename string) (*Net, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println("read conf.json failed, err:", err)
		return nil, err
	}

	var net Net
	err = sonic.Unmarshal(bytes, &net)
	if err != nil {
		fmt.Println("unmarshal net failed, err:", err)
		return nil, err
	}
	return &net, nil
}
