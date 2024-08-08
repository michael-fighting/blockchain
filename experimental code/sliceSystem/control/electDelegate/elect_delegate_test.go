package electDelegate

import (
	"fmt"
	"github.com/DE-labtory/cleisthenes"
	"math/rand"
	"testing"
)

func TestElectDelegate(t *testing.T) {
	addrs := make([]cleisthenes.Address, 0)
	var ctrlInfo cleisthenes.CtrlInfo
	ctrlInfo.CtrlMemberMap = make(map[cleisthenes.Address]cleisthenes.CtrlSubmitInfo, 0)
	for port := 6000; port < 6010; port += 2 {
		addr, _ := cleisthenes.ToAddress(fmt.Sprintf("127.0.0.1:%d", port))
		info := cleisthenes.CtrlMemberInfo{
			Random: rand.Intn(100),
			TxSig:  nil,
		}
		ctrlInfo.CtrlMemberMap[addr] = cleisthenes.CtrlSubmitInfo{
			MsgType:        cleisthenes.CtrlInfoType,
			CtrlMemberInfo: info,
		}
		addrs = append(addrs, addr)
	}

	res := ElectDelegate(addrs, &ctrlInfo)
	fmt.Println("res:", res)
}
