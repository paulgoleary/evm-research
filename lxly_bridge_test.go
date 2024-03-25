package evm_research

import (
	"cmp"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/umbracle/ethgo/abi"
	"io"
	"math/big"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/umbracle/ethgo"
	"github.com/umbracle/ethgo/jsonrpc"
)

var lxlyEVMBridgeEthMainnetAddr = ethgo.HexToAddress("0x2a3DD3EB832aF982ec71669E178424b10Dca2EDe")
var lxlyEVMGlobalExitRootAddr = ethgo.HexToAddress("0x580bda1e7A0CFAe92Fa7F6c20A3794F169CE3CFb")
var lxlyEVMBridgeDeployBlock = 16896718
var lxlyEVMV2UpgradeBlock = 19100076
var firstDepositEventBlock = 16898815

var startBlock = lxlyEVMV2UpgradeBlock

var (
	// New Ger event
	updateL1InfoTreeSignatureHash = ethgo.Hash(ethgo.Keccak256([]byte("UpdateL1InfoTree(bytes32,bytes32)")))
	l1InfoTreeEvent               = abi.MustNewEvent(`event UpdateL1InfoTree(
        bytes32 indexed mainnetExitRoot,
        bytes32 indexed rollupExitRoot
	)`)

	// PreLxLy events
	updateGlobalExitRootSignatureHash = ethgo.Hash(ethgo.Keccak256([]byte("UpdateGlobalExitRoot(bytes32,bytes32)")))
	v1GEREvent                        = abi.MustNewEvent(`event UpdateGlobalExitRoot(
        bytes32 indexed mainnetExitRoot,
        bytes32 indexed rollupExitRoot
	)`)

	// New Bridge events
	depositEventSignatureHash = ethgo.Hash(ethgo.Keccak256([]byte("BridgeEvent(uint8,uint32,address,uint32,address,uint256,bytes,uint32)"))) // Used in oldBridge as well
	depositEvent              = abi.MustNewEvent(`event BridgeEvent(
	   uint8 leafType,
	   uint32 originNetwork,
	   address originAddress,
	   uint32 destinationNetwork,
	   address destinationAddress,
	   uint256 amount,
	   bytes metadata,
	   uint32 depositCount
	)`)

	//     * @param globalIndex Global index is defined as:
	//     * | 191 bits |    1 bit     |   32 bits   |     32 bits    |
	//     * |    0     |  mainnetFlag | rollupIndex | localRootIndex |
	//     * note that only the rollup index will be used only in case the mainnet flag is 0
	//     * note that global index do not assert the unused bits to 0.
	//     * This means that when synching the events, the globalIndex must be decoded the same way that in the Smart contract
	//     * to avoid possible synch attacks

	claimEventSignatureHash = ethgo.Hash(ethgo.Keccak256([]byte("ClaimEvent(uint256,uint32,address,address,uint256)")))
	claimEvent              = abi.MustNewEvent(`event ClaimEvent(
        uint256 globalIndex,
        uint32 originNetwork,
        address originAddress,
        address destinationAddress,
        uint256 amount
	)`)

	// Old Bridge events
	oldClaimEventSignatureHash = ethgo.Hash(ethgo.Keccak256([]byte("ClaimEvent(uint32,uint32,address,address,uint256)")))
	oldClaimEvent              = abi.MustNewEvent(`event ClaimEvent(
        uint32 index,
        uint32 originNetwork,
        address originAddress,
        address destinationAddress,
        uint256 amount
	)`)
)

const (
	BridgeEventL1InfoTree = iota
	BridgeEventV1GER
	BridgeEventDeposit
	BridgeEventV2Claim
	BridgeEventV1Claim
)

var (
	bridgeEventTypeMap = map[ethgo.Hash]int{
		l1InfoTreeEvent.ID(): BridgeEventL1InfoTree,
		v1GEREvent.ID():      BridgeEventV1GER,
		depositEvent.ID():    BridgeEventDeposit,
		claimEvent.ID():      BridgeEventV2Claim,
		oldClaimEvent.ID():   BridgeEventV1Claim,
	}

	bridgeEventParseMap = map[int]func(log *ethgo.Log) (map[string]interface{}, error){
		BridgeEventL1InfoTree: l1InfoTreeEvent.ParseLog,
		BridgeEventV1GER:      v1GEREvent.ParseLog,
		BridgeEventDeposit:    depositEvent.ParseLog,
		BridgeEventV2Claim:    claimEvent.ParseLog,
		BridgeEventV1Claim:    oldClaimEvent.ParseLog,
	}
)

func maybeFromLog(l *ethgo.Log) *BridgeEvent {
	if et, ok := bridgeEventTypeMap[l.Topics[0]]; !ok {
		return nil
	} else {
		data, _ := bridgeEventParseMap[et](l)
		be := BridgeEvent{
			Removed:          l.Removed,
			BlockNumber:      l.BlockNumber,
			TransactionIndex: l.TransactionIndex,
			LogIndex:         l.LogIndex,
			TransactionHash:  l.TransactionHash,
			EventType:        uint8(et),
			Data:             data,
		}
		return &be
	}
}

type BridgeEvent struct {
	Removed          bool                   `json:"removed"`
	BlockNumber      uint64                 `json:"block_number"`
	TransactionIndex uint64                 `json:"transaction_index"`
	LogIndex         uint64                 `json:"log_index"`
	TransactionHash  ethgo.Hash             `json:"transaction_hash"`
	EventType        uint8                  `json:"event_type"`
	Data             map[string]interface{} `json:"event_data"`
}

func (be BridgeEvent) toDeposit() Deposit {
	leafType := be.Data["leafType"].(float64)
	originNetwork := be.Data["originNetwork"].(float64)
	originAddress := be.Data["originAddress"].(string)
	amount := be.Data["amount"].(float64)
	destinationNetwork := be.Data["destinationNetwork"].(float64)
	destinationAddress := be.Data["destinationAddress"].(string)
	depositCount := be.Data["depositCount"].(float64)
	dep := Deposit{
		LeafType:           uint8(leafType),
		OriginNetwork:      uint(originNetwork),
		OriginAddress:      common.HexToAddress(originAddress),
		Amount:             big.NewInt(int64(amount)),
		DestinationNetwork: uint(destinationNetwork),
		DestinationAddress: common.HexToAddress(destinationAddress),
		DepositCount:       uint(depositCount),
		Metadata:           nil,
	}
	return dep
}

func TestEventDefCompat(t *testing.T) {
	// this checks the compatibility of ethgo-based event defs and their equivalent from zkevm-bridge-service
	require.Equal(t, updateL1InfoTreeSignatureHash.Bytes(), l1InfoTreeEvent.ID().Bytes())
	require.Equal(t, updateGlobalExitRootSignatureHash.Bytes(), v1GEREvent.ID().Bytes())
	require.Equal(t, depositEventSignatureHash.Bytes(), depositEvent.ID().Bytes())
	require.Equal(t, claimEventSignatureHash.Bytes(), claimEvent.ID().Bytes())
	require.Equal(t, oldClaimEventSignatureHash.Bytes(), oldClaimEvent.ID().Bytes())
}

func TestBridgeExtractEvents(t *testing.T) {

	ec, err := jsonrpc.NewClient(os.Getenv("ETH_URL"))
	require.NoError(t, err)

	bridgeABI, err := LoadABI("zkevm/PolygonZkEVMBridgeV2")
	require.NoError(t, err)
	_ = bridgeABI

	fromBlockNum := ethgo.BlockNumber(startBlock)

	f, err := os.Create("./bridge_events.ndjson")
	require.NoError(t, err)
	defer f.Close()

	cntEvent := 0
	cntMap := make(map[uint8]int)

	for {
		toBlockNum := fromBlockNum + 999

		filter := ethgo.LogFilter{
			// Address:   []ethgo.Address{lxlyEVMBridgeEthMainnetAddr, lxlyEVMGlobalExitRootAddr},
			// Topics:    topics,
			BlockHash: nil,
			From:      &fromBlockNum,
			To:        &toBlockNum,
		}

		// this seems to be the most efficient way to query ...?

		filter.Address = []ethgo.Address{lxlyEVMBridgeEthMainnetAddr}
		llBridge, err := ec.Eth().GetLogs(&filter)
		require.NoError(t, err)

		filter.Address = []ethgo.Address{lxlyEVMGlobalExitRootAddr}
		llGER, err := ec.Eth().GetLogs(&filter)
		require.NoError(t, err)

		ll := append(llBridge, llGER...)

		fmt.Printf("queried blocks %v to %v, %v events\n", int(fromBlockNum), int(toBlockNum), len(ll))

		if len(ll) > 0 {
			for _, l := range ll {
				if maybeBridgeEvent := maybeFromLog(l); maybeBridgeEvent != nil {
					json, err := json.Marshal(maybeBridgeEvent)
					require.NoError(t, err)
					f.WriteString(string(json) + "\n")
					cntEvent++
					cntMap[maybeBridgeEvent.EventType] += 1
				}
			}
		}

		if cntEvent >= 10_000 {
			break
		}

		fromBlockNum = toBlockNum + 1
	}

	fmt.Printf("type count summary: 0: %v, 1: %v, 2: %v, 3: %v, 5: %v\n", cntMap[0], cntMap[1], cntMap[2], cntMap[3], cntMap[4])
}

func TestBridgeProcessEvents(t *testing.T) {

	f, err := os.Open("./bridge_events_10k.ndjson")
	require.NoError(t, err)

	d := json.NewDecoder(f)
	var bevs []BridgeEvent
	cnt := 0
	for {
		var be BridgeEvent
		if err := d.Decode(&be); err == io.EOF {
			break
		} else {
			require.NoError(t, err)
			bevs = append(bevs, be)
			cnt++
		}
	}

	sort.Slice(bevs, func(i, j int) bool {
		switch cmp.Compare(bevs[i].BlockNumber, bevs[j].BlockNumber) {
		case -1:
			return true
		case 0:
			{
				switch cmp.Compare(bevs[i].TransactionIndex, bevs[j].TransactionIndex) {
				case -1:
					return true
				case 0:
					{
						switch cmp.Compare(bevs[i].LogIndex, bevs[j].LogIndex) {
						case -1:
							return true
						}
					}
				}
			}
		}
		return false
	})
	require.Equal(t, cnt, len(bevs))

	checkSort := sort.SliceIsSorted(bevs, func(i, j int) bool {
		return bevs[i].BlockNumber < bevs[j].BlockNumber &&
			bevs[i].TransactionIndex < bevs[j].TransactionIndex &&
			bevs[i].LogIndex < bevs[j].LogIndex
	})
	require.True(t, checkSort)

	checkDepositCount := -1
	depositCount := 0
	for _, ev := range bevs {
		if ev.EventType == BridgeEventDeposit {
			dc, ok := ev.Data["depositCount"].(float64)
			require.True(t, ok)
			if checkDepositCount == -1 {
				checkDepositCount = int(dc)
			}
			require.True(t, int(dc) == checkDepositCount)
			checkDepositCount++
			depositCount++
		}
	}
	fmt.Printf("found %v deposits\n", depositCount+1)
	fmt.Printf("last block found %v\n", bevs[len(bevs)-1].BlockNumber)

	// TODO: quick-and-dirty test!
	// MATCHED with Rust impl ...
	require.Equal(t, BridgeEventDeposit, int(bevs[0].EventType))
	dep := bevs[0].toDeposit()
	depHash := hashDeposit(&dep)
	require.Equal(t, "b7cd745b9fc33c6e233768f51f262865c8cdff188d4e63c16709e389c11d5cd8", hex.EncodeToString(depHash[:]))

	nodes := generateZeroHashes(32)
	rootHash := calculateRoot(depHash, nodes, 0, 32)
	require.Equal(t, "927e6ceecb5b20935d26fbfc57002a59298b6e82640e8d652809e06854d7a81f", hex.EncodeToString(rootHash[:]))

	require.Equal(t, BridgeEventV1GER, int(bevs[1].EventType))
	bevsHash := bevs[1].Data["mainnetExitRoot"].([]interface{})
	// TODO: obv this needs to be cleaned up :)
	require.True(t, byte(bevsHash[0].(float64)) == rootHash[0])
	require.True(t, byte(bevsHash[1].(float64)) == rootHash[1])
	require.True(t, byte(bevsHash[2].(float64)) == rootHash[2])
	require.True(t, byte(bevsHash[3].(float64)) == rootHash[3])
}
