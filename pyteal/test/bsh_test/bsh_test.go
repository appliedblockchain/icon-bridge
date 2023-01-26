package tests

import (
	"log"
	"path/filepath"
	"testing"

	"appliedblockchain.com/icon-bridge/config"
	contracts "appliedblockchain.com/icon-bridge/contracts"
	bmcmethods "appliedblockchain.com/icon-bridge/contracts/methods/bmc"
	bshmethods "appliedblockchain.com/icon-bridge/contracts/methods/bsh"
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
var bshContract *abi.Contract
var bshMcp future.AddMethodCallParams
var err error

const bmcIcon = "btp://0x1.icon/0x12333";


func Test_Init(t *testing.T) {
	client, deployer, txParams = tools.Init(t)

	bshAppId = tools.BshTestInit(t, client, config.BshTealDir, deployer, txParams)
	bmcAppId = tools.BmcTestInit(t, client, config.BmcTealDir, deployer, txParams)

	bmcContract, bmcMcp, err = contracts.InitABIContract(client, deployer, filepath.Join(config.BmcTealDir, "contract.json"), bmcAppId)

	if err != nil {
		t.Fatalf("Failed to init BMC contract: %+v", err)
	}

	bshContract, bshMcp, err = contracts.InitABIContract(client, deployer, filepath.Join(config.BshTealDir, "contract.json"), bshAppId)

	if err != nil {
		t.Fatalf("Failed to init BSH contract: %+v", err)
	}
}

func Test_BshSendServiceMessage(t *testing.T) {
	_, err = bmcmethods.RegisterBSHContract(client, bshAppId, bmcContract, bmcMcp)

	if err != nil {
		t.Fatalf("Failed to execute RegisterBSHContract: %+v", err)
	}

	_, err = bshmethods.SendServiceMessage(client, bshContract, bshMcp, []interface{}{bmcAppId, bmcIcon})

	if err != nil {
		t.Fatalf("Failed to execute SendServiceMessage: %+v", err)
	}
}

func Test_GetMessagePushedFromBmcToRelayer(t *testing.T) {
	round := tools.GetLatestRound(t, client)

	newBlock := tools.GetBlock(t, client, round)

	for _, stxn := range newBlock.Payset {
		for _, innertxn := range stxn.EvalDelta.InnerTxns {
			for _, l := range innertxn.EvalDelta.Logs {
				tupleType, err := abi.TypeOf("(string,string,string,uint64,byte[])")

				if err != nil {
					t.Fatalf("Failed to get ABI type: %+v", err)
				}

				decoded, err := tupleType.Decode([]byte(l))

				if err != nil {
					t.Fatalf("Failed to decode tuple type: %+v", err)
				}

				type BMCMessage struct {
					Src     string //  an address of BMC (i.e. btp://1234.PARA/0x1234)
					Dst     string //  an address of destination BMC
					Svc     string //  service name of BSH
					Sn      uint64 //  sequence number of BMC
					Message []byte //  serialized Service Message from BSH
				}

				var bmcMessage BMCMessage

				if val, ok := decoded.([]interface{}); ok {
					var byteArray []byte

					for _, v := range val[4].([]interface{}) {
						byteArray = append(byteArray, byte(v.(uint8)))
					}
				
					bmcMessage.Src = val[0].(string)
					bmcMessage.Dst = val[1].(string)
					bmcMessage.Svc = val[2].(string)
					bmcMessage.Sn = val[3].(uint64)
					bmcMessage.Message = byteArray
				}

				log.Print(bmcMessage)
			}
		}
	}
}
