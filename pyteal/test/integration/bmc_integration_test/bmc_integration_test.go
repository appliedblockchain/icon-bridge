package tests

import (
	"path/filepath"
	"testing"

	"appliedblockchain.com/icon-bridge/config"
	contracts "appliedblockchain.com/icon-bridge/contracts"
	tools "appliedblockchain.com/icon-bridge/testtools"
	"github.com/algorand/go-algorand-sdk/abi"
	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/future"
	"github.com/algorand/go-algorand-sdk/types"
)

var client *algod.Client
var deployer crypto.Account
var txParams types.SuggestedParams
var bshAppId uint64
var bmcAppId uint64
var bmcContract *abi.Contract
var bmcMcp future.AddMethodCallParams
var err error

func Test_Init(t *testing.T) {
	client, deployer, txParams = tools.Init(t)

	bshAppId = tools.BshTestInit(t, client, config.BshTealDir, deployer, txParams)
	bmcAppId = tools.BmcTestInit(t, client, config.BmcTealDir, deployer, txParams)

	bmcContract, bmcMcp, err = contracts.InitABIContract(client, deployer, filepath.Join(config.BmcTealDir, "contract.json"), bmcAppId)

	if err != nil {
		t.Fatalf("Failed to init ABI contract: %+v", err)
	}
}
