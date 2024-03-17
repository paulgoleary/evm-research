package evm_research

import (
	"bytes"
	"encoding/hex"
	"math/big"
)

// cribbed from https://gist.github.com/miguelmota/bc4304bb21a8f4cc0a37a0f9347b8bbb

func encodePacked(input ...[]byte) []byte {
	return bytes.Join(input, nil)
}

func encodeBytesString(v string) []byte {
	decoded, err := hex.DecodeString(v)
	if err != nil {
		panic(err)
	}
	return decoded
}

func encodeUint256(v string) []byte {
	bn := new(big.Int)
	bn.SetString(v, 10)
	return padUint256(bn)
}

func encodeUint64(v string) []byte {
	bn := new(big.Int)
	bn.SetString(v, 10)
	return padUint64(bn)
}

func padUint256(bn *big.Int) []byte {
	var b [32]byte
	copy(b[32-len(bn.Bytes()):], bn.Bytes())
	return b[:]
}

func padUint64(bn *big.Int) []byte {
	var b [8]byte
	copy(b[8-len(bn.Bytes()):], bn.Bytes())
	return b[:]
}

func encodeUint256Array(arr []string) []byte {
	var res [][]byte
	for _, v := range arr {
		b := encodeUint256(v)
		res = append(res, b)
	}

	return bytes.Join(res, nil)
}
