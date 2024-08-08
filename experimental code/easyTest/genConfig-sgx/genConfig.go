package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"github.com/DE-labtory/tpke"
	"github.com/bytedance/sonic"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	bn256 "github.com/ethereum/go-ethereum/crypto/bn256/cloudflare"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"time"
)

const Interval time.Duration = 0 * time.Second

type MyTss struct {
	Nstm     int
	Thres    int
	Degree   int
	Pubkey   []bn256.G1
	Prikey   []big.Int
	x        []big.Int
	s        []*big.Int
	gpk      bn256.G1
	gskshare []*big.Int
	gpkshare []*bn256.G1
	rshare   []*big.Int
	rpk      bn256.G1
}

type MyTpke struct {
	threshold    int
	SecretKeySet *tpke.SecretKeySet
}

func NewMyTpke(t int) *MyTpke {
	secretKeySet := tpke.RandomSecretKeySet(t)

	return &MyTpke{
		threshold:    t,
		SecretKeySet: secretKeySet,
	}
}

func NewMyTss(N, T int) (*MyTss, error) {
	t := &MyTss{
		Nstm:     N,
		Thres:    T,
		Degree:   T - 1,
		Pubkey:   make([]bn256.G1, N),
		Prikey:   make([]big.Int, N),
		s:        make([]*big.Int, N),
		x:        make([]big.Int, N),
		gskshare: make([]*big.Int, N),
		gpkshare: make([]*bn256.G1, N),
		rshare:   make([]*big.Int, N),
	}

	for i := 0; i < t.Nstm; i++ {
		Pri, Pub, err := bn256.RandomG1(rand.Reader)
		if err != nil {
			return nil, err
		}
		t.Prikey[i] = *Pri
		t.Pubkey[i] = *Pub
	}

	for i := 0; i < t.Nstm; i++ {
		t.x[i].SetBytes(crypto.Keccak256(t.Pubkey[i].Marshal()))
		t.x[i].Mod(&t.x[i], bn256.Order)
	}

	return t, nil
}

func (t *MyTss) SetUp() {
	for i := 0; i < t.Nstm; i++ {
		t.s[i], _ = rand.Int(rand.Reader, bn256.Order)
	}

	// Each storeman node conducts the shamir secret sharing process
	poly := make([]Polynomial, t.Nstm)

	sshare := make([][]big.Int, t.Nstm)
	for i := 0; i < t.Nstm; i++ {
		sshare[i] = make([]big.Int, t.Nstm)
	}

	for i := 0; i < t.Nstm; i++ {
		poly[i] = RandPoly(t.Degree, *t.s[i]) // fi(x), set si as its constant term
		for j := 0; j < t.Nstm; j++ {
			sshare[i][j] = EvaluatePoly(poly[i], &t.x[j], t.Degree) // share for j is fi(x) evaluation result on x[j]=Hash(Pub[j])
		}
	}

	// every storeman node sends the secret shares to other nodes in secret!
	// Attention! IN SECRET!

	// After reveiving the secret shares, each node computes its group private key share

	for i := 0; i < t.Nstm; i++ {
		t.gskshare[i] = big.NewInt(0)
		for j := 0; j < t.Nstm; j++ {
			t.gskshare[i].Add(t.gskshare[i], &sshare[j][i])
		}
	}

	// Each storeman node publishs the scalar point of its group private key share

	for i := 0; i < t.Nstm; i++ {
		t.gpkshare[i] = new(bn256.G1).ScalarBaseMult(t.gskshare[i])
	}

	// Each storeman node computes the group public key by Lagrange's polynomial interpolation

	t.gpk = LagrangeECC(t.gpkshare, t.x, t.Degree)
}

func (t *MyTss) PrintAll() {
	fmt.Println("gpk", hexutil.Encode(t.gpk.Marshal()))
	fmt.Println("rpk", hexutil.Encode(t.rpk.Marshal()))
	for i := 0; i < t.Nstm; i++ {
		fmt.Println(i, "gpkshare:", hexutil.Encode(t.gpkshare[i].Marshal()), "gskshare:", hexutil.EncodeBig(t.gskshare[i]), "rshare:", hexutil.EncodeBig(t.rshare[i]))
	}
}

func GenerateRShare() {

}

func (t *MyTss) ShareSig(data []byte, gskshare *big.Int, rshare *big.Int) *big.Int {
	var buffer bytes.Buffer
	buffer.Write(data)
	M := crypto.Keccak256(buffer.Bytes())
	m := new(big.Int).SetBytes(M)

	sig := SchnorrSign(*gskshare, *rshare, *m)
	return &sig
}

func (t *MyTss) ShareSigVerify(gpkshare *bn256.G1, rshare *big.Int, sigshare *big.Int, data []byte) bool {
	var buffer bytes.Buffer
	buffer.Write(data)
	M := crypto.Keccak256(buffer.Bytes())
	m := new(big.Int).SetBytes(M)
	return SchnorrVerify(*gpkshare, *rshare, *sigshare, *m)
}

func (t *MyTss) CombineSig(data []byte) (*big.Int, *bn256.G1) {
	// 1st step: each storeman node decides a random number r using shamir secret sharing

	rr := make([]*big.Int, t.Nstm)

	for i := 0; i < t.Nstm; i++ {
		rr[i], _ = rand.Int(rand.Reader, bn256.Order)
	}

	poly1 := make([]Polynomial, t.Nstm)

	rrshare := make([][]big.Int, t.Nstm)
	for i := 0; i < t.Nstm; i++ {
		rrshare[i] = make([]big.Int, t.Nstm)
	}

	for i := 0; i < t.Nstm; i++ {
		poly1[i] = RandPoly(t.Degree, *t.s[i]) // fi(x), set si as its constant term
		for j := 0; j < t.Nstm; j++ {
			rrshare[i][j] = EvaluatePoly(poly1[i], &t.x[j], t.Degree) // share for j is fi(x) evaluation result on x[j]=Hash(Pub[j])
		}
	}

	// every storeman node sends the secret shares to other nodes in secret!
	// Attention! IN SECRET!

	for i := 0; i < t.Nstm; i++ {
		t.rshare[i] = big.NewInt(0)
		for j := 0; j < t.Nstm; j++ {
			t.rshare[i].Add(t.rshare[i], &rrshare[j][i])
		}
	}

	// Compute the scalar point of r
	rpkshare := make([]*bn256.G1, t.Nstm)

	for i := 0; i < t.Nstm; i++ {
		rpkshare[i] = new(bn256.G1).ScalarBaseMult(t.rshare[i])
	}

	t.rpk = LagrangeECC(rpkshare, t.x, t.Degree)

	// Forming the m: hash(message||rpk)
	var buffer bytes.Buffer
	buffer.Write(data)
	//buffer.Write(rpk.Marshal())

	M := crypto.Keccak256(buffer.Bytes())
	m := new(big.Int).SetBytes(M)

	// Each storeman node computes the signature share
	sigshare := make([]big.Int, t.Nstm)

	for i := 0; i < t.Thres; i++ {
		sigshare[i] = SchnorrSign(*t.gskshare[i], *t.rshare[i], *m)
	}

	// Compute the signature using Lagrange's polynomial interpolation

	ss := Lagrange(sigshare, t.x, t.Degree)

	return &ss, &t.rpk
}

func (t *MyTss) CombineVerify(ss *big.Int, rpk *bn256.G1, data []byte) bool {

	var buffer bytes.Buffer
	buffer.Write(data)
	M := crypto.Keccak256(buffer.Bytes())
	m := new(big.Int).SetBytes(M)

	ssG := new(bn256.G1).ScalarBaseMult(ss)

	mgpk := new(bn256.G1).ScalarMult(&t.gpk, m)

	temp := new(bn256.G1).Add(mgpk, rpk)

	return CompareG1(*ssG, *temp)
}

var bigZero = big.NewInt(0)

var bigOne = big.NewInt(1)

// Generator of ECC
var gbase = new(bn256.G1).ScalarBaseMult(big.NewInt(int64(1)))

// Structure defination for polynomial
type Polynomial []big.Int

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

func RandPoly(degree int, constant big.Int) Polynomial {

	poly := make(Polynomial, degree+1)

	poly[0].Mod(&constant, bn256.Order)

	for i := 1; i < degree+1; i++ {

		temp, _ := rand.Int(rand.Reader, bn256.Order)

		// in case of polynomial degenerating
		poly[i] = *temp.Add(temp, bigOne)
	}
	return poly
}

// Calculate polynomial's evaluation at some point
func EvaluatePoly(f Polynomial, x *big.Int, degree int) big.Int {

	sum := big.NewInt(0)

	for i := 0; i < degree+1; i++ {

		temp1 := new(big.Int).Exp(x, big.NewInt(int64(i)), bn256.Order)

		temp1.Mod(temp1, bn256.Order)

		temp2 := new(big.Int).Mul(&f[i], temp1)

		temp2.Mod(temp2, bn256.Order)

		sum.Add(sum, temp2)

		sum.Mod(sum, bn256.Order)
	}
	return *sum
}

// Calculate the b coefficient in Lagrange's polynomial interpolation algorithm

func evaluateB(x []big.Int, degree int) []big.Int {

	//k := len(x)

	k := degree + 1

	b := make([]big.Int, k)

	for i := 0; i < k; i++ {
		b[i] = evaluateb(x, i, degree)
	}
	return b
}

// sub-function for evaluateB

func evaluateb(x []big.Int, i int, degree int) big.Int {

	//k := len(x)

	k := degree + 1

	sum := big.NewInt(1)

	for j := 0; j < k; j++ {

		if j != i {

			temp1 := new(big.Int).Sub(&x[j], &x[i])

			temp1.ModInverse(temp1, bn256.Order)

			temp2 := new(big.Int).Mul(&x[j], temp1)

			sum.Mul(sum, temp2)

			sum.Mod(sum, bn256.Order)

		} else {
			continue
		}
	}
	return *sum
}

// Lagrange's polynomial interpolation algorithm: working in ECC points
func LagrangeECC(sig []*bn256.G1, x []big.Int, degree int) bn256.G1 {

	b := evaluateB(x, degree)

	sum := new(bn256.G1).ScalarBaseMult(big.NewInt(int64(0)))

	for i := 0; i < degree+1; i++ {
		temp := new(bn256.G1).ScalarMult(sig[i], &b[i])
		sum.Add(sum, temp)
	}
	return *sum
}

func SchnorrSign(psk big.Int, r big.Int, m big.Int) big.Int {
	sum := big.NewInt(1)
	sum.Mul(&psk, &m)
	sum.Mod(sum, bn256.Order)
	sum.Add(sum, &r)
	sum.Mod(sum, bn256.Order)
	return *sum
}

func SchnorrVerify(pub bn256.G1, r big.Int, sig big.Int, m big.Int) bool {
	left := new(bn256.G1).ScalarBaseMult(&sig)
	R := new(bn256.G1).ScalarBaseMult(&r)
	right := new(bn256.G1).ScalarMult(&pub, &m)
	right = new(bn256.G1).Add(right, R)
	return CompareG1(*left, *right)
}

// Lagrange's polynomial interpolation algorithm
func Lagrange(f []big.Int, x []big.Int, degree int) big.Int {

	b := evaluateB(x, degree)

	s := big.NewInt(0)

	for i := 0; i < degree+1; i++ {

		temp1 := new(big.Int).Mul(&f[i], &b[i])

		s.Add(s, temp1)

		s.Mod(s, bn256.Order)
	}
	return *s
}

// The comparison function of G1
func CompareG1(a bn256.G1, b bn256.G1) bool {
	return a.String() == b.String()
}

func init() {
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.StampMicro}
	log.Logger = zerolog.New(output).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
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

var MyNet Net
var ConfMap = make(map[string]map[string]Config)

func GenConfigV2() {

	ConfMap = make(map[string]map[string]Config)

	sliceNum := len(MyNet.Branches)
	dn := MyNet.MainChain.NetworkSize
	if dn == 0 {
		dn = sliceNum
	}
	df := MyNet.MainChain.Byzantine
	db := MyNet.MainChain.BatchSize
	dc := MyNet.MainChain.CommitteeSize

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
		branchNet := MyNet.Branches[branchIndex]
		bn := len(branchNet.Nodes)
		bf := branchNet.Byzantine
		bb := branchNet.BatchSize
		bc := branchNet.CommitteeSize

		bt := newTss(bn, bf)
		tp := NewMyTpke(bf)

		for nodeIndex := 0; nodeIndex < bn; nodeIndex++ {
			port := branchPortStart + 2*nodeIndex
			conf := Config{}
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
				member := Member{
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
				delegateMember := MemberInfo{
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

			// Slices
			slices := make([]Slice, 0)
			p := mainChainPortStart
			for i := 0; i < sliceNum; i++ {
				tmpbn := len(MyNet.Branches[i].Nodes)
				var slice Slice
				for j := 0; j < tmpbn; j++ {
					slice.Slice = append(slice.Slice, fmt.Sprintf("%s:%d", MyNet.Branches[i].Nodes[j], p))
					p += 2
				}
				slice.NetworkSize = tmpbn
				slice.Byzantine = bf
				slices = append(slices, slice)
			}
			conf.Slices = slices

			// CtrlSlices
			ctrlSlices := make([]Slice, 0)
			p = ctrlPortStart
			for i := 0; i < sliceNum; i++ {
				tmpbn := len(MyNet.Branches[i].Nodes)
				var slice Slice
				for j := 0; j < tmpbn; j++ {
					slice.Slice = append(slice.Slice, fmt.Sprintf("%s:%d", MyNet.Branches[i].Nodes[j], p))
					p += 2
				}
				slice.NetworkSize = tmpbn
				slice.Byzantine = bf
				ctrlSlices = append(ctrlSlices, slice)
			}
			conf.CtrlSlices = ctrlSlices

			log.Debug().Msgf("正在生成node%d的配置", allNodeIndex)
			if ConfMap[string(nodeIp)] == nil {
				ConfMap[string(nodeIp)] = map[string]Config{
					fmt.Sprintf("node%d", allNodeIndex): conf,
				}
			} else {
				ConfMap[string(nodeIp)][fmt.Sprintf("node%d", allNodeIndex)] = conf
			}
			//writeConf(dir, allNodeIndex, conf)

			mainChainPort += 2
			ctrlPort += 2
			allNodeIndex++
		}

		// LAST
		branchPortStart += 2 * bn
	}
	log.Info().Msg("生成配置完成")
}
