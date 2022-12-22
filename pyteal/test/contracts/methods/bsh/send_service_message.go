package contracts

import (
	"context"
	"fmt"

	"appliedblockchain.com/icon-bridge/config"
	tools "appliedblockchain.com/icon-bridge/contracts/tools"
	"github.com/algorand/go-algorand-sdk/abi"
	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/future"
)

func SendServiceMessage(client *algod.Client, bmcId uint64, bshContract *abi.Contract, mcp future.AddMethodCallParams) (ret future.ExecuteResult, err error) {
	var atc = future.AtomicTransactionComposer{}

	err = atc.AddMethodCall(tools.CombineMethod(mcp, tools.GetMethod(bshContract, "sendServiceMessage"), []interface{}{bmcId, "ICON", "TOKEN_TRANSFER_SERVICE", 3}))

	if err != nil {
		fmt.Printf("Failed to add method SendServiceMessage call into BSH contract: %+v \n", err)
		return
	}

	ret, err = atc.Execute(client, context.Background(), config.TransactionWaitRounds)

	if err != nil {
		fmt.Printf("Failed to execute call: %+v \n", err)
		return
	}

	return
}
