package config

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/spf13/viper"
)

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
	Passports         []string
}

// TODO: change default config
var defaultConfig = &Config{}

var once sync.Once

var configPath = os.Args[3]

func Path() string {
	return configPath
}

// 读取指定文件的配置
func Get() *Config {
	once.Do(func() {
		viper.SetConfigFile(configPath)
		if err := viper.ReadInConfig(); err != nil {
			panic("cannot read config")
		}
		err := viper.Unmarshal(&defaultConfig)
		if err != nil {
			panic(fmt.Sprintf("error in read config, err: %s", err))
		}
	})
	return defaultConfig
}
