package electDelegate

import (
	"fmt"
	"github.com/DE-labtory/cleisthenes"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog"
	"math/big"
)

var (
	log zerolog.Logger
)

func init() {
	log = cleisthenes.NewLoggerWithHead("ED")
}

func ElectDelegate(addrs []cleisthenes.Address, info *cleisthenes.CtrlInfo) cleisthenes.Address {
	if len(addrs) == 1 {
		return addrs[0]
	}

	//sum := &big.Int{}
	//for _, addr := range addrs {
	//	// FIXME 当前info的addr是ctrl端口，但是addrs是主链端口，需要进行转换，当前转换规则是port+1000
	//	ctrlAddr, _ := cleisthenes.ToAddress(fmt.Sprintf("%s:%d", addr.Ip, addr.Port+1000))
	//	if memberInfo, ok := info.CtrlMemberMap[ctrlAddr]; ok {
	//		log.Debug().Msgf("addr:%s hash:%v", ctrlAddr.String(), memberInfo.CtrlMemberInfo.Hash)
	//		bs := hexutil.MustDecode(memberInfo.CtrlMemberInfo.Hash)
	//		sum.SetBytes(bs)
	//		break
	//	}
	//}
	//k := sum.Mod(sum, big.NewInt(int64(len(addrs)))).Int64()
	//
	//log.Info().Msgf("addrs:%v k:%d", addrs, k)
	//
	//for i := 0; i < len(addrs); i++ {
	//	var cnt int64 = 0
	//	for j := 0; j < len(addrs); j++ {
	//		if addrs[j].Port < addrs[i].Port {
	//			cnt++
	//		}
	//	}
	//	if cnt == k {
	//		return addrs[i]
	//	}
	//}

	return addrs[0]
}

func SelectMainChainNodeFromDelegate(addrs []cleisthenes.Address, info *cleisthenes.CtrlInfo) cleisthenes.Address {
	if len(addrs) == 1 {
		return addrs[0]
	}
	sum := &big.Int{}
	for _, addr := range addrs {
		// FIXME 当前info的addr是ctrl端口，但是addrs是主链端口，需要进行转换，当前转换规则是port+1000
		ctrlAddr, _ := cleisthenes.ToAddress(fmt.Sprintf("%s:%d", addr.Ip, addr.Port+1000))
		if memberInfo, ok := info.CtrlMemberMap[ctrlAddr]; ok {
			tmp := &big.Int{}
			log.Debug().Msgf("addr:%s hash:%v", ctrlAddr.String(), memberInfo.CtrlMemberInfo.Hash)
			bs := hexutil.MustDecode(memberInfo.CtrlMemberInfo.Hash)
			tmp.SetBytes(bs)
			sum = sum.Add(sum, tmp)
		}
	}

	k := sum.Mod(sum, big.NewInt(int64(len(addrs)))).Int64()
	log.Info().Msgf("addrs:%v k:%d", addrs, k)

	for i := 0; i < len(addrs); i++ {
		var cnt int64 = 0
		for j := 0; j < len(addrs); j++ {
			if addrs[j].Port < addrs[i].Port {
				cnt++
			}
		}
		if cnt == k {
			return addrs[i]
		}
	}

	return addrs[0]
}
