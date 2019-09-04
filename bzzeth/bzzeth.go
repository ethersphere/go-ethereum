// Copyright 2019 The Swarm Authors
// This file is part of the Swarm library.
//
// The Swarm library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Swarm library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Swarm library. If not, see <http://www.gnu.org/licenses/>.

package bzzeth

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/crypto"
	"sync"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethersphere/swarm/chunk"
	"github.com/ethersphere/swarm/log"
	"github.com/ethersphere/swarm/network"
	"github.com/ethersphere/swarm/p2p/protocols"
	"github.com/ethersphere/swarm/storage"
)

var (
	errUnsolicitedHeader = errors.New("unsolicited header received")
	errDuplicateHeader   = errors.New("duplicate header received")
)

var (
	errRcvdMsgFromSwarmNode = errors.New("received message from Swarm node")
)

// BzzEth implements node.Service
var _ node.Service = &BzzEth{}

// BzzEth is a global module handling ethereum state on swarm
type BzzEth struct {
	peers    *peers            // bzzeth peer pool
	netStore *storage.NetStore // netstore to retrieve and store
	kad      *network.Kademlia // kademlia to determine if a header chunk belongs to us
	quit     chan struct{}     // quit channel to close go routines
}

// New constructs the BzzEth node service
func New(ns *storage.NetStore, kad *network.Kademlia) *BzzEth {
	return &BzzEth{
		peers:    newPeers(),
		netStore: ns,
		kad:      kad,
		quit:     make(chan struct{}),
	}
}

// Run is the bzzeth protocol run function.
// - creates a peer
// - checks if it is a swarm node, put the protocol in idle mode
// - performs handshake
// - adds peer to the peerpool
// - starts incoming message handler loop
func (b *BzzEth) Run(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	peer := protocols.NewPeer(p, rw, Spec)
	bp := NewPeer(peer)

	// perform handshake and register if peer serves headers
	handshake, err := bp.Handshake(context.TODO(), Handshake{ServeHeaders: true}, nil)
	if err != nil {
		return err
	}
	bp.serveHeaders = handshake.(*Handshake).ServeHeaders
	log.Debug("handshake", "hs", handshake, "peer", bp)

	b.peers.add(bp)
	defer b.peers.remove(bp)

	// This protocol is all about interaction between an Eth node and a Swarm Node.
	// If another swarm node tries to connect then the protocol goes into idle
	if isSwarmNodeFunc(bp) {
		return peer.Run(b.handleMsgFromSwarmNode(bp))
	}

	return peer.Run(b.handleMsg(bp))
}

// handleMsg is the message handler that delegates incoming messages
// handlers are called asynchronously so handler calls do not block incoming msg processing
func (b *BzzEth) handleMsg(p *Peer) func(context.Context, interface{}) error {
	return func(ctx context.Context, msg interface{}) error {
		p.logger.Debug("bzzeth.handleMsg")
		switch msg := msg.(type) {
		case *NewBlockHeaders:
			go b.handleNewBlockHeaders(ctx, p, msg)
		case *BlockHeaders:
			go b.handleBlockHeaders(ctx, p, msg)
		default:
			p.logger.Error("Invalid msg")
		}
		return nil
	}
}


// handleMsgFromSwarmNode is used in the case if this node is connected to a Swarm node
// If any message is received in this case, the peer needs to be dropped
func (b *BzzEth) handleMsgFromSwarmNode(p *Peer) func(context.Context, interface{}) error {
	return func(ctx context.Context, msg interface{}) error {
		p.logger.Warn("bzzeth.handleMsgFromSwarmNode")
		return errRcvdMsgFromSwarmNode
	}
}

// handles new header hashes - strategy; only request headers that are in Kad Nearest Neighbourhood
func (b *BzzEth) handleNewBlockHeaders(ctx context.Context, p *Peer, msg *NewBlockHeaders) {
	p.logger.Debug("bzzeth.handleNewBlockHeaders")

	// collect the addresses of blocks that are not in our localstore
	var addresses []chunk.Address
	for _, h := range msg.Headers {
		addresses = append(addresses, h.Hash)
	}
	yes, err := b.netStore.Store.HasMulti(ctx, addresses...)

	// collect the hashes of block headers we want
	var hashes [][]byte
	for i, y := range yes {
		// ignore hashes already present in localstore
		if y {
			continue
		}

		// collect hash based on proximity
		vhash := addresses[i]
		if wantHeaderFunc(vhash, b.kad) {
			hashes = append(hashes, vhash)
		}
	}

	// request them from the offering peer and deliver in a channel
	deliveries := make(chan []byte)
	req, err := p.getBlockHeaders(ctx, hashes, deliveries)
	if err != nil {
		p.logger.Error("Error sending GetBlockHeader message", "Reason", err)
		return
	}
	defer req.cancel()

	// this loop blocks until all delivered or context done
	// only needed to log results
	for {
		select {
		case hash, ok := <-deliveries:
			// calculate the delivery count by looking at the status in the request
			deliveredCnt := getDeliveryCount(req)

			if !ok {
				p.logger.Debug("bzzeth.handleNewBlockHeaders", "hash", hash, "delivered", deliveredCnt)
				return
			}

			if deliveredCnt == len(req.hashes) {
				p.logger.Debug("bzzeth.handleNewBlockHeaders", "hash", hash, "delivered", deliveredCnt)
				return
			}
		case <-ctx.Done():
			// calculate the delivery count by looking at the status in the request
			deliveredCnt := getDeliveryCount(req)
			p.logger.Debug("bzzeth.handleNewBlockHeaders", "delivered", deliveredCnt, "err", err)
			return
		}
	}
}

// wantHeaderFunc is used to determine if we need a particular header offered as latest
// by an eth fullnode
// tests reassign this to control
var wantHeaderFunc = wantHeader

// wantHeader returns true iff the hash argument falls in the NN of kademlia
func wantHeader(hash []byte, kad *network.Kademlia) bool {
	return chunk.Proximity(kad.BaseAddr(), hash) >= kad.NeighbourhoodDepth()
}

// Calculates the no of headers delivered in a given request
func getDeliveryCount(req *request) int {
	deliveredCnt := 0
	req.lock.Lock()
	defer req.lock.Unlock()
	for _, done := range req.hashes {
		if done {
			deliveredCnt++
		}
	}
	return deliveredCnt
}

// handleBlockHeaders handles block headers message
func (b *BzzEth) handleBlockHeaders(ctx context.Context, p *Peer, msg *BlockHeaders) {
	p.logger.Debug("bzzeth.handleBlockHeaders", "id", msg.ID)

	// retrieve the request for this id
	req, ok := p.requests.get(msg.ID)
	if !ok {
		p.logger.Warn("bzzeth.handleBlockHeaders: nonexisting request id", "id", msg.ID)
		p.Drop()
		return
	}
	err := b.deliverAndStoreAll(ctx, req, msg.Headers)
	if err != nil {
		p.logger.Warn("bzzeth.handleBlockHeaders: fatal dropping peer", "id", msg.ID, "err", err)
		p.Drop()
	}
}

// Validates and headers asynchronously and stores the valid chunks in one go
func (b *BzzEth) deliverAndStoreAll(ctx context.Context, req *request, headers [][]byte) error {
	errC := make(chan error, len(headers))
	chunksC := make(chan chunk.Chunk, len(headers))

	var wg sync.WaitGroup
	for _, h := range headers {
		hdr := make([]byte, len(h))
		copy(hdr, h)
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch, err := b.validateHeader(ctx, hdr, req)
			if err != nil {
				errC <- err
				return
			}
			chunksC <- ch
		}()
	}

	// wait for all validations to get over and close the channels
	wg.Wait()
	close(chunksC)
	close(errC)

	// Store all the valid header chunks in one shot
	chunks := make([]chunk.Chunk, 0)
	for c := range chunksC {
		chunks = append(chunks, c)
	}
	results, err := b.netStore.Put(ctx, chunk.ModePutUpload, chunks...)
	if err != nil {
		for i, _ := range results {
			ch := chunks[i]
			log.Warn("bzzeth.store", "hash", ch.Address().Hex(), "err", err)
			// ignore all other errors, but invalid chunk incurs peer drop
			if err == chunk.ErrChunkInvalid {
				return err
			}
		}
	}
	return <-errC
}

// validateHeader check for correctness and validity of the header
// this also informs the delivery channel about the received header
func (b *BzzEth) validateHeader(ctx context.Context, header []byte, req *request) (chunk.Chunk, error) {
	ch := newChunk(header)
	headerAlreadyReceived, expected := isHeaderExpected(req, ch.Address().Hex())
	if expected {
		if headerAlreadyReceived {
			// header already received
			return nil, errDuplicateHeader
		} else {
			setHeaderAsReceived(req, ch.Address().Hex())
			req.c <- ch.Address()
			return ch, nil
		}
	} else {
		// header is not present in the request hash.
		return nil, errUnsolicitedHeader
	}

	// TODO: Ethereum block header validation should come here
}

// Checks if the given hash is expected in this request
func isHeaderExpected(req *request, addr string) (rcvdFlag bool, ok bool) {
	req.lock.RLock()
	defer req.lock.RUnlock()
	rcvdFlag, ok = req.hashes[addr]
	return
}

// Set the given hash as received in the request
func setHeaderAsReceived(req *request, addr string) {
	req.lock.Lock()
	defer req.lock.Unlock()
	req.hashes[addr] = true
}

// newChunk creates a new content addressed chunk from data using Keccak256  SHA3 hash
func newChunk(data []byte) chunk.Chunk {
	hash := crypto.Keccak256(data)
	return chunk.NewChunk(hash, data)

}

// Protocols returns the p2p protocol
func (b *BzzEth) Protocols() []p2p.Protocol {
	return []p2p.Protocol{
		{
			Name:    Spec.Name,
			Version: Spec.Version,
			Length:  Spec.Length(),
			Run:     b.Run,
		},
	}
}

// APIs return APIs defined on the node service
func (b *BzzEth) APIs() []rpc.API {
	return nil
}

// Start starts the BzzEth node service
func (b *BzzEth) Start(server *p2p.Server) error {
	log.Info("bzzeth starting...")
	return nil
}

// Stop stops the BzzEth node service
func (b *BzzEth) Stop() error {
	log.Info("bzzeth shutting down...")
	close(b.quit)
	return nil
}
