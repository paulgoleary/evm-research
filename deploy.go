package evm_research

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"github.com/paulgoleary/evm-research/crypto"
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
