// Copyright 2016 The go-ethereum Authors
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

package network

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

/*
Hive is the logistic manager of the swarm
it uses an Overlay Topology driver (e.g., generic kademlia nodetable)
to find best peer list for any target
this is used by the netstore to search for content in the swarm

It handles the hive protocol getPeersMsg peersMsg exchange
and relay the peer request process to the Overlay module

peer connections and disconnections are reported and registered
to keep the nodetable uptodate
*/

// Overlay is the interface to Jaak ahd ka)a
type Overlay interface {
	Register(chan OverlayAddr) error

	On(OverlayConn)
	Off(OverlayConn)

	EachConn([]byte, int, func(OverlayConn, int, bool) bool)
	EachAddr([]byte, int, func(OverlayAddr, int) bool)

	SuggestPeer() (OverlayAddr, int, bool)

	String() string
	BaseAddr() []byte
	Healthy(*PeerPot) bool
}

// HiveParams holds the config options to hive
type HiveParams struct {
	Discovery             bool  // if want discovery of not
	PeersBroadcastSetSize uint8 // how many peers to use when relaying
	MaxPeersPerRequest    uint8 // max size for peer address batches
	KeepAliveInterval     time.Duration
}

// NewHiveParams returns hive config with only the
func NewHiveParams() *HiveParams {
	return &HiveParams{
		Discovery:             true,
		PeersBroadcastSetSize: 2,
		MaxPeersPerRequest:    5,
		KeepAliveInterval:     1000 * time.Millisecond,
	}
}

// Hive implements the PeerPool interface
type Hive struct {
	*HiveParams // settings
	Overlay     // the overlay topology driver
	store       StateStore

	// bookkeeping
	lock sync.Mutex
	quit chan bool
	more chan bool

	tick <-chan time.Time
}

// NewHive constructs a new hive
// HiveParams: config parameters
// Overlay: Topology Driver Interface
// StateStore: to save peers across sessions
func NewHive(params *HiveParams, overlay Overlay, store StateStore) *Hive {
	return &Hive{
		HiveParams: params,
		Overlay:    overlay,
		store:      store,
	}
}

// Start receives network info only at startup
// server is used to connect to a peer based on its NodeID or enode URL
// these are called on the p2p.Server which runs on the node
// af() returns an arbitrary ticker channel
// rw is a read writer for json configs
func (h *Hive) Start(server *p2p.Server) error {
	if h.store != nil {
		if err := h.loadPeers(); err != nil {
			return err
		}
	}
	h.more = make(chan bool, 1)
	h.quit = make(chan bool)
	// this loop is doing bootstrapping and maintains a healthy table
	go h.connect(server)
	return nil
}

func (h *Hive) connect(server *p2p.Server) {
	ticker := time.NewTicker(h.KeepAliveInterval)
	defer ticker.Stop()
	// each iteration, ask kademlia about most preferred peer to connect to
	for {
		log.Trace(fmt.Sprintf("%08x: hive delegate to overlay driver: suggest addr to connect to", h.BaseAddr()[:4]))
		// log.Trace(fmt.Sprintf("%s", h))
		addr, order, want := h.SuggestPeer()
		if addr != nil {
			under, err := discover.ParseNode(string(addr.(Addr).Under()))
			if err != nil {
				log.Error(fmt.Sprintf("%08x unable to connect to bee %08x: invalid node URL: %v", h.BaseAddr()[:4], addr.Address()[:4], err))
			} else {
				log.Trace(fmt.Sprintf("%08x ========> connect to bee %08x", h.BaseAddr()[:4], addr.Address()[:4]))
				go server.AddPeer(under)
			}
		} else {
			log.Trace(fmt.Sprintf("%08x unable to suggest peers", h.BaseAddr()[:4]))
		}

		// if there is a need for more peers in some PO bin and discovery is enabled
		// then request peers
		if h.Discovery && want {
			log.Trace(fmt.Sprintf("%08x ========> request peers for PO%0d", h.BaseAddr()[:4], order))
			RequestOrder(h.Overlay, uint8(order), h.PeersBroadcastSetSize, h.MaxPeersPerRequest)
		}

		select {
		case <-h.quit:
			return
			// default:
		case <-ticker.C:
		}
	}
}

// Stop terminates the updateloop and saves the peers
func (h *Hive) Stop() {
	if h.store != nil {
		h.savePeers()
	}
	// closing toggle channel quits the updateloop
	close(h.quit)
}

// Run protocol run function
func (h *Hive) Run(p *bzzPeer) error {
	dp := newDiscovery(p, h)
	log.Debug(fmt.Sprintf("to add new bee %v", p))
	h.On(dp)
	defer h.Off(dp)
	return p.Run(dp.HandleMsg)
}

// NodeInfo function is used by the p2p.server RPC interface to display
// protocol specific node information
func (h *Hive) NodeInfo() interface{} {
	return h.String()
}

// PeerInfo function is used by the p2p.server RPC interface to display
// protocol specific information any connected peer referred to by their NodeID
func (h *Hive) PeerInfo(id discover.NodeID) interface{} {
	h.lock.Lock()
	defer h.lock.Unlock()
	addr := NewAddrFromNodeID(id)
	return interface{}(addr)
}

// ToAddr returns the serialisable version of u
func ToAddr(pa OverlayPeer) *bzzAddr {
	if addr, ok := pa.(*bzzAddr); ok {
		return addr
	}
	if p, ok := pa.(*discPeer); ok {
		return p.bzzAddr
	}
	return pa.(*bzzPeer).bzzAddr
}

// loadPeers, savePeer implement persistence callback/
func (h *Hive) loadPeers() error {
	data, err := h.store.Load("peers")
	if err != nil {
		return err
	}
	if data == nil {
		return nil
	}
	var as []*bzzAddr
	if err := json.Unmarshal(data, &as); err != nil {
		return err
	}

	c := make(chan OverlayAddr)
	go func() {
		defer close(c)
		for _, a := range as {
			c <- a
		}
	}()
	return h.Overlay.Register(c)
}

// savePeers, savePeer implement persistence callback/
func (h *Hive) savePeers() error {
	var peers []*bzzAddr
	h.Overlay.EachAddr(nil, 256, func(pa OverlayAddr, i int) bool {
		if pa == nil {
			log.Warn(fmt.Sprintf("empty addr: %v", i))
			return true
		}
		peers = append(peers, ToAddr(pa))
		return true
	})
	data, err := json.Marshal(peers)
	if err != nil {
		return fmt.Errorf("could not encode peers: %v", err)
	}
	if err := h.store.Save("peers", data); err != nil {
		return fmt.Errorf("could not save peers: %v", err)
	}
	return nil
}
