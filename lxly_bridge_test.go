package evm_research

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/umbracle/ethgo"
	"github.com/umbracle/ethgo/jsonrpc"
)

var lxlyEVMBridgeEthMainnetAddr = ethgo.HexToAddress("0x2a3DD3EB832aF982ec71669E178424b10Dca2EDe")
var lxlyEVMBridgeDeployBlock = 16896718

func TestBridgeBasics(t *testing.T) {

	ec, err := jsonrpc.NewClient(os.Getenv("ETH_URL"))
	require.NoError(t, err)

	//sk, err := crypto.SKFromHex(os.Getenv("ETH_SK"))
	//require.NoError(t, err)
	//k := &ecdsaKey{k: sk}
	//
	//lxlyBridge, err := loadArtifact(ec, "zkevm/PolygonZkEVMBridgeV2", k, lxlyEVMBridgeEthMainnetAddr)
	//require.NoError(t, err)
	//_ = lxlyBridge

	fromBlockNum := ethgo.BlockNumber(lxlyEVMBridgeDeployBlock)
	toBlockNum := ethgo.BlockNumber(lxlyEVMBridgeDeployBlock + 100)

	filter := ethgo.LogFilter{
		Address:   []ethgo.Address{lxlyEVMBridgeEthMainnetAddr},
		Topics:    nil,
		BlockHash: nil,
		From:      &fromBlockNum,
		To:        &toBlockNum,
	}

	ll, err := ec.Eth().GetLogs(&filter)

	require.NoError(t, err)

	for i := 0; i < len(ll); i++ {
		fmt.Println("log", i+1)
		log := ll[i]

		tx := log.TransactionHash.String()
		fmt.Println("tx:", tx)

		address := log.Address.String()
		fmt.Println("address:", address)

		block := log.BlockHash.String()
		fmt.Println("block:", block)

		t := log.Topics

		for j := 0; j < len(t); j++ {
			block := t[j].String()
			fmt.Printf("topic %d: %s\n", j+1, block)
		}

		data := ethgo.BytesToHash(log.Data)
		fmt.Println("data:", data)

	}

	_ = ll
}
