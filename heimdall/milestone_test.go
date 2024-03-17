package heimdall

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	tmTypes "github.com/tendermint/tendermint/types"
	"github.com/umbracle/ethgo"
	"github.com/umbracle/ethgo/abi"
	"github.com/umbracle/ethgo/jsonrpc"
	"github.com/umbracle/ethgo/wallet"
	"math/big"
	"net/http"
	"testing"
	"time"
)

const BorAmoyRpc = "https://rpc-amoy.polygon.technology/"

func getBlockRangeAmoy() ([]*ethgo.Block, error) {
	c, err := jsonrpc.NewClient(BorAmoyRpc)
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

func TestGetETHBlockRange(t *testing.T) {
	ec, err := ethclient.Dial(BorAmoyRpc)
	require.NoError(t, err)

	h, err := ec.HeaderByNumber(context.Background(), big.NewInt(3887762))
	require.NoError(t, err)

	println(h.Hash().String())
	println(h.Root.String())

	bw := bytes.NewBuffer(nil)
	err = h.EncodeRLP(bw)
	require.NoError(t, err)
	println(hex.EncodeToString(bw.Bytes()))
}

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

		packedSig := packSig(r, s, v)
		signerAddr, err := wallet.EcrecoverMsg(sideTxResultWithData.GetBytes(), packedSig)
		require.NoError(t, err)
		println(signerAddr.String())
		println(hex.EncodeToString(packedSig))
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
