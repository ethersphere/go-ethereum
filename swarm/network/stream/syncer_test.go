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
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/network/simulation"
	"github.com/ethereum/go-ethereum/swarm/state"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/testutil"
)

const dataChunkCount = 1000

func TestSyncerSimulation(t *testing.T) {
	testSyncBetweenNodes(t, 2, dataChunkCount, true, 1)
	// This test uses much more memory when running with
	// race detector. Allow it to finish successfully by
	// reducing its scope, and still check for data races
	// with the smallest number of nodes.
	/*if !testutil.RaceEnabled {
		testSyncBetweenNodes(t, 4, dataChunkCount, true, 1)
		testSyncBetweenNodes(t, 8, dataChunkCount, true, 1)
		testSyncBetweenNodes(t, 16, dataChunkCount, true, 1)
	}*/
}

func testSyncBetweenNodes(t *testing.T, nodes, chunkCount int, skipCheck bool, po uint8) {

	sim := simulation.New(map[string]simulation.ServiceFunc{
		"streamer": func(ctx *adapters.ServiceContext, bucket *sync.Map) (s node.Service, cleanup func(), err error) {
			addr := network.NewAddr(ctx.Config.Node())
			//hack to put addresses in same space
			addr.OAddr[0] = byte(0)

			netStore, delivery, clean, err := newNetStoreAndDeliveryWithBzzAddr(ctx, bucket, addr)
			if err != nil {
				return nil, nil, err
			}

			var dir string
			var store *state.DBStore
			if testutil.RaceEnabled {
				// Use on-disk DBStore to reduce memory consumption in race tests.
				dir, err = ioutil.TempDir("", "swarm-stream-")
				if err != nil {
					return nil, nil, err
				}
				store, err = state.NewDBStore(dir)
				if err != nil {
					return nil, nil, err
				}
			} else {
				store = state.NewInmemoryStore()
			}

			r := NewRegistry(addr.ID(), delivery, netStore, store, &RegistryOptions{
				Syncing:         SyncingAutoSubscribe,
				SyncUpdateDelay: 50 * time.Millisecond,
				SkipCheck:       skipCheck,
			}, nil)

			cleanup = func() {
				r.Close()
				clean()
				if dir != "" {
					os.RemoveAll(dir)
				}
			}

			return r, cleanup, nil
		},
	})
	defer sim.Close()

	// create context for simulation run
	timeout := 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	// defer cancel should come before defer simulation teardown
	defer cancel()

	_, err := sim.AddNodesAndConnectChain(nodes)
	if err != nil {
		t.Fatal(err)
	}
	result := sim.Run(ctx, func(ctx context.Context, sim *simulation.Simulation) (err error) {
		nodeIDs := sim.UpNodeIDs()

		nodeIndex := make(map[enode.ID]int)
		for i, id := range nodeIDs {
			nodeIndex[id] = i
		}

		disconnected := watchDisconnections(ctx, sim)
		defer func() {
			if err != nil && disconnected.bool() {
				err = errors.New("disconnect events received")
			}
		}()

		// each node Subscribes to each other's swarmChunkServerStreamName
		item, ok := sim.NodeItem(nodeIDs[0], bucketKeyFileStore)
		if !ok {
			return fmt.Errorf("No filestore")
		}
		fileStore := item.(*storage.FileStore)
		size := chunkCount * chunkSize
		_, wait1, err := fileStore.Store(ctx, testutil.RandomReader(0, size), int64(size), false)
		if err != nil {
			return fmt.Errorf("fileStore.Store: %v", err)
		}
		// here we distribute chunks of a random file into stores 1...nodes
		// collect hashes in po 1 bin for each node
		_, wait2, err := fileStore.Store(ctx, testutil.RandomReader(10, size), int64(size), false)
		if err != nil {
			return fmt.Errorf("fileStore.Store: %v", err)
		}
		wait1(ctx)
		wait2(ctx)
		time.Sleep(2 * time.Second)

		log.Warn("uploader node", "enode", nodeIDs[0])
		item, ok = sim.NodeItem(nodeIDs[0], bucketKeyStore)
		if !ok {
			return fmt.Errorf("No DB")
		}
		store := item.(chunk.Store)
		until, err := store.LastPullSubscriptionBinID(po)
		if err != nil {
			return err
		}

		for idx, node := range nodeIDs {
			if nodeIDs[idx] == nodeIDs[0] {
				continue
			}

			i := nodeIndex[node]

			log.Warn("compare to", "enode", nodeIDs[idx])
			item, ok = sim.NodeItem(nodeIDs[idx], bucketKeyStore)
			if !ok {
				return fmt.Errorf("No DB")
			}
			db := item.(chunk.Store)
			shouldUntil, err := db.LastPullSubscriptionBinID(po)
			if err != nil {
				t.Fatal(err)
			}
			log.Warn("last pull subscription bin id", "shouldUntil", shouldUntil, "until", until, "po", po)
			if shouldUntil != until {
				t.Fatalf("did not get correct bin index from peer. got %d want %d", shouldUntil, until)
			}

			log.Warn("sync check", "node", node, "index", i, "bin", po)
		}
		return nil
	})

	if result.Error != nil {
		t.Fatal(result.Error)
	}
}

//TestSameVersionID just checks that if the version is not changed,
//then streamer peers see each other
func TestSameVersionID(t *testing.T) {
	//test version ID
	v := uint(1)
	sim := simulation.New(map[string]simulation.ServiceFunc{
		"streamer": func(ctx *adapters.ServiceContext, bucket *sync.Map) (s node.Service, cleanup func(), err error) {
			addr, netStore, delivery, clean, err := newNetStoreAndDelivery(ctx, bucket)
			if err != nil {
				return nil, nil, err
			}

			r := NewRegistry(addr.ID(), delivery, netStore, state.NewInmemoryStore(), &RegistryOptions{
				Syncing: SyncingAutoSubscribe,
			}, nil)
			bucket.Store(bucketKeyRegistry, r)

			//assign to each node the same version ID
			r.spec.Version = v

			cleanup = func() {
				r.Close()
				clean()
			}

			return r, cleanup, nil
		},
	})
	defer sim.Close()

	//connect just two nodes
	log.Info("Adding nodes to simulation")
	_, err := sim.AddNodesAndConnectChain(2)
	if err != nil {
		t.Fatal(err)
	}

	log.Info("Starting simulation")
	ctx := context.Background()
	//make sure they have time to connect
	time.Sleep(200 * time.Millisecond)
	result := sim.Run(ctx, func(ctx context.Context, sim *simulation.Simulation) error {
		//get the pivot node's filestore
		nodes := sim.UpNodeIDs()

		item, ok := sim.NodeItem(nodes[0], bucketKeyRegistry)
		if !ok {
			return fmt.Errorf("No filestore")
		}
		registry := item.(*Registry)

		//the peers should connect, thus getting the peer should not return nil
		if registry.getPeer(nodes[1]) == nil {
			return errors.New("Expected the peer to not be nil, but it is")
		}
		return nil
	})
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	log.Info("Simulation ended")
}

//TestDifferentVersionID proves that if the streamer protocol version doesn't match,
//then the peers are not connected at streamer level
func TestDifferentVersionID(t *testing.T) {
	//create a variable to hold the version ID
	v := uint(0)
	sim := simulation.New(map[string]simulation.ServiceFunc{
		"streamer": func(ctx *adapters.ServiceContext, bucket *sync.Map) (s node.Service, cleanup func(), err error) {
			addr, netStore, delivery, clean, err := newNetStoreAndDelivery(ctx, bucket)
			if err != nil {
				return nil, nil, err
			}

			r := NewRegistry(addr.ID(), delivery, netStore, state.NewInmemoryStore(), &RegistryOptions{
				Syncing: SyncingAutoSubscribe,
			}, nil)
			bucket.Store(bucketKeyRegistry, r)

			//increase the version ID for each node
			v++
			r.spec.Version = v

			cleanup = func() {
				r.Close()
				clean()
			}

			return r, cleanup, nil
		},
	})
	defer sim.Close()

	//connect the nodes
	log.Info("Adding nodes to simulation")
	_, err := sim.AddNodesAndConnectChain(2)
	if err != nil {
		t.Fatal(err)
	}

	log.Info("Starting simulation")
	ctx := context.Background()
	//make sure they have time to connect
	time.Sleep(200 * time.Millisecond)
	result := sim.Run(ctx, func(ctx context.Context, sim *simulation.Simulation) error {
		//get the pivot node's filestore
		nodes := sim.UpNodeIDs()

		item, ok := sim.NodeItem(nodes[0], bucketKeyRegistry)
		if !ok {
			return fmt.Errorf("No filestore")
		}
		registry := item.(*Registry)

		//getting the other peer should fail due to the different version numbers
		if registry.getPeer(nodes[1]) != nil {
			return errors.New("Expected the peer to be nil, but it is not")
		}
		return nil
	})
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	log.Info("Simulation ended")

}

// Tests that when two nodes connect:
// 1. All subscriptions are created
// 2. All chunks are transferred from one node to another
func TestTwoNodesFullSync(t *testing.T) { //
	const chunkCount = 1000

	sim := simulation.New(map[string]simulation.ServiceFunc{
		"streamer": func(ctx *adapters.ServiceContext, bucket *sync.Map) (s node.Service, cleanup func(), err error) {
			addr := network.NewAddr(ctx.Config.Node())
			//hack to put addresses in same space
			addr.OAddr[0] = byte(0)

			netStore, delivery, clean, err := newNetStoreAndDeliveryWithBzzAddr(ctx, bucket, addr)
			if err != nil {
				return nil, nil, err
			}

			var dir string
			var store *state.DBStore
			if testutil.RaceEnabled {
				// Use on-disk DBStore to reduce memory consumption in race tests.
				dir, err = ioutil.TempDir("", "swarm-stream-")
				if err != nil {
					return nil, nil, err
				}
				store, err = state.NewDBStore(dir)
				if err != nil {
					return nil, nil, err
				}
			} else {
				store = state.NewInmemoryStore()
			}

			r := NewRegistry(addr.ID(), delivery, netStore, store, &RegistryOptions{
				Syncing:         SyncingAutoSubscribe,
				SyncUpdateDelay: 50 * time.Millisecond,
				SkipCheck:       true,
			}, nil)

			cleanup = func() {
				r.Close()
				clean()
				if dir != "" {
					os.RemoveAll(dir)
				}
			}

			return r, cleanup, nil
		},
	})
	defer sim.Close()

	// create context for simulation run
	timeout := 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	// defer cancel should come before defer simulation teardown
	defer cancel()

	_, err := sim.AddNodesAndConnectChain(2)
	if err != nil {
		t.Fatal(err)
	}

	result := sim.Run(ctx, func(ctx context.Context, sim *simulation.Simulation) (err error) {
		nodeIDs := sim.UpNodeIDs()

		nodeIndex := make(map[enode.ID]int)
		for i, id := range nodeIDs {
			nodeIndex[id] = i
		}

		disconnected := watchDisconnections(ctx, sim)
		defer func() {
			if err != nil && disconnected.bool() {
				err = errors.New("disconnect events received")
			}
		}()

		// each node Subscribes to each other's swarmChunkServerStreamName
		item, ok := sim.NodeItem(nodeIDs[0], bucketKeyFileStore)
		if !ok {
			return fmt.Errorf("No filestore")
		}
		fileStore := item.(*storage.FileStore)
		size := chunkCount * chunkSize

		_, wait1, err := fileStore.Store(ctx, testutil.RandomReader(0, size), int64(size), false)
		if err != nil {
			return fmt.Errorf("fileStore.Store: %v", err)
		}

		_, wait2, err := fileStore.Store(ctx, testutil.RandomReader(10, size), int64(size), false)
		if err != nil {
			return fmt.Errorf("fileStore.Store: %v", err)
		}

		wait1(ctx)
		wait2(ctx)
		time.Sleep(5 * time.Second)

		log.Warn("uploader node", "enode", nodeIDs[0])
		item, ok = sim.NodeItem(nodeIDs[0], bucketKeyStore)
		if !ok {
			return fmt.Errorf("No DB")
		}
		store := item.(chunk.Store)
		uploaderNodeBinIDs := make([]uint64, 17)

		log.Debug("checking pull subscription bin ids")
		for po := 0; po <= 16; po++ {
			until, err := store.LastPullSubscriptionBinID(uint8(po))
			if err != nil {
				t.Fatal(err)
			}

			uploaderNodeBinIDs[po] = until
		}

		for idx, _ := range nodeIDs {
			if nodeIDs[idx] == nodeIDs[0] {
				continue
			}

			//nodeIdx := nodeIndex[node]

			log.Warn("compare to", "enode", nodeIDs[idx])
			item, ok = sim.NodeItem(nodeIDs[idx], bucketKeyStore)
			if !ok {
				return fmt.Errorf("No DB")
			}
			db := item.(chunk.Store)

			time.Sleep(5 * time.Second)
			uploaderSum, otherSum := 0, 0
			for po, uploaderUntil := range uploaderNodeBinIDs {
				shouldUntil, err := db.LastPullSubscriptionBinID(uint8(po))
				if err != nil {
					t.Fatal(err)
				}
				otherSum += int(shouldUntil)
				uploaderSum += int(uploaderUntil)
			}
			//				log.Warn("last pull subscription bin id", "shouldUntil", shouldUntil, "uploader node until", uploaderUntil, "po", po)
			if uploaderSum != otherSum {
				t.Fatalf("did not get correct bin index from peer. got %d want %d", uploaderSum, otherSum)
			}
			//log.Warn("sync check", "node", node, "index", nodeIdx, "bin", po)

		}
		return nil
	})

	if result.Error != nil {
		t.Fatal(result.Error)
	}
}

// connect simulation of X nodes in a star topology (min 8 nodes)
// get all chunk refs from the smoke test util
// let them sync
// iterate over all chunk refs
// for each chunk, check pos with all nodes, for the node with highest PO - check if that node has the chunk in its localstore (with a GET)
// iterate over the subscirptions and if the node has only subscription 0, it should not have chunks
// Anton Evangelatov @nonsense 17:39
// exclusivity test (after we do the most prox inclusivity test, similar to the smoke test)
// ---
// uploader node 0
// node 1 -> 0 -> does not have chunks from 1 2 3 4 5 .. 16
//  - pick random (or any) chunk from bin 1 2 3 4 5 .. 16 from uploader node
//  - localstore get on node 1 (without caring about po of node 1 and the chunk)
//node 2 -> 1 -> does not have chunks from 0 2 3 4 5 .. 16
//node 3 -> 2 3 4 5 .. 16 -> does not have chunks from 0 1
/*

   //create rpc client
   client, err := node.Client()
   if err != nil {
       return fmt.Errorf("create node 1 rpc client fail: %v", err)
   }

   //ask it for subscriptions
   pstreams := make(map[string][]string)
   err = client.Call(&pstreams, "stream_getPeerServerSubscriptions")
   if err != nil {
       return fmt.Errorf("client call stream_getPeerSubscriptions: %v", err)
   }
*/
