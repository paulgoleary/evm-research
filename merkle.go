package evm_research

import (
	"encoding/binary"
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

func calculateRoot(leafHash common.Hash, smtProof [][KeyLen]byte, index uint, height uint8) common.Hash {
	var node [KeyLen]byte
	copy(node[:], leafHash[:])

	// Check merkle proof
	var h uint8
	for h = 0; h < height; h++ {
		if ((index >> h) & 1) == 1 {
			node = Hash(smtProof[h], node)
		} else {
			node = Hash(node, smtProof[h])
		}
	}
	return common.BytesToHash(node[:])
}

type Deposit struct {
	LeafType           uint8          `mapstructure:"leafType"`
	OriginNetwork      uint           `mapstructure:"originNetwork"`
	OriginAddress      common.Address `mapstructure:"originAddress"`
	Amount             *big.Int       `mapstructure:"amount"`
	DestinationNetwork uint           `mapstructure:"destinationNetwork"`
	DestinationAddress common.Address `mapstructure:"destinationAddress"`
	DepositCount       uint           `mapstructure:"depositCount"`
	//BlockID              uint64         `json:"removed"`
	//BlockNumber          uint64         `json:"removed"`
	//OriginNetwork        uint           `json:"removed"`
	//TxHash   common.Hash `json:"removed"`
	Metadata []byte `mapstructure:"metadata"`
	// it is only used for the bridge service
	//ReadyForClaim bool
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
