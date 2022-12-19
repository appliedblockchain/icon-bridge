package algo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/algorand/go-algorand-sdk/types"
	"github.com/icon-project/icon-bridge/cmd/iconbridge/chain"
	"github.com/icon-project/icon-bridge/common"
	"github.com/icon-project/icon-bridge/common/log"
	"github.com/pkg/errors"
)

// TODO adjust settings for algo
const (
	MonitorBlockMaxConcurrency = 300 // number of concurrent requests to synchronize older blocks from source chain
	RPCCallRetry               = 5
)

func NewReceiver(
	src, dst chain.BTPAddress,
	algodAccess []string,
	rawOpts json.RawMessage, l log.Logger) (chain.Receiver, error) {
	r := &receiver{
		log: l,
		src: src,
		dst: dst,
	}

	if len(algodAccess) < 2 {
		return nil, fmt.Errorf("Invalid algorand credentials")
	}

	err := json.Unmarshal(rawOpts, &r.opts)
	if err != nil {
		return nil, err
	}
	if r.opts.SyncConcurrency < 1 {
		r.opts.SyncConcurrency = 1
	} else if r.opts.SyncConcurrency > MonitorBlockMaxConcurrency {
		r.opts.SyncConcurrency = MonitorBlockMaxConcurrency
	}

	r.cl, err = newClient(algodAccess, r.log)
	if err != nil {
		return nil, err
	}
	return r, nil

}

type ReceiverOptions struct {
	SyncConcurrency uint64           `json:"syncConcurrency"`
	Verifier        *VerifierOptions `json:"verifier"`
}

// TODO move struct to verifier.go
type VerifierOptions struct {
	BlockHeight   uint64          `json:"blockHeight"`
	BlockHash     common.HexBytes `json:"parentHash"`
	ValidatorData common.HexBytes `json:"validatorData"`
}

func (opts *ReceiverOptions) Unmarshal(v map[string]interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, opts)
}

type receiver struct {
	log  log.Logger
	src  chain.BTPAddress
	dst  chain.BTPAddress
	opts ReceiverOptions
	cl   *Client
}

func (r *receiver) Subscribe(
	ctx context.Context, msgCh chan<- *chain.Message,
	subOpts chain.SubscribeOptions) (errCh <-chan error, err error) {

	subOpts.Seq++
	_errCh := make(chan error)

	if subOpts.Seq <= 0 || subOpts.Height <= 0 {
		return _errCh, errors.New("receiveLoop: invalid options: <nil>")
	}

	//TODO add verifier logic

	curRound := subOpts.Height
	latestRound, err := r.cl.GetLatestRound(ctx)
	if err != nil {
		r.log.WithFields(log.Fields{"error": err}).Error("receiveLoop: error failed to getLatestRound-")
		return _errCh, err
	}

	if err != nil {
		r.log.WithFields(log.Fields{"error": err}).Error("receiveLoop: failed to GetLatestRound")
		return _errCh, err
	}

	go func() {
		defer func() {
			_errCh <- errors.New("aborting receiveloop")
			close(_errCh)
		}()
	receiveLoop:
		for {
			select {
			case <-ctx.Done():
				fmt.Println("ctx canceled aborting")
				break receiveLoop
			default:
				if curRound >= latestRound {
					time.Sleep(500 * time.Millisecond)

					latestRound, err = r.cl.GetLatestRound(ctx)
					if err != nil {
						r.log.WithFields(log.Fields{"error": err}).Error("receiveLoop: error failed to getLatestRound-")
						_errCh <- err
					}
					continue
				}
				//Check the latest block for txns addressed to this BMC
				r.inspectBlock(ctx, curRound, &subOpts, msgCh, _errCh)
				curRound++
			}
		}
	}()
	return _errCh, err
}

// Inspects the latest block created for new relay messages
func (r *receiver) inspectBlock(ctx context.Context, round uint64, subOpts *chain.SubscribeOptions,
	msgCh chan<- *chain.Message, _errCh chan error) {
	newBlock, err := r.cl.GetBlockbyRound(ctx, round)
	if err != nil {
		_errCh <- err
		return
	}
	bmcTxns := r.GetBMCTxns(newBlock)
	if len(*bmcTxns) <= 0 {
		fmt.Println("new block doesnt have SC txns")

		return
	}

	relayRcps, err := r.getRelayReceipts(bmcTxns, round)
	if err != nil {
		_errCh <- err
		return
	}

	err = r.validateEvents(&relayRcps, subOpts)
	if err != nil {
		_errCh <- err
		return
	}
	msgCh <- &chain.Message{Receipts: relayRcps}
}

// Check if the new block has any transaction meant to be sent across the relayer
func (r *receiver) GetBMCTxns(block *types.Block) *[]types.SignedTxnWithAD {
	fmt.Println("READIN TXNS FROM NEW BLOCK")

	txns := make([]types.SignedTxnWithAD, 0)
	for _, signedTxnInBlock := range block.Payset {
		signedTxnWithAD := signedTxnInBlock.SignedTxnWithAD
		//TODO review the way of properly identify a bmc txn once we have a proper BMC
		//This block is now only adding txns with payload to test the receiveloop
		/* 		if signedTxnWithAD.SignedTxn.Txn.Type == types.ApplicationCallTx &&
		signedTxnWithAD.SignedTxn.Txn.ApplicationID == types.AppIndex(r.cl.bmc.appID) &&
		signedTxnWithAD.SignedTxn.AuthAddr.String() == r.src.ContractAddress() &&
		string(signedTxnWithAD.SignedTxn.Txn.ApplicationArgs[0]) == r.dst.ContractAddress() { */

		if signedTxnWithAD.SignedTxn.Txn.Type == types.ApplicationCallTx {
			txns = append(txns, signedTxnWithAD)
		}
	}
	return &txns
}

func (r *receiver) getRelayReceipts(txns *[]types.SignedTxnWithAD, round uint64) ([]*chain.Receipt, error) {
	var receipts []*chain.Receipt
	var events []*chain.Event
	for i, txn := range *txns {
		events := events[:0]
		for _, log := range txn.ApplyData.EvalDelta.Logs {
			if txn.Txn.Header.Sender.String() != r.src.ContractAddress() {
				continue
			}
			decodedMsg, err := r.cl.DecodeBtpMsg(log)
			if err == nil {
				events = append(events, decodedMsg)
			}
		}
		if len(events) > 0 {
			rcp := &chain.Receipt{}
			rcp.Index, rcp.Height = uint64(i), round
			rcp.Events = append(rcp.Events, events...)
			receipts = append(receipts, rcp)
		}
	}
	if len(receipts) <= 0 {
		return receipts, errors.New("Couldn't retrieve any receipt from the new block")
	}
	return receipts, nil
}

func (r *receiver) validateEvents(rcps *[]*chain.Receipt, subOpts *chain.SubscribeOptions) error {
	for _, receipt := range *rcps {
		events := receipt.Events[:0]
		for _, event := range receipt.Events {
			switch {
			case event.Sequence == subOpts.Seq:
				events = append(events, event)
				subOpts.Seq++
			case event.Sequence > subOpts.Seq:
				r.log.WithFields(log.Fields{
					"seq": log.Fields{"got": event.Sequence, "expected": subOpts.Seq},
				}).Error("invalid event seq")
				return fmt.Errorf("invalid event seq")
			}
		}
		receipt.Events = events
	}
	return nil
}
