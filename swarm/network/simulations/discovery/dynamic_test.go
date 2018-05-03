package discovery

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/state"
)

const (
	bootNodeCount             = 3
	defaultHealthCheckRetries = 10
	defaultHealthCheckDelay   = time.Millisecond * 250
)

var (
	bootNodes       []*discover.NodeID
	ids             []discover.NodeID
	addrIdx         map[discover.NodeID][]byte
	dynamicServices adapters.Services
)

// Test to verify that restarted nodes will reach healthy state
//
// First, it brings up and connects bootnodes, and connects each of the further nodes
// to a random bootnode
//
// if network is healthy, it proceeds to stop and start a selection of nodes
// upon stop, it checks the health of the remaining nodes in the network that are up
// after starting, it performs new health checks after they have connected
// to one of their previous peers
func TestDynamicDiscovery(t *testing.T) {
	t.Run("16/4/sim", dynamicDiscoverySimulation)
}

func dynamicDiscoverySimulation(t *testing.T) {

	// this directory will keep the state store dbs
	dir, err := ioutil.TempDir("", "dynamic-discovery")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// discovery service
	dynamicServices := adapters.Services{
		"discovery": newDynamicServices(dir),
	}

	// set up locals we need
	paramstring := strings.Split(t.Name(), "/")
	nodeCount, _ := strconv.ParseInt(paramstring[1], 10, 0)
	numUpDowns, _ := strconv.ParseInt(paramstring[2], 10, 0)
	adapter := paramstring[3]

	if nodeCount < bootNodeCount {
		t.Fatalf("nodeCount must be bigger than bootnodeCount (%d < %d)", nodeCount, bootNodeCount)
	}

	bootNodes = make([]*discover.NodeID, bootNodeCount)
	ids = make([]discover.NodeID, nodeCount)
	addrIdx = make(map[discover.NodeID][]byte)

	healthCheckDelay := defaultHealthCheckDelay
	healthCheckRetries := defaultHealthCheckRetries

	log.Info("starting dynamic test", "nodecount", nodeCount, "adaptertype", adapter)

	// select adapter
	var a adapters.NodeAdapter
	if adapter == "exec" {
		dirname, err := ioutil.TempDir(".", "")
		if err != nil {
			t.Fatal(err)
		}
		a = adapters.NewExecAdapter(dirname)
	} else if adapter == "sock" {
		a = adapters.NewSocketAdapter(dynamicServices)
	} else if adapter == "tcp" {
		a = adapters.NewTCPAdapter(dynamicServices)
	} else if adapter == "sim" {
		a = adapters.NewSimAdapter(dynamicServices)
	}

	// create network
	net := simulations.NewNetwork(a, &simulations.NetworkConfig{
		ID:             "0",
		DefaultService: "discovery",
	})
	defer net.Shutdown()

	// create simnodes
	for i := 0; i < int(nodeCount); i++ {
		conf := adapters.RandomNodeConfig()
		node, err := net.NewNodeWithConfig(conf)
		if err != nil {
			t.Fatalf("error starting node: %s", err)
		}
		ids[i] = node.ID()
		log.Debug("new node", "id", ids[i])
		if i < bootNodeCount {
			bootNodes[i] = &ids[i]
		}
		addrIdx[node.ID()] = network.ToOverlayAddr(node.ID().Bytes()) //fmt.Sprintf("%x", network.ToOverlayAddr(node.ID().Bytes()))
	}

	// sim step 1
	// start nodes, trigger them on node up event from sim
	trigger := make(chan discover.NodeID)
	events := make(chan *simulations.Event)
	defer func() {
		go func() {
			for range events {
			}
		}()
	}()
	sub := net.Events().Subscribe(events)
	defer sub.Unsubscribe()

	// quitC stops the event listener loops
	// inside the step action method after step is complete
	log.Info("starting step 1")
	quitC := make(chan struct{})

	action := func(ctx context.Context) error {
		go func() {
			for {
				select {
				case ev := <-events:
					if ev.Type == simulations.EventTypeNode {
						if ev.Node.Up {
							log.Info("got node up event", "event", ev, "node", ev.Node.Config.ID)
							trigger <- ev.Node.Config.ID
						}
					}
				case <-ctx.Done():
					return
				case <-quitC:
					return
				}

			}
		}()
		go func() {
			for _, n := range ids {
				if err := net.Start(n); err != nil {
					t.Fatalf("error starting node: %s", err)
				}
				log.Info("network start returned", "node", n)
			}
		}()
		return nil

	}

	check := func(ctx context.Context, nodeId discover.NodeID) (bool, error) {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}
		log.Info("trigger expect up", "node", nodeId)
		return true, nil
	}

	timeout := 10 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	result := simulations.NewSimulation(net).Run(ctx, &simulations.Step{
		Action:  action,
		Trigger: trigger,
		Expect: &simulations.Expectation{
			Nodes: ids,
			Check: check,
		},
	})

	if result.Error != nil {
		t.Fatal(result.Error)
	}

	// sim step 2
	// connect the three bootnodes together 1 -> 2 -> .. -> n
	// connect each of the other nodes to a random bootnode
	// triggers on connection event from sim
	close(quitC)
	log.Info("starting step 2")
	quitC = make(chan struct{})
	action = func(ctx context.Context) error {
		go func() {
			for {
				select {
				case ev := <-events:
					if ev.Type == simulations.EventTypeConn {
						if ev.Conn.Up {
							log.Info(fmt.Sprintf("got conn up event %v", ev))
							trigger <- ev.Conn.One
						}
					}
				case <-ctx.Done():
					return
				case <-quitC:
					return
				}
			}
		}()
		go func() {
			for i := range ids {
				var j int
				if i == 0 {
					continue
				}
				if i < len(bootNodes) {
					j = i - 1
				} else {
					j = rand.Intn(len(bootNodes) - 1)
				}

				if err := net.Connect(ids[i], ids[j]); err != nil {
					t.Fatalf("error connecting node %x => bootnode %x: %s", ids[i], ids[j], err)
				}
				log.Info("network connect returned", "one", ids[i], "other", ids[j])
			}
		}()
		return nil
	}

	check = func(ctx context.Context, id discover.NodeID) (bool, error) {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}
		log.Info("trigger expect conn", "node", id)
		return true, nil
	}

	timeout = 10 * time.Second
	ctx, cancel = context.WithTimeout(context.Background(), timeout)
	defer cancel()
	result = simulations.NewSimulation(net).Run(ctx, &simulations.Step{
		Action:  action,
		Trigger: trigger,
		Expect: &simulations.Expectation{
			Nodes: ids[1:],
			Check: check,
		},
	})
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	// sim step 3
	// now all nodes are up, all nodes are connected to network
	// so we check health of all nodes
	close(quitC)
	log.Info("starting step 3")
	quitC = make(chan struct{})
	action = func(ctx context.Context) error {
		for _, n := range net.GetNodes() {
			go func(n *simulations.Node) {
				tick := time.NewTicker(healthCheckDelay)
				for {
					select {
					case <-events:
					case <-tick.C:
						trigger <- n.ID()
					case <-ctx.Done():
						return
					case <-quitC:
						return
					}
				}
			}(n)
		}
		return nil
	}

	check = func(ctx context.Context, id discover.NodeID) (bool, error) {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}
		log.Info("health ok", "node", id)
		return checkHealth(net, id)
	}

	timeout = 10 * time.Second
	ctx, cancel = context.WithTimeout(context.Background(), timeout)
	defer cancel()
	result = simulations.NewSimulation(net).Run(ctx, &simulations.Step{
		Action:  action,
		Trigger: trigger,
		Expect: &simulations.Expectation{
			Nodes: ids[:],
			Check: check,
		},
	})
	if result.Error != nil {
		t.Fatal(result.Error)
	}

	log.Info("starting step 4")
	// sim step 4
	// bring the nodes up and down
	// if any health checks fail, the step will fail
	// check will be triggered when the first node up event is received
	close(quitC)
	quitC = make(chan struct{})
	victimSliceOffset := rand.Intn(len(ids) - int(numUpDowns) - 1)
	victimNodes := ids[victimSliceOffset : victimSliceOffset+int(numUpDowns)]

	wg := sync.WaitGroup{}
	wg.Add(len(victimNodes))

	action = func(ctx context.Context) error {
		for _, nid := range victimNodes {
			stopC := make(chan struct{})
			go func(nid discover.NodeID, stopC chan struct{}) {
				var stopped bool
				var upped bool
				for {
					select {
					case ev := <-events:
						if ev.Type == simulations.EventTypeNode {
							if ev.Node.Config.ID == nid {
								if ev.Node.Up && stopped && !upped {
									log.Info(fmt.Sprintf("got node up event %v", ev))
									upped = true
									trigger <- nid

								} else {
									if !stopped {
										log.Info(fmt.Sprintf("got node down event %v", ev))

										stopped = true
										close(stopC)
									}
								}
							}
						}
					case <-ctx.Done():
						return
					case <-quitC:
						return
					}
				}
			}(nid, stopC)

			// stop the node
			log.Info("restarting: stop", "node", nid, "addr", fmt.Sprintf("%x", addrIdx[nid]))
			err := net.Stop(nid)
			if err != nil {
				t.Fatal(err)
			}

			// wait for the stop event
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-quitC:
				return nil
			case <-stopC:
			}

			// check health of remaining nodes
		OUTER:
			for _, n := range net.GetUpNodes() {
				tick := time.NewTicker(healthCheckDelay)
				for i := 0; ; i++ {
					select {
					case <-tick.C:
						log.Debug("health check other node after stop", "stoppednode", nid, "checknode", n.ID(), "attempt", i)
						ok, err := checkHealth(net, n.ID())
						if ok {
							log.Info("health ok other node after stop", "stoppednode", nid, "checknode", n.ID())
							continue OUTER
						} else if err != nil {
							return err
						}
					case <-ctx.Done():
						return fmt.Errorf("health not reached for node %s (addr %s) after stopped node %s (addr %s)", n.ID().TerminalString(), fmt.Sprintf("%x", addrIdx[n.ID()][:8]), nid.TerminalString(), fmt.Sprintf("%x", addrIdx[nid][:8]))
					case <-quitC:
						return nil
					}
				}
			}

			// bring the node back up
			log.Info("restarting: start", "node", nid, "addr", fmt.Sprintf("%x", addrIdx[nid]))
			err = net.Start(nid)
			if err != nil {
				t.Fatal(err)
			}

		}
		return nil
	}

	check = func(ctx context.Context, id discover.NodeID) (bool, error) {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		// health check for the restarted node
		for i := 0; i < healthCheckRetries; i++ {
			log.Debug("health check after restart", "node", id, "attempt", i)
			ok, err := checkHealth(net, id)
			if ok {
				log.Info("health ok after restart", "node", id)
				return true, nil
			} else if err != nil {
				return false, err
			}
			time.Sleep(healthCheckDelay)
		}
		return false, fmt.Errorf("health not reached for node %s (addr %s)", id.TerminalString(), fmt.Sprintf("%x", addrIdx[id][:8]))
	}

	timeout = 300 * time.Second
	ctx, cancel = context.WithTimeout(context.Background(), timeout)
	defer cancel()
	result = simulations.NewSimulation(net).Run(ctx, &simulations.Step{
		Action:  action,
		Trigger: trigger,
		Expect: &simulations.Expectation{
			Nodes: victimNodes,
			Check: check,
		},
	})
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	close(quitC)

	log.Warn("exiting test")
}

func randomDelay(maxDuration int) time.Duration {
	if maxDuration == 0 {
		maxDuration = 1000000000
	}
	timeout := rand.Intn(maxDuration) + 10000000
	ns := fmt.Sprintf("%dns", timeout)
	dur, _ := time.ParseDuration(ns)
	return dur
}

func checkHealth(net *simulations.Network, id discover.NodeID) (bool, error) {
	healthy := &network.Health{}

	var upAddrs [][]byte
	for _, n := range net.GetUpNodes() {
		upAddrs = append(upAddrs, addrIdx[n.ID()])
	}
	log.Debug("generating new peerpotmap", "node", id, "addr", fmt.Sprintf("%x", addrIdx[id]))
	hotPot := network.NewPeerPotMap(testMinProxBinSize, upAddrs)
	addrHex := fmt.Sprintf("%x", addrIdx[id])

	if _, ok := hotPot[addrHex]; !ok {
		log.Debug("missing pot", "node", id, "addr", fmt.Sprintf("%x", addrHex[:8]))
		return false, nil
	}
	client, err := net.GetNode(id).Client()
	if err != nil {
		return false, fmt.Errorf("error getting node client for node %v: %v", id, err)
	}
	if err := client.Call(&healthy, "hive_healthy", hotPot[addrHex]); err != nil {
		return false, fmt.Errorf("error retrieving node health by rpc for node %v: %v", id, err)
	}
	if !(healthy.KnowNN && healthy.GotNN && healthy.Full) {
		log.Debug(fmt.Sprintf("healthy not yet reached\n%s", healthy.Hive), "id", id, "addr", fmt.Sprintf("%x", addrIdx[id][:8]), "missing", network.LogAddrs(healthy.CulpritsNN), "knowNN", healthy.KnowNN, "gotNN", healthy.GotNN, "countNN", healthy.CountNN, "full", healthy.Full)
		return false, nil
	}
	return true, nil

}

func newDynamicServices(storePath string) func(*adapters.ServiceContext) (node.Service, error) {
	return func(ctx *adapters.ServiceContext) (node.Service, error) {
		host := adapters.ExternalIP()

		addr := network.NewAddrFromNodeIDAndPort(ctx.Config.ID, host, ctx.Config.Port)

		kp := network.NewKadParams()
		kp.MinProxBinSize = testMinProxBinSize
		kp.MaxBinSize = 3
		kp.MinBinSize = 1
		kp.MaxRetries = 10
		kp.RetryExponent = 2
		kp.RetryInterval = 50000000
		if ctx.Config.Reachable != nil {
			kp.Reachable = func(o network.OverlayAddr) bool {
				return ctx.Config.Reachable(o.(*network.BzzAddr).ID())
			}
		}
		kad := network.NewKademlia(addr.Over(), kp)

		hp := network.NewHiveParams()
		hp.KeepAliveInterval = 200 * time.Millisecond

		config := &network.BzzConfig{
			OverlayAddr:  addr.Over(),
			UnderlayAddr: addr.Under(),
			HiveParams:   hp,
		}

		stateStore, err := state.NewDBStore(filepath.Join(storePath, fmt.Sprintf("state-store-%s.db", ctx.Config.ID)))
		if err != nil {
			return nil, err
		}
		return network.NewBzz(config, kad, stateStore, nil, nil), nil
	}
}
