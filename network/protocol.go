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
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethersphere/swarm/log"
	"github.com/ethersphere/swarm/p2p/protocols"
	"github.com/ethersphere/swarm/state"
)

var (
	capabilitiesFlagRetrieve      = []byte{0x00, 0x01} // node retrieves data for itself
	capabilitiesFlagPush          = []byte{0x00, 0x02} // node pushes own data to network
	capabilitiesFlagRelayRetrieve = []byte{0x00, 0x10} // node relays retrieve requests for the network
	capabilitiesFlagRelayPush     = []byte{0x00, 0x20} // node relays push requests for the network
	capabilitiesFlagStorer        = []byte{0x80, 0x00} // node is part of network storage (sync)

	// temporary presets to emulate the legacy LightNode/full node regime
	fullCapability  capability
	lightCapability capability
)

const (
	DefaultNetworkID = 4
	// timeout for waiting
	bzzHandshakeTimeout = 3000 * time.Millisecond
)

var DefaultTestNetworkID = rand.Uint64()

// BzzSpec is the spec of the generic swarm handshake
var BzzSpec = &protocols.Spec{
	Name:       "bzz",
	Version:    12,
	MaxMsgSize: 10 * 1024 * 1024,
	Messages: []interface{}{
		HandshakeMsg{},
	},
}

// DiscoverySpec is the spec for the bzz discovery subprotocols
var DiscoverySpec = &protocols.Spec{
	Name:       "hive",
	Version:    10,
	MaxMsgSize: 10 * 1024 * 1024,
	Messages: []interface{}{
		peersMsg{},
		subPeersMsg{},
	},
}

// temporary capabilities presets for current notions of "light" and "full" nodes
func init() {
	fullCapability = newFullCapability()
	lightCapability = newLightCapability()
}

// temporary convenience functions for legacy "LightNode"
func newLightCapability() capability {
	c := newCapability(0, 2)
	c.set(capabilitiesFlagRetrieve)
	c.set(capabilitiesFlagPush)
	return c
}
func isLightCapability(c capability) bool {
	return bytes.Equal(c, lightCapability)
}

// temporary convenience functions for legacy "full node"
func newFullCapability() capability {
	c := newCapability(0, 2)
	c.set(capabilitiesFlagRetrieve)
	c.set(capabilitiesFlagPush)
	c.set(capabilitiesFlagRelayRetrieve)
	c.set(capabilitiesFlagRelayPush)
	c.set(capabilitiesFlagStorer)
	return c
}

func isFullCapability(c capability) bool {
	return bytes.Equal(c, fullCapability)
}

// BzzConfig captures the config params used by the hive
type BzzConfig struct {
	OverlayAddr  []byte // base address of the overlay network
	UnderlayAddr []byte // node's underlay address
	HiveParams   *HiveParams
	NetworkID    uint64
	LightNode    bool // temporarily kept as we still only define light/full on operational level
	BootnodeMode bool
}

// Bzz is the swarm protocol bundle
type Bzz struct {
	*Hive
	NetworkID     uint64
	localAddr     *BzzAddr
	mtx           sync.Mutex
	handshakes    map[enode.ID]*HandshakeMsg
	streamerSpec  *protocols.Spec
	streamerRun   func(*BzzPeer) error
	capabilities  *Capabilities     // capabilities control and state
	capabilitiesC <-chan capability // reports changes in capabilities
}

// NewBzz is the swarm protocol constructor
// arguments
// * bzz config
// * overlay driver
// * peer store
func NewBzz(config *BzzConfig, kad *Kademlia, store state.Store, streamerSpec *protocols.Spec, streamerRun func(*BzzPeer) error) *Bzz {
	capabilitiesC := make(chan capability)
	bzz := &Bzz{
		Hive:          NewHive(config.HiveParams, kad, store),
		NetworkID:     config.NetworkID,
		localAddr:     &BzzAddr{config.OverlayAddr, config.UnderlayAddr},
		handshakes:    make(map[enode.ID]*HandshakeMsg),
		streamerRun:   streamerRun,
		streamerSpec:  streamerSpec,
		capabilities:  NewCapabilities(capabilitiesC),
		capabilitiesC: capabilitiesC,
	}

	if config.BootnodeMode {
		bzz.streamerRun = nil
		bzz.streamerSpec = nil
	}

	// temporary legacy light/full, as above
	if config.LightNode {
		bzz.capabilities.add(newLightCapability())
	} else {
		bzz.capabilities.add(newFullCapability())
	}

	return bzz
}

// Stop Implements node.Service
func (b *Bzz) Stop() error {
	err := b.Hive.Stop()
	b.capabilities.destroy()
	return err
}

// UpdateLocalAddr updates underlayaddress of the running node
func (b *Bzz) UpdateLocalAddr(byteaddr []byte) *BzzAddr {
	b.localAddr = b.localAddr.Update(&BzzAddr{
		UAddr: byteaddr,
		OAddr: b.localAddr.OAddr,
	})

	return b.localAddr
}

// NodeInfo returns the node's overlay address
func (b *Bzz) NodeInfo() interface{} {
	return b.localAddr.Address()
}

// Protocols return the protocols swarm offers
// Bzz implements the node.Service interface
// * handshake/hive
// * discovery
func (b *Bzz) Protocols() []p2p.Protocol {
	protocol := []p2p.Protocol{
		{
			Name:     BzzSpec.Name,
			Version:  BzzSpec.Version,
			Length:   BzzSpec.Length(),
			Run:      b.runBzz,
			NodeInfo: b.NodeInfo,
		},
		{
			Name:     DiscoverySpec.Name,
			Version:  DiscoverySpec.Version,
			Length:   DiscoverySpec.Length(),
			Run:      b.RunProtocol(DiscoverySpec, b.Hive.Run),
			NodeInfo: b.Hive.NodeInfo,
			PeerInfo: b.Hive.PeerInfo,
		},
	}
	if b.streamerSpec != nil && b.streamerRun != nil {
		protocol = append(protocol, p2p.Protocol{
			Name:    b.streamerSpec.Name,
			Version: b.streamerSpec.Version,
			Length:  b.streamerSpec.Length(),
			Run:     b.RunProtocol(b.streamerSpec, b.streamerRun),
		})
	}
	return protocol
}

// APIs returns the APIs offered by bzz
// * hive
// Bzz implements the node.Service interface
func (b *Bzz) APIs() []rpc.API {
	return []rpc.API{{
		Namespace: "hive",
		Version:   "3.0",
		Service:   b.Hive,
	}}
}

// RunProtocol is a wrapper for swarm subprotocols
// returns a p2p protocol run function that can be assigned to p2p.Protocol#Run field
// arguments:
// * p2p protocol spec
// * run function taking BzzPeer as argument
//   this run function is meant to block for the duration of the protocol session
//   on return the session is terminated and the peer is disconnected
// the protocol waits for the bzz handshake is negotiated
// the overlay address on the BzzPeer is set from the remote handshake
func (b *Bzz) RunProtocol(spec *protocols.Spec, run func(*BzzPeer) error) func(*p2p.Peer, p2p.MsgReadWriter) error {
	return func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
		// wait for the bzz protocol to perform the handshake
		handshake, _ := b.GetOrCreateHandshake(p.ID())
		defer b.removeHandshake(p.ID())
		select {
		case <-handshake.done:
		case <-time.After(bzzHandshakeTimeout):
			return fmt.Errorf("%08x: %s protocol timeout waiting for handshake on %08x", b.BaseAddr()[:4], spec.Name, p.ID().Bytes()[:4])
		}
		if handshake.err != nil {
			return fmt.Errorf("%08x: %s protocol closed: %v", b.BaseAddr()[:4], spec.Name, handshake.err)
		}

		// the handshake has succeeded so construct the BzzPeer and run the protocol
		peer := &BzzPeer{
			Peer:       protocols.NewPeer(p, rw, spec),
			BzzAddr:    handshake.peerAddr,
			lastActive: time.Now(),
			LightNode:  isLightCapability(handshake.Capabilities.get(0)), // this is a temporary member kept until kademlia code accommodates Capabilities instead
		}

		log.Debug("peer created", "addr", handshake.peerAddr.String())

		return run(peer)
	}
}

// performHandshake implements the negotiation of the bzz handshake
// shared among swarm subprotocols
func (b *Bzz) performHandshake(p *protocols.Peer, handshake *HandshakeMsg) error {
	ctx, cancel := context.WithTimeout(context.Background(), bzzHandshakeTimeout)
	defer func() {
		close(handshake.done)
		cancel()
	}()
	rsh, err := p.Handshake(ctx, handshake, b.checkHandshake)
	if err != nil {
		handshake.err = err
		return err
	}
	handshake.peerAddr = rsh.(*HandshakeMsg).Addr
	handshake.Capabilities = rsh.(*HandshakeMsg).Capabilities
	return nil
}

// runBzz is the p2p protocol run function for the bzz base protocol
// that negotiates the bzz handshake
func (b *Bzz) runBzz(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	handshake, _ := b.GetOrCreateHandshake(p.ID())
	if !<-handshake.init {
		return fmt.Errorf("%08x: bzz already started on peer %08x", b.localAddr.Over()[:4], p.ID().Bytes()[:4])
	}
	close(handshake.init)
	defer b.removeHandshake(p.ID())
	peer := protocols.NewPeer(p, rw, BzzSpec)
	err := b.performHandshake(peer, handshake)
	if err != nil {
		log.Warn(fmt.Sprintf("%08x: handshake failed with remote peer %08x: %v", b.localAddr.Over()[:4], p.ID().Bytes()[:4], err))

		return err
	}
	// fail if we get another handshake
	msg, err := rw.ReadMsg()
	if err != nil {
		return err
	}
	msg.Discard()
	return errors.New("received multiple handshakes")
}

// BzzPeer is the bzz protocol view of a protocols.Peer (itself an extension of p2p.Peer)
// implements the Peer interface and all interfaces Peer implements: Addr, OverlayPeer
type BzzPeer struct {
	*protocols.Peer           // represents the connection for online peers
	*BzzAddr                  // remote address -> implements Addr interface = protocols.Peer
	lastActive      time.Time // time is updated whenever mutexes are releasing
	LightNode       bool
}

func NewBzzPeer(p *protocols.Peer) *BzzPeer {
	return &BzzPeer{Peer: p, BzzAddr: NewAddr(p.Node())}
}

// ID returns the peer's underlay node identifier.
func (p *BzzPeer) ID() enode.ID {
	// This is here to resolve a method tie: both protocols.Peer and BzzAddr are embedded
	// into the struct and provide ID(). The protocols.Peer version is faster, ensure it
	// gets used.
	return p.Peer.ID()
}

/*
 Handshake

* Version: 8 byte integer version of the protocol
* NetworkID: 8 byte integer network identifier
* Addr: the address advertised by the node including underlay and overlay connecctions
* Capabilities: the capabilities bitvector
*/
type HandshakeMsg struct {
	Version      uint64
	NetworkID    uint64
	Addr         *BzzAddr
	Capabilities Capabilities

	// peerAddr is the address received in the peer handshake
	peerAddr *BzzAddr

	init chan bool
	done chan struct{}
	err  error
}

// String pretty prints the handshake
func (bh *HandshakeMsg) String() string {
	return fmt.Sprintf("Handshake: Version: %v, NetworkID: %v, Addr: %v, peerAddr: %v, caps: %s", bh.Version, bh.NetworkID, bh.Addr, bh.peerAddr, bh.Capabilities)
}

// Perform initiates the handshake and validates the remote handshake message
func (b *Bzz) checkHandshake(hs interface{}) error {
	rhs := hs.(*HandshakeMsg)
	if rhs.NetworkID != b.NetworkID {
		return fmt.Errorf("network id mismatch %d (!= %d)", rhs.NetworkID, b.NetworkID)
	}
	if rhs.Version != uint64(BzzSpec.Version) {
		return fmt.Errorf("version mismatch %d (!= %d)", rhs.Version, BzzSpec.Version)
	}
	// temporary check for valid capability settings, legacy full/light
	if !isFullCapability(rhs.Capabilities.get(0)) && !isLightCapability(rhs.Capabilities.get(0)) {
		return fmt.Errorf("invalid capabilities setting: %s", rhs.Capabilities)
	}
	return nil
}

// removeHandshake removes handshake for peer with peerID
// from the bzz handshake store
func (b *Bzz) removeHandshake(peerID enode.ID) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	delete(b.handshakes, peerID)
}

// GetHandshake returns the bzz handhake that the remote peer with peerID sent
func (b *Bzz) GetOrCreateHandshake(peerID enode.ID) (*HandshakeMsg, bool) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	handshake, found := b.handshakes[peerID]
	if !found {
		handshake = &HandshakeMsg{
			Version:      uint64(BzzSpec.Version),
			NetworkID:    b.NetworkID,
			Addr:         b.localAddr,
			Capabilities: *b.capabilities,
			init:         make(chan bool, 1),
			done:         make(chan struct{}),
		}
		// when handhsake is first created for a remote peer
		// it is initialised with the init
		handshake.init <- true
		b.handshakes[peerID] = handshake
	}

	return handshake, found
}
