package tests

import (
	"encoding/binary"
	"fmt"
	"log"
	"path/filepath"
	"testing"

	"appliedblockchain.com/icon-bridge/algorand"
	"appliedblockchain.com/icon-bridge/config"
	contracts "appliedblockchain.com/icon-bridge/contracts"
	xcallmethods "appliedblockchain.com/icon-bridge/contracts/methods/xcall"
	tools "appliedblockchain.com/icon-bridge/testtools"
	"github.com/algorand/go-algorand-sdk/abi"
	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/future"
	"github.com/algorand/go-algorand-sdk/types"
)

var client *algod.Client
var deployer crypto.Account
var fundingAccount crypto.Account
var txParams types.SuggestedParams
var xcallAppId uint64
var bmcAppId uint64
var bmcContract *abi.Contract
var bmcMcp future.AddMethodCallParams
var xcallContract *abi.Contract
var xcallMcp future.AddMethodCallParams
var err error

func Test_Init(t *testing.T) {
	client, deployer, fundingAccount, txParams = tools.Init(t)

	xcallAppId = tools.XcallTestInit(t, client, config.XcallTealDir, deployer, txParams)
	bmcAppId = tools.BmcTestInit(t, client, config.BmcTealDir, deployer, txParams)

	bmcContract, bmcMcp, err = contracts.InitABIContract(client, deployer, filepath.Join(config.BmcTealDir, "contract.json"), bmcAppId)
	
	if err != nil {
		t.Fatalf("Failed to init ABI contract: %+v", err)
	}
	
	xcallContract, xcallMcp, err = contracts.InitABIContract(client, deployer, filepath.Join(config.XcallTealDir, "contract.json"), xcallAppId)

	if err != nil {
		t.Fatalf("Failed to init ABI contract: %+v", err)
	}
}

func Test_SendMessageFromDApp(t *testing.T) {
	xcallAddress := crypto.GetApplicationAddress(xcallAppId)
	txnIds := tools.TransferAlgos(t, client, txParams, fundingAccount, []types.Address{xcallAddress}, 141700)
	tools.WaitForConfirmationsT(t, client, txnIds)

	to := "btp://0x3.icon/cx10f228c2372abf4517685526317a7e43eed1bf57"
	data := []byte{97, 98, 99, 100, 101, 102}
	rollback := []byte{97, 98, 99, 100, 101, 102}

	lastSnBytes := tools.GetGlobalStateByKey(t, client, xcallAppId, "last_sn")
	lastSn := binary.BigEndian.Uint64(lastSnBytes)
	boxName := make([]byte, 8)
	binary.BigEndian.PutUint64(boxName[0:], lastSn + 1)

	xcallMcp.BoxReferences = []types.AppBoxReference{{AppID: xcallAppId, Name: boxName}}
	_, err := xcallmethods.SendCallMessage(client, xcallContract, xcallMcp, to, data, rollback)

	if err != nil {
		t.Fatalf("Failed to add method call: %+v", err)
	}
}


func Test_GetMessagePushedFromXcall(t *testing.T) {
	round := tools.GetLatestRound(t, client)

	newBlock := tools.GetBlock(t, client, round)

	txns := algorand.GetTxns(&newBlock, xcallAppId)

	if txns == nil {
		t.Fatalf("No txns containing btp msgs")
	}

	for _, txn := range *txns {
		addressType := tools.GetAbiType(t, "address")
		stringType := tools.GetAbiType(t, "string")
		byteArrayType := tools.GetAbiType(t, "byte[]")
		booleanType := tools.GetAbiType(t, "bool")

		tupleType, err := abi.MakeTupleType([]abi.Type{addressType, stringType, byteArrayType, booleanType})
		
		if err != nil {
			t.Fatalf("Failed to get ABI type: %+v", err)
		}

		fmt.Print(tupleType.)

		log.Printf("%+v\n", txn.EvalDelta.Logs[0])
	}
}
