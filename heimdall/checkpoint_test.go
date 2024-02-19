package heimdall

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/require"
	httpClient "github.com/tendermint/tendermint/rpc/client"
	"github.com/umbracle/ethgo"
	"testing"
)

// const HeimdallTestRpc = "https://heimdall-api-testnet.polygon.technology:"
const HeimdallTestRpc = "http://149.28.197.92:26657"

func TestCheckpointBasics(t *testing.T) {
	c := httpClient.NewHTTP(HeimdallTestRpc, "/websocket")

	err := c.Start()
	require.NoError(t, err)

	// votes []*tmTypes.CommitSig, sigs []byte, chainID string, err error
	votes, sigs, chainId, err := FetchVotes(c, 1495098)
	require.NoError(t, err)
	_ = votes
	_ = sigs
	_ = chainId

	qr, err := c.ABCIQuery("custom/checkpoint/ack-count", nil)
	require.NoError(t, err)

	var ackCount uint64
	err = jsoniter.ConfigFastest.Unmarshal(qr.Response.Value, &ackCount)
	require.NoError(t, err)
}

// {"height":"1495098","result":{"id":2809,"proposer":"0x6dc2dd54f24979ec26212794c71afefed722280c",
// "start_block":3639411,"end_block":3639922,"root_hash":"0x80df8b6d4fa3731c4b4960522efba1602e23ee1ebff9ac5f237a540de04df4cc","bor_chain_id":"80002","timestamp":1708275729}}

// 	getBlockRange := func() ([]*ethgo.Block, error) {
//		var blocks []*ethgo.Block
//		for num := 3639411; num <= 3639922; num++ {
//			b, err := c.Eth().GetBlockByNumber(ethgo.BlockNumber(num), false)
//			if err != nil {
//				return nil, err
//			}
//			blocks = append(blocks, b)
//
//			jsonBlock, err := json.Marshal(b)
//			if err != nil {
//				return nil, err
//			}
//			println(string(jsonBlock))
//		}

func TestCalcCheckpoint(t *testing.T) {

	//c, err := jsonrpc.NewClient("https://rpc-amoy.polygon.technology/")
	//require.NoError(t, err)

	getBlockRange := func() (blocks []*ethgo.Block, err error) {
		r := bytes.NewReader(checkPointData)
		scanner := bufio.NewScanner(r)
		for {
			if !scanner.Scan() {
				break
			}
			var b ethgo.Block
			if err = json.Unmarshal(scanner.Bytes(), &b); err != nil {
				return
			}
			blocks = append(blocks, &b)
		}
		if len(blocks) != 512 || blocks[0].Number != 3639411 || blocks[len(blocks)-1].Number != 3639922 {
			err = fmt.Errorf("should not happen - invalid or unexpected test data")
		}
		return
	}

	rootHash, err := getRootHash(getBlockRange)
	require.NoError(t, err)
	require.Equal(t, "80df8b6d4fa3731c4b4960522efba1602e23ee1ebff9ac5f237a540de04df4cc", rootHash)
}
