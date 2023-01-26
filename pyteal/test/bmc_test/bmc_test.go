package tests

import (
	"bytes"
	"path/filepath"
	"testing"

	"appliedblockchain.com/icon-bridge/config"
	contracts "appliedblockchain.com/icon-bridge/contracts"
	bmcmethods "appliedblockchain.com/icon-bridge/contracts/methods/bmc"
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

func Test_RelayerAsDeployer(t *testing.T) {
	appRelayerAddress := tools.GetGlobalStateByKey(t, client, bmcAppId, "relayer_acc_address")

	if !bytes.Equal(appRelayerAddress, deployer.Address[:]) {
		t.Fatal("Failed to align relayer address to address in global state of BMC application")
	}
}

func Test_RegisterBSHContract(t *testing.T) {
	_, err = bmcmethods.RegisterBSHContract(client, bshAppId, bmcContract, bmcMcp)

	if err != nil {
		t.Fatalf("Failed to add method call: %+v", err)
	}
	
	bshAddress := crypto.GetApplicationAddress(bshAppId)
	globalBshAddress := tools.GetGlobalStateByKey(t, client, bmcAppId, "bsh_app_address")

	if !bytes.Equal(globalBshAddress, bshAddress[:]) {
		t.Fatal("Failed to align BSH address to address in global state of BMC application")
	}
}

func Test_CallSendMessageFromOutsideOfBsh(t *testing.T) {
	_, err = bmcmethods.SendMessage(client, bmcContract, bmcMcp, []interface{}{"0x1.icon", "bts", 2, []byte("hello world")})

	if err == nil {
		t.Fatal("SendMessage should throw error, as it's not been called from BSH contract")
	}
}