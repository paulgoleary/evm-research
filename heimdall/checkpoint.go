package heimdall

import (
	"encoding/hex"
	"github.com/umbracle/ethgo"
	"github.com/xsleonard/go-merkle"
	"golang.org/x/crypto/sha3"
	"math/big"
)

// checkpoint root hash logic is cribbed from (bor)/consensus/bor/api.go GetRootHash

func convertTo32(input []byte) (output [32]byte) {
	l := len(input)
	if l > 32 || l == 0 {
		return
	}

	copy(output[32-l:], input[:])

	return
}

func appendBytes32(data ...[]byte) []byte {
	var result []byte

	for _, v := range data {
		paddedV := convertTo32(v)
		result = append(result, paddedV[:]...)
	}

	return result
}

// 	header := crypto.Keccak256(appendBytes32(
//		blockHeader.Number.Bytes(),
//		new(big.Int).SetUint64(blockHeader.Time).Bytes(),
//		blockHeader.TxHash.Bytes(),
//		blockHeader.ReceiptHash.Bytes(),

func calcHeaderHash(b *ethgo.Block) []byte {
	return ethgo.Keccak256(appendBytes32(
		new(big.Int).SetUint64(b.Number).Bytes(), // blockHeader.Number.Bytes(),
		new(big.Int).SetUint64(b.Timestamp).Bytes(),
		b.TransactionsRoot.Bytes(), // blockHeader.TxHash.Bytes(),
		b.ReceiptsRoot.Bytes(),
	))
}

func nextPowerOfTwo(n uint64) uint64 {
	if n == 0 {
		return 1
	}
	// http://graphics.stanford.edu/~seander/bithacks.html#RoundUpPowerOf2
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	n++

	return n
}

func calcCheckpointFromBlocks(b []*ethgo.Block) [][]byte {
	headers := make([][]byte, nextPowerOfTwo(uint64(len(b))))
	for i := range b {
		headers[i] = calcHeaderHash(b[i])
	}
	return headers
}

func getRootHash(getBlockRange func() ([]*ethgo.Block, error)) (root string, err error) {

	var blocks []*ethgo.Block
	if blocks, err = getBlockRange(); err != nil {
		return
	}

	headers := calcCheckpointFromBlocks(blocks)

	tree := merkle.NewTreeWithOpts(merkle.TreeOptions{EnableHashSorting: false, DisableHashLeaves: true})
	if err = tree.Generate(headers, sha3.NewLegacyKeccak256()); err != nil {
		return
	}

	root = hex.EncodeToString(tree.Root().Hash)
	return

}
