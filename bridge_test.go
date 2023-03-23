package hub_research

import (
	"github.com/paulgoleary/hub-research/crypto"
	"github.com/stretchr/testify/require"
	"github.com/umbracle/ethgo"
	"github.com/umbracle/ethgo/jsonrpc"
	"os"
	"testing"
)

var zkRootStateSenderAddr = ethgo.HexToAddress("0x6978BD695CCAc50350fe9974bBae5D29ec2Ae4D7")

func TestBridgeBasics(t *testing.T) {

	ecRoot, err := jsonrpc.NewClient(os.Getenv("ZKEVM_URL"))
	require.NoError(t, err)

	sk, err := crypto.SKFromHex(os.Getenv("ZKEVM_SK"))
	require.NoError(t, err)
	k := &ecdsaKey{k: sk}

	rootState, err := loadArtifact(ecRoot, "root/StateSender.sol/StateSender", k, zkRootStateSenderAddr)
	require.NoError(t, err)

	maybeTxOutput = func(s string) {
		println(s)
	}
	testData := []byte{0xde, 0xad, 0xbe, 0xef}
	err = TxnDoWait(rootState.Txn("syncState",
		edgeChildValidatorSetAddr,
		testData,
	))
	require.NoError(t, err)

}
