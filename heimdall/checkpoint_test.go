package heimdall

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/require"
	httpClient "github.com/tendermint/tendermint/rpc/client"
	"github.com/umbracle/ethgo"
	"github.com/xsleonard/go-merkle"
	"golang.org/x/crypto/sha3"
	"testing"
)

// const HeimdallTestRpc = "https://heimdall-api-testnet.polygon.technology:"
const HeimdallTestRpc = "http://149.28.197.92:26657"
const HeimdallAmoyRpc = "https://heimdall-api-amoy.polygon.technology/"

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

// TODO: we need an efficient way to find milestones / checkpoint event by block range. this can scan block by block ...
func TestGetEvents(t *testing.T) {
	c := httpClient.NewHTTP(HeimdallTestRpc, "/websocket")
	err := c.Start()
	require.NoError(t, err)

	for i := int64(1588000); i < 1588611; i++ {
		events, err := GetBeginBlockEvents(c, i)
		require.NoError(t, err)
		if len(events) > 0 {
			println(events[0].String())
		}
	}
}

// testing with this Amoy testnet checkpoint
// {
//   "height":"1495098",
//   "result":{
//      "id":2809,
//      "proposer":"0x6dc2dd54f24979ec26212794c71afefed722280c",
//      "start_block":3639411,
//      "end_block":3639922,
//      "root_hash":"0x80df8b6d4fa3731c4b4960522efba1602e23ee1ebff9ac5f237a540de04df4cc",
//      "bor_chain_id":"80002",
//      "timestamp":1708275729
//   }
// }

func TestCalcCheckpoint(t *testing.T) {

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

func Test2Leaves(t *testing.T) {
	headers := make([][]byte, 2)
	headers[0], _ = hex.DecodeString("fc905b8816642b177111968433a6aea8ea790ad2ea7c164de1625eaf01270f88")
	headers[1], _ = hex.DecodeString("cadfe86c5a7b1f839bfa2b7a11e5f3599b4d793daf50e690d1acbd8751175bfd")

	tree := merkle.NewTreeWithOpts(merkle.TreeOptions{EnableHashSorting: false, DisableHashLeaves: true})
	if err := tree.Generate(headers, sha3.NewLegacyKeccak256()); err != nil {
		return
	}

	root := hex.EncodeToString(tree.Root().Hash)
	println(root)
}

func TestHeaderHash(t *testing.T) {
	b := &ethgo.Block{
		Number:           3639411,
		TransactionsRoot: ethgo.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"),
		ReceiptsRoot:     ethgo.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"),
		Timestamp:        1708274440,
	}
	header := calcHeaderHash(b)
	println(hex.EncodeToString(header))
}
