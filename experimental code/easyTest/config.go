package easyTest

import (
	"time"
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

type PassportInfo struct {
	Pub string
	Pri string
}

type PassportConfig struct {
	Passports []PassportInfo
}
