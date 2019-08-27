// Copyright 2019 The go-ethereum Authors
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

package pushsync

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethersphere/swarm/chunk"
	"github.com/ethersphere/swarm/log"
	"github.com/ethersphere/swarm/spancontext"
	"github.com/ethersphere/swarm/storage"
	"github.com/opentracing/opentracing-go"
	olog "github.com/opentracing/opentracing-go/log"
)

// DB interface implemented by localstore
type DB interface {
	// subscribe to chunk to be push synced - iterates from earliest to newest
	SubscribePush(context.Context) (<-chan storage.Chunk, func())
	// called to set a chunk as synced - and allow it to be garbage collected
	// TODO this should take ... last argument to delete many in one batch
	Set(context.Context, chunk.ModeSet, ...storage.Address) error
}

// Pusher takes care of the push syncing
type Pusher struct {
	store    DB                     // localstore DB
	tags     *chunk.Tags            // tags to update counts
	quit     chan struct{}          // channel to signal quitting on all loops
	pushed   map[string]*pushedItem // cache of items push-synced
	receipts chan []byte            // channel to receive receipts
	ps       PubSub                 // PubSub interface to send chunks and receive receipts
}

var (
	retryInterval = 100 * time.Millisecond // seconds to wait before retry sync
)

// pushedItem captures the info needed for the pusher about a chunk during the
// push-sync--receipt roundtrip
type pushedItem struct {
	tag         *chunk.Tag       // tag for the chunk
	shortcut    bool             // if the chunk receipt was sent by self
	firstSentAt time.Time        // first sent at time
	lastSentAt  time.Time        // most recently sent at time
	synced      bool             // set when chunk got synced
	span        opentracing.Span // roundtrip span
}

// NewPusher contructs a Pusher and starts up the push sync protocol
// takes
// - a DB interface to subscribe to push sync index to allow iterating over recently stored chunks
// - a pubsub interface to send chunks and receive statements of custody
// - tags that hold the tags
func NewPusher(store DB, ps PubSub, tags *chunk.Tags) *Pusher {
	p := &Pusher{
		store:    store,
		tags:     tags,
		quit:     make(chan struct{}),
		pushed:   make(map[string]*pushedItem),
		receipts: make(chan []byte),
		ps:       ps,
	}
	go p.sync()
	return p
}

// Close closes the pusher
func (p *Pusher) Close() {
	close(p.quit)
}

// sync starts a forever loop that pushes chunks to their neighbourhood
// and receives receipts (statements of custody) for them.
// chunks that are not acknowledged with a receipt are retried
// not earlier than retryInterval after they were last pushed
// the routine also updates counts of states on a tag in order
// to monitor the proportion of saved, sent and synced chunks of
// a file or collection
func (p *Pusher) sync() {
	var chunks <-chan chunk.Chunk
	var unsubscribe func()
	var syncedAddrs []storage.Address
	var syncedItems []*pushedItem

	// timer, initially set to 0 to fall through select case on timer.C for initialisation
	timer := time.NewTimer(0)
	defer timer.Stop()

	// register handler for pssReceiptTopic on pss pubsub
	deregister := p.ps.Register(pssReceiptTopic, false, func(msg []byte, _ *p2p.Peer) error {
		return p.handleReceiptMsg(msg)
	})
	defer deregister()

	chunksInBatch := -1
	var batchStartTime time.Time
	ctx := context.Background()

	var average uint64 = 1000
	var measurements uint64

	for {
		select {

		// retry interval timer triggers starting from new
		case <-timer.C:
			// initially timer is set to go off as well as every time we hit the end of push index
			// so no wait for retryInterval needed to set  items synced
			metrics.GetOrRegisterCounter("pusher.subscribe-push", nil).Inc(1)
			// if subscribe was running, stop it
			if unsubscribe != nil {
				unsubscribe()
			}

			log.Debug("set chunk status to synced, insert to db GC index")
			syncedTags := make(map[uint32]int)
			// set chunk status to synced, insert to db GC index
			if err := p.store.Set(ctx, chunk.ModeSetSync, syncedAddrs...); err != nil {
				log.Error("error setting chunks to synced", "err", err)
				continue
			}
			for i, item := range syncedItems {
				// increment synced count for the tag if exists
				tag := item.tag
				if tag != nil {
					syncedTags[tag.Uid] = syncedTags[tag.Uid] + 1
					item.span.Finish()
				}
				delete(p.pushed, syncedAddrs[i].Hex())
			}
			// iterate over tags in this batch
			for uid, n := range syncedTags {
				tag, _ := p.tags.Get(uid)
				tag.IncN(chunk.StateSynced, n)
				if tag.Done(chunk.StateSynced) {
					log.Info("closing root span for tag", "taguid", tag.Uid, "tagname", tag.Name)
					tag.FinishRootSpan()
				}
			}
			// reset synced list
			syncedAddrs = nil
			syncedItems = nil

			// we don't want to record the first iteration
			if chunksInBatch != -1 {
				// this measurement is not a timer, but we want a histogram, so it fits the data structure
				metrics.GetOrRegisterResettingTimer("pusher.subscribe-push.chunks-in-batch.hist", nil).Update(time.Duration(chunksInBatch))
				metrics.GetOrRegisterResettingTimer("pusher.subscribe-push.chunks-in-batch.time", nil).UpdateSince(batchStartTime)
				metrics.GetOrRegisterCounter("pusher.subscribe-push.chunks-in-batch", nil).Inc(int64(chunksInBatch))
			}
			chunksInBatch = 0
			batchStartTime = time.Now()

			// and start iterating on Push index from the beginning
			chunks, unsubscribe = p.store.SubscribePush(ctx)
			// reset timer to go off after retryInterval
			timer.Reset(retryInterval)

		// handle incoming chunks
		case ch, more := <-chunks:
			// if no more, set to nil, reset timer to 0 to finalise batch immediately
			if !more {
				chunks = nil
				timer.Reset(0)
				break
			}

			chunksInBatch++
			metrics.GetOrRegisterCounter("pusher.send-chunk", nil).Inc(1)
			// if no need to sync this chunk then continue
			if !p.needToSync(ch) {
				break
			}

			metrics.GetOrRegisterCounter("pusher.send-chunk.send-to-sync", nil).Inc(1)
			// send the chunk and ignore the error
			// go func(ch chunk.Chunk) {
			if err := p.sendChunkMsg(ch); err != nil {
				log.Error("error sending chunk", "addr", ch.Address().Hex(), "err", err)
			}

		// handle incoming receipts
		case addr := <-p.receipts:
			hexaddr := hex.EncodeToString(addr)
			log.Debug("synced", "addr", hexaddr)
			metrics.GetOrRegisterCounter("pusher.receipts.all", nil).Inc(1)
			// ignore if already received receipt
			item, found := p.pushed[hexaddr]
			if !found {
				metrics.GetOrRegisterCounter("pusher.receipts.not-found", nil).Inc(1)
				log.Debug("not wanted or already got... ignore", "addr", hexaddr)
				break
			}
			if item.synced {
				metrics.GetOrRegisterCounter("pusher.receipts.already-synced", nil).Inc(1)
				log.Debug("just synced... ignore", "addr", hexaddr)
				break
			}

			totalDuration := time.Since(item.firstSentAt)
			if !item.shortcut {
				roundtripDuration := time.Since(item.lastSentAt)
				measurement := uint64(roundtripDuration) / 1000
				if 2*measurement < 3*average {
					average = (measurements*average + measurement) / (measurements + 1)
					measurements++
					retryInterval = time.Duration(average*2) * time.Millisecond
					log.Debug("time to sync", "addr", hexaddr, "total duration", totalDuration, "roundtrip duration", roundtripDuration, "n", measurements, "average", average, "retry", retryInterval)
				}
			}
			metrics.GetOrRegisterResettingTimer("pusher.chunk.roundtrip", nil).Update(totalDuration)
			metrics.GetOrRegisterCounter("pusher.receipts.synced", nil).Inc(1)
			// collect synced addresses and corresponding items to do subsequent batch operations
			syncedAddrs = append(syncedAddrs, addr)
			syncedItems = append(syncedItems, item)
			// set synced flag
			item.synced = true

		case <-p.quit:
			// if subscribe was running, stop it
			if unsubscribe != nil {
				unsubscribe()
			}
			return
		}
	}
}

// handleReceiptMsg is a handler for pssReceiptTopic that
// - deserialises receiptMsg and
// - sends the receipted address on a channel
func (p *Pusher) handleReceiptMsg(msg []byte) error {
	receipt, err := decodeReceiptMsg(msg)
	if err != nil {
		return err
	}
	log.Debug("Handler", "receipt", label(receipt.Addr), "self", label(p.ps.BaseAddr()))
	p.pushReceipt(receipt.Addr)
	return nil
}

// pushReceipt just inserts the address into the channel
func (p *Pusher) pushReceipt(addr []byte) {
	select {
	case p.receipts <- addr:
	case <-p.quit:
	}
}

// sendChunkMsg sends chunks to their destination
// using the PubSub interface Send method (e.g., pss neighbourhood addressing)
func (p *Pusher) sendChunkMsg(ch chunk.Chunk) error {
	rlpTimer := time.Now()

	cmsg := &chunkMsg{
		Origin: p.ps.BaseAddr(),
		Addr:   ch.Address()[:],
		Data:   ch.Data(),
		Nonce:  newNonce(),
	}
	msg, err := rlp.EncodeToBytes(cmsg)
	if err != nil {
		return err
	}
	log.Debug("send chunk", "addr", label(ch.Address()), "self", label(p.ps.BaseAddr()))

	metrics.GetOrRegisterResettingTimer("pusher.send.chunk.rlp", nil).UpdateSince(rlpTimer)

	defer metrics.GetOrRegisterResettingTimer("pusher.send.chunk.pss", nil).UpdateSince(time.Now())
	return p.ps.Send(ch.Address()[:], pssChunkTopic, msg)
}

// needToSync checks if a chunk needs to be push-synced:
// * if not sent yet OR
// * if sent but more than retryInterval ago, so need resend OR
// * if self is closest node to chunk TODO: and not light node
//   in this case send receipt to self to trigger synced state on chunk
func (p *Pusher) needToSync(ch chunk.Chunk) bool {
	item, found := p.pushed[ch.Address().Hex()]
	// has been pushed already
	if found {
		// has synced already since subscribe called
		if item.synced {
			return false
		}
		item.lastSentAt = time.Now()
		// first time encountered
	} else {

		addr := ch.Address()
		hexaddr := addr.Hex()
		// remember item
		tag, _ := p.tags.Get(ch.TagID())
		now := time.Now()
		item = &pushedItem{
			tag:         tag,
			firstSentAt: now,
			lastSentAt:  now,
		}

		// increment SENT count on tag  if it exists
		if tag != nil {
			tag.Inc(chunk.StateSent)
			// opentracing for chunk roundtrip
			_, span := spancontext.StartSpan(tag.Context(), "chunk.sent")
			span.LogFields(olog.String("ref", hexaddr))
			span.SetTag("addr", hexaddr)

			item.span = span
		}

		// remember the item
		p.pushed[hexaddr] = item
		if p.ps.IsClosestTo(addr) {
			log.Debug("self is closest to ref: push receipt locally", "ref", hexaddr, "self", hex.EncodeToString(p.ps.BaseAddr()))
			item.shortcut = true
			go p.pushReceipt(addr)
			return false
		}
		log.Debug("self is not the closest to ref: send chunk to neighbourhood", "ref", hexaddr, "self", hex.EncodeToString(p.ps.BaseAddr()))
	}
	return true
}