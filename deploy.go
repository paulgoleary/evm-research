package hub_research

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/paulgoleary/hub-research/crypto"
	"github.com/umbracle/ethgo"
	"github.com/umbracle/ethgo/abi"
	"github.com/umbracle/ethgo/compiler"
	"github.com/umbracle/ethgo/contract"
	"github.com/umbracle/ethgo/jsonrpc"
	"os"
	"path/filepath"
	"strings"
)

type jsonArtifact struct {
	Bytecode string          `json:"bytecode"`
	Abi      json.RawMessage `json:"abi"`
}

func getBuildArtifact(name string) (art *compiler.Artifact, err error) {

	if !strings.HasSuffix(name, ".json") {
		name = name + ".json"
	}
	var jsonBytes []byte
	if jsonBytes, err = os.ReadFile(filepath.Join("build/contracts", name)); err != nil {
		return
	}
	var jart jsonArtifact
	if err = json.Unmarshal(jsonBytes, &jart); err != nil {
		return
	}

	bc := jart.Bytecode
	if !strings.HasPrefix(bc, "0x") {
		bc = "0x" + bc
	}
	art = &compiler.Artifact{
		Abi: string(jart.Abi),
		Bin: bc,
	}
	return
}

type ecdsaKey struct {
	k *ecdsa.PrivateKey
}

func (e *ecdsaKey) Address() ethgo.Address {
	return ethgo.Address(crypto.PubKeyToAddress(&e.k.PublicKey))
}

func (e *ecdsaKey) Sign(hash []byte) ([]byte, error) {
	return crypto.Sign(e.k, hash)
}

var _ ethgo.Key = &ecdsaKey{}

func loadArtifact(ec *jsonrpc.Client, name string, withKey ethgo.Key, addr ethgo.Address) (loaded *contract.Contract, err error) {
	var art *compiler.Artifact
	if art, err = getBuildArtifact(name); err != nil {
		return
	}
	var theAbi *abi.ABI
	if theAbi, err = abi.NewABI(art.Abi); err != nil {
		return
	}

	loaded = contract.NewContract(addr, theAbi,
		contract.WithJsonRPC(ec.Eth()),
		contract.WithSender(withKey),
	)

	return
}

func deployArtifact(ec *jsonrpc.Client, name string, withKey ethgo.Key, args []interface{}) (deployed *contract.Contract, addr ethgo.Address, err error) {
	var art *compiler.Artifact
	if art, err = getBuildArtifact(name); err != nil {
		return
	}
	var theAbi *abi.ABI
	if theAbi, err = abi.NewABI(art.Abi); err != nil {
		return
	}

	var rcpt *ethgo.Receipt
	var artBin []byte
	if artBin, err = hex.DecodeString(strings.TrimPrefix(art.Bin, "0x")); err != nil {
		return
	}
	var txn contract.Txn
	if txn, err = contract.DeployContract(theAbi, artBin, args,
		contract.WithJsonRPC(ec.Eth()), contract.WithSender(withKey)); err != nil {
		return
	} else {
		if err = txn.Do(); err != nil {
			return
		}
		if rcpt, err = txn.Wait(); err != nil {
			return
		}
	}

	deployed = contract.NewContract(rcpt.ContractAddress, theAbi,
		contract.WithJsonRPC(ec.Eth()),
		contract.WithSender(withKey),
	)
	addr = rcpt.ContractAddress

	return
}

type Deployer struct {
	ec *jsonrpc.Client
	k  ethgo.Key

	registryAddr     ethgo.Address
	registryContract *contract.Contract

	report func(string)
}

func MakeDeployer(ec *jsonrpc.Client, registerAddr ethgo.Address, k ethgo.Key, maybeReport func(string)) (d *Deployer, err error) {
	d = &Deployer{
		ec:           ec,
		k:            k,
		registryAddr: registerAddr,
		report:       maybeReport,
	}
	d.registryContract, err = loadArtifact(d.ec, "Registry.json", d.k, d.registryAddr)
	return
}

func (d *Deployer) checkRegistry(key string) (ethgo.Address, error) {
	checkMapRet, err := d.registryContract.Call("contractMap", ethgo.Latest, ethgo.Keccak256([]byte("eventsHub")))
	if addr, ok := checkMapRet["0"].(ethgo.Address); err == nil && ok && addr != ethgo.ZeroAddress {
		// already deployed and registered...
		return addr, nil
	}
	return ethgo.ZeroAddress, err
}

func (d *Deployer) deployEventsHub() error {

	if addr, err := d.checkRegistry("contractMap"); err != nil {
		return err
	} else if addr != ethgo.ZeroAddress {
		// already deployed and registered...
		return nil
	}

	eventsHubImpl, eventsHubAddr, err := deployArtifact(d.ec, "EventsHub.json", d.k, nil)
	if err != nil {
		return err
	}
	proxy, proxyAddr, err := deployArtifact(d.ec, "EventsHubProxy.json", d.k, []interface{}{ethgo.ZeroAddress})
	if err != nil {
		return err
	}

	encInit, err := eventsHubImpl.GetABI().GetMethod("initialize").Encode([]interface{}{d.registryAddr})
	if err != nil {
		return err
	}
	txn, err := proxy.Txn("updateAndCall", eventsHubAddr, encInit)
	if err != nil {
		return err
	}
	if err = txn.Do(); err != nil {
		return err
	}
	rcptInit, err := txn.Wait()
	if err != nil {
		return err
	}
	_ = rcptInit

	return d.updateContractMap(ethgo.Keccak256([]byte("eventsHub")), proxyAddr)
}

func (d *Deployer) updateContractMap(key []byte, addr ethgo.Address) error {

	txn, err := d.registryContract.Txn("updateContractMap", key, addr)
	if err != nil {
		return err
	}
	if err = txn.Do(); err != nil {
		return err
	}
	rcptUpdate, err := txn.Wait()
	if err != nil {
		return err
	}
	if d.report != nil {
		d.report(fmt.Sprintf("update registry contract map, key %v, value %v, hash %v",
			hex.EncodeToString(key), addr.String(), rcptUpdate.TransactionHash.String()))
	}
	return nil
}

func (d *Deployer) deployStakeManager() (err error) {

	if err = d.deployEventsHub(); err != nil {
		return
	}

	var validatorShareFactoryAddr, validatorShareAddr ethgo.Address

	if _, validatorShareFactoryAddr, err = deployArtifact(d.ec, "ValidatorShareFactory.json", d.k, nil); err != nil {
		return
	}
	if _, validatorShareAddr, err = deployArtifact(d.ec, "ValidatorShare.json", d.k, nil); err != nil {
		return
	}

	var rootChainAddr, rootChainProxyAddr ethgo.Address
	if _, rootChainAddr, err = deployArtifact(d.ec, "RootChain.json", d.k, nil); err != nil {
		return
	}
	if _, rootChainProxyAddr, err = deployArtifact(d.ec, "RootChainProxy.json", d.k,
		[]interface{}{rootChainAddr, d.registryAddr, "heimdall-P5rXwg"}); err != nil {
		return err
	}

	// TODO: change root chain contract to proxy
	// this.rootChain = await contracts.RootChain.at(rootChainProxy.address)

	var stakingNFT *contract.Contract
	var stakingInfoAddr, stakeTokenAddr, stakingNFTAddr ethgo.Address

	if _, stakingInfoAddr, err = deployArtifact(d.ec, "StakingInfo.json", d.k, []interface{}{d.registryAddr}); err != nil {
		return
	}
	if _, stakeTokenAddr, err = deployArtifact(d.ec, "TestToken.json", d.k,
		[]interface{}{"Stake Token", "STEAK"}); err != nil {
		return
	}
	if stakingNFT, stakingNFTAddr, err = deployArtifact(d.ec, "StakingNFT.json", d.k,
		[]interface{}{"Test Validator", "TV"}); err != nil {
		return
	}

	var stakeManager, stakeManagerProxy *contract.Contract
	var stakeManagerAddr, stakeManagerProxyAddr, auctionImplAddr ethgo.Address

	if stakeManager, stakeManagerAddr, err = deployArtifact(d.ec, "StakeManagerTestable.json", d.k, nil); err != nil {
		return
	}
	if stakeManagerProxy, stakeManagerProxyAddr, err = deployArtifact(d.ec, "StakeManagerProxy.json", d.k, []interface{}{ethgo.ZeroAddress}); err != nil {
		return
	}
	if _, auctionImplAddr, err = deployArtifact(d.ec, "StakeManagerExtension.json", d.k, nil); err != nil {
		return
	}

	encInit, err := stakeManager.GetABI().GetMethod("initialize").Encode(
		[]interface{}{
			d.registryAddr,            // this.registry.address,
			d.k.Address(),             // rootChainOwner.getAddressString(),
			stakeTokenAddr,            // this.stakeToken.address,
			stakingNFTAddr,            // this.stakingNFT.address,
			stakingInfoAddr,           // this.stakingInfo.address,
			validatorShareFactoryAddr, // this.validatorShareFactory.address,
			d.k.Address(),             // this.governance.address,
			d.k.Address().String(),    // wallets[0].getAddressString(),
			auctionImplAddr,           // auctionImpl.address
		})
	if err != nil {
		return err
	}
	// TODO: this doesn't fail but also doesn't seem to call through to initialize
	txn, err := stakeManagerProxy.Txn("updateAndCall", stakeManagerAddr, encInit)
	if err != nil {
		return err
	}
	if err = txn.Do(); err != nil {
		return err
	}
	rcptInit, err := txn.Wait()
	if err != nil {
		return err
	}
	_ = rcptInit

	// this.stakeManager = await contracts.StakeManagerTestable.at(proxy.address)
	if stakeManager, err = loadArtifact(d.ec, "StakeManagerTestable.json", d.k, stakeManagerProxyAddr); err != nil {
		return
	}

	if stakeManager, err = d.buildStakeManagerObject(stakeManager); err != nil {
		return
	}

	//     this.slashingManager = await contracts.SlashingManager.new(this.registry.address, this.stakingInfo.address, 'heimdall-P5rXwg')
	var slashingManagerAddr ethgo.Address
	if _, slashingManagerAddr, err = deployArtifact(d.ec, "SlashingManager.json", d.k,
		[]interface{}{d.registryAddr, stakingInfoAddr, "heimdall-P5rXwg"}); err != nil {
		return
	}

	// await this.stakingNFT.transferOwnership(this.stakeManager.address)
	if txn, err = stakingNFT.Txn("transferOwnership", stakeManagerAddr); err != nil {
		return
	}
	if txn.Do(); err != nil {
		return
	}
	if _, err = txn.Wait(); err != nil {
		return
	}

	if err = d.updateContractMap(ethgo.Keccak256([]byte("stakeManager")), stakeManagerAddr); err != nil {
		return
	}
	if err = d.updateContractMap(ethgo.Keccak256([]byte("validatorShare")), validatorShareAddr); err != nil {
		return
	}
	if err = d.updateContractMap(ethgo.Keccak256([]byte("slashingManager")), slashingManagerAddr); err != nil {
		return
	}

	d.report(fmt.Sprintf("root chain proxy: %v", rootChainProxyAddr.String()))
	d.report(fmt.Sprintf("stake manager: %v", stakeManagerAddr.String()))
	d.report(fmt.Sprintf("stake token: %v", stakeTokenAddr.String()))
	d.report(fmt.Sprintf("slashing manager: %v", slashingManagerAddr.String()))
	d.report(fmt.Sprintf("staking info: %v", stakingInfoAddr.String()))
	d.report(fmt.Sprintf("staking nft: %v", stakingNFTAddr.String()))
	d.report(fmt.Sprintf("stake manager: %v", stakeManagerAddr.String()))
	d.report(fmt.Sprintf("stake manager proxy: %v", stakeManagerProxyAddr.String()))

	return nil
}

func (d *Deployer) buildStakeManagerObject(stakeManager *contract.Contract) (*contract.Contract, error) {
	// in the js impl this method wraps methods of the stake manager to be proxied through a governance contract
	// since - for convenience - we don't use a governance contract, we can just call staking manager methods directly.
	// FWIW, this method-level proxying is easier to do in js - not sure what the best way to do it in Go would be ...?
	return stakeManager, nil
}
