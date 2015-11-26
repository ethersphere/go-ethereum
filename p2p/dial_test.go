// Copyright 2015 The go-ethereum Authors
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

package p2p

import (
	"encoding/binary"
	"reflect"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

func init() {
	spew.Config.Indent = "\t"
	spew.Config.DisableMethods = true
	// glog.SetV(8)
	// glog.SetToStderr(true)
}

type dialtest struct {
	init   *dialstate // state before and after the test.
	rounds []round
}

type round struct {
	peers []*Peer // current peer set
	done  []task  // tasks that got done this round
	new   []task  // the result must match this one
}

func runDialTest(t *testing.T, test dialtest) {
	var (
		vtime   time.Time
		running int
	)
	pm := func(ps []*Peer) map[discover.NodeID]*Peer {
		m := make(map[discover.NodeID]*Peer)
		for _, p := range ps {
			m[p.conn.id] = p
		}
		return m
	}
	for i, round := range test.rounds {
		for _, task := range round.done {
			running--
			if running < 0 {
				panic("running task counter underflow")
			}
			test.init.taskDone(task, vtime)
		}

		new := test.init.newTasks(running, pm(round.peers), vtime)
		if !sametasks(new, round.new) {
			t.Errorf("round %d: new tasks mismatch:\ngot %v\nwant %v\nstate: %v\nrunning: %v\n",
				i, spew.Sdump(new), spew.Sdump(round.new), spew.Sdump(test.init), spew.Sdump(running))
		}

		// Time advances by 16 seconds on every round.
		vtime = vtime.Add(16 * time.Second)
		running += len(new)
	}
}

type fakeTable []*discover.Node

func (t fakeTable) Self() *discover.Node                     { return new(discover.Node) }
func (t fakeTable) Close()                                   {}
func (t fakeTable) Bootstrap([]*discover.Node)               {}
func (t fakeTable) Lookup(discover.NodeID) []*discover.Node  { return nil }
func (t fakeTable) Resolve(discover.NodeID) *discover.Node   { return nil }
func (t fakeTable) ReadRandomNodes(buf []*discover.Node) int { return copy(buf, t) }

// This test checks that dynamic dials are launched from discovery results.
func TestDialStateDynDial(t *testing.T) {
	runDialTest(t, dialtest{
		init: newDialState(nil, fakeTable{}, 5),
		rounds: []round{
			// A discovery query is launched.
			{
				peers: []*Peer{
					{conn: &conn{flags: staticDialedConn, id: uintID(0)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(1)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(2)}},
				},
				new: []task{&discoverTask{bootstrap: true}},
			},
			// Dynamic dials are launched when it completes.
			{
				peers: []*Peer{
					{conn: &conn{flags: staticDialedConn, id: uintID(0)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(1)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(2)}},
				},
				done: []task{
					&discoverTask{bootstrap: true, results: []*discover.Node{
						{ID: uintID(2)}, // this one is already connected and not dialed.
						{ID: uintID(3)},
						{ID: uintID(4)},
						{ID: uintID(5)},
						{ID: uintID(6)}, // these are not tried because max dyn dials is 5
						{ID: uintID(7)}, // ...
					}},
				},
				new: []task{
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(3)}, nil},
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(4)}, nil},
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(5)}, nil},
				},
			},
			// Some of the dials complete but no new ones are launched yet because
			// the sum of active dial count and dynamic peer count is == maxDynDials.
			{
				peers: []*Peer{
					{conn: &conn{flags: staticDialedConn, id: uintID(0)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(1)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(2)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(3)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(4)}},
				},
				done: []task{
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(3)}, nil},
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(4)}, nil},
				},
			},
			// No new dial tasks are launched in the this round because
			// maxDynDials has been reached.
			{
				peers: []*Peer{
					{conn: &conn{flags: staticDialedConn, id: uintID(0)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(1)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(2)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(3)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(4)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(5)}},
				},
				done: []task{
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(5)}, nil},
				},
				new: []task{
					&waitExpireTask{Duration: 14 * time.Second},
				},
			},
			// In this round, the peer with id 2 drops off. The query
			// results from last discovery lookup are reused.
			{
				peers: []*Peer{
					{conn: &conn{flags: staticDialedConn, id: uintID(0)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(1)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(3)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(4)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(5)}},
				},
				new: []task{
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(6)}, nil},
				},
			},
			// More peers (3,4) drop off and dial for ID 6 completes.
			// The last query result from the discovery lookup is reused
			// and a new one is spawned because more candidates are needed.
			{
				peers: []*Peer{
					{conn: &conn{flags: staticDialedConn, id: uintID(0)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(1)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(5)}},
				},
				done: []task{
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(6)}, nil},
				},
				new: []task{
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(7)}, nil},
					&discoverTask{},
				},
			},
			// Peer 7 is connected, but there still aren't enough dynamic peers
			// (4 out of 5). However, a discovery is already running, so ensure
			// no new is started.
			{
				peers: []*Peer{
					{conn: &conn{flags: staticDialedConn, id: uintID(0)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(1)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(5)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(7)}},
				},
				done: []task{
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(7)}, nil},
				},
			},
			// Finish the running node discovery with an empty set. A new lookup
			// should be immediately requested.
			{
				peers: []*Peer{
					{conn: &conn{flags: staticDialedConn, id: uintID(0)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(1)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(5)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(7)}},
				},
				done: []task{
					&discoverTask{},
				},
				new: []task{
					&discoverTask{},
				},
			},
		},
	})
}

func TestDialStateDynDialFromTable(t *testing.T) {
	// This table always returns the same random nodes
	// in the order given below.
	table := fakeTable{
		{ID: uintID(1)},
		{ID: uintID(2)},
		{ID: uintID(3)},
		{ID: uintID(4)},
		{ID: uintID(5)},
		{ID: uintID(6)},
		{ID: uintID(7)},
		{ID: uintID(8)},
	}

	runDialTest(t, dialtest{
		init: newDialState(nil, table, 10),
		rounds: []round{
			// Discovery bootstrap is launched.
			{
				new: []task{&discoverTask{bootstrap: true}},
			},
			// 5 out of 8 of the nodes returned by ReadRandomNodes are dialed.
			{
				done: []task{
					&discoverTask{bootstrap: true},
				},
				new: []task{
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(1)}, nil},
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(2)}, nil},
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(3)}, nil},
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(4)}, nil},
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(5)}, nil},
					&discoverTask{bootstrap: false},
				},
			},
			// Dialing nodes 1,2 succeeds. Dials from the lookup are launched.
			{
				peers: []*Peer{
					{conn: &conn{flags: dynDialedConn, id: uintID(1)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(2)}},
				},
				done: []task{
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(1)}, nil},
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(2)}, nil},
					&discoverTask{results: []*discover.Node{
						{ID: uintID(10)},
						{ID: uintID(11)},
						{ID: uintID(12)},
					}},
				},
				new: []task{
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(10)}, nil},
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(11)}, nil},
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(12)}, nil},
					&discoverTask{bootstrap: false},
				},
			},
			// Dialing nodes 3,4,5 fails. The dials from the lookup succeed.
			{
				peers: []*Peer{
					{conn: &conn{flags: dynDialedConn, id: uintID(1)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(2)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(10)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(11)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(12)}},
				},
				done: []task{
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(3)}, nil},
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(4)}, nil},
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(5)}, nil},
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(10)}, nil},
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(11)}, nil},
					&dialTask{dynDialedConn, &discover.Node{ID: uintID(12)}, nil},
				},
			},
			// Waiting for expiry. No waitExpireTask is launched because the
			// discovery query is still running.
			{
				peers: []*Peer{
					{conn: &conn{flags: dynDialedConn, id: uintID(1)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(2)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(10)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(11)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(12)}},
				},
			},
			// Nodes 3,4 are not tried again because only the first two
			// returned random nodes (nodes 1,2) are tried and they're
			// already connected.
			{
				peers: []*Peer{
					{conn: &conn{flags: dynDialedConn, id: uintID(1)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(2)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(10)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(11)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(12)}},
				},
			},
		},
	})
}

// This test checks that static dials are launched.
func TestDialStateStaticDial(t *testing.T) {
	wantStatic := []*discover.Node{
		{ID: uintID(1)},
		{ID: uintID(2)},
		{ID: uintID(3)},
		{ID: uintID(4)},
		{ID: uintID(5)},
	}

	runDialTest(t, dialtest{
		init: newDialState(wantStatic, fakeTable{}, 0),
		rounds: []round{
			// Static dials are launched for the nodes that
			// aren't yet connected.
			{
				peers: []*Peer{
					{conn: &conn{flags: dynDialedConn, id: uintID(1)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(2)}},
				},
				new: []task{
					&dialTask{staticDialedConn, &discover.Node{ID: uintID(3)}, nil},
					&dialTask{staticDialedConn, &discover.Node{ID: uintID(4)}, nil},
					&dialTask{staticDialedConn, &discover.Node{ID: uintID(5)}, nil},
				},
			},
			// No new tasks are launched in this round because all static
			// nodes are either connected or still being dialed.
			{
				peers: []*Peer{
					{conn: &conn{flags: dynDialedConn, id: uintID(1)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(2)}},
					{conn: &conn{flags: staticDialedConn, id: uintID(3)}},
				},
				done: []task{
					&dialTask{staticDialedConn, &discover.Node{ID: uintID(3)}, nil},
				},
			},
			// No new dial tasks are launched because all static
			// nodes are now connected.
			{
				peers: []*Peer{
					{conn: &conn{flags: dynDialedConn, id: uintID(1)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(2)}},
					{conn: &conn{flags: staticDialedConn, id: uintID(3)}},
					{conn: &conn{flags: staticDialedConn, id: uintID(4)}},
					{conn: &conn{flags: staticDialedConn, id: uintID(5)}},
				},
				done: []task{
					&dialTask{staticDialedConn, &discover.Node{ID: uintID(4)}, nil},
					&dialTask{staticDialedConn, &discover.Node{ID: uintID(5)}, nil},
				},
				new: []task{
					&waitExpireTask{Duration: 14 * time.Second},
				},
			},
			// Wait a round for dial history to expire, no new tasks should spawn.
			{
				peers: []*Peer{
					{conn: &conn{flags: dynDialedConn, id: uintID(1)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(2)}},
					{conn: &conn{flags: staticDialedConn, id: uintID(3)}},
					{conn: &conn{flags: staticDialedConn, id: uintID(4)}},
					{conn: &conn{flags: staticDialedConn, id: uintID(5)}},
				},
			},
			// If a static node is dropped, it should be immediately redialed,
			// irrespective whether it was originally static or dynamic.
			{
				peers: []*Peer{
					{conn: &conn{flags: dynDialedConn, id: uintID(1)}},
					{conn: &conn{flags: staticDialedConn, id: uintID(3)}},
					{conn: &conn{flags: staticDialedConn, id: uintID(5)}},
				},
				new: []task{
					&dialTask{staticDialedConn, &discover.Node{ID: uintID(2)}, nil},
					&dialTask{staticDialedConn, &discover.Node{ID: uintID(4)}, nil},
				},
			},
		},
	})
}

// This test checks that past dials are not retried for some time.
func TestDialStateCache(t *testing.T) {
	wantStatic := []*discover.Node{
		{ID: uintID(1)},
		{ID: uintID(2)},
		{ID: uintID(3)},
	}

	runDialTest(t, dialtest{
		init: newDialState(wantStatic, fakeTable{}, 0),
		rounds: []round{
			// Static dials are launched for the nodes that
			// aren't yet connected.
			{
				peers: nil,
				new: []task{
					&dialTask{staticDialedConn, &discover.Node{ID: uintID(1)}, nil},
					&dialTask{staticDialedConn, &discover.Node{ID: uintID(2)}, nil},
					&dialTask{staticDialedConn, &discover.Node{ID: uintID(3)}, nil},
				},
			},
			// No new tasks are launched in this round because all static
			// nodes are either connected or still being dialed.
			{
				peers: []*Peer{
					{conn: &conn{flags: staticDialedConn, id: uintID(1)}},
					{conn: &conn{flags: staticDialedConn, id: uintID(2)}},
				},
				done: []task{
					&dialTask{staticDialedConn, &discover.Node{ID: uintID(1)}, nil},
					&dialTask{staticDialedConn, &discover.Node{ID: uintID(2)}, nil},
				},
			},
			// A salvage task is launched to wait for node 3's history
			// entry to expire.
			{
				peers: []*Peer{
					{conn: &conn{flags: dynDialedConn, id: uintID(1)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(2)}},
				},
				done: []task{
					&dialTask{staticDialedConn, &discover.Node{ID: uintID(3)}, nil},
				},
				new: []task{
					&waitExpireTask{Duration: 14 * time.Second},
				},
			},
			// Still waiting for node 3's entry to expire in the cache.
			{
				peers: []*Peer{
					{conn: &conn{flags: dynDialedConn, id: uintID(1)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(2)}},
				},
			},
			// The cache entry for node 3 has expired and is retried.
			{
				peers: []*Peer{
					{conn: &conn{flags: dynDialedConn, id: uintID(1)}},
					{conn: &conn{flags: dynDialedConn, id: uintID(2)}},
				},
				new: []task{
					&dialTask{staticDialedConn, &discover.Node{ID: uintID(3)}, nil},
				},
			},
		},
	})
}

// compares task lists but doesn't care about the order.
func sametasks(a, b []task) bool {
	if len(a) != len(b) {
		return false
	}
next:
	for _, ta := range a {
		for _, tb := range b {
			if reflect.DeepEqual(ta, tb) {
				continue next
			}
		}
		return false
	}
	return true
}

func uintID(i uint32) discover.NodeID {
	var id discover.NodeID
	binary.BigEndian.PutUint32(id[:], i)
	return id
}
