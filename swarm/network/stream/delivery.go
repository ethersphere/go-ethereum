// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package stream

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const (
	swarmChunkServerStreamName = "RETRIEVE_REQUEST"
	deliveryCap                = 32
)

var (
	processReceivedChunksCount    = metrics.NewRegisteredCounter("network.stream.received_chunks.count", nil)
	handleRetrieveRequestMsgCount = metrics.NewRegisteredCounter("network.stream.handle_retrieve_request_msg.count", nil)

	requestFromPeersCount     = metrics.NewRegisteredCounter("network.stream.request_from_peers.count", nil)
	requestFromPeersEachCount = metrics.NewRegisteredCounter("network.stream.request_from_peers_each.count", nil)
)

type Delivery struct {
	fileStore storage.FileStore
	overlay   network.Overlay
	receiveC  chan *ChunkDeliveryMsg
	getPeer   func(discover.NodeID) *Peer
}

func NewDelivery(overlay network.Overlay, fileStore FileStore) *Delivery {
	d := &Delivery{
		fileStore: fileStore,
		overlay:   overlay,
		receiveC:  make(chan *ChunkDeliveryMsg, deliveryCap),
	}

	go d.processReceivedChunks()
	return d
}

// SwarmChunkServer implements Server
type SwarmChunkServer struct {
	deliveryC  chan []byte
	batchC     chan []byte
	fileStore  storage.FileStore
	currentLen uint64
	quit       chan struct{}
}

// NewSwarmChunkServer is SwarmChunkServer constructor
func NewSwarmChunkServer(fileStore FileStore) *SwarmChunkServer {
	s := &SwarmChunkServer{
		deliveryC: make(chan []byte, deliveryCap),
		batchC:    make(chan []byte),
		fileStore: FileStore,
		quit:      make(chan struct{}),
	}
	go s.processDeliveries()
	return s
}

func (s *SwarmChunkServer) context(req *RetrieveRequestMsg) context.Context {
	var cancel func()
	ctx := context.Background()
	// if req.Timeout > 0 {
	// ctx, cancel = context.WithTimeout(ctx, req.Timeout)
	// }
	ctx, cancel = context.WithCancel(ctx)

	go func() {
		select {
		case <-ctx.Done():
		case <-s.quit:
			cancel()
		}
	}()

	return ctx
}

// processDeliveries handles delivered chunk hashes
func (s *SwarmChunkServer) processDeliveries() {
	var hashes []byte
	var batchC chan []byte
	for {
		select {
		case <-s.quit:
			return
		case hash := <-s.deliveryC:
			hashes = append(hashes, hash...)
			batchC = s.batchC
		case batchC <- hashes:
			hashes = nil
			batchC = nil
		}
	}
}

// SetNextBatch
func (s *SwarmChunkServer) SetNextBatch(_, _ uint64) (hashes []byte, from uint64, to uint64, proof *HandoverProof, err error) {
	select {
	case hashes = <-s.batchC:
	case <-s.quit:
		return
	}

	from = s.currentLen
	s.currentLen += uint64(len(hashes))
	to = s.currentLen
	return
}

// Close needs to be called on a stream server
func (s *SwarmChunkServer) Close() {
	close(s.quit)
}

// GetData retrives chunk data from db store
func (s *SwarmChunkServer) GetData(key []byte) ([]byte, error) {
	chunk, err := s.fileStore.Get(immediately, storage.Address(key))
	if err != nil {
		return nil, err
	}
	return chunk.Data(), nil
}

// RetrieveRequestMsg is the protocol msg for chunk retrieve requests
type RetrieveRequestMsg struct {
	Addr      storage.Address
	SkipCheck bool
}

func (d *Delivery) handleRetrieveRequestMsg(sp *Peer, req *RetrieveRequestMsg) error {
	log.Trace("received request", "peer", sp.ID(), "hash", req.Addr)
	handleRetrieveRequestMsgCount.Inc(1)

	s, err := sp.getServer(NewStream(swarmChunkServerStreamName, "", false))
	if err != nil {
		return err
	}
	streamer := s.Server.(*SwarmChunkServer)
	go func() {
		chunk, err := d.fileStore.Get(streamer.context(req), req.Addr)
		if err != nil {
			return
		}
		if req.SkipCheck {
			err = sp.Deliver(chunk, s.priority)
			if err != nil {
				log.Warn("ERROR in handleRetrieveRequestMsg, DROPPING peer!", "err", err)
				sp.Drop(err)
				return
			}
		}
		streamer.deliveryC <- chunk.Address()[:]
	}()

	return nil
}

type ChunkDeliveryMsg struct {
	Addr  storage.Address
	SData []byte // the stored chunk Data (incl size)
	peer  *Peer  // set in handleChunkDeliveryMsg
}

func (d *Delivery) handleChunkDeliveryMsg(sp *Peer, req *ChunkDeliveryMsg) error {
	req.peer = sp
	d.receiveC <- req
	return nil
}

var immediately context.Context

func init() {
	var cancel func()
	immediately, cancel = context.WithCancel(context.Background())
	cancel()
}

func (d *Delivery) processReceivedChunks() {
	for req := range d.receiveC {
		processReceivedChunksCount.Inc(1)

		_, err := d.fileStore.Put(storage.NewChunk(req.Addr, req.SData))
		if err != nil {
			if err == storage.ErrChunkInvalid {
				req.peer.Drop(err)
			}
			return
		}
	}
}

// RequestFromPeers sends a chunk retrieve request to
func (d *Delivery) RequestFromPeers(ctx context.Context, addr storage.Address, source storage.Address, skipCheck bool, peersToSkip *sync.Map) (storage.Address, chan struct{}, error) {
	requestFromPeersCount.Inc(1)

	//
	var sp *Peer
	var spAddr storage.Address

	if source != nil {
		var found bool
		d.overlay.EachConn(source[:], 256, func(p network.OverlayConn, po int, nn bool) bool {
			found = bytes.Equal(p.Address(), source[:])
			spId := p.(network.Peer).ID()
			sp = d.getPeer(spId)
			if sp == nil {
				log.Warn("Delivery.RequestFromPeers: peer not found", "id", spId)
				return true
			}
			return false
		})
		if !found {
			return nil, nil, fmt.Errorf("source peer %v not found", spAddr.Hex())
		}
	} else {
		d.overlay.EachConn(addr[:], 255, func(p network.OverlayConn, po int, nn bool) bool {
			spAddr = storage.Address(p.Address())
			// TODO: skip light nodes that do not accept retrieve requests
			if _, ok := peersToSkip.Load(spAddr); ok {
				log.Trace("Delivery.RequestFromPeers: skip peer", "peer addr", spAddr.Hex())
				return true
			}
			spId := p.(network.Peer).ID()
			sp = d.getPeer(spId)
			if sp == nil {
				log.Warn("Delivery.RequestFromPeers: peer not found", "id", spId)
				return true
			}
			return false
		})
	}

	if sp == nil {
		return nil, nil, errors.New("no peer found")
	}
	err := sp.SendPriority(&RetrieveRequestMsg{
		Addr:      addr,
		SkipCheck: skipCheck,
	}, Top)
	if err != nil {
		return nil, nil, err
	}
	requestFromPeersEachCount.Inc(1)

	return spAddr, sp.quit, nil
}
