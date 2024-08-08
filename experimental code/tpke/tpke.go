package tpke

import (
	"github.com/DE-labtory/cleisthenes"
)

type MockTpke struct{}

func (t *MockTpke) Encrypt(data []byte) ([]byte, error) {

	//playGround := setUp()
	////pk为主公钥
	//pk := playGround.publishPubKey()
	//msg, _ := json.Marshal(data)
	//iLogger.Infof(nil, "序列化后的交易为msg : %v", msg)
	////使用主公钥对消息msg进行加密，得到密文cipherText
	//cipherText, err := pk.Encrypt(msg)
	//cipherData, _ := json.Marshal(cipherText)
	//iLogger.Infof(nil, "门限加密的交易为cipherData : %v", cipherData)
	//return cipherData, err

	//原设计：没有实现门限签名，只是序列化了
	//return json.Marshal(data)
	return data, nil
}

func (t *MockTpke) Decrypt(enc []byte) ([]byte, error) {
	//var contribution cleisthenes.Contribution
	//err := json.Unmarshal(enc, &contribution.TxList)
	//if err != nil {
	//	return nil, err
	//}
	//return contribution.TxList, nil
	return enc, nil
}

func (t *MockTpke) DecShare(ctBytes cleisthenes.CipherText) cleisthenes.DecryptionShare {
	return [96]byte{}
}

func (t *MockTpke) AcceptDecShare(addr cleisthenes.Address, decShare cleisthenes.DecryptionShare) {

}

func (t *MockTpke) ClearDecShare() {}
