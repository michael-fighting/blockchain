package cleisthenes

import "math/big"

type Tss interface {
	Sign(data []byte) []byte
	Verify(address Address, sig []byte, data []byte) bool

	CombineSig(sigs map[Address][]byte) *big.Int

	VerifyCombineSig(ss *big.Int, data []byte) bool
}
