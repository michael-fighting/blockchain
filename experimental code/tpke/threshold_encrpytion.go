package tpke

import (
	"fmt"
	"github.com/DE-labtory/cleisthenes"
	tpk "github.com/DE-labtory/tpke"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"
	"net"
)

type Config struct {
	threshold   int
	participant int
}

type DefaultTpke struct {
	threshold    int //
	publicKey    *tpk.PublicKey
	publicKeySet *tpk.PublicKeySet
	secretKey    *tpk.SecretKeyShare
	decShares    map[string]*tpk.DecryptionShare
}

func NewDefaultTpke(th int, skStr cleisthenes.SecretKey, pksStr cleisthenes.PublicKey) (*DefaultTpke, error) {
	sk := tpk.NewSecretKeyFromBytes(skStr)
	sks := tpk.NewSecretKeyShare(sk)

	pks, err := tpk.NewPublicKeySetFromBytes(pksStr)
	if err != nil {
		return nil, err
	}

	return &DefaultTpke{
		threshold:    th,
		publicKeySet: pks,
		publicKey:    pks.PublicKey(),
		secretKey:    sks,
		decShares:    make(map[string]*tpk.DecryptionShare),
	}, nil
}

func (t *DefaultTpke) Copy() *DefaultTpke {
	return &DefaultTpke{
		threshold:    t.threshold,
		publicKey:    t.publicKey,
		publicKeySet: t.publicKeySet,
		secretKey:    t.secretKey,
		decShares:    make(map[string]*tpk.DecryptionShare),
	}
}

func (t *DefaultTpke) AcceptDecShare(addr cleisthenes.Address, decShare cleisthenes.DecryptionShare) {
	ds := tpk.NewDecryptionShareFromBytes(decShare)
	// 将addr转为数字字符串
	i := big.NewInt(0).SetBytes(net.ParseIP(addr.Ip).To4()).Int64()
	i <<= 16
	i += int64(addr.Port)
	iStr := fmt.Sprintf("%d", i)
	t.decShares[iStr] = ds
}

func (t *DefaultTpke) ClearDecShare() {
	t.decShares = make(map[string]*tpk.DecryptionShare)
}

// Encrypt encrypts some byte array message.
func (t *DefaultTpke) Encrypt(msg []byte) ([]byte, error) {
	encrypted, err := t.publicKey.Encrypt(msg)
	if err != nil {
		return nil, err
	}
	return encrypted.Serialize(), nil
}

// DecShare makes decryption share using each secret key.
func (t *DefaultTpke) DecShare(ctb cleisthenes.CipherText) cleisthenes.DecryptionShare {
	ct := tpk.NewCipherTextFromBytes(ctb)
	ds := t.secretKey.DecryptShare(ct)
	return ds.Serialize()
}

// Decrypt collects decryption share, and combine it for decryption.
//func (t *DefaultTpke) Decrypt(decShares map[string]cleisthenes.DecryptionShare, ctBytes []byte) ([]byte, error) {
//	ct := tpk.NewCipherTextFromBytes(ctBytes)
//	ds := make(map[string]*tpk.DecryptionShare)
//	for id, decShare := range decShares {
//		ds[id] = tpk.NewDecryptionShareFromBytes(decShare)
//	}
//	return t.publicKeySet.DecryptUsingStringMap(ds, ct)
//}

func (t *DefaultTpke) Decrypt(ctBytes []byte) ([]byte, error) {
	ct := tpk.NewCipherTextFromBytes(ctBytes)
	return t.publicKeySet.DecryptUsingStringMap(t.decShares, ct)
}

func (t *DefaultTpke) Sign(msg []byte) (cleisthenes.SignatureShare, error) {
	encrypted, err := t.publicKey.Encrypt(msg)
	if err != nil {
		return cleisthenes.SignatureShare{}, err
	}
	ds := t.secretKey.DecryptShare(encrypted)
	dss := ds.Serialize()
	pkss := t.publicKey.Serialize()
	fmt.Println("信息:", string(msg), "密钥:", t.secretKey.String(), "公钥:", hexutil.Encode(pkss[:]), "密文:", encrypted.String(), "签名结果:", hexutil.Encode(dss[:]))
	return ds.Serialize(), nil
}

// TODO 门限签名
func (t *DefaultTpke) Combine(sig map[cleisthenes.Address]cleisthenes.SignatureShare) {
}

func (t *DefaultTpke) CombineVerify(sigs map[cleisthenes.Address]cleisthenes.SignatureShare, msg []byte) bool {
	t.ClearDecShare()
	for addr, sig := range sigs {
		t.AcceptDecShare(addr, cleisthenes.DecryptionShare(sig))
	}
	ct, err := t.publicKey.Encrypt(msg)
	if err != nil {
		fmt.Println("ERROR, cannot encrypt msg")
		return false
	}
	decMsg, err := t.publicKeySet.DecryptUsingStringMap(t.decShares, ct)
	if err != nil {
		fmt.Println("ERROR, cannot decrypt decshares")
		return false
	}
	fmt.Println("msg:", string(msg), "decMsg:", string(decMsg))
	return string(msg) == string(decMsg)
}
