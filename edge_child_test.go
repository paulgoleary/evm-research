package evm_research

import (
	"github.com/stretchr/testify/require"
	"github.com/umbracle/ethgo"
	"github.com/umbracle/ethgo/jsonrpc"
	"os"
	"testing"
)

var edgeChildValidatorSetAddr = ethgo.HexToAddress("0x101")

func TestChildStaking(t *testing.T) {

	ec, err := jsonrpc.NewClient(os.Getenv("EDGE_CHILD_URL"))
	require.NoError(t, err)

	blk, err := ec.Eth().BlockNumber()
	require.NoError(t, err)
	require.True(t, blk > 0)

	chainId, err := ec.Eth().ChainID()
	require.NoError(t, err)
	require.Equal(t, uint64(1706), chainId.Uint64())

	msgSender := ethgo.HexToAddress("0x4756bc913ea1f59e8c046d53e09e8cb6191d4d9e") // TODO: nope
	k := &extKey{msgSender}

	cvs, err := loadArtifact(ec, "child/ChildValidatorSet.sol/ChildValidatorSet", k, edgeChildValidatorSetAddr)
	require.NoError(t, err)

	// getEpochByBlock
	// res, err := cvs.Call("getEpochByBlock", ethgo.Latest, blk)
	res, err := cvs.Call("getCurrentValidatorSet", ethgo.Latest)
	require.NoError(t, err)
	validatorAddrs, ok := res["0"].([]ethgo.Address)
	require.True(t, ok)

	for i := 0; i < len(validatorAddrs); i++ {
		res, err = cvs.Call("getValidator", ethgo.Latest, validatorAddrs[i])
		require.NoError(t, err)
		resMap, ok := res["0"].(map[string]interface{})
		require.True(t, ok)

		v, err := ValidatorFromMap(resMap)
		require.NoError(t, err)
		_ = v
	}
}
