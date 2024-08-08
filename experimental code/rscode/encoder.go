package rscode

import "io"

type MockEncoder struct {
	dataShards   int
	parityShards int
	totalShards  int
}

func NewMockEncoder(dataShards, parityShards int) (*MockEncoder, error) {
	return &MockEncoder{
		dataShards:   dataShards,
		parityShards: parityShards,
		totalShards:  dataShards + parityShards,
	}, nil
}

func (enc *MockEncoder) Encode(shards [][]byte) error {
	return nil
}

func (enc *MockEncoder) EncodeIdx(dataShard []byte, idx int, parity [][]byte) error {
	return nil
}

func (enc *MockEncoder) Verify(shards [][]byte) (bool, error) {
	return true, nil
}

func (enc *MockEncoder) Reconstruct(shards [][]byte) error {
	foundIdx := 0
	for i := 0; i < len(shards); i++ {
		if len(shards[i]) != 0 {
			foundIdx = i
			break
		}
	}
	if foundIdx != 0 {
		copy(shards[0], shards[foundIdx])
	}

	for i := 1; i < len(shards); i++ {
		shards[i] = nil
	}
	return nil
}

func (enc *MockEncoder) ReconstructData(shards [][]byte) error {
	return nil
}

func (enc *MockEncoder) ReconstructSome(shards [][]byte, required []bool) error {
	return nil
}

func (enc *MockEncoder) Update(shards [][]byte, newDatashards [][]byte) error {
	return nil
}

func (enc *MockEncoder) Split(data []byte) ([][]byte, error) {
	dst := make([][]byte, enc.totalShards)
	for i := 0; i < len(dst); i++ {
		dst[i] = data
	}
	return dst, nil
}

func (enc *MockEncoder) Join(dst io.Writer, shards [][]byte, outSize int) error {
	return nil
}
