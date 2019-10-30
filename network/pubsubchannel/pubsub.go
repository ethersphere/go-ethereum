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
package pubsubchannel

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethersphere/swarm/log"
)

//PubSubChannel represents a pubsub system where subscriber can .Subscribe() and publishers can .Publish() or .Close().
type PubSubChannel struct {
	subscriptions []*Subscription
	subsMutex     sync.RWMutex
	nextId        int
	quitC         chan struct{}
}

// Subscription is created in PubSubChannel using pubSub.Subscribe(). Subscribers can receive using .ReceiveChannel().
// or .Unsubscribe()
type Subscription struct {
	closed    bool
	pubSubC   *PubSubChannel
	signal    chan interface{}
	closeOnce sync.Once
	id        string
	lock      sync.RWMutex
	quitC     chan struct{} // close channel for publisher goroutines
}

// New creates a new PubSubChannel.
func New() *PubSubChannel {
	return &PubSubChannel{
		subscriptions: make([]*Subscription, 0),
		quitC:         make(chan struct{}),
	}
}

// Subscribe creates a subscription to a channel, each subscriber should keep its own Subscription instance.
func (psc *PubSubChannel) Subscribe() *Subscription {
	psc.subsMutex.Lock()
	defer psc.subsMutex.Unlock()
	newSubscription := newSubscription(strconv.Itoa(psc.nextId), psc)
	psc.nextId++
	psc.subscriptions = append(psc.subscriptions, &newSubscription)

	return &newSubscription
}

func (psc *PubSubChannel) removeSub(s *Subscription) {
	psc.subsMutex.Lock()
	defer psc.subsMutex.Unlock()

	for i, subscription := range psc.subscriptions {
		if subscription.signal == s.signal {
			log.Debug("Unsubscribing", "id", subscription.id)
			subscription.lock.Lock()
			subscription.closed = true
			subscription.lock.Unlock()
			psc.subscriptions = append(psc.subscriptions[:i], psc.subscriptions[i+1:]...)
		}
	}
}

// Publish broadcasts a message asynchronously to each subscriber.
// If some of the subscriptions(channels) has been marked as closeable, it does it now.
func (psc *PubSubChannel) Publish(msg interface{}) {
	psc.subsMutex.RLock()
	defer psc.subsMutex.RUnlock()
	for _, sub := range psc.subscriptions {
		go func(sub *Subscription) {
			sub.lock.Lock()
			defer sub.lock.Unlock()
			metrics.GetOrRegisterCounter(fmt.Sprintf("pubsubchannel.%v.pending", sub.id), nil).Inc(1)
			defer metrics.GetOrRegisterCounter(fmt.Sprintf("pubsubchannel.%v.pending", sub.id), nil).Inc(-1)
			//atomic.AddInt64(sub.pending, 1)
			//defer atomic.AddInt64(sub.pending, -1)
			if sub.closed {
				log.Debug("Subscription was closed", "id", sub.id)
				sub.closeChannel()
			} else {
				select {
				case sub.signal <- msg:
					metrics.GetOrRegisterCounter(fmt.Sprintf("pubsubchannel.%v.delivered", sub.id), nil).Inc(1)
					//atomic.AddInt64(sub.msgCount, 1)
				case <-psc.quitC:
				case <-sub.quitC:
				}
			}

		}(sub)
	}
}

// NumSubscriptions returns how many subscriptions are currently active.
func (psc *PubSubChannel) NumSubscriptions() int {
	psc.subsMutex.RLock()
	defer psc.subsMutex.RUnlock()
	return len(psc.subscriptions)
}

// Close cancels all subscriptions closing the channels associated with them.
// Usually the publisher is in charge of calling Close().
func (psc *PubSubChannel) Close() {
	psc.subsMutex.Lock()
	defer psc.subsMutex.Unlock()
	for _, sub := range psc.subscriptions {
		sub.lock.Lock()
		sub.closed = true
		sub.closeChannel()
		sub.lock.Unlock()
	}
	close(psc.quitC)
}

// Unsubscribe cancels subscription from the subscriber side. Channel is marked as closed but only writer should close it.
func (sub *Subscription) Unsubscribe() {
	close(sub.quitC)
	sub.pubSubC.removeSub(sub)
}

// ReceiveChannel returns the channel where the subscriber will receive messages.
func (sub *Subscription) ReceiveChannel() <-chan interface{} {
	return sub.signal
}

// IsClosed returns if the subscription is closed via Unsubscribe() or Close() in the pubSub that creates it.
func (sub *Subscription) IsClosed() bool {
	sub.lock.RLock()
	defer sub.lock.RUnlock()
	return sub.closed
}

// ID returns a unique id in the PubSubChannel of this subscription. Useful for debugging.
func (sub *Subscription) ID() string {
	return sub.id
}

func (sub *Subscription) closeChannel() {
	sub.closeOnce.Do(func() {
		close(sub.signal)
	})
}

func (sub *Subscription) MessageCount() int64 {
	return metrics.GetOrRegisterCounter(fmt.Sprintf("pubsubchannel.%v.delivered", sub.id), nil).Count()
}

func (sub *Subscription) Pending() int64 {
	return metrics.GetOrRegisterCounter(fmt.Sprintf("pubsubchannel.%v.pending", sub.id), nil).Count()
}

func newSubscription(id string, psc *PubSubChannel) Subscription {
	return Subscription{
		closed:    false,
		pubSubC:   psc,
		signal:    make(chan interface{}),
		closeOnce: sync.Once{},
		id:        id,
		quitC:     make(chan struct{}),
	}
}
