package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"github.com/DE-labtory/tpke"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	bn256 "github.com/ethereum/go-ethereum/crypto/bn256/cloudflare"
	"math/big"
)

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
