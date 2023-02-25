package hub_research

import (
	"encoding/hex"
	"fmt"
	"github.com/paulgoleary/hub-research/crypto"
	"github.com/stretchr/testify/require"
	"github.com/umbracle/ethgo"
	"github.com/umbracle/ethgo/abi"
	"github.com/umbracle/ethgo/jsonrpc"
	"math/big"
	"os"
	"testing"
)

var zkEVMRegistryAddr = ethgo.HexToAddress("0x6978BD695CCAc50350fe9974bBae5D29ec2Ae4D7")

// update registry contract map, key a1ed0e7a71ca197f0dfc1206d3fcb9c6b88b70f1c3a11268f9b6ed75e8cabd08, value 0xBA305cBed8a253345aeD5a42D3736F9407d24d1b, hash 0x35a38f6b7df89a95c27ed427f835879553aa1484a72e316b6e74b2c0f80542e9
// update registry contract map, key 56e86af72b94d3aa725a2e35243d6acbf3dc1ada7212033defd5140c5fcb6a9d, value 0x6E63b3F24dABde0DDef88Fd0A59e86201D62971F, hash 0x02fcdf9baffaaf96cce228fcaa93f5be1fb0bf27a864fd81be1053af5d77b524
// update registry contract map, key f32233bced9bbd82f0754425f51b5ffaf897dacec3c8ac3384a66e38ea701ec8, value 0x0135f11cD2451E620977D01A3D33c534F91239De, hash 0x8f7c55f9e17b368bc6e9902893371a7dec9ef138ebfe4f21b3db5fbe1ee122f3
// update registry contract map, key 1ca32e38cf142cb762bc7468b9a3eac49626b43585fcbd6d3b807227216286c2, value 0x360A0377F731D163985B7C70d08029C3866872D4, hash 0x39f28a3047683cbecb42389468b002e4dc51ef4fdcca4244201c775428d9030d
// root chain proxy: 0x94078c2AEccE36DfdaD4999F8C903CC9184dF78f
// stake manager: 0x6E63b3F24dABde0DDef88Fd0A59e86201D62971F
// stake token: 0x5728197742b1dFd7660AEc1D4dD77e37268361f1
// slashing manager: 0x360A0377F731D163985B7C70d08029C3866872D4
// staking info: 0x2BE7CCbc0DcdAd1FF6cdcCaFA79a2920d40E2B14
// staking nft: 0xC4fC8a4A5b1a1B2d7d8893c72727a3fAafAe099b
// stake manager: 0x6E63b3F24dABde0DDef88Fd0A59e86201D62971F
// stake manager proxy: 0xE9614f75672cb63ef4171bFbA5242066695B2838

// HubProviderSet on zkEVM: 0x8CCE00685A46aEFcCe602FAa35e59669F3395B95

var zkEVMHPSAddr = ethgo.HexToAddress("0x8CCE00685A46aEFcCe602FAa35e59669F3395B95")
var zkEVMBLSAddr = ethgo.HexToAddress("0xBAdF63888C173168f6F020B6E22B9ff7FF9B8806")

var kzEVMProviderBLSHex = "2b172bd46566451c81d13795b902d315bba47a4033774fd0e46a127d400837da0bb816f67ef39b73ef9629ff7dcb475e3afe5a76d45e858a1a9187ba78a154b20dcbd19d7d902ede2b45f65b2291396824ce6c556284f4338a031d1b39f5b087028c934724305ea98dd783fe9d22b7392268ae230bd309f1423c0899831095f5"

func TestZKEVMDeploy(t *testing.T) {

	ec, err := jsonrpc.NewClient(os.Getenv("ZKEVM_URL"))
	require.NoError(t, err)

	sk, err := crypto.SKFromHex(os.Getenv("ZKEVM_SK"))
	require.NoError(t, err)
	skAddr := crypto.PubKeyToAddress(&sk.PublicKey)
	println(skAddr.String())
	k := &ecdsaKey{k: sk}

	if false {
		deployed, addr, err := deployArtifact(ec, "hub/HubProviderSet.sol/HubProviderSet", k, []interface{}{skAddr})
		require.NoError(t, err)
		require.NotNil(t, deployed)
		require.NotNil(t, addr)
		println(addr.String())
	}
}

type InitStruct struct {
	EpochReward   *big.Int `abi:"epochReward"`
	MinStake      *big.Int `abi:"minStake"`
	MinDelegation *big.Int `abi:"minDelegation"`
	EpochSize     *big.Int `abi:"epochSize"`
}

var InitStructBIType = abi.MustNewType("tuple(uint256 epochReward,uint256 minStake,uint256 minDelegation,uint256 epochSize)")

func (e *InitStruct) EncodeAbi() ([]byte, error) {
	return InitStructBIType.Encode(e)
}

func (e *InitStruct) Encode() ([]byte, error) {
	return InitStructBIType.Encode(e)
}

func TestZKEVMInit(t *testing.T) {

	ec, err := jsonrpc.NewClient(os.Getenv("ZKEVM_URL"))
	require.NoError(t, err)

	sk, err := crypto.SKFromHex(os.Getenv("ZKEVM_SK"))
	require.NoError(t, err)
	skAddr := crypto.PubKeyToAddress(&sk.PublicKey)
	k := &ecdsaKey{k: sk}

	if false {
		deployed, addr, err := deployArtifact(ec, "common/BLS.sol/BLS", k, nil)
		require.NoError(t, err)
		require.NotNil(t, deployed)
		require.NotNil(t, addr)
	}

	hps, err := loadArtifact(ec, "hub/HubProviderSet.sol/HubProviderSet", k, zkEVMHPSAddr)
	require.NoError(t, err)

	pskd := new(big.Int).Add(big.NewInt(1), sk.D)
	psk, err := crypto.SKFromInt(pskd)
	require.NoError(t, err)

	blsSK, err := crypto.GenerateBlsKey()
	require.NoError(t, err)
	blsPK := blsSK.PublicKey()
	blsBytes := blsPK.Marshal()
	println(hex.EncodeToString(blsBytes))

	providerAddresses := []ethgo.Address{crypto.PubKeyToAddress(&psk.PublicKey)}
	providerPK := [4]*big.Int{
		new(big.Int).SetBytes(blsBytes[:32]),
		new(big.Int).SetBytes(blsBytes[32:64]),
		new(big.Int).SetBytes(blsBytes[64:96]),
		new(big.Int).SetBytes(blsBytes[96:]),
	}
	providerPubkeys := [][4]*big.Int{providerPK}
	providerStakes := []*big.Int{big.NewInt(1_000_000_000)}

	newMessage := [2]*big.Int{
		new(big.Int).SetBytes([]byte{0xde, 0xad, 0xbe, 0xef}),
		new(big.Int).SetBytes([]byte{0xca, 0xfe, 0xba, 0xbe}),
	}

	is := &InitStruct{
		EpochReward:   big.NewInt(1),
		MinStake:      big.NewInt(0),
		MinDelegation: big.NewInt(0),
		EpochSize:     big.NewInt(16),
	}
	if false {
		err = TxnDoWait(hps.Txn("initialize",
			is,
			providerAddresses,
			providerPubkeys,
			providerStakes,
			zkEVMBLSAddr,
			newMessage,
			skAddr,
		))
		require.NoError(t, err)
	}

}

func TestZKEVMCheckHPS(t *testing.T) {

	ec, err := jsonrpc.NewClient(os.Getenv("ZKEVM_URL"))
	require.NoError(t, err)

	sk, err := crypto.SKFromHex(os.Getenv("ZKEVM_SK"))
	require.NoError(t, err)
	// skAddr := crypto.PubKeyToAddress(&sk.PublicKey)
	k := &ecdsaKey{k: sk}

	hps, err := loadArtifact(ec, "hub/HubProviderSet.sol/HubProviderSet", k, zkEVMHPSAddr)
	require.NoError(t, err)

	res, err := hps.Call("sortedProviders", ethgo.Latest, big.NewInt(1))
	require.NoError(t, err)
	checkAddrs, ok := res["0"].([]ethgo.Address)
	require.True(t, ok)
	require.Equal(t, 1, len(checkAddrs))
	println(checkAddrs[0].String())

	res, err = hps.Call("totalStake", ethgo.Latest)
	require.NoError(t, err)
	require.True(t, len(res) > 0)

	pskd := new(big.Int).Add(big.NewInt(1), sk.D)
	psk, err := crypto.SKFromInt(pskd)
	require.NoError(t, err)
	println(crypto.PubKeyToAddress(&psk.PublicKey).String())

	res, err = hps.Call("getProvider", ethgo.Latest, crypto.PubKeyToAddress(&psk.PublicKey))
	require.NoError(t, err)

	latest, err := ec.Eth().BlockNumber()
	require.NoError(t, err)

	for i := uint64(330909); i <= latest; i++ {
		blk, err := ec.Eth().GetBlockByNumber(ethgo.BlockNumber(i), false)
		require.NoError(t, err)
		println(fmt.Sprintf("%v: %v", i, blk.StateRoot.String()))
	}
}
