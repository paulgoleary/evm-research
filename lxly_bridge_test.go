package evm_research

import (
	"github.com/stretchr/testify/require"
	"github.com/umbracle/ethgo"
	"github.com/umbracle/ethgo/jsonrpc"
	"os"
	"testing"
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
	_ = ll

}
