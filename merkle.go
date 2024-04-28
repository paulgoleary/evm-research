package evm_research

import (
	"encoding/binary"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/umbracle/ethgo"
	"golang.org/x/crypto/sha3"
	"math/big"
)

const (
	// KeyLen is the length of key and value in the Merkle Tree
	KeyLen = 32
)

// Hash calculates  the keccak hash of elements.
func Hash(data ...[KeyLen]byte) [KeyLen]byte {
	var res [KeyLen]byte
	hash := sha3.NewLegacyKeccak256()
	for _, d := range data {
		hash.Write(d[:]) //nolint:errcheck,gosec
	}
	copy(res[:], hash.Sum(nil))
	return res
}

// HashZero is an empty hash
var HashZero = [KeyLen]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

func generateZeroHashes(height uint8) [][KeyLen]byte {
	var zeroHashes = [][KeyLen]byte{
		HashZero,
	}
	// This generates a leaf = HashZero in position 0. In the rest of the positions that are equivalent to the ascending levels,
	// we set the hashes of the nodes. So all nodes from level i=5 will have the same value and same children nodes.
	for i := 1; i <= int(height); i++ {
		zeroHashes = append(zeroHashes, Hash(zeroHashes[i-1], zeroHashes[i-1]))
	}
	return zeroHashes
}

func calculateRoot(frontier [][KeyLen]byte, index uint, height uint8) common.Hash {
	var node, currentZero [KeyLen]byte
	var h uint8
	for h = 0; h < height; h++ {
		if ((index >> h) & 1) == 1 {
			node = Hash(frontier[h], node)
		} else {
			node = Hash(node, currentZero)
		}
		currentZero = Hash(currentZero, currentZero)
	}
	return common.BytesToHash(node[:])
}

func addLeaf(leafHash common.Hash, frontier [][KeyLen]byte, index uint, height uint8) {
	var node [KeyLen]byte
	copy(node[:], leafHash[:])
	var h uint8
	for h = 0; h < height; h++ {
		if ((index >> h) & 1) == 1 {
			copy(frontier[h][:], node[:])
			return
		}
		node = Hash(frontier[h], node)
	}
	panic("should not get here")
}

type Deposit struct {
	LeafType           uint8          `json:"leafType"`
	OriginNetwork      uint           `json:"originNetwork"`
	OriginAddress      common.Address `json:"originAddress"`
	Amount             *big.Int       `json:"amount"`
	DestinationNetwork uint           `json:"destinationNetwork"`
	DestinationAddress common.Address `json:"destinationAddress"`
	DepositCount       uint           `json:"depositCount"`
	Metadata           []byte         `json:"metadata"`
}

func (d *Deposit) JSON() string {
	jsonBytes, _ := json.Marshal(d)
	return string(jsonBytes)
}

func hashDeposit(deposit *Deposit) [KeyLen]byte {
	var res [KeyLen]byte
	origNet := make([]byte, 4) //nolint:gomnd
	binary.BigEndian.PutUint32(origNet, uint32(deposit.OriginNetwork))
	destNet := make([]byte, 4) //nolint:gomnd
	binary.BigEndian.PutUint32(destNet, uint32(deposit.DestinationNetwork))
	var buf [KeyLen]byte
	metaHash := ethgo.Keccak256(deposit.Metadata)
	copy(res[:], ethgo.Keccak256([]byte{deposit.LeafType}, origNet, deposit.OriginAddress[:], destNet, deposit.DestinationAddress[:], deposit.Amount.FillBytes(buf[:]), metaHash))
	return res
}
