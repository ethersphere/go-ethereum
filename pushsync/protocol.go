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

package pushsync

import (
	"crypto/rand"
	"io"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	pssChunkTopic   = "PUSHSYNC_CHUNKS"   // pss topic for chunks
	pssReceiptTopic = "PUSHSYNC_RECEIPTS" // pss topic for statement of custody receipts
)

// PubSub is a Postal Service interface needed to send/receive chunks and receipts for push syncing
type PubSub interface {
	Register(topic string, prox bool, handler func(msg []byte, p *p2p.Peer) error) func()
	Send(to []byte, topic string, msg []byte) error
	BaseAddr() []byte
	IsClosestTo([]byte) bool
}

// chunkMsg is the message construct to send chunks to their local neighbourhood
type chunkMsg struct {
	Addr   []byte // chunk address
	Data   []byte // chunk data
	Origin []byte // originator - need this for sending receipt back to origin
	Nonce  []byte // nonce to make multiple instances of send immune to deduplication cache
}

// receiptMsg is a statement of custody response to receiving a push-synced chunk
// it is currently a notification only (contains no proof) sent to the originator
// Nonce is there to make multiple responses immune to deduplication cache
type receiptMsg struct {
	Addr  []byte // chunk address
	Nonce []byte // nonce to make multiple instances of send immune to deduplication cache
}

func decodeChunkMsg(msg []byte) (*chunkMsg, error) {
	var chmsg chunkMsg
	err := rlp.DecodeBytes(msg, &chmsg)
	if err != nil {
		return nil, err
	}
	return &chmsg, nil
}

func decodeReceiptMsg(msg []byte) (*receiptMsg, error) {
	var rmsg receiptMsg
	err := rlp.DecodeBytes(msg, &rmsg)
	if err != nil {
		return nil, err
	}
	return &rmsg, nil
}

// newNonce creates a random nonce;
// even without POC it is important otherwise resending a chunk is deduplicated by pss
func newNonce() []byte {
	buf := make([]byte, 32)
	io.ReadFull(rand.Reader, buf)
	return buf
}

func label(b []byte) string {
	l := len(b)
	if l > 8 {
		l = 8
	}
	return hexutil.Encode(b[:l])
}
