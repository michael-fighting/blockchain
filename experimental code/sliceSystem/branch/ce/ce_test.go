package ce

import (
	"fmt"
	"github.com/DE-labtory/cleisthenes"
	"testing"
)

func TestCtoss(t *testing.T) {
	memberMap := cleisthenes.NewMemberMap()
	memberMap.Add(cleisthenes.NewMember("127.0.0.1", 5000))
	memberMap.Add(cleisthenes.NewMember("127.0.0.1", 5002))
	memberMap.Add(cleisthenes.NewMember("127.0.0.1", 5004))
	memberMap.Add(cleisthenes.NewMember("127.0.0.1", 5006))
	memberMap.Add(cleisthenes.NewMember("127.0.0.1", 5008))
	memberMap.Add(cleisthenes.NewMember("127.0.0.1", 5010))
	ce, _ := New(6, 3, 4, 0, nil, cleisthenes.Member{}, cleisthenes.Member{}, nil, *memberMap, nil)
	fmt.Println(ce.ctoss(""))
	fmt.Println(ce.ctoss(""))
	fmt.Println(ce.ctoss(""))
}
