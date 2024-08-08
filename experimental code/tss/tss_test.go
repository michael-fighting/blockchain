package tss

import (
	"fmt"
	"github.com/DE-labtory/cleisthenes"
	"testing"
)

func TestSign(t *testing.T) {
	tss, err := NewDefaultTss(
		5,
		3,
		"0x02f0be0d20bb2f64271b63c6d5ba1709486f9338f8f4b322c86602fbc88e15df22d616781f314c03199e898fe739c3c2f9ed6d053235fb346da93dbaf9cda269",
		"0x02f0be0d20bb2f64271b63c6d5ba1709486f9338f8f4b322c86602fbc88e15df22d616781f314c03199e898fe739c3c2f9ed6d053235fb346da93dbaf9cda269",
		"0x2b023ae926fe8ff85ddc1a36c9ff3bf9a466a98cfb9fb45e2305b768070d5f0215dc14c7663de379ce65ad385dff9684bbedd774c26d36fe3c60c4df664600db",
		"0x606ebf775d15b896649dc7d359a3b30183e2c6835cd032b58d518c8b540b53b5",
		"0x6ace3cff14996be59ee444de46bc4c8dffab5dca01deb20f1a235249da10de7d",
		map[cleisthenes.Address]string{
			cleisthenes.Address{Ip: "127.0.0.1", Port: 5000}: "0x2b023ae926fe8ff85ddc1a36c9ff3bf9a466a98cfb9fb45e2305b768070d5f0215dc14c7663de379ce65ad385dff9684bbedd774c26d36fe3c60c4df664600db",
			cleisthenes.Address{Ip: "127.0.0.1", Port: 5002}: "0x24244f9cfe8933e04133d403a36df43e33957a5301bb86f7e41df528a3eeba891cdc1ffcd1cebc7eef40a29e6be3e09b98889785157dfd02afa8360636b050e4",
			cleisthenes.Address{Ip: "127.0.0.1", Port: 5004}: "0x05c7ac35e24e33fdf40646460d0aa45cbcae4d4119964f5e6b92218a889998fd1d221211850a3e9288681f5628ace00bedd79c695286951bab720cadbdff9ed8",
			cleisthenes.Address{Ip: "127.0.0.1", Port: 5006}: "0x18dd6b4f6e52c80f4ff3785574bea184d43db782dc4846d68b0e09cccec384110e90e3a17f9f7e9adfb2d3a03e4baadb29480a55f7c4424d6aad88b31d06a4eb",
			cleisthenes.Address{Ip: "127.0.0.1", Port: 5008}: "0x2c2362cf3d60ba18b1691b8a4b34d48a24ceeba286fc215089e988040d6e6b76162023c4224487b54c218118299283bf23942c1ced37557b5a5e08e362985d34",
		},
		map[cleisthenes.Address]string{
			cleisthenes.Address{Ip: "127.0.0.1", Port: 5000}: "0x6ace3cff14996be59ee444de46bc4c8dffab5dca01deb20f1a235249da10de7d",
			cleisthenes.Address{Ip: "127.0.0.1", Port: 5002}: "0x7c667ede9097a1aadd3f28b07562ebaa2eadc7421659c67166f0c7c405c64927",
			cleisthenes.Address{Ip: "127.0.0.1", Port: 5004}: "0x6e5e62cdf118ee2f32251d2bd298b272c4e1533e9ac05c1d2aca3cc3df54c54a",
			cleisthenes.Address{Ip: "127.0.0.1", Port: 5006}: "0x53db35126ee1c16d3ba38307a820edcdbbf178a9ca95e2e023fa840b240add88",
			cleisthenes.Address{Ip: "127.0.0.1", Port: 5008}: "0x71a5e0d2f1bb08b2e009f08bfdecc1b181dc24224a5ef93585a4b6c0fa4e95a5",
		},
	)
	if err != nil {
		fmt.Println("init failed")
		return
	}

	sig := tss.Sign([]byte("hello world"))
	fmt.Println(string(sig))

	verify := tss.Verify(cleisthenes.Address{Ip: "127.0.0.1", Port: 5000}, sig, []byte("hello world"))
	if verify {
		fmt.Println("验签成功")
	} else {
		fmt.Println("验签失败")
	}
}

func TestVerify(t *testing.T) {

}
