package hub_research

import (
	"github.com/paulgoleary/hub-research/crypto"
	"github.com/stretchr/testify/require"
	"github.com/umbracle/ethgo"
	"github.com/umbracle/ethgo/contract"
	"github.com/umbracle/ethgo/jsonrpc"
	"os"
	"testing"
)

var ethPOLTokenAddr = ethgo.HexToAddress("0x455e53CBB86018Ac2B8092FdCd39d8444aFFC3F6")
var ethPOLMigrateAddr = ethgo.HexToAddress("0x29e7DF7b6A1B2b07b731457f499E1696c60E2C4e")
var ethPOLEmissionAddr = ethgo.HexToAddress("0xbC9f74b3b14f460a6c47dCdDFd17411cBc7b6c53")
var ethMATICAddr = ethgo.HexToAddress("0x7D1AfA7B718fb893dB30A3aBc0Cfc608AaCfeBB0")

func TestPOLMigrate(t *testing.T) {
	ecEth, err := jsonrpc.NewClient(os.Getenv("ETH_URL"))
	require.NoError(t, err)

	sk, err := crypto.SKFromHex(os.Getenv("ETH_BASE_SK"))
	require.NoError(t, err)
	k := &ecdsaKey{k: sk}

	matic, err := loadArtifact(ecEth, "pol-token/out/ERC20.sol/ERC20", k, ethMATICAddr)
	require.NoError(t, err)
	_ = matic

	pol, err := loadArtifact(ecEth, "pol-token/out/PolygonEcosystemToken.sol/PolygonEcosystemToken", k, ethPOLTokenAddr)
	require.NoError(t, err)

	migrate, err := loadArtifact(ecEth, "pol-token/out/PolygonMigration.sol/PolygonMigration", k, ethPOLMigrateAddr)
	require.NoError(t, err)

	res, err := pol.Call("decimals", ethgo.Latest)
	require.NoError(t, err)
	_ = res

	//err = TxnDoWait(matic.Txn("approve",
	//	ethPOLMigrateAddr,
	//	ethgo.Ether(10),
	//))
	//require.NoError(t, err)

	// function migrate(uint256 amount) external {
	txn, _ := migrate.Txn("migrate", ethgo.Ether(10))
	txn.WithOpts(&contract.TxnOpts{GasPrice: 40_000_000_000})
	err = TxnDoWait(txn, nil)
	require.NoError(t, err)

	res, err = pol.Call("balanceOf", ethgo.Latest, k.Address())
	require.NoError(t, err)
	_ = res

}
