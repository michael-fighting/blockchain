package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/DE-labtory/cleisthenes/easyTest"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"os"
	"path"
	"time"
)

const Interval time.Duration = 1 * time.Second

func init() {
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.StampMicro}
	log.Logger = zerolog.New(output).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func main() {
	dir := flag.String("dir", "allConfig", "all Config save dir")
	confPath := flag.String("conf", "conf.json", "conf.json file path")
	useSgx := flag.Bool("sgx", false, "use sgx server to gen config")
	addr := flag.String("sgxAddr", "127.0.0.1:8090", "sgxServer")
	pn := flag.Int("pn", 10, "passport num")
	flag.Parse()

	if *useSgx {
		genConfigSgx(*addr, *confPath)
	} else {
		genConfigV2(*dir, *confPath, *pn)
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

func transAddrToIntStr(ip string, port int) string {
	// 将addr转为数字字符串
	i := big.NewInt(0).SetBytes(net.ParseIP(ip).To4()).Int64()
	i <<= 16
	i += int64(port)
	iStr := fmt.Sprintf("%d", i)
	return iStr
}

func newTss(n, f int) *MyTss {
	t, _ := NewMyTss(n, f+1)
	t.SetUp()
	sig, R := t.CombineSig([]byte("hello world"))
	if t.CombineVerify(sig, R, []byte("hello world")) {
		log.Debug().Msg("验签成功")
	} else {
		log.Fatal().Msg("验签失败")
	}
	return t
}

func genPassports(n int) []easyTest.PassportInfo {
	res := make([]easyTest.PassportInfo, 0)
	for i := 0; i < n; i++ {
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			log.Fatal().Msgf("生成账户失败, err:%s", err.Error())
		}

		info := easyTest.PassportInfo{
			Pub: base64.StdEncoding.EncodeToString(publicKey),
			Pri: base64.StdEncoding.EncodeToString(privateKey),
		}

		res = append(res, info)
	}

	return res
}

func genConfigV2(dir, confPath string, pn int) {
	myNet, err := easyTest.ParseNetConf(confPath)
	if err != nil {
		log.Fatal().Msgf("解析配置文件失败, err:%s", err.Error())
	}

	log.Debug().Msgf("解析文件成功, myNet:%v", myNet)

	if err := os.RemoveAll(dir); err != nil {
		log.Fatal().Msgf("删除文件夹失败, err:%s", err.Error())
	}

	if err := os.Mkdir(dir, 0755); err != nil {
		log.Fatal().Msgf("创建文件夹失败, err:%s", err.Error())
	}

	// 生成账户
	passports := genPassports(pn)
	// 写账户文件，供后续发送请求签名使用
	writePassportConf(dir, easyTest.PassportConfig{Passports: passports})
	// 获取账户的所有公钥保存到pubs
	pubs := make([]string, 0)
	for _, passport := range passports {
		pubs = append(pubs, passport.Pub)
	}

	sliceNum := len(myNet.Branches)
	dn := myNet.MainChain.NetworkSize
	if dn == 0 {
		dn = sliceNum
	}
	df := myNet.MainChain.Byzantine
	db := myNet.MainChain.BatchSize
	dc := myNet.MainChain.CommitteeSize

	gap := (sliceNum + dn - 1) / dn

	dt := newTss(dn, df)
	mtp := NewMyTpke(df)
	allBranchPortStart := 15000
	mainChainPortStart := 16000
	ctrlPortStart := 17000

	branchPortStart := allBranchPortStart
	mainChainPort := mainChainPortStart
	ctrlPort := ctrlPortStart
	allNodeIndex := 0

	for branchIndex := 0; branchIndex < sliceNum; branchIndex++ {
		branchNet := myNet.Branches[branchIndex]
		bn := len(branchNet.Nodes)
		bf := branchNet.Byzantine
		bb := branchNet.BatchSize
		bc := branchNet.CommitteeSize

		bt := newTss(bn, bf)
		tp := NewMyTpke(bf)

		for nodeIndex := 0; nodeIndex < bn; nodeIndex++ {
			port := branchPortStart + 2*nodeIndex
			conf := easyTest.Config{}
			nodeIp := branchNet.Nodes[nodeIndex]

			// BranchIdentity init
			identity := conf.BranchIdentity
			identity.Address = fmt.Sprintf("%s:%d", nodeIp, port)
			identity.ExternalAddress = fmt.Sprintf("%s:%d", nodeIp, port)
			identity.Gpk = hexutil.Encode(bt.gpk.Marshal())
			identity.Rpk = hexutil.Encode(bt.rpk.Marshal())
			identity.GpkShare = hexutil.Encode(bt.gpkshare[nodeIndex].Marshal())
			identity.GskShare = hexutil.Encode(bt.gskshare[nodeIndex].Bytes())
			identity.RShare = hexutil.Encode(bt.rshare[nodeIndex].Bytes())
			conf.BranchIdentity = identity

			// Branch init
			dumbo1 := conf.Branch.Dumbo1
			dumbo1.NetworkSize = bn
			dumbo1.Byzantine = bf
			dumbo1.BatchSize = bb
			dumbo1.CommitteeSize = bc
			dumbo1.ProposeInterval = Interval
			conf.Branch.Dumbo1 = dumbo1

			members := conf.Branch.Members
			for i := 0; i < bn; i++ {
				memberIp := branchNet.Nodes[i]
				memberPort := branchPortStart + 2*i
				member := easyTest.Member{
					Address:  fmt.Sprintf("%s:%d", memberIp, memberPort),
					GpkShare: hexutil.Encode(bt.gpkshare[i].Marshal()),
					RShare:   hexutil.Encode(bt.rshare[i].Bytes()),
					X:        hexutil.Encode(bt.x[i].Bytes()),
				}
				members = append(members, member)
			}
			conf.Branch.Members = members

			// MainChainIdentity init
			delegateIdentity := conf.MainChainIdentity
			delegateIdentity.Address = fmt.Sprintf("%s:%d", nodeIp, mainChainPort)
			delegateIdentity.ExternalAddress = fmt.Sprintf("%s:%d", nodeIp, mainChainPort)
			delegateIdentity.Gpk = hexutil.Encode(dt.gpk.Marshal())
			delegateIdentity.Rpk = hexutil.Encode(dt.rpk.Marshal())
			delegateIdentity.GpkShare = hexutil.Encode(dt.gpkshare[branchIndex/gap].Marshal())
			delegateIdentity.GskShare = hexutil.Encode(dt.gskshare[branchIndex/gap].Bytes())
			delegateIdentity.RShare = hexutil.Encode(dt.rshare[branchIndex/gap].Bytes())
			conf.MainChainIdentity = delegateIdentity

			// MainChain init
			delegateDumbo1 := conf.MainChain.Dumbo1
			delegateDumbo1.NetworkSize = dn
			delegateDumbo1.Byzantine = df
			delegateDumbo1.BatchSize = db
			delegateDumbo1.CommitteeSize = dc
			delegateDumbo1.ProposeInterval = Interval
			conf.MainChain.Dumbo1 = delegateDumbo1

			delegateMembers := conf.MainChain.Members
			for i := 0; i < dn; i++ {
				delegateMember := easyTest.MemberInfo{
					Index:    i,
					GpkShare: hexutil.Encode(dt.gpkshare[i].Marshal()),
					RShare:   hexutil.Encode(dt.rshare[i].Bytes()),
					X:        hexutil.Encode(dt.x[i].Bytes()),
				}
				delegateMembers = append(delegateMembers, delegateMember)
			}
			conf.MainChain.Members = delegateMembers

			// ctrl Identity
			conf.CtrlIdentity.Address = fmt.Sprintf("%s:%d", nodeIp, ctrlPort)
			conf.CtrlIdentity.ExternalAddress = fmt.Sprintf("%s:%d", nodeIp, ctrlPort)

			// Tpke init, no use
			tpke := conf.Tpke
			mpkBytes := tp.SecretKeySet.PublicKeySet().Serialize()
			tpke.MasterPublicKey = hexutil.Encode(mpkBytes[:])
			// 将addr转为数字字符串
			iStr := transAddrToIntStr(string(nodeIp), port)
			sksBytes := tp.SecretKeySet.KeyShareUsingString(iStr).Serialize()
			tpke.SecretKeyShare = hexutil.Encode(sksBytes[:])
			tpke.Threshold = bf
			conf.Tpke = tpke

			mTpke := conf.MTpke
			mpkBytes = mtp.SecretKeySet.PublicKeySet().Serialize()
			mTpke.MasterPublicKey = hexutil.Encode(mpkBytes[:])
			// 将addr转为数字字符串
			// FIXME 主链的代表节点变化，那么主链的tpke会存在问题，需要对mTpke做处理
			iStr = transAddrToIntStr(string(nodeIp), mainChainPort)
			sksBytes = mtp.SecretKeySet.KeyShareUsingString(iStr).Serialize()
			mTpke.SecretKeyShare = hexutil.Encode(sksBytes[:])
			mTpke.Threshold = df
			conf.MTpke = mTpke

			// Passports
			conf.Passports = pubs

			// Slices
			slices := make([]easyTest.Slice, 0)
			p := mainChainPortStart
			for i := 0; i < sliceNum; i++ {
				tmpbn := len(myNet.Branches[i].Nodes)
				var slice easyTest.Slice
				for j := 0; j < tmpbn; j++ {
					slice.Slice = append(slice.Slice, fmt.Sprintf("%s:%d", myNet.Branches[i].Nodes[j], p))
					p += 2
				}
				slice.NetworkSize = tmpbn
				slice.Byzantine = bf
				slices = append(slices, slice)
			}
			conf.Slices = slices

			// CtrlSlices
			ctrlSlices := make([]easyTest.Slice, 0)
			p = ctrlPortStart
			for i := 0; i < sliceNum; i++ {
				tmpbn := len(myNet.Branches[i].Nodes)
				var slice easyTest.Slice
				for j := 0; j < tmpbn; j++ {
					slice.Slice = append(slice.Slice, fmt.Sprintf("%s:%d", myNet.Branches[i].Nodes[j], p))
					p += 2
				}
				slice.NetworkSize = tmpbn
				slice.Byzantine = bf
				ctrlSlices = append(ctrlSlices, slice)
			}
			conf.CtrlSlices = ctrlSlices

			log.Debug().Msgf("正在生成node%d的配置", allNodeIndex)
			writeConf(dir, allNodeIndex, conf)

			mainChainPort += 2
			ctrlPort += 2
			allNodeIndex++
		}

		// LAST
		branchPortStart += 2 * bn
	}
	log.Info().Msg("生成配置完成")
}

func writeConf(dir string, nodeIndex int, conf easyTest.Config) {
	yamlConf, err := yaml.Marshal(conf)
	if err != nil {
		log.Fatal().Msgf("序列化成yaml格式错误, err:%s", err.Error())
	}

	nodeDir := path.Join(dir, fmt.Sprintf("node%d", nodeIndex))
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

func writePassportConf(dir string, conf easyTest.PassportConfig) {
	return
	yamlConf, err := yaml.Marshal(conf)
	if err != nil {
		log.Fatal().Msgf("序列化成yaml格式错误, err:%s", err.Error())
	}
	cPath := path.Join(dir, "passports.yml")
	cFile, err := os.OpenFile(cPath, os.O_CREATE|os.O_RDWR, 0766)
	if err != nil {
		log.Fatal().Msgf("创建passport config文件失败, err:%s", err.Error())
	}
	_, err = cFile.Write(yamlConf)
	if err != nil {
		cFile.Close()
		log.Fatal().Msgf("写文件失败, err:%s", err.Error())
	}
	cFile.Close()
}
