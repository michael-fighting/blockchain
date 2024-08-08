package tss

import (
	"bytes"
	"fmt"
	"github.com/DE-labtory/cleisthenes"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	bn256 "github.com/ethereum/go-ethereum/crypto/bn256/cloudflare"
	"math/big"
)

type Config struct {
	threshold   int
	participant int
}

type DefaultTss struct {
	n           int
	t           int
	gpk         bn256.G1
	rpk         bn256.G1
	gpkshare    bn256.G1
	gskshare    big.Int
	rshare      big.Int
	gpkshareMap map[cleisthenes.Address]bn256.G1
	rshareMap   map[cleisthenes.Address]big.Int
	xMap        map[cleisthenes.Address]big.Int
}

func NewDefaultTss(
	n, t int,
	gpk, rpk, gpkshare, gskshare, rshare string,
	gpkshareMap, rshareMap, xMap map[cleisthenes.Address]string) (cleisthenes.Tss, error) {

	_gpk := bn256.G1{}
	_, err := _gpk.Unmarshal(hexutil.MustDecode(gpk))
	if err != nil {
		return nil, err
	}

	_rpk := bn256.G1{}
	_, err = _rpk.Unmarshal(hexutil.MustDecode(rpk))
	if err != nil {
		return nil, err
	}

	_gpkshare := bn256.G1{}
	_, err = _gpkshare.Unmarshal(hexutil.MustDecode(gpkshare))
	if err != nil {
		return nil, err
	}

	_gskshare := big.Int{}
	_gskshare.SetBytes(hexutil.MustDecode(gskshare))
	// mustDecodeBig最高只能解析256bit的big.Int，会有问题
	//hexutil.MustDecodeBig(gskshare)
	_rshare := big.Int{}
	_rshare.SetBytes(hexutil.MustDecode(rshare))
	//hexutil.MustDecodeBig(rshare)

	_gpkshareMap := make(map[cleisthenes.Address]bn256.G1)
	for addr, val := range gpkshareMap {
		g := bn256.G1{}
		_, err = g.Unmarshal(hexutil.MustDecode(val))
		if err != nil {
			return nil, err
		}
		_gpkshareMap[addr] = g
	}

	_rshareMap := make(map[cleisthenes.Address]big.Int)
	for addr, val := range rshareMap {
		r := big.Int{}
		r.SetBytes(hexutil.MustDecode(val))
		//hexutil.MustDecodeBig(val)
		_rshareMap[addr] = r
	}

	_xMap := make(map[cleisthenes.Address]big.Int)
	for addr, val := range xMap {
		r := big.Int{}
		r.SetBytes(hexutil.MustDecode(val))
		_xMap[addr] = r
	}

	tss := &DefaultTss{
		n:           n,
		t:           t,
		gpk:         _gpk,
		rpk:         _rpk,
		gpkshare:    _gpkshare,
		gskshare:    _gskshare,
		rshare:      _rshare,
		gpkshareMap: _gpkshareMap,
		rshareMap:   _rshareMap,
		xMap:        _xMap,
	}

	return tss, nil
}

func (t *DefaultTss) Sign(data []byte) []byte {
	M := crypto.Keccak256(data)
	m := big.Int{}
	m.SetBytes(M)

	sig := SchnorrSign(t.gskshare, t.rshare, m)
	return []byte(hexutil.Encode(sig.Bytes()))
}

func (t *DefaultTss) CombineSig(sigs map[cleisthenes.Address][]byte) *big.Int {
	sigShares := make([]big.Int, 0)
	x := make([]big.Int, 0)
	for addr, bytes := range sigs {
		sigShare := big.Int{}
		sigShare.SetBytes(hexutil.MustDecode(string(bytes)))
		sigShares = append(sigShares, sigShare)
		x = append(x, t.xMap[addr])
	}

	//fmt.Println("len sigshares:", len(sigShares), "len x", len(x), "degree:", t.t)
	ss := Lagrange(sigShares, x, t.t)
	return &ss
}

func (t *DefaultTss) VerifyCombineSig(ss *big.Int, data []byte) bool {
	var buffer bytes.Buffer
	buffer.Write(data)
	M := crypto.Keccak256(buffer.Bytes())
	m := new(big.Int).SetBytes(M)

	ssG := new(bn256.G1).ScalarBaseMult(ss)

	mgpk := new(bn256.G1).ScalarMult(&t.gpk, m)

	temp := new(bn256.G1).Add(mgpk, &t.rpk)

	return CompareG1(*ssG, *temp)

	//return true
}

func (t *DefaultTss) Verify(address cleisthenes.Address, sig []byte, data []byte) bool {
	M := crypto.Keccak256(data)
	m := big.Int{}
	m.SetBytes(M)
	sigshare := big.Int{}
	sigshare.SetBytes(hexutil.MustDecode(string(sig)))

	gpkshare, ok := t.gpkshareMap[address]
	if !ok {
		fmt.Println("gpkshare not found for addr - ", address)
		return false
	}
	rshare, ok := t.rshareMap[address]
	if !ok {
		fmt.Println("rshare not found for addr - ", address)
		return false
	}

	return SchnorrVerify(gpkshare, rshare, sigshare, m)
}
