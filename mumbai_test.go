package hub_research

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"github.com/paulgoleary/hub-research/crypto"
	"github.com/stretchr/testify/require"
	"github.com/umbracle/ethgo"
	"github.com/umbracle/ethgo/abi"
	"github.com/umbracle/ethgo/contract"
	"github.com/umbracle/ethgo/jsonrpc"
	"math/big"
	"os"
	"sort"
	"testing"
)

var registryAddr = ethgo.HexToAddress("0x6E63b3F24dABde0DDef88Fd0A59e86201D62971F")

func TestMumbaiDeploy(t *testing.T) {

	ec, err := jsonrpc.NewClient(os.Getenv("MUMBAI_URL"))
	require.NoError(t, err)

	sk, err := crypto.SKFromHex(os.Getenv("MUMBAI_SK"))
	require.NoError(t, err)
	skAddr := crypto.PubKeyToAddress(&sk.PublicKey)
	println(skAddr.String())
	k := &ecdsaKey{k: sk}

	loaded, err := loadArtifact(ec, "Registry.json", k, registryAddr)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	ret, err := loaded.Call("governance", ethgo.Latest)
	require.NoError(t, err)
	checkGovAddr, ok := ret["0"].(ethgo.Address)
	require.True(t, ok)
	require.Equal(t, skAddr, checkGovAddr)

	if false {
		deployed, addr, err := deployArtifact(ec, "Registry.json", k, []interface{}{skAddr})
		require.NoError(t, err)
		require.NotNil(t, deployed)
		require.NotNil(t, addr)
	}

	report := func(r string) {
		println(r)
	}
	d, err := MakeDeployer(ec, registryAddr, k, report)
	require.NoError(t, err)

	err = d.deployStakeManager()
	require.NoError(t, err)
}

// root chain proxy: 0x06c3909415918ddc86CC6e7f0219B57125926cE5
// stake token: 0xD47c2bC7c2E8E36884b53971Fb50F04537835E75
// slashing manager: 0xf369ED0563807FE4E51b4fe391C8C21A6386A96d
// staking info: 0x6F1e7093d19FCB5f31e7bA585BDff9Fc1071004A
// staking nft: 0x2381746Fe8DABF8a26837A0585C08f40328E3f37
// stake manager: 0xB2238d7f76EA36133c72C917a9Bb268223e74074
// stake manager proxy: 0x9d7652EaC859b36d08e807Fe99a15b4d5840D6fD

var stakeManagerAddr = ethgo.HexToAddress("0xB2238d7f76EA36133c72C917a9Bb268223e74074")
var stakeManagerProxyAddr = ethgo.HexToAddress("0x9d7652EaC859b36d08e807Fe99a15b4d5840D6fD")

var slashingManagerAddr = ethgo.HexToAddress("0xf369ED0563807FE4E51b4fe391C8C21A6386A96d")

var stakeTokenAddr = ethgo.HexToAddress("0xD47c2bC7c2E8E36884b53971Fb50F04537835E75")
var stakingNFTAddr = ethgo.HexToAddress("0x2381746Fe8DABF8a26837A0585C08f40328E3f37")
var stakingInfoAddr = ethgo.HexToAddress("0x6F1e7093d19FCB5f31e7bA585BDff9Fc1071004A")

var DYNASTY = big.NewInt(8)

func inspectContract(c *contract.Contract, out func(string)) {
	for _, v := range []string{"governance", "owner", "rootChain"} {
		if checkRet, err := c.Call(v, ethgo.Latest); err == nil {
			checkAddr, ok := checkRet["0"].(ethgo.Address)
			if ok {
				out(fmt.Sprintf("%v: %v", v, checkAddr.String()))
			}
		}
	}
}

func deriveStakerKeys(parent *ecdsa.PrivateKey) []*ecdsa.PrivateKey {
	stakerKeys := [2]*ecdsa.PrivateKey{}
	for i := int64(0); i < 2; i++ {
		ssk := new(big.Int).Add(parent.D, big.NewInt(i+1))
		stakerKeys[i], _ = crypto.SKFromHex(hex.EncodeToString(ssk.Bytes()))
	}
	return stakerKeys[:]
}

type sortKeysByAddress []*ecdsaKey

func (s *sortKeysByAddress) Len() int {
	return len(*s)
}

func (s *sortKeysByAddress) Less(i, j int) bool {
	return bytes.Compare((*s)[i].Address().Bytes(), (*s)[j].Address().Bytes()) < 0
}

func (s *sortKeysByAddress) Swap(i, j int) {
	temp := (*s)[i]
	(*s)[i] = (*s)[j]
	(*s)[j] = temp
}

var _ sort.Interface = &sortKeysByAddress{}

func TestStakeSetup(t *testing.T) {

	ec, err := jsonrpc.NewClient(os.Getenv("MUMBAI_URL"))
	require.NoError(t, err)

	sk, err := crypto.SKFromHex(os.Getenv("MUMBAI_SK"))
	require.NoError(t, err)
	skAddr := crypto.PubKeyToAddress(&sk.PublicKey)
	k := &ecdsaKey{k: sk}

	stakeManager, err := loadArtifact(ec, "StakeManager", k, stakeManagerAddr)
	require.NoError(t, err)

	out := func(s string) { println(s) }
	inspectContract(stakeManager, out)

	if false {
		var validatorShareFactoryAddr, auctionImplAddr ethgo.Address
		_, validatorShareFactoryAddr, err = deployArtifact(ec, "ValidatorShareFactory.json", k, nil)
		require.NoError(t, err)
		_, auctionImplAddr, err = deployArtifact(ec, "StakeManagerExtension.json", k, nil)
		require.NoError(t, err)
		println(validatorShareFactoryAddr.String())
		println(auctionImplAddr.String())

		err = TxnDoWait(stakeManager.Txn("initialize",
			registryAddr,
			skAddr,
			stakeTokenAddr,
			stakingNFTAddr,
			stakingInfoAddr,
			validatorShareFactoryAddr,
			skAddr,
			skAddr,
			auctionImplAddr,
		))
		require.NoError(t, err)

		err = TxnDoWait(stakeManager.Txn("updateDynastyValue", DYNASTY))
		require.NoError(t, err)

		err = TxnDoWait(stakeManager.Txn("updateCheckPointBlockInterval", big.NewInt(1)))
		require.NoError(t, err)

		// for some reason this little guy can't estimate gas ...?
		txn, err := stakeManager.Txn("changeRootChain", skAddr)
		if err == nil {
			txn.WithOpts(&contract.TxnOpts{
				GasLimit: 200_000,
			})
			err = TxnDoWait(txn, nil)
			require.NoError(t, err)
		}
	}

	stakeToken, err := loadArtifact(ec, "TestToken", k, stakeTokenAddr)
	require.NoError(t, err)
	inspectContract(stakeToken, out)

	totalAmount, _ := new(big.Int).SetString("12000000000000000000", 10) // 12 eth
	stakeAmount, _ := new(big.Int).SetString("10000000000000000000", 10) // 10 eth
	heimdallFee, _ := new(big.Int).SetString("2000000000000000000", 10) // 2 eth
	rewardsAmount, _ := new(big.Int).SetString("10000000000000000000", 10) // 100 eth

	err = TxnDoWait(stakeToken.Txn("mint", stakeManagerAddr, rewardsAmount))
	require.NoError(t, err)

	var stakerKeys []*ecdsaKey
	for i, k := range deriveStakerKeys(sk) {
		sk := &ecdsaKey{k: k}
		stakerKeys = append(stakerKeys, sk)
		out(fmt.Sprintf("staker address %v: %v", i, sk.Address().String()))
	}

	var c *contract.Contract

	for i := 0; i < len(stakerKeys); i++ {
		// mint from stake token to staker account
		err = TxnDoWait(stakeToken.Txn("mint", stakerKeys[i].Address(), totalAmount))
		require.NoError(t, err)

		// switch to stake token contract with staker key
		c = contract.NewContract(stakeTokenAddr, stakeToken.GetABI(),
			contract.WithJsonRPC(ec.Eth()),
			contract.WithSender(stakerKeys[i]),
		)
		// approve stake manager to transfer staked tokens
		err = TxnDoWait(c.Txn("approve", stakeManagerAddr, totalAmount))
		require.NoError(t, err)

		// switch to stake manager contract with staker key
		c = contract.NewContract(stakeManagerAddr, stakeManager.GetABI(),
			contract.WithJsonRPC(ec.Eth()),
			contract.WithSender(stakerKeys[i]),
		)

		// stake tokens to manager
		pk := stakerKeys[i].k.PublicKey
		err = TxnDoWait(c.Txn("stakeFor", stakerKeys[i].Address(), stakeAmount, heimdallFee, false,
			crypto.MarshalPublicKey(&pk)[1:]))
		require.NoError(t, err)
	}
}

func inspectValidatorContract(c *contract.Contract, out func(string)) {
	// currentValidatorSetSize
	if retSize, err := c.Call("currentValidatorSetSize", ethgo.Latest); err == nil {
		if sz, ok := retSize["0"].(*big.Int); ok {
			out(fmt.Sprintf("validator set size: %v", sz.String()))
		}
	}

	// currentValidatorSetTotalStake
	if retStake, err := c.Call("currentValidatorSetTotalStake", ethgo.Latest); err == nil {
		if st, ok := retStake["0"].(*big.Int); ok {
			out(fmt.Sprintf("validator set total stake: %v", st.String()))
		}
	}
}

func TestStakeSlashing(t *testing.T) {

	ec, err := jsonrpc.NewClient(os.Getenv("MUMBAI_URL"))
	require.NoError(t, err)

	sk, err := crypto.SKFromHex(os.Getenv("MUMBAI_SK"))
	require.NoError(t, err)
	// skAddr := crypto.PubKeyToAddress(&sk.PublicKey)
	k := &ecdsaKey{k: sk}

	var stakerKeys []*ecdsaKey
	for _, k := range deriveStakerKeys(sk) {
		sk := &ecdsaKey{k: k}
		stakerKeys = append(stakerKeys, sk)
	}

	sortKeys := sortKeysByAddress(stakerKeys)
	sort.Sort(&sortKeys)

	registry, err := loadArtifact(ec, "Registry.json", k, registryAddr)
	require.NoError(t, err)
	require.NotNil(t, registry)
	retCheckStake, err := registry.Call("getStakeManagerAddress", ethgo.Latest)
	require.NoError(t, err)
	checkState, ok := retCheckStake["0"].(ethgo.Address)
	require.True(t, ok && checkState == stakeManagerAddr)

	stakeManager, err := loadArtifact(ec, "StakeManager", k, stakeManagerAddr)
	require.NoError(t, err)

	out := func(s string) { println(s) }
	inspectValidatorContract(stakeManager, out)

	slashingManager, err := loadArtifact(ec, "SlashingManager", k, slashingManagerAddr)
	require.NoError(t, err)

	// function verifyConsensus(bytes32 voteHash, bytes memory sigs) public view returns (bool) {

	voteHash := [32]byte{0xDE, 0xAD, 0xBE, 0xEF}
	var catSigs []byte
	for _, k := range stakerKeys {
		sig, err := k.Sign(voteHash[:])
		require.NoError(t, err)
		catSigs = append(catSigs, sig...)

		retValidator, err := stakeManager.Call("signerToValidator", ethgo.Latest, k.Address())
		require.NoError(t, err)
		checkValidator, ok := retValidator["0"].(*big.Int)
		if ok {
			out(fmt.Sprintf("validator id %v, addr %v", checkValidator.String(), k.Address().String()))
		}
	}

	retVerify, err := slashingManager.Call("verifyConsensus", ethgo.Latest, voteHash, catSigs)
	require.NoError(t, err)
	checkVerify, ok := retVerify["0"].(bool)
	require.True(t, ok)
	require.True(t, checkVerify)
}

type slashData struct {
	ValidatorId *big.Int `abi:"validatorId"`
	Amount *big.Int `abi:"amount"`
	Jailed bool `abi:"jailed"`
}

func TestSlashEncode(t *testing.T) {

	// TODO: for some reason 'validatorid' has to lower-cased here - but didn't have to in other projects?
	et, err := abi.NewType("tuple(uint256 validatorid, uint256 amount, bool jailed)")
	require.NoError(t, err)

	sd := slashData{
		ValidatorId: big.NewInt(1),
		Amount:      big.NewInt(1000),
		Jailed:      false,
	}
	encSlashData, err := abi.Encode(&sd, et)
	require.NoError(t, err)
	println(hex.EncodeToString(encSlashData))

}