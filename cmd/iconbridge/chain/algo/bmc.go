package algo

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/algorand/go-algorand-sdk/abi"
	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/future"
	"github.com/algorand/go-algorand-sdk/types"
)

const contractDir = "bmc/contract.json"
const approvalPath = "bmc/approval.teal"
const clearPath = "bmc/clear.teal"

func getMethod(c *abi.Contract, name string) (abi.Method, error) {
	m, err := c.GetMethodByName(name)
	if err != nil {
		return abi.Method{}, err
	}
	return m, nil
}

func combine(mcp future.AddMethodCallParams, m abi.Method,
	a []interface{}) future.AddMethodCallParams {
	mcp.Method = m
	mcp.MethodArgs = a
	return mcp
}

func deployBmc(algodAccess []string, account crypto.Account) (uint64, error) {
	client, err := algod.MakeClient(algodAccess[0], algodAccess[1])
	if err != nil {
		return 0, fmt.Errorf("Bmc couldn't create algod: %w", err)
	}
	params, err := client.SuggestedParams().Do(context.Background())
	if err != nil {
		return 0, fmt.Errorf("Error getting params: %w", err)
	}
	approvalProgram, err := compileTeal(client, approvalPath)
	if err != nil {
		return 0, fmt.Errorf("Approval compile err: %w", err)
	}
	clearProgram, err := compileTeal(client, clearPath)
	if err != nil {
		return 0, fmt.Errorf("Clear compile err: %w", err)
	}
	txn, err := future.MakeApplicationCreateTx(
		false,
		approvalProgram,
		clearProgram,
		types.StateSchema{NumUint: 4, NumByteSlice: 4},
		types.StateSchema{NumUint: 4, NumByteSlice: 4},
		[][]byte{},
		nil,
		nil,
		nil,
		params,
		account.Address,
		nil,
		types.Digest{},
		[32]byte{},
		types.Address{},
	)
	if err != nil {
		return 0, fmt.Errorf("Failed to make bmc: %w", err)
	}
	txID, signedTxn, err := crypto.SignTransaction(account.PrivateKey, txn)
	if err != nil {
		return 0, fmt.Errorf("Failed to sign transaction: %w", err)
	}
	_, err = client.SendRawTransaction(signedTxn).Do(context.Background())
	if err != nil {
		return 0, fmt.Errorf("Failed to send transaction: %w", err)
	}
	deployRes, err := future.WaitForConfirmation(client, txID, 4, context.Background())
	if err != nil {
		return 0, fmt.Errorf("Error waiting for confirmation: %w", err)
	}
	return deployRes.ApplicationIndex, nil
}

func compileTeal(client *algod.Client, filePath string) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return []byte{}, err
	}
	compileResponse, err := client.TealCompile(content).Do(context.Background())
	if err != nil {
		return []byte{}, err
	}

	decodedProgram, err := base64.StdEncoding.DecodeString(compileResponse.Result)
	if err != nil {
		return []byte{}, err
	}
	return decodedProgram, nil
}
