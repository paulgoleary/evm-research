package heimdall

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	httpClient "github.com/tendermint/tendermint/rpc/client"
	tmTypes "github.com/tendermint/tendermint/types"
	"github.com/umbracle/ethgo"
	"github.com/umbracle/ethgo/abi"
	"github.com/umbracle/ethgo/jsonrpc"
	"github.com/umbracle/ethgo/wallet"
	"github.com/xsleonard/go-merkle"
	"golang.org/x/crypto/sha3"
	"math/big"
	"net/http"
	"testing"
	"time"
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

var milestoneTxHash = "2e65d38c422e31f220b05fbc24328a77d034c1a9a099c57ff90693ded8579614"

var myClient = &http.Client{Timeout: 10 * time.Second}

func getJson(url string, target interface{}) error {
	r, err := myClient.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}

func TestMilestoneTx(t *testing.T) {
	var txMap map[string]interface{}
	err := getJson(fmt.Sprintf("%v/txs/%v", HeimdallAmoyRpc, milestoneTxHash), &txMap)
	require.NoError(t, err)

	var sideTxMap map[string]interface{}
	err = getJson(fmt.Sprintf("%v/txs/%v/side-tx", HeimdallAmoyRpc, milestoneTxHash), &sideTxMap)
	require.NoError(t, err)

	jsonOut, err := json.Marshal(sideTxMap)
	require.NoError(t, err)
	println(string(jsonOut))
}

func getBlockRangeAmoy() ([]*ethgo.Block, error) {
	c, err := jsonrpc.NewClient("https://rpc-amoy.polygon.technology/")
	if err != nil {
		return nil, err
	}

	var blocks []*ethgo.Block
	for num := 3639411; num <= 3639922; num++ {
		b, err := c.Eth().GetBlockByNumber(ethgo.BlockNumber(num), false)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, b)

		jsonBlock, err := json.Marshal(b)
		if err != nil {
			return nil, err
		}
		println(string(jsonBlock))
	}
	return blocks, nil
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

// value = {interface{} | []interface{}} len:5, cap:8
// 0 = {interface{} | map[string]interface{}}
//
//	0 = key -> module
//	1 = value -> checkpoint
//
// 1 = {interface{} | map[string]interface{}}
//
//	0 = key -> proposer
//	1 = value -> 0x4ad84f7014b7b44f723f284a85b1662337971439
//
// 2 = {interface{} | map[string]interface{}}
//
//	0 = key -> start-block
//	1 = value -> 3887749
//
// 3 = {interface{} | map[string]interface{}}
//
//	0 = key -> end-block
//	1 = value -> 3887762
//
// 4 = {interface{} | map[string]interface{}}
//
//	0 = key -> hash
//	1 = value -> 0x6f73bdeda24c8d6b978628e10c425f5a8bbf181a547dafdf5eb156135626728e

// TODO: again???
func packSig(r, s, v *big.Int) []byte {
	var b [65]byte
	r32 := convertTo32(r.Bytes())
	copy(b[:32], r32[:])
	s32 := convertTo32(s.Bytes())
	copy(b[32:64], s32[:])
	b[64] = byte(v.Uint64())
	return b[:]
}

// proposer? 0x4ad84f7014b7b44f723f284a85b1662337971439

func TestMilestoneSigs(t *testing.T) {

	var res map[string]interface{}
	err := json.Unmarshal(sideTxData, &res)
	require.NoError(t, err)

	sideTxMap, ok := res["result"].(map[string]interface{})
	require.True(t, ok)

	sideTxHex, ok := sideTxMap["data"].(string)
	require.True(t, ok)
	sideTxData, _ := hex.DecodeString(sideTxHex)

	txHash, _ := hex.DecodeString("2e65d38c422e31f220b05fbc24328a77d034c1a9a099c57ff90693ded8579614")

	sideTxResultWithData := tmTypes.SideTxResultWithData{
		SideTxResult: tmTypes.SideTxResult{
			TxHash: txHash,
			Result: int32(abci.SideTxResultType_Yes),
		},
		Data: sideTxData,
	}

	tt, _ := abi.NewType("(address, uint256, uint256, bytes32, uint256, uint256)")
	dd, err := abi.Decode(tt, sideTxData)
	require.NoError(t, err)
	_ = dd

	// require.Equal(t, "0x4ad84f7014b7b44f723f284a85b1662337971439", dd[0].(ethgo.Address).String())

	sideTxSigs, ok := sideTxMap["sigs"].([]interface{})
	require.True(t, ok)

	for i := 0; i < len(sideTxSigs); i++ {
		sig, ok := sideTxSigs[i].([]interface{})
		require.True(t, ok)
		require.Equal(t, 3, len(sig))

		r, _ := new(big.Int).SetString(sig[0].(string), 10)
		s, _ := new(big.Int).SetString(sig[1].(string), 10)
		v, _ := new(big.Int).SetString(sig[2].(string), 10)

		if v.Uint64() >= 27 {
			v.Sub(v, big.NewInt(27))
		}

		signerAddr, err := wallet.EcrecoverMsg(sideTxResultWithData.GetBytes(), packSig(r, s, v))
		require.NoError(t, err)
		println(signerAddr.String())
	}
}
