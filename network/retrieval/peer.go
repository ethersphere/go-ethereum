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

package retrieval

import (
	"bytes"
	"encoding/hex"
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethersphere/swarm/chunk"
	"github.com/ethersphere/swarm/network"
	"github.com/ethersphere/swarm/storage"
)

// Peer wraps BzzPeer with a contextual logger and tracks open
// retrievals for that peer
type Peer struct {
	*network.BzzPeer
	logger     log.Logger             // logger with base and peer address
	mtx        sync.Mutex             // synchronize retrievals
	retrievals map[uint]chunk.Address // current ongoing retrievals
}

// NewPeer is the constructor for Peer
func NewPeer(peer *network.BzzPeer, baseKey []byte) *Peer {
	return &Peer{
		BzzPeer:    peer,
		logger:     log.New("base", hex.EncodeToString(baseKey)[:16], "peer", peer.ID().String()[:16]),
		retrievals: make(map[uint]chunk.Address),
	}
}

// chunkRequested adds a new retrieval to the retrievals map
// this is in order to identify unsolicited chunk deliveries
func (p *Peer) addRetrieval(ruid uint, addr storage.Address) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	p.retrievals[ruid] = addr
}

// chunkReceived is called upon ChunkDelivery message reception
// it is meant to idenfify unsolicited chunk deliveries
func (p *Peer) checkRequest(ruid uint, addr storage.Address) error {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	v, ok := p.retrievals[ruid]
	if !ok {
		return errors.New("cannot find ruid")
	}
	delete(p.retrievals, ruid) // since we got the delivery we wanted - it is safe to delete the retrieve request
	if !bytes.Equal(v, addr) {
		return errors.New("retrieve request found but address does not match")
	}

	return nil
}
