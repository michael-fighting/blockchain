package cleisthenes

import (
	"fmt"
	"github.com/DE-labtory/cleisthenes/pb"
	"testing"
	"time"
)

func TestReqRepo_Save(t *testing.T) {
	reqRepo := NewReqRepo()
	msg := pb.Message{Proposer: "msg"}
	msg1 := msg
	msg1.Proposer = "msg1"
	msg2 := msg
	msg2.Proposer = "msg2"
	msg3 := msg
	msg3.Proposer = "msg3"

	reqRepo.Save(9, msg1)
	reqRepo.Save(9, msg2)
	reqRepo.Save(9, msg3)

	msgs := reqRepo.FindMsgsByEpochAndDelete(9)
	for _, msg := range msgs {
		go fmt.Println(msg)
	}

	msgs = reqRepo.FindMsgsByEpochAndDelete(9)
	for _, msg := range msgs {
		fmt.Println(msg)
	}
	time.Sleep(1)
}
