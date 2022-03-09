/*
 * Copyright 2020 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package chain

import (
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/icon-project/btp/cmd/btpsimple/module"
	"github.com/icon-project/btp/common/db"
	"github.com/icon-project/btp/common/errors"
	"github.com/icon-project/btp/common/log"
	"github.com/icon-project/btp/common/mta"
	"github.com/icon-project/btp/common/wallet"
)

const (
	DefaultDBDir  = "db"
	DefaultDBType = db.GoLevelDBBackend
	// DefaultBufferScaleOfBlockProof Base64 in:out = 6:8
	DefaultBufferScaleOfBlockProof  = 0.5
	DefaultBufferNumberOfBlockProof = 100
	DefaultBufferInterval           = 5 * time.Second
	DefaultReconnectDelay           = time.Second
	DefaultRelayReSendInterval      = time.Second
)

type SimpleChain struct {
	s       module.Sender
	r       module.Receiver
	w       wallet.Wallet
	src     module.BtpAddress
	acc     *mta.ExtAccumulator
	dst     module.BtpAddress
	bs      *module.BMCLinkStatus //getstatus(dst.src)
	relayCh chan *module.RelayMessage
	l       log.Logger
	cfg     *Config

	rms             []*module.RelayMessage
	rmsMtx          sync.RWMutex
	rmSeq           uint64
	heightOfDst     int64
	lastBlockUpdate *module.BlockUpdate

	bmrIndex       int
	relayble       bool
	relaybleIndex  int
	relaybleHeight int64
}

func (s *SimpleChain) _hasWait(rm *module.RelayMessage) bool {
	for _, segment := range rm.Segments {
		if segment != nil && segment.GetResultParam != nil && segment.TransactionResult == nil {
			return true
		}
	}
	return false
}

func (s *SimpleChain) _log(prefix string, rm *module.RelayMessage, segment *module.Segment, segmentIdx int) {
	if segment == nil {
		s.l.Debugf("%s rm:%d bu:%d ~ %d rps:%d",
			prefix,
			rm.Seq,
			rm.BlockUpdates[0].Height,
			rm.BlockUpdates[len(rm.BlockUpdates)-1].Height,
			len(rm.ReceiptProofs))
	} else {
		s.l.Debugf("%s rm:%d [i:%d,h:%d,bu:%d,seq:%d,evt:%d,txh:%v]",
			prefix,
			rm.Seq,
			segmentIdx,
			segment.Height,
			segment.NumberOfBlockUpdate,
			segment.EventSequence,
			segment.NumberOfEvent,
			segment.GetResultParam)
	}
}

func (s *SimpleChain) _relay() {
	s.rmsMtx.RLock()
	defer s.rmsMtx.RUnlock()
	var err error
	for _, rm := range s.rms {
		if (len(rm.ReceiptProofs) == 0) ||
			s._hasWait(rm) ||
			(!s._skippable(rm)) {
			break
		} else {
			if len(rm.Segments) == 0 {
				//TODO: change the segment method signature
				if rm.Segments, err = s.s.Segment(rm, 0); err != nil {
					s.l.Panicf("fail to segment err:%+v", err)
				}
			}
			//s._log("before relay", rm, nil, -1)
			reSegment := true
			for j, segment := range rm.Segments {
				if segment == nil {
					continue
				}
				reSegment = false

				if segment.GetResultParam == nil {
					segment.TransactionResult = nil
					s._log("Going to relay now", rm, segment, j)
					if segment.GetResultParam, err = s.s.Relay(segment); err != nil {
						s.l.Panicf("fail to Relay err:%+v", err)
					}
					s._log("after relay", rm, segment, j)
					go s.result(rm, segment)
				}
			}
			if reSegment {
				rm.Segments = rm.Segments[:0]
			}
		}
	}
}

func (s *SimpleChain) result(rm *module.RelayMessage, segment *module.Segment) {
	var err error
	segment.TransactionResult, err = s.s.GetResult(segment.GetResultParam)
	if err != nil {
		if ec, ok := errors.CoderOf(err); ok {
			s.l.Debugf("fail to GetResult GetResultParam:%v ErrorCoder:%+v",
				segment.GetResultParam, ec)
			switch ec.ErrorCode() {
			case module.BMVRevertInvalidSequence, module.BMVRevertInvalidBlockUpdateLower:
				for i := 0; i < len(rm.Segments); i++ {
					if rm.Segments[i] == segment {
						rm.Segments[i] = nil
						break
					}
				}
			case module.BMVRevertInvalidSequenceHigher, module.BMVRevertInvalidBlockUpdateHigher, module.BMVRevertInvalidBlockProofHigher:
				segment.GetResultParam = nil
			case module.BMCRevertUnauthorized:
				segment.GetResultParam = nil
			default:
				s.l.Panicf("fail to GetResult GetResultParam:%v ErrorCoder:%+v",
					segment.GetResultParam, ec)
			}
		} else {
			//TODO: commented temporarily to keep the relayer running
			//s.l.Panicf("fail to GetResult GetResultParam:%v err:%+v",
			//	segment.GetResultParam, err)
			s.l.Debugf("fail to GetResult GetResultParam:%v err:%+v", segment.GetResultParam, err)
		}
	}
}

func (s *SimpleChain) _rm() *module.RelayMessage {
	rm := &module.RelayMessage{
		From:         s.src,
		BlockUpdates: make([]*module.BlockUpdate, 0),
		Seq:          s.rmSeq,
	}
	s.rms = append(s.rms, rm)
	s.rmSeq += 1
	return rm
}

func (s *SimpleChain) addRelayMessage(bu *module.BlockUpdate, rps []*module.ReceiptProof) {
	s.rmsMtx.Lock()
	defer s.rmsMtx.Unlock()

	rm := s.rms[len(s.rms)-1]
	if len(rm.Segments) > 0 {
		rm = s._rm()
	}
	if len(rps) > 0 {
		rm.BlockUpdates = append(rm.BlockUpdates, bu)
		rm.ReceiptProofs = rps
		s.l.Debugf("addRelayMessage rms:%d rps:%d HeightOfDst:%d", len(s.rms), len(rps), rm.HeightOfDst)
		rm = s._rm()
	}
}

func (s *SimpleChain) updateRelayMessage(seq int64) (err error) {
	s.rmsMtx.Lock()
	defer s.rmsMtx.Unlock()

	s.l.Debugf("updateRelayMessage seq:%d monitorHeight:%d", seq, s.monitorHeight())

	rrm := 0
	for i, rm := range s.rms {
		if len(rm.ReceiptProofs) > 0 {
			rrp := 0
		rpLoop:
			for j, rp := range rm.ReceiptProofs {
				revt := seq - rp.Events[0].Sequence + 1
				if revt < 1 {
					break rpLoop
				}
				if revt >= int64(len(rp.Events)) {
					rrp = j + 1
				} else {
					s.l.Debugf("updateRelayMessage rm:%d rp:%d removeEventProofs %d ~ %d",
						rm.Seq,
						rp.Index,
						rp.Events[0].Sequence,
						rp.Events[revt-1].Sequence)
					rp.Events = rp.Events[revt:]
				}
			}
			if rrp > 0 {
				s.l.Debugf("updateRelayMessage rm:%d removeReceiptProofs %d ~ %d",
					rm.Seq,
					rm.ReceiptProofs[0].Index,
					rm.ReceiptProofs[rrp-1].Index)
				rm.ReceiptProofs = rm.ReceiptProofs[rrp:]
			}
		}

		if len(rm.ReceiptProofs) <= 0 {
			rrm = i + 1
		}
	}
	if rrm > 0 {
		s.l.Debugf("updateRelayMessage rms:%d removeRelayMessage %d ~ %d",
			len(s.rms),
			s.rms[0].Seq,
			s.rms[rrm-1].Seq)
		s.rms = s.rms[rrm:]
		if len(s.rms) == 0 {
			s._rm()
		}
	}
	return nil
}

func (s *SimpleChain) OnBlockOfDst(height int64) error {
	s.l.Tracef("OnBlockOfDst height:%d", height)
	atomic.StoreInt64(&s.heightOfDst, height)
	seq := s.bs.RxSeq
	if err := s.RefreshStatus(); err != nil {
		return err
	}
	if seq != s.bs.RxSeq {
		seq = s.bs.RxSeq
		if err := s.updateRelayMessage(seq); err != nil {
			return err
		}
		s.relayCh <- nil
	}
	return nil
}

func (s *SimpleChain) OnBlockOfSrc(bu *module.BlockUpdate, rps []*module.ReceiptProof) {
	s.l.Tracef("OnBlockOfSrc")
	s.addRelayMessage(bu, rps)
	s.relayCh <- nil
}

func (s *SimpleChain) prepareDatabase(offset int64) error {
	s.l.Debugln("open database", filepath.Join(s.cfg.AbsBaseDir(), s.cfg.Dst.Address.NetworkAddress()))
	database, err := db.Open(s.cfg.AbsBaseDir(), string(DefaultDBType), s.cfg.Dst.Address.NetworkAddress())
	if err != nil {
		return errors.Wrap(err, "fail to open database")
	}
	defer func() {
		if err != nil {
			database.Close()
		}
	}()
	var bk db.Bucket
	if bk, err = database.GetBucket("Accumulator"); err != nil {
		return err
	}
	k := []byte("Accumulator")
	if offset < 0 {
		offset = 0
	}
	s.acc = mta.NewExtAccumulator(k, bk, offset)
	if bk.Has(k) {
		if err = s.acc.Recover(); err != nil {
			return errors.Wrapf(err, "fail to acc.Recover cause:%v", err)
		}
		s.l.Debugf("recover Accumulator offset:%d, height:%d", s.acc.Offset(), s.acc.Height())
	}
	return nil
}

func (s *SimpleChain) _skippable(rm *module.RelayMessage) bool {
	if len(rm.ReceiptProofs) > 0 {
		return true
	}
	return false
}

func (s *SimpleChain) RefreshStatus() error {
	bmcStatus, err := s.s.GetStatus()
	if err != nil {
		return err
	}
	s.bs = bmcStatus
	return nil
}

func (s *SimpleChain) init() error {
	if err := s.RefreshStatus(); err != nil {
		return err
	}
	atomic.StoreInt64(&s.heightOfDst, s.bs.CurrentHeight)
	if s.relayCh == nil {
		s.relayCh = make(chan *module.RelayMessage, 2)
		go func() {
			s.l.Debugln("start relayLoop")
			defer func() {
				s.l.Debugln("stop relayLoop")
			}()
			for {
				select {
				case _, ok := <-s.relayCh:
					if !ok {
						return
					}
					s._relay()
				}
			}
		}()
	}
	s.l.Debugf("_init height:%d, dst(%s, seq:%d), receive:%d",
		s.acc.Height(), s.dst, s.bs.RxSeq, s.receiveHeight())
	return nil
}

func (s *SimpleChain) receiveHeight() int64 {
	//TODO: check this logic
	//min(max(s.acc.Height(), s.bs.Verifier.Offset), s.bs.Verifier.LastHeight)
	max := s.acc.Height()
	/* if max < s.bs.Verifier.Offset {
		max = s.bs.Verifier.Offset
	} */
	max += 1
	/* min := s.bs.Verifier.LastHeight
	if max < min {
		min = max
	} */
	return max
}

func (s *SimpleChain) monitorHeight() int64 {
	return atomic.LoadInt64(&s.heightOfDst)
}

func (s *SimpleChain) Serve() error {
	if err := s.init(); err != nil {
		return err
	}
	errCh := make(chan error)
	go func() {
		err := s.s.MonitorLoop(
			s.bs.CurrentHeight,
			s.OnBlockOfDst,
			func() {
				s.l.Debugf("Connect MonitorLoop")
				errCh <- nil
			})
		select {
		case errCh <- err:
		default:
		}
	}()
	go func() {
		err := s.r.ReceiveLoop(
			s.receiveHeight(),
			s.bs.RxSeq,
			s.OnBlockOfSrc,
			func() {
				s.l.Debugf("Connect ReceiveLoop")
				errCh <- nil
			})
		select {
		case errCh <- err:
		default:
		}
	}()
	for {
		select {
		case err := <-errCh:
			if err != nil {
				return err
			}
		}
	}
}

func NewChain(cfg *Config, w wallet.Wallet, l log.Logger) (*SimpleChain, error) {
	s := &SimpleChain{
		src: cfg.Src.Address,
		dst: cfg.Dst.Address,
		w:   w,
		l:   l.WithFields(log.Fields{log.FieldKeyChain:
		//fmt.Sprintf("%s->%s", cfg.Src.Address.NetworkAddress(), cfg.Dst.Address.NetworkAddress())}),
		fmt.Sprintf("%s", cfg.Dst.Address.NetworkID())}),
		cfg: cfg,
		rms: make([]*module.RelayMessage, 0),
	}
	s._rm()

	s.s, s.r = newSenderAndReceiver(cfg, w, l)

	if err := s.prepareDatabase(cfg.Offset); err != nil {
		return nil, err
	}
	return s, nil
}
