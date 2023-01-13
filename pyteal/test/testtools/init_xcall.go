package testtools

import (
	"fmt"
	"testing"

	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/types"
)

func XcallTestInit(t *testing.T, client *algod.Client, tealDir string, deployer crypto.Account,
	txParams types.SuggestedParams,
) uint64 {
	t.Helper()

	appCreationTx := MakeXcallDeployTx(t, client, tealDir, deployer, txParams)
	creationTxId := SendTransaction(t, client, deployer.PrivateKey, appCreationTx)
	deployRes := WaitForConfirmationsT(t, client, []string{creationTxId})

	fmt.Printf("xcall App ID: %d \n", deployRes[0].ApplicationIndex)
	return deployRes[0].ApplicationIndex
}
