package control

import (
	"fmt"
	"github.com/DE-labtory/cleisthenes"
	"github.com/bytedance/sonic"
	"testing"
)

func TestGetNormalBroadcastType(t *testing.T) {
	info := cleisthenes.CtrlSubmitInfo{
		MsgType: cleisthenes.DataType,
		Data:    []byte("hello world"),
	}

	bs, _ := sonic.ConfigFastest.Marshal(info)

	fmt.Println(string(bs))

	ctrl := DefaultControl{log: cleisthenes.NewLoggerWithHead("test")}
	ctrl.getNormalBroadcastType(bs)
}
