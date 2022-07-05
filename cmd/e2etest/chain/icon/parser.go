package icon

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/icon-project/icon-bridge/cmd/e2etest/chain"
)

type parser struct {
	addressToContractName map[string]chain.ContractName
}

func NewParser(nameToAddr map[chain.ContractName]string) (*parser, error) {
	addrToName := map[string]chain.ContractName{}
	for name, addr := range nameToAddr {
		addrToName[addr] = name
	}
	return &parser{addressToContractName: addrToName}, nil
}

func (p *parser) Parse(log *TxnEventLog) (resLog interface{}, eventType chain.EventLogType, err error) {
	eventName := strings.Split(log.Indexed[0], "(")
	eventType = chain.EventLogType(strings.TrimSpace(eventName[0]))
	if eventType == chain.TransferStart {
		resLog, err = parseTransferStart(log)
	} else if eventType == chain.TransferReceived {
		resLog, err = parseTransferReceived(log)
	} else if eventType == chain.TransferEnd {
		resLog, err = parseTransferEnd(log)
	} else {
		err = fmt.Errorf("No matching signature for event log of type %v generated by contract address %v", eventType, log.Addr)
	}
	return
}

func rlpDecodeHex(str string, out interface{}) error {
	if strings.HasPrefix(str, "0x") {
		str = str[2:]
	}
	input, err := hex.DecodeString(str)
	if err != nil {
		return errors.Wrap(err, "hex.DecodeString ")
	}
	err = rlp.Decode(bytes.NewReader(input), out)
	if err != nil {
		return errors.Wrap(err, "rlp.Decode ")
	}
	return nil
}

func parseTransferStart(log *TxnEventLog) (*chain.TransferStartEvent, error) {
	if len(log.Data) != 3 {
		return nil, fmt.Errorf("Unexpected length of log.Data. Got %d. Expected 3", len(log.Data))
	}
	data := log.Data
	res := &[]chain.AssetTransferDetails{}
	err := rlpDecodeHex(data[len(data)-1], res)
	if err != nil {
		return nil, errors.Wrap(err, "rlpDecodeHex ")
	}
	sn := new(big.Int)
	if strings.HasPrefix(data[1], "0x") {
		data[1] = data[1][2:]
	}
	sn.SetString(data[1], 16)
	ts := &chain.TransferStartEvent{
		From:   log.Indexed[1],
		To:     data[0],
		Sn:     sn,
		Assets: *res,
	}
	return ts, nil
}

func parseTransferReceived(log *TxnEventLog) (*chain.TransferReceivedEvent, error) {
	if len(log.Data) != 2 || len(log.Indexed) != 3 {
		return nil, fmt.Errorf("Unexpected length. Got %v and %v. Expected 2 and 3", len(log.Data), len(log.Indexed))
	}
	data := log.Data
	res := &[]chain.AssetDetails{}
	err := rlpDecodeHex(data[len(data)-1], res)
	if err != nil {
		return nil, errors.Wrap(err, "rlp.DecodeHex ")
	}
	sn := new(big.Int)
	if strings.HasPrefix(data[0], "0x") {
		data[0] = data[0][2:]
	}
	sn.SetString(data[0], 16)
	newAssetDetails := make([]chain.AssetTransferDetails, len(*res))
	for i, v := range *res {
		newAssetDetails[i].Name = v.Name
		newAssetDetails[i].Value = v.Value
	}
	ts := &chain.TransferReceivedEvent{
		From:   log.Indexed[1],
		To:     log.Indexed[2],
		Sn:     sn,
		Assets: newAssetDetails,
	}
	return ts, nil
}

func parseTransferEnd(log *TxnEventLog) (*chain.TransferEndEvent, error) {
	data := log.Data
	sn := new(big.Int)
	if strings.HasPrefix(data[0], "0x") {
		data[0] = data[0][2:]
	}
	sn.SetString(data[0], 16)

	cd := new(big.Int)
	if strings.HasPrefix(data[1], "0x") {
		data[1] = data[1][2:]
	}
	cd.SetString(data[1], 16)
	te := &chain.TransferEndEvent{
		From: log.Indexed[1],
		Sn:   sn,
		Code: cd,
	}
	return te, nil
}
