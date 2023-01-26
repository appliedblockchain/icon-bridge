package testtools

import (
	"testing"

	"github.com/algorand/avm-abi/abi"
)

func GetAbiType(t *testing.T, arc4Type string) abi.Type {
	abiType, err := abi.TypeOf(arc4Type)

	if err != nil {
		t.Fatalf("Failed to get ABI type: %+v", err)
	}

	return abiType
}