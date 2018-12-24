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

package simulations

import (
	"testing"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
)

func newTestNetwork(t *testing.T, nodeCount int) (*Network, []enode.ID) {
	adapter := adapters.NewSimAdapter(adapters.Services{
		"noopwoop": func(ctx *adapters.ServiceContext) (node.Service, error) {
			return NewNoopService(nil), nil
		},
	})

	// create network
	network := NewNetwork(adapter, &NetworkConfig{
		DefaultService: "noopwoop",
	})

	// create and start nodes
	ids := make([]enode.ID, nodeCount)
	for i := range ids {
		conf := adapters.RandomNodeConfig()
		node, err := network.NewNodeWithConfig(conf)
		if err != nil {
			t.Fatalf("error creating node: %s", err)
		}
		if err := network.Start(node.ID()); err != nil {
			t.Fatalf("error starting node: %s", err)
		}
		ids[i] = node.ID()
	}

	if len(network.Conns) > 0 {
		t.Fatal("no connections should exist after just adding nodes")
	}

	return network, ids
}

func TestConnectToLastNode(t *testing.T) {
	net, ids := newTestNetwork(t, 10)
	defer net.Shutdown()

	first := ids[0]
	if err := net.ConnectToLastNode(first); err != nil {
		t.Fatal(err)
	}

	last := ids[len(ids)-1]
	for i, id := range ids {
		if id == first || id == last {
			continue
		}

		if net.GetConn(first, id) != nil {
			t.Errorf("connection must not exist with node(ind: %v, id: %v)", i, id)
		}
	}

	if net.GetConn(first, last) == nil {
		t.Error("first and last node must be connected")
	}
}

func TestConnectToRandomNode(t *testing.T) {
	net, ids := newTestNetwork(t, 10)
	defer net.Shutdown()

	err := net.ConnectToRandomNode(ids[0])
	if err != nil {
		t.Fatal(err)
	}

	var cc int
	for i, a := range ids {
		for _, b := range ids[i:] {
			if net.GetConn(a, b) != nil {
				cc++
			}
		}
	}

	if cc != 1 {
		t.Errorf("expected one connection, got %v", cc)
	}
}

func TestConnectNodesFull(t *testing.T) {
	tests := []struct {
		name      string
		nodeCount int
	}{
		{name: "no node", nodeCount: 0},
		{name: "single node", nodeCount: 1},
		{name: "2 nodes", nodeCount: 2},
		{name: "3 nodes", nodeCount: 2},
		{name: "even number of nodes", nodeCount: 12},
		{name: "odd number of nodes", nodeCount: 13},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			net, ids := newTestNetwork(t, test.nodeCount)
			defer net.Shutdown()

			err := net.ConnectNodesFull(ids)
			if err != nil {
				t.Fatal(err)
			}

			verifyFull(t, net, ids)
		})
	}
}

func verifyFull(t *testing.T, net *Network, ids []enode.ID) {
	t.Helper()
	n := len(ids)
	var connections int
	for i, lid := range ids {
		for _, rid := range ids[i+1:] {
			if net.GetConn(lid, rid) != nil {
				connections++
			}
		}
	}

	want := n * (n - 1) / 2
	if connections != want {
		t.Errorf("wrong number of connections, got: %v, want: %v", connections, want)
	}
}

func TestConnectNodesChain(t *testing.T) {
	net, ids := newTestNetwork(t, 10)
	defer net.Shutdown()

	err := net.ConnectNodesChain(ids)
	if err != nil {
		t.Fatal(err)
	}

	verifyChain(t, net, ids)
}

func verifyChain(t *testing.T, net *Network, ids []enode.ID) {
	t.Helper()
	n := len(ids)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			c := net.GetConn(ids[i], ids[j])
			if i == j-1 {
				if c == nil {
					t.Errorf("nodes %v and %v are not connected, but they should be", i, j)
				}
			} else {
				if c != nil {
					t.Errorf("nodes %v and %v are connected, but they should not be", i, j)
				}
			}
		}
	}
}

func TestConnectNodesRing(t *testing.T) {
	net, ids := newTestNetwork(t, 10)
	defer net.Shutdown()

	err := net.ConnectNodesRing(ids)
	if err != nil {
		t.Fatal(err)
	}

	verifyRing(t, net, ids)
}

func verifyRing(t *testing.T, net *Network, ids []enode.ID) {
	t.Helper()
	n := len(ids)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			c := net.GetConn(ids[i], ids[j])
			if i == j-1 || (i == 0 && j == n-1) {
				if c == nil {
					t.Errorf("nodes %v and %v are not connected, but they should be", i, j)
				}
			} else {
				if c != nil {
					t.Errorf("nodes %v and %v are connected, but they should not be", i, j)
				}
			}
		}
	}
}

func TestConnectNodesStar(t *testing.T) {
	net, ids := newTestNetwork(t, 10)
	defer net.Shutdown()

	pivotIndex := 2

	err := net.ConnectNodesStar(ids, ids[pivotIndex])
	if err != nil {
		t.Fatal(err)
	}

	verifyStar(t, net, ids, pivotIndex)
}

func verifyStar(t *testing.T, net *Network, ids []enode.ID, centerIndex int) {
	t.Helper()
	n := len(ids)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			c := net.GetConn(ids[i], ids[j])
			if i == centerIndex || j == centerIndex {
				if c == nil {
					t.Errorf("nodes %v and %v are not connected, but they should be", i, j)
				}
			} else {
				if c != nil {
					t.Errorf("nodes %v and %v are connected, but they should not be", i, j)
				}
			}
		}
	}
}
